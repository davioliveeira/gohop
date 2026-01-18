package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueInfo contém informações sobre uma fila
type QueueInfo struct {
	Name       string
	Messages   int    // Mensagens prontas
	Consumers  int    // Número de consumers
	Unacked    int    // Mensagens não confirmadas
	Type       string // classic ou quorum
	Durable    bool
	AutoDelete bool
	VHost      string
}

// CreateQueueOptions são opções para criar uma fila
type CreateQueueOptions struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Type       string // classic ou quorum
	Arguments  map[string]interface{}
}

// CreateQueue cria uma nova fila
func (c *Client) CreateQueue(opts CreateQueueOptions) error {
	args := make(amqp.Table)
	if opts.Arguments != nil {
		for k, v := range opts.Arguments {
			args[k] = v
		}
	}

	// Definir tipo de fila (quorum ou classic)
	if opts.Type == "quorum" {
		args["x-queue-type"] = "quorum"
	} else {
		args["x-queue-type"] = "classic"
	}

	_, err := c.channel.QueueDeclare(
		opts.Name,
		opts.Durable,
		opts.AutoDelete,
		opts.Exclusive,
		opts.NoWait,
		args,
	)

	if err != nil {
		return fmt.Errorf("erro ao criar fila %s: %w", opts.Name, err)
	}

	return nil
}

// DeleteQueue deleta uma fila
func (c *Client) DeleteQueue(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	messages, err := c.channel.QueueDelete(
		name,
		ifUnused,
		ifEmpty,
		noWait,
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao deletar fila %s: %w", name, err)
	}

	return messages, nil
}

// ListQueues retorna lista de todas as filas (requer Management API - será implementado depois)
// Por enquanto, retorna apenas filas conhecidas via AMQP
func (c *Client) ListQueues() ([]QueueInfo, error) {
	// Nota: AMQP não tem comando nativo para listar filas
	// Precisamos usar Management API HTTP
	// Esta função será implementada quando adicionarmos suporte à Management API
	return []QueueInfo{}, fmt.Errorf("listagem de filas requer Management API (não implementado ainda)")
}

// GetQueueInfo obtém informações sobre uma fila específica
func (c *Client) GetQueueInfo(name string) (*QueueInfo, error) {
	// Tentar declarar a fila passiva (não cria, apenas obtém info)
	queue, err := c.channel.QueueDeclarePassive(
		name,
		false, // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações da fila %s: %w", name, err)
	}

	// Obter informações adicionais via Management API seria ideal
	// Por enquanto, retornamos o que conseguimos via AMQP
	info := &QueueInfo{
		Name:       queue.Name,
		Messages:   queue.Messages,
		Consumers:  queue.Consumers,
		Durable:    false, // Não conseguimos saber via declarar passivo
		AutoDelete: false,
		Type:       "classic", // Assumimos classic por padrão
	}

	return info, nil
}

// PurgeQueue remove todas as mensagens de uma fila
func (c *Client) PurgeQueue(name string, noWait bool) (int, error) {
	messages, err := c.channel.QueuePurge(name, noWait)
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar fila %s: %w", name, err)
	}
	return messages, nil
}

// QueueExists verifica se uma fila existe
func (c *Client) QueueExists(name string) (bool, error) {
	_, err := c.channel.QueueDeclarePassive(
		name,
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		// Se a fila não existe, retorna erro específico
		if err == amqp.ErrClosed {
			return false, fmt.Errorf("conexão fechada")
		}
		// Assumimos que é erro de fila não encontrada
		return false, nil
	}

	return true, nil
}

// SavedMessage representa uma mensagem salva para migração
type SavedMessage struct {
	Body        []byte
	ContentType string
	Headers     map[string]interface{}
	Priority    uint8
	MessageId   string
	Timestamp   int64
}

// DrainQueue consome todas as mensagens de uma fila e retorna elas
// As mensagens são ACK'ed após serem lidas (removidas da fila)
func (c *Client) DrainQueue(queueName string, onProgress func(current, total int)) ([]SavedMessage, error) {
	// Primeiro, verificar quantas mensagens tem (estimativa para progresso)
	queue, err := c.channel.QueueDeclarePassive(queueName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar fila: %w", err)
	}

	estimatedTotal := queue.Messages
	if estimatedTotal == 0 {
		// Tentar ler mesmo assim - a contagem pode estar desatualizada
		estimatedTotal = 1
	}

	// Consumir mensagens uma a uma até não haver mais (basic.get ao invés de basic.consume)
	var messages []SavedMessage
	count := 0
	
	for {
		msg, ok, err := c.channel.Get(queueName, false) // autoAck = false
		if err != nil {
			return messages, fmt.Errorf("erro ao obter mensagem %d: %w", count+1, err)
		}
		if !ok {
			break // Não há mais mensagens
		}

		count++

		// Salvar mensagem
		savedMsg := SavedMessage{
			Body:        msg.Body,
			ContentType: msg.ContentType,
			Priority:    msg.Priority,
			MessageId:   msg.MessageId,
			Timestamp:   msg.Timestamp.Unix(),
		}

		// Copiar headers se existirem
		if msg.Headers != nil {
			savedMsg.Headers = make(map[string]interface{})
			for k, v := range msg.Headers {
				savedMsg.Headers[k] = v
			}
		}

		messages = append(messages, savedMsg)

		// ACK a mensagem (remove da fila)
		if err := msg.Ack(false); err != nil {
			return messages, fmt.Errorf("erro ao confirmar mensagem %d: %w", count, err)
		}

		// Callback de progresso (usar estimativa, mas pode exceder)
		if onProgress != nil {
			total := estimatedTotal
			if count > total {
				total = count
			}
			onProgress(count, total)
		}
	}

	return messages, nil
}

// PublishMessages republica mensagens em uma fila com confirmação síncrona
func (c *Client) PublishMessages(queueName string, messages []SavedMessage, onProgress func(current, total int)) error {
	total := len(messages)
	if total == 0 {
		return nil
	}

	// Habilitar modo de confirmação para garantir que mensagens sejam persistidas
	if err := c.channel.Confirm(false); err != nil {
		return fmt.Errorf("erro ao habilitar modo de confirmação: %w", err)
	}

	// Canal para receber confirmações
	confirms := c.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	for i, msg := range messages {
		// Construir headers AMQP
		var headers amqp.Table
		if msg.Headers != nil {
			headers = make(amqp.Table)
			for k, v := range msg.Headers {
				headers[k] = v
			}
		}

		// Garantir ContentType
		contentType := msg.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Publicar diretamente na fila (exchange vazio, routing key = nome da fila)
		err := c.channel.Publish(
			"",        // exchange (default)
			queueName, // routing key = nome da fila
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:  contentType,
				Body:         msg.Body,
				Headers:      headers,
				Priority:     msg.Priority,
				MessageId:    msg.MessageId,
				DeliveryMode: amqp.Persistent, // Mensagens persistentes
			},
		)
		if err != nil {
			return fmt.Errorf("erro ao publicar mensagem %d: %w", i+1, err)
		}

		// Aguardar confirmação do broker (síncrono, uma de cada vez)
		select {
		case confirmed := <-confirms:
			if !confirmed.Ack {
				return fmt.Errorf("mensagem %d não foi confirmada pelo broker (NACK)", i+1)
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout aguardando confirmação da mensagem %d", i+1)
		}

		// Callback de progresso
		if onProgress != nil {
			onProgress(i+1, total)
		}
	}

	return nil
}
