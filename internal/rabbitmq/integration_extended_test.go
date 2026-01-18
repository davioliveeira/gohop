//go:build integration
// +build integration

package rabbitmq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateQueue_WithArguments_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_args_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
		Arguments: map[string]interface{}{
			"x-max-length":      1000,
			"x-message-ttl":     3600000,
			"x-dead-letter-exchange": "dlx",
		},
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Limpar
	client.DeleteQueue(queueName, false, false, false)
}

func TestCreateQueue_AutoDelete_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_autodel_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:       queueName,
		Type:       "classic",
		Durable:    false,
		AutoDelete: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Limpar
	client.DeleteQueue(queueName, false, false, false)
}

func TestCreateQueue_Exclusive_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_excl_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:      queueName,
		Type:      "classic",
		Durable:   false,
		Exclusive: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	exists, err := client.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Limpar
	client.DeleteQueue(queueName, false, false, false)
}

func TestDeleteQueue_WithFlags_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_del_flags_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)

	// Deletar com flags
	messages, err := client.DeleteQueue(queueName, false, false, false)
	require.NoError(t, err)
	assert.Equal(t, 0, messages)
}

func TestGetQueueInfo_WithMessages_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_info_msgs_" + time.Now().Format("20060102150405")

	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)
	defer client.DeleteQueue(queueName, false, false, false)

	// Obter informações (fila vazia)
	info, err := client.GetQueueInfo(queueName)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, queueName, info.Name)
	assert.Equal(t, 0, info.Messages)
}

func TestPurgeQueue_EmptyQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	queueName := "test_queue_purge_empty_" + time.Now().Format("20060102150405")

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

func TestClient_IsConnected_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	assert.True(t, client.IsConnected())
	
	// Fechar e verificar
	client.Close()
	assert.False(t, client.IsConnected())
}

func TestClient_GetChannel_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	channel := client.GetChannel()
	assert.NotNil(t, channel)
}

func TestClient_GetConnection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)
	defer client.Close()

	conn := client.GetConnection()
	assert.NotNil(t, conn)
	assert.False(t, conn.IsClosed())
}

func TestClient_Close_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	client := setupTestClient(t)

	err := client.Close()
	require.NoError(t, err)
	assert.False(t, client.IsConnected())
}
