package rabbitmq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/davioliveeira/gohop/internal/config"
)

// ManagementClient é um cliente para a Management API do RabbitMQ
type ManagementClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewManagementClient cria um novo cliente para Management API
func NewManagementClient(cfg config.RabbitMQConfig) *ManagementClient {
	protocol := "http"
	if cfg.UseTLS {
		protocol = "https"
	}

	baseURL := fmt.Sprintf("%s://%s:%d/api", protocol, cfg.Host, cfg.ManagementPort)

	return &ManagementClient{
		baseURL:  baseURL,
		username: cfg.Username,
		password: cfg.Password,
		client:   &http.Client{},
	}
}

// QueueInfoManagement representa informações completas de uma fila via Management API
type QueueInfoManagement struct {
	Name              string `json:"name"`
	VHost             string `json:"vhost"`
	Durable           bool   `json:"durable"`
	AutoDelete        bool   `json:"auto_delete"`
	Exclusive         bool   `json:"exclusive"`
	Type              string `json:"type"` // classic, quorum, stream
	Messages          int    `json:"messages"`
	MessagesReady     int    `json:"messages_ready"`
	MessagesUnacked   int    `json:"messages_unacked"`
	Consumers         int    `json:"consumers"`
	ConsumerUtilisation float64 `json:"consumer_utilisation"`
	Memory            int64  `json:"memory"`
	MessageStats      struct {
		Publish    int `json:"publish"`
		PublishIn  int `json:"publish_in"`
		PublishOut int `json:"publish_out"`
		Deliver    int `json:"deliver"`
		DeliverGet int `json:"deliver_get"`
		Get        int `json:"get"`
		Ack        int `json:"ack"`
	} `json:"message_stats"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ListQueues retorna lista de todas as filas via Management API
func (m *ManagementClient) ListQueues() ([]QueueInfoManagement, error) {
	endpoint := fmt.Sprintf("%s/queues", m.baseURL)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.SetBasicAuth(m.username, m.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro na API: status %d, body: %s", resp.StatusCode, string(body))
	}

	var queues []QueueInfoManagement
	if err := json.NewDecoder(resp.Body).Decode(&queues); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return queues, nil
}

// GetQueue retorna informações detalhadas de uma fila específica
func (m *ManagementClient) GetQueue(vhost, queueName string) (*QueueInfoManagement, error) {
	// Escapar vhost para URL (geralmente "/" fica como "%2F")
	// Se vhost for "/", usar "%2F" diretamente
	vhostForURL := strings.TrimPrefix(vhost, "/")
	if vhostForURL == "" {
		vhostForURL = "%2F"
	} else {
		vhostForURL = url.PathEscape(vhostForURL)
	}
	queueEscaped := url.PathEscape(queueName)
	
	endpoint := fmt.Sprintf("%s/queues/%s/%s", m.baseURL, vhostForURL, queueEscaped)
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.SetBasicAuth(m.username, m.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("fila não encontrada: %s/%s", vhost, queueName)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro na API: status %d, body: %s", resp.StatusCode, string(body))
	}

	var queue QueueInfoManagement
	if err := json.NewDecoder(resp.Body).Decode(&queue); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &queue, nil
}

// DeleteQueueViaAPI deleta uma fila via Management API
func (m *ManagementClient) DeleteQueueViaAPI(vhost, queueName string) error {
	// Escapar vhost para URL (geralmente "/" fica como "%2F")
	vhostForURL := strings.TrimPrefix(vhost, "/")
	if vhostForURL == "" {
		vhostForURL = "%2F"
	} else {
		vhostForURL = url.PathEscape(vhostForURL)
	}
	queueEscaped := url.PathEscape(queueName)
	
	endpoint := fmt.Sprintf("%s/queues/%s/%s", m.baseURL, vhostForURL, queueEscaped)
	
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.SetBasicAuth(m.username, m.password)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro ao deletar fila: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
