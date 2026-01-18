package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/davioliveeira/gohop/internal/config"
)

// Client representa um cliente RabbitMQ
type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  config.RabbitMQConfig
}

// NewClient cria um novo cliente RabbitMQ
func NewClient(cfg config.RabbitMQConfig) (*Client, error) {
	// Construir URL de conexão
	protocol := "amqp"
	if cfg.UseTLS {
		protocol = "amqps"
	}
	
	// Garantir que vhost sempre comece com "/"
	vhost := cfg.VHost
	if vhost == "" {
		vhost = "/"
	} else if vhost[0] != '/' {
		vhost = "/" + vhost
	}
	
	url := fmt.Sprintf("%s://%s:%s@%s:%d%s",
		protocol,
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		vhost,
	)

	// Conectar
	conn, err := amqp.DialConfig(url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		// TODO: Adicionar configuração TLS adequada quando necessário
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar ao RabbitMQ: %w", err)
	}

	// Criar canal
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("erro ao criar canal: %w", err)
	}

	return &Client{
		conn:    conn,
		channel: channel,
		config:  cfg,
	}, nil
}

// Close fecha a conexão e o canal
func (c *Client) Close() error {
	var errs []error

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("erro ao fechar canal: %w", err))
		}
	}

	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("erro ao fechar conexão: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("erros ao fechar cliente: %v", errs)
	}

	return nil
}

// GetChannel retorna o canal AMQP
func (c *Client) GetChannel() *amqp.Channel {
	return c.channel
}

// GetConnection retorna a conexão AMQP
func (c *Client) GetConnection() *amqp.Connection {
	return c.conn
}

// IsConnected verifica se está conectado
func (c *Client) IsConnected() bool {
	return c.conn != nil && !c.conn.IsClosed()
}
