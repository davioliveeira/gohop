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

func setupTestRetryClient(t *testing.T) *rabbitmq.Client {
	cfg := config.RabbitMQConfig{
		Host:            "localhost",
		Port:            5672,
		ManagementPort: 15672,
		Username:        "test_user",
		Password:        "test_pass",
		VHost:           "/", // Usar VHost padrão para testes
		UseTLS:          false,
	}

	client, err := rabbitmq.NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	return client
}

func TestSetupRetry_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestRetryClient(t)
	defer client.Close()

	queueName := "test_retry_" + time.Now().Format("20060102150405")

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

	// Verificar se componentes foram criados
	waitQueueName := queueName + ".wait"
	dlqName := queueName + ".dlq"

	waitExists, err := client.QueueExists(waitQueueName)
	require.NoError(t, err)
	assert.True(t, waitExists, "wait queue deve existir")

	dlqExists, err := client.QueueExists(dlqName)
	require.NoError(t, err)
	assert.True(t, dlqExists, "DLQ deve existir")

	// Limpar
	client.DeleteQueue(waitQueueName, false, false, false)
	client.DeleteQueue(dlqName, false, false, false)
}

func TestRecreateQueueWithDLX_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestRetryClient(t)
	defer client.Close()

	queueName := "test_dlx_" + time.Now().Format("20060102150405")
	waitExchangeName := queueName + ".wait.exchange"

	// Criar wait exchange primeiro (necessário para DLX)
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
	err = RecreateQueueWithDLX(client, queueName, "classic")
	require.NoError(t, err)

	// Verificar se fila existe
	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Limpar
	client.DeleteQueue(queueName, false, false, false)
}

func TestRetryComponentNames_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestRetryClient(t)
	defer client.Close()

	queueName := "test_component_names_" + time.Now().Format("20060102150405")

	opts := SetupOptions{
		QueueName:  queueName,
		QueueType:  "classic",
		MaxRetries: 3,
		RetryDelay: 5,
		DLQTTL:     604800000,
	}

	err := SetupRetry(client, opts)
	require.NoError(t, err)

	// Verificar nomes dos componentes (não usado, mas mantido para documentação)
	_ = queueName + ".wait"
	_ = queueName + ".wait.exchange"
	_ = queueName + ".retry"
	waitQueueName := queueName + ".wait"
	dlqName := queueName + ".dlq"

	// Verificar que componentes existem
	waitExists, _ := client.QueueExists(waitQueueName)
	assert.True(t, waitExists, "wait queue deve existir: %s", waitQueueName)

	dlqExists, _ := client.QueueExists(dlqName)
	assert.True(t, dlqExists, "DLQ deve existir: %s", dlqName)

	// Limpar
	client.DeleteQueue(waitQueueName, false, false, false)
	client.DeleteQueue(dlqName, false, false, false)
}
