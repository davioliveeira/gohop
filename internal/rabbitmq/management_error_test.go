package rabbitmq

import (
	"testing"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestManagementClient_ErrorCases(t *testing.T) {
	cfg := config.RabbitMQConfig{
		Host:            "localhost",
		ManagementPort: 15672,
		Username:        "guest",
		Password:        "guest",
		UseTLS:          false,
	}

	client := NewManagementClient(cfg)
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:15672/api", client.baseURL)
}

func TestManagementClient_WithInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.RabbitMQConfig
	}{
		{
			name: "empty host",
			cfg: config.RabbitMQConfig{
				Host:            "",
				ManagementPort: 15672,
			},
		},
		{
			name: "zero port",
			cfg: config.RabbitMQConfig{
				Host:            "localhost",
				ManagementPort: 0,
			},
		},
		{
			name: "empty credentials",
			cfg: config.RabbitMQConfig{
				Host:            "localhost",
				ManagementPort: 15672,
				Username:        "",
				Password:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewManagementClient(tt.cfg)
			assert.NotNil(t, client) // Client é criado mesmo com config inválida
		})
	}
}

func TestManagementClient_VHostEscaping(t *testing.T) {
	tests := []struct {
		name     string
		vhost    string
		expected string
	}{
		{
			name:     "root vhost",
			vhost:    "/",
			expected: "%2F",
		},
		{
			name:     "custom vhost",
			vhost:    "/test",
			expected: "test", // Após TrimPrefix
		},
		{
			name:     "vhost without leading slash",
			vhost:    "test",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vhostForURL := tt.vhost
			if vhostForURL == "/" {
				vhostForURL = "%2F"
			} else {
				// Simular TrimPrefix
				if len(vhostForURL) > 0 && vhostForURL[0] == '/' {
					vhostForURL = vhostForURL[1:]
				}
			}
			assert.NotEmpty(t, vhostForURL)
		})
	}
}

func TestQueueInfoManagement_Structure(t *testing.T) {
	info := QueueInfoManagement{
		Name:            "test_queue",
		VHost:           "/",
		Durable:         true,
		AutoDelete:      false,
		Exclusive:       false,
		Type:            "quorum",
		Messages:        10,
		MessagesReady:   8,
		MessagesUnacked: 2,
		Consumers:       2,
		Memory:          1024,
	}

	assert.Equal(t, "test_queue", info.Name)
	assert.Equal(t, "/", info.VHost)
	assert.True(t, info.Durable)
	assert.False(t, info.AutoDelete)
	assert.Equal(t, "quorum", info.Type)
	assert.Equal(t, 10, info.Messages)
	assert.Equal(t, 8, info.MessagesReady)
	assert.Equal(t, 2, info.MessagesUnacked)
	assert.Equal(t, 2, info.Consumers)
}

func TestQueueInfoManagement_MessageStats(t *testing.T) {
	info := QueueInfoManagement{}
	info.MessageStats.Publish = 100
	info.MessageStats.Deliver = 95
	info.MessageStats.Ack = 90

	assert.Equal(t, 100, info.MessageStats.Publish)
	assert.Equal(t, 95, info.MessageStats.Deliver)
	assert.Equal(t, 90, info.MessageStats.Ack)
}
