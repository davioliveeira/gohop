package ui

import (
	"testing"
	"time"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewDashboard(t *testing.T) {
	cfg := &config.Config{
		RabbitMQ: config.RabbitMQConfig{
			Host: "localhost",
		},
	}

	model := NewDashboard("test_queue", cfg, 2*time.Second)

	assert.Equal(t, "test_queue", model.queueName)
	assert.Equal(t, cfg, model.cfg)
	assert.Equal(t, 2*time.Second, model.interval)
	assert.True(t, model.loading)
	assert.NotNil(t, model.spinner)
}

func TestDashboardModel_InitialState(t *testing.T) {
	cfg := &config.Config{
		RabbitMQ: config.RabbitMQConfig{
			Host: "localhost",
		},
	}

	model := NewDashboard("test_queue", cfg, 1*time.Second)

	assert.Equal(t, "test_queue", model.queueName)
	assert.NotNil(t, model.cfg)
	assert.Equal(t, 1*time.Second, model.interval)
	assert.True(t, model.loading)
}

func TestDashboardStyles(t *testing.T) {
	// Testar que os estilos são inicializados corretamente
	assert.NotNil(t, titleStyle)
	assert.NotNil(t, boxStyle)
	assert.NotNil(t, labelStyle)
	assert.NotNil(t, valueStyle)
	assert.NotNil(t, warningStyle)
	assert.NotNil(t, helpStyle)
}

func TestQueueData(t *testing.T) {
	data := QueueData{
		Name:           "test_queue",
		MessagesReady:  10,
		MessagesUnacked: 5,
		TotalMessages:  15,
		Consumers:      2,
		Type:          "quorum",
		VHost:         "/",
	}

	assert.Equal(t, "test_queue", data.Name)
	assert.Equal(t, 10, data.MessagesReady)
	assert.Equal(t, 5, data.MessagesUnacked)
	assert.Equal(t, 15, data.TotalMessages)
	assert.Equal(t, 2, data.Consumers)
}

func TestRetryData(t *testing.T) {
	data := RetryData{
		MainQueueExists: true,
		WaitQueueExists: true,
		DLQExists:       true,
		MainQueueMsgs:   10,
		WaitQueueMsgs:   3,
		DLQMsgs:         2,
		MaxRetries:      3,
		RetryDelay:      5,
	}

	assert.True(t, data.MainQueueExists)
	assert.True(t, data.WaitQueueExists)
	assert.True(t, data.DLQExists)
	assert.Equal(t, 10, data.MainQueueMsgs)
	assert.Equal(t, 3, data.WaitQueueMsgs)
	assert.Equal(t, 2, data.DLQMsgs)
}
