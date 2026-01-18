package retry

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/davioliveeira/rabbit/internal/config"
	"github.com/davioliveeira/rabbit/internal/rabbitmq"
)

// SetupOptions são opções para configurar o sistema de retry
type SetupOptions struct {
	QueueName  string // Nome da fila principal
	MaxRetries int    // Número máximo de tentativas
	RetryDelay int    // Delay entre tentativas em segundos (será convertido para TTL em ms)
	DLQTTL     int    // TTL de mensagens na DLQ em milissegundos
	Force      bool   // Recriar mesmo se já existir
}

// RetrySystemInfo contém informações sobre o sistema de retry configurado
type RetrySystemInfo struct {
	QueueName       string
	MainQueue       bool
	WaitQueue       bool
	WaitExchange    bool
	RetryExchange   bool
	DLQ             bool
	MaxRetries      int
	RetryDelay      int
	DLQTTL          int
	MainQueueMsgs   int
	WaitQueueMsgs   int
	DLQMsgs         int
}

// SetupRetry configura o sistema completo de retry para uma fila
func SetupRetry(client *rabbitmq.Client, opts SetupOptions) error {
	queueName := opts.QueueName
	
	// Nomes dos componentes do sistema de retry
	waitQueueName := fmt.Sprintf("%s.wait", queueName)
	waitExchangeName := fmt.Sprintf("%s.wait.exchange", queueName)
	retryExchangeName := fmt.Sprintf("%s.retry", queueName)
	dlqName := fmt.Sprintf("%s.dlq", queueName)

	channel := client.GetChannel()

	// 1. Criar Wait Exchange (recebe mensagens rejeitadas)
	if err := channel.ExchangeDeclare(
		waitExchangeName,
		"fanout", // tipo fanout
		true,     // durable
		false,    // auto-delete
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	); err != nil {
		return fmt.Errorf("erro ao criar wait exchange: %w", err)
	}

	// 2. Criar Wait Queue (delays mensagem antes de retry)
	// TTL é em milissegundos
	retryDelayMs := opts.RetryDelay * 1000
	
	waitQueueArgs := amqp.Table{
		"x-message-ttl":           int32(retryDelayMs),
		"x-dead-letter-exchange":  retryExchangeName,
		"x-queue-type":            "classic",
	}

	if _, err := channel.QueueDeclare(
		waitQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		waitQueueArgs,
	); err != nil {
		return fmt.Errorf("erro ao criar wait queue: %w", err)
	}

	// 3. Bind Wait Queue to Wait Exchange
	if err := channel.QueueBind(
		waitQueueName,
		"", // routing key (fanout não usa)
		waitExchangeName,
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("erro ao fazer bind wait queue: %w", err)
	}

	// 4. Criar Retry Exchange (rota para main queue ou DLQ baseado em retry count)
	// Tipo headers para permitir roteamento baseado em headers
	if err := channel.ExchangeDeclare(
		retryExchangeName,
		"headers", // tipo headers
		true,      // durable
		false,     // auto-delete
		false,     // internal
		false,     // no-wait
		nil,       // arguments
	); err != nil {
		return fmt.Errorf("erro ao criar retry exchange: %w", err)
	}

	// 5. Criar DLQ (destino final após max retries)
	dlqArgs := amqp.Table{
		"x-message-ttl": int32(opts.DLQTTL),
		"x-queue-type":  "classic",
	}

	if _, err := channel.QueueDeclare(
		dlqName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		dlqArgs,
	); err != nil {
		return fmt.Errorf("erro ao criar DLQ: %w", err)
	}

	// 6. Bind DLQ to Retry Exchange
	// Nota: O roteamento baseado em retry count é feito pelo consumer
	// Este bind aceita todas as mensagens que chegam no retry exchange
	bindingArgs := amqp.Table{
		"x-match": "any", // Aceita qualquer mensagem
	}
	if err := channel.QueueBind(
		dlqName,
		"", // routing key (headers não usa routing key tradicional)
		retryExchangeName,
		false,        // no-wait
		bindingArgs,  // arguments para headers
	); err != nil {
		return fmt.Errorf("erro ao fazer bind DLQ: %w", err)
	}

	return nil
}

// RecreateQueueWithDLX recria a fila principal com DLX apontando para wait exchange
func RecreateQueueWithDLX(client *rabbitmq.Client, queueName string, queueType string) error {
	channel := client.GetChannel()
	waitExchangeName := fmt.Sprintf("%s.wait.exchange", queueName)

	// Argumentos da fila com DLX
	args := amqp.Table{
		"x-dead-letter-exchange": waitExchangeName,
	}

	// Tipo de fila
	if queueType == "quorum" {
		args["x-queue-type"] = "quorum"
	} else {
		args["x-queue-type"] = "classic"
	}

	_, err := channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		args,
	)

	if err != nil {
		return fmt.Errorf("erro ao recriar fila com DLX: %w", err)
	}

	return nil
}

// GetRetrySystemInfo obtém informações sobre o sistema de retry
func GetRetrySystemInfo(mgmtClient *rabbitmq.ManagementClient, cfg config.RabbitMQConfig, queueName string) (*RetrySystemInfo, error) {
	info := &RetrySystemInfo{
		QueueName: queueName,
	}

	vhost := cfg.VHost
	if vhost == "" {
		vhost = "/"
	} else if vhost[0] != '/' {
		vhost = "/" + vhost
	}

	// Verificar componentes
	mainQueue, err := mgmtClient.GetQueue(vhost, queueName)
	if err == nil {
		info.MainQueue = true
		info.MainQueueMsgs = mainQueue.MessagesReady
	}

	waitQueueName := fmt.Sprintf("%s.wait", queueName)
	waitQueue, err := mgmtClient.GetQueue(vhost, waitQueueName)
	if err == nil {
		info.WaitQueue = true
		info.WaitQueueMsgs = waitQueue.MessagesReady
	}

	dlqName := fmt.Sprintf("%s.dlq", queueName)
	dlq, err := mgmtClient.GetQueue(vhost, dlqName)
	if err == nil {
		info.DLQ = true
		info.DLQMsgs = dlq.MessagesReady
	}

	// Verificar exchanges (Management API tem endpoint para exchanges também)
	// Por enquanto, assumimos que se as filas existem, os exchanges também existem
	info.WaitExchange = info.WaitQueue
	info.RetryExchange = info.DLQ || info.WaitQueue

	// TODO: Obter MaxRetries, RetryDelay, DLQTTL dos arguments das filas
	// Por enquanto, usamos valores padrão
	info.MaxRetries = 3
	info.RetryDelay = 5
	info.DLQTTL = 604800000

	return info, nil
}
