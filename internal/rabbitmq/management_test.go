package rabbitmq

import (
	"testing"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewManagementClient(t *testing.T) {
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
	assert.Equal(t, "guest", client.username)
	assert.Equal(t, "guest", client.password)
}

func TestNewManagementClient_WithTLS(t *testing.T) {
	cfg := config.RabbitMQConfig{
		Host:            "localhost",
		ManagementPort: 15672,
		Username:        "guest",
		Password:        "guest",
		UseTLS:          true,
	}

	client := NewManagementClient(cfg)
	assert.NotNil(t, client)
	assert.Equal(t, "https://localhost:15672/api", client.baseURL)
}

func TestManagementClient_BaseURLFormat(t *testing.T) {
	tests := []struct {
		name           string
		cfg            config.RabbitMQConfig
		expectedPrefix string
	}{
		{
			name: "HTTP base URL",
			cfg: config.RabbitMQConfig{
				Host:            "localhost",
				ManagementPort: 15672,
				UseTLS:          false,
			},
			expectedPrefix: "http://",
		},
		{
			name: "HTTPS base URL",
			cfg: config.RabbitMQConfig{
				Host:            "localhost",
				ManagementPort: 15672,
				UseTLS:          true,
			},
			expectedPrefix: "https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewManagementClient(tt.cfg)
			assert.Contains(t, client.baseURL, tt.expectedPrefix)
			assert.Contains(t, client.baseURL, "/api")
		})
	}
}
