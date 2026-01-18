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

func setupTestManagementClient(t *testing.T) *ManagementClient {
	cfg := config.RabbitMQConfig{
		Host:            "localhost",
		Port:            5672,
		ManagementPort: 15672,
		Username:        "test_user",
		Password:        "test_pass",
		VHost:           "/", // Usar VHost padrão para testes
		UseTLS:          false,
	}

	return NewManagementClient(cfg)
}

func setupTestQueueForManagement(t *testing.T, client *Client, queueName string) {
	opts := CreateQueueOptions{
		Name:    queueName,
		Type:    "classic",
		Durable: true,
	}

	err := client.CreateQueue(opts)
	require.NoError(t, err)
}

func TestManagementClient_ListQueues_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	mgmtClient := setupTestManagementClient(t)

	// Criar fila via AMQP primeiro
	rabbitClient := setupTestClient(t)
	defer rabbitClient.Close()

	queueName := "test_mgmt_list_" + time.Now().Format("20060102150405")
	setupTestQueueForManagement(t, rabbitClient, queueName)
	defer rabbitClient.DeleteQueue(queueName, false, false, false)

	// Listar filas via Management API
	queues, err := mgmtClient.ListQueues()
	require.NoError(t, err)
	assert.NotNil(t, queues)

	// Verificar se nossa fila está na lista
	found := false
	for _, queue := range queues {
		if queue.Name == queueName {
			found = true
			assert.Equal(t, "classic", queue.Type)
			break
		}
	}
	assert.True(t, found, "fila deve estar na lista")
}

func TestManagementClient_GetQueue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	mgmtClient := setupTestManagementClient(t)

	// Criar fila via AMQP primeiro
	rabbitClient := setupTestClient(t)
	defer rabbitClient.Close()

	queueName := "test_mgmt_get_" + time.Now().Format("20060102150405")
	setupTestQueueForManagement(t, rabbitClient, queueName)
	defer rabbitClient.DeleteQueue(queueName, false, false, false)

	vhost := "/" // Usar VHost padrão

	// Obter informações via Management API
	queue, err := mgmtClient.GetQueue(vhost, queueName)
	require.NoError(t, err)
	assert.NotNil(t, queue)
	assert.Equal(t, queueName, queue.Name)
	assert.Equal(t, "classic", queue.Type)
	assert.Equal(t, vhost, queue.VHost)
	assert.Equal(t, 0, queue.MessagesReady)
	assert.Equal(t, 0, queue.Consumers)
}

func TestManagementClient_GetQueue_NotFound_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	mgmtClient := setupTestManagementClient(t)

	// Tentar obter fila que não existe
	_, err := mgmtClient.GetQueue("/", "non_existent_queue_xyz123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "não encontrada")
}

func TestManagementClient_DeleteQueueViaAPI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração em modo short")
	}

	mgmtClient := setupTestManagementClient(t)

	// Criar fila via AMQP primeiro
	rabbitClient := setupTestClient(t)
	defer rabbitClient.Close()

	queueName := "test_mgmt_delete_" + time.Now().Format("20060102150405")
	setupTestQueueForManagement(t, rabbitClient, queueName)

	// Aguardar um pouco para garantir que a fila foi criada
	time.Sleep(100 * time.Millisecond)

	// Verificar que existe
	exists, err := rabbitClient.QueueExists(queueName)
	require.NoError(t, err)
	assert.True(t, exists, "fila deve existir antes de deletar")

	// Deletar via Management API
	err = mgmtClient.DeleteQueueViaAPI("/", queueName)
	require.NoError(t, err)

	// Verificar que não existe mais
	exists, err = rabbitClient.QueueExists(queueName)
	require.NoError(t, err)
	assert.False(t, exists)
}
