//go:build integration
// +build integration

package retry

import (
	"testing"
	"time"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/davioliveeira/gohop/internal/rabbitmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRetrySystemInfo_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestRetryClient(t)
	defer client.Close()

	mgmtClient := rabbitmq.NewManagementClient(config.RabbitMQConfig{
		Host:            "localhost",
		ManagementPort: 15672,
		Username:        "test_user",
		Password:        "test_pass",
		VHost:           "/",
		UseTLS:          false,
	})

	queueName := "test_retry_info_" + time.Now().Format("20060102150405")

	opts := SetupOptions{
		QueueName:  queueName,
		QueueType:  "classic",
		MaxRetries: 3,
		RetryDelay: 5,
		DLQTTL:     604800000,
		Force:      false,
	}

	// Setup retry system
	err := SetupRetry(client, opts)
	require.NoError(t, err)

	// Recriar fila principal com DLX
	err = RecreateQueueWithDLX(client, queueName, "classic")
	require.NoError(t, err)

	// Obter informações do sistema de retry
	cfg := config.RabbitMQConfig{
		Host:     "localhost",
		VHost:    "/",
		Username: "test_user",
		Password: "test_pass",
	}

	info, err := GetRetrySystemInfo(mgmtClient, cfg, queueName)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, queueName, info.QueueName)
	assert.True(t, info.MainQueue)
	assert.True(t, info.WaitQueue)
	assert.True(t, info.DLQ)

	// Limpar
	client.DeleteQueue(queueName, false, false, false)
	client.DeleteQueue(queueName+".wait", false, false, false)
	client.DeleteQueue(queueName+".dlq", false, false, false)
}

func TestSetupRetry_WithDifferentOptions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestRetryClient(t)
	defer client.Close()

	tests := []struct {
		name      string
		maxRetries int
		retryDelay int
		dlqTTL     int
	}{
		{
			name:       "default options",
			maxRetries: 3,
			retryDelay: 5,
			dlqTTL:     604800000,
		},
		{
			name:       "high retries",
			maxRetries: 10,
			retryDelay: 10,
			dlqTTL:     86400000,
		},
		{
			name:       "short delay",
			maxRetries: 5,
			retryDelay: 1,
			dlqTTL:     3600000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queueName := "test_retry_" + tt.name + "_" + time.Now().Format("20060102150405")

			opts := SetupOptions{
				QueueName:  queueName,
				QueueType:  "classic",
				MaxRetries: tt.maxRetries,
				RetryDelay: tt.retryDelay,
				DLQTTL:     tt.dlqTTL,
				Force:      false,
			}

			err := SetupRetry(client, opts)
			require.NoError(t, err)

			// Verificar componentes
			waitExists, _ := client.QueueExists(queueName + ".wait")
			dlqExists, _ := client.QueueExists(queueName + ".dlq")

			assert.True(t, waitExists)
			assert.True(t, dlqExists)

			// Limpar
			client.DeleteQueue(queueName+".wait", false, false, false)
			client.DeleteQueue(queueName+".dlq", false, false, false)
		})
	}
}

func TestRecreateQueueWithDLX_DifferentTypes_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestRetryClient(t)
	defer client.Close()

	tests := []struct {
		name      string
		queueType string
	}{
		{"classic", "classic"},
		{"quorum", "quorum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queueName := "test_dlx_" + tt.queueType + "_" + time.Now().Format("20060102150405")
			waitExchangeName := queueName + ".wait.exchange"

			// Criar wait exchange primeiro
			channel := client.GetChannel()
			err := channel.ExchangeDeclare(
				waitExchangeName,
				"fanout",
				true,
				false,
				false,
				false,
				nil,
			)
			require.NoError(t, err)

			// Recriar fila com DLX
			err = RecreateQueueWithDLX(client, queueName, tt.queueType)
			require.NoError(t, err)

			// Verificar que fila existe
			exists, err := client.QueueExists(queueName)
			require.NoError(t, err)
			assert.True(t, exists)

			// Limpar
			client.DeleteQueue(queueName, false, false, false)
		})
	}
}
