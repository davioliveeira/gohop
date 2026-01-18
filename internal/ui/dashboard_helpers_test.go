package ui

import (
	"testing"
	"time"

	"github.com/davioliveeira/gohop/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestQueueData_Structure(t *testing.T) {
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
	assert.Equal(t, "quorum", data.Type)
	assert.Equal(t, "/", data.VHost)
}

func TestQueueData_EmptyQueue(t *testing.T) {
	data := QueueData{
		Name:           "empty_queue",
		MessagesReady:  0,
		MessagesUnacked: 0,
		TotalMessages:  0,
		Consumers:      0,
		Type:          "classic",
		VHost:         "/",
	}

	assert.Equal(t, "empty_queue", data.Name)
	assert.Equal(t, 0, data.MessagesReady)
	assert.Equal(t, 0, data.MessagesUnacked)
	assert.Equal(t, 0, data.TotalMessages)
	assert.Equal(t, 0, data.Consumers)
}

func TestRetryData_Structure(t *testing.T) {
	data := RetryData{
		MainQueueExists: true,
		WaitQueueExists: true,
		DLQExists:       true,
		MainQueueMsgs:  10,
		WaitQueueMsgs:  3,
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
	assert.Equal(t, 3, data.MaxRetries)
	assert.Equal(t, 5, data.RetryDelay)
}

func TestRetryData_NoRetrySystem(t *testing.T) {
	data := RetryData{
		MainQueueExists: true,
		WaitQueueExists: false,
		DLQExists:       false,
		MainQueueMsgs:  10,
		WaitQueueMsgs:  0,
		DLQMsgs:         0,
		MaxRetries:      0,
		RetryDelay:      0,
	}

	assert.True(t, data.MainQueueExists)
	assert.False(t, data.WaitQueueExists)
	assert.False(t, data.DLQExists)
	assert.Equal(t, 0, data.WaitQueueMsgs)
	assert.Equal(t, 0, data.DLQMsgs)
}

func TestNewDashboard_Initialization(t *testing.T) {
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

func TestNewDashboard_DifferentIntervals(t *testing.T) {
	cfg := &config.Config{
		RabbitMQ: config.RabbitMQConfig{
			Host: "localhost",
		},
	}

	tests := []struct {
		interval time.Duration
	}{
		{1 * time.Second},
		{5 * time.Second},
		{10 * time.Second},
		{30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.interval.String(), func(t *testing.T) {
			model := NewDashboard("test_queue", cfg, tt.interval)
			assert.Equal(t, tt.interval, model.interval)
		})
	}
}

func TestDashboardStyles_Initialization(t *testing.T) {
	// Verificar que os estilos são inicializados
	assert.NotNil(t, titleStyle)
	assert.NotNil(t, boxStyle)
	assert.NotNil(t, labelStyle)
	assert.NotNil(t, valueStyle)
	assert.NotNil(t, warningStyle)
	assert.NotNil(t, helpStyle)
}

func TestModel_InitialState(t *testing.T) {
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
	assert.Nil(t, model.queueData)
	assert.Nil(t, model.retryData)
	assert.Empty(t, model.error)
}
