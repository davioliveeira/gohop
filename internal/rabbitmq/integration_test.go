//go:build integration
// +build integration

package rabbitmq

import (
	"testing"
	"time"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestClient cria um cliente de teste conectado ao RabbitMQ em Docker
func setupTestClient(t *testing.T) *Client {
	cfg := config.RabbitMQConfig{
		Host:            "localhost",
		Port:            5672,
		ManagementPort: 15672,
		Username:        "test_user",
		Password:        "test_pass",
		VHost:           "/", // Usar VHost padrão para testes
		UseTLS:          false,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err, "deve conectar ao RabbitMQ")
	require.NotNil(t, client)

	return client
}

func TestNewClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	cfg := config.RabbitMQConfig{
		Host:            "localhost",
		Port:            5672,
		ManagementPort: 15672,
		Username:        "test_user",
		Password:        "test_pass",
		VHost:           "/", // Usar VHost padrão para testes
		UseTLS:          false,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.True(t, client.IsConnected())
}

func TestCreateQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_create_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:       queueName,
		Type:       "classic",
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Arguments:  nil,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err, "deve criar fila com sucesso")

	// Verificar se fila existe
	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists, "fila deve existir após criação")
}

func TestCreateQueue_Quorum_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_quorum_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:       queueName,
		Type:       "quorum",
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Arguments:  nil,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestQueueExists_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	// Criar fila primeiro
	client := setupTestClient(t)
	queueName := "test_queue_exists_" + time.Now().Format("20060102150405")
	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	// Verificar que existe
	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Deletar fila
	client.DeleteQueue(queueName, false, false, false)

	// Verificar que não existe mais (usar novo cliente para evitar problema de canal fechado)
	client2 := setupTestClient(t)
	defer client2.Close()
	
	exists, err = client2.QueueExists(queueName)
	require.NoError(t, err)
	assert.False(t, exists)

	client.Close()
}

func TestDeleteQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	// Criar fila primeiro
	queueName := "test_queue_delete_" + time.Now().Format("20060102150405")
	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	// Verificar que existe
	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Deletar fila
	messages, err := client.DeleteQueue(queueName, false, false, false)
	require.NoError(t, err)
	assert.Equal(t, 0, messages) // Fila vazia

	// Verificar que não existe mais
	exists, err = client.QueueExists(queueName)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestGetQueueInfo_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	// Criar fila primeiro
	queueName := "test_queue_info_" + time.Now().Format("20060102150405")
	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)
	defer client.DeleteQueue(queueName, false, false, false)

	// Obter informações
	info, err := client.GetQueueInfo(queueName)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, queueName, info.Name)
	assert.Equal(t, 0, info.Messages)
	assert.Equal(t, 0, info.Consumers)
}

func TestPurgeQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	// Criar fila
	queueName := "test_queue_purge_" + time.Now().Format("20060102150405")
	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)
	defer client.DeleteQueue(queueName, false, false, false)

	// Purge fila vazia
	messages, err := client.PurgeQueue(queueName, false)
	require.NoError(t, err)
	assert.Equal(t, 0, messages)
}
