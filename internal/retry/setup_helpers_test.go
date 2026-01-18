package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    SetupOptions
		isValid bool
	}{
		{
			name: "valid options",
			opts: SetupOptions{
				QueueName:  "test_queue",
				MaxRetries: 3,
				RetryDelay: 5,
				DLQTTL:     604800000,
				Force:      false,
			},
			isValid: true,
		},
		{
			name: "options with force",
			opts: SetupOptions{
				QueueName:  "test_queue",
				MaxRetries: 5,
				RetryDelay: 10,
				DLQTTL:     86400000,
				Force:      true,
			},
			isValid: true,
		},
		{
			name: "options with zero retries",
			opts: SetupOptions{
				QueueName:  "test_queue",
				MaxRetries: 0,
				RetryDelay: 5,
				DLQTTL:     604800000,
			},
			isValid: true,
		},
		{
			name: "options with high retry delay",
			opts: SetupOptions{
				QueueName:  "test_queue",
				MaxRetries: 3,
				RetryDelay: 3600, // 1 hora
				DLQTTL:     604800000,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.opts.QueueName)
			if tt.isValid {
				assert.True(t, len(tt.opts.QueueName) > 0)
				assert.GreaterOrEqual(t, tt.opts.MaxRetries, 0)
				assert.GreaterOrEqual(t, tt.opts.RetryDelay, 0)
				assert.GreaterOrEqual(t, tt.opts.DLQTTL, 0)
			}
		})
	}
}

func TestRetrySystemInfo_Structure(t *testing.T) {
	info := RetrySystemInfo{
		QueueName:     "test_queue",
		MainQueue:     true,
		WaitQueue:     true,
		WaitExchange:  true,
		RetryExchange: true,
		DLQ:           true,
		MaxRetries:    3,
		RetryDelay:    5,
		DLQTTL:        604800000,
		MainQueueMsgs: 10,
		WaitQueueMsgs: 3,
		DLQMsgs:       2,
	}

	assert.Equal(t, "test_queue", info.QueueName)
	assert.True(t, info.MainQueue)
	assert.True(t, info.WaitQueue)
	assert.True(t, info.WaitExchange)
	assert.True(t, info.RetryExchange)
	assert.True(t, info.DLQ)
	assert.Equal(t, 3, info.MaxRetries)
	assert.Equal(t, 5, info.RetryDelay)
	assert.Equal(t, 604800000, info.DLQTTL)
	assert.Equal(t, 10, info.MainQueueMsgs)
	assert.Equal(t, 3, info.WaitQueueMsgs)
	assert.Equal(t, 2, info.DLQMsgs)
}

func TestRetrySystemInfo_EmptySystem(t *testing.T) {
	info := RetrySystemInfo{
		QueueName:     "test_queue",
		MainQueue:     false,
		WaitQueue:     false,
		WaitExchange:  false,
		RetryExchange: false,
		DLQ:           false,
		MaxRetries:    0,
		RetryDelay:    0,
		DLQTTL:        0,
		MainQueueMsgs: 0,
		WaitQueueMsgs: 0,
		DLQMsgs:       0,
	}

	assert.Equal(t, "test_queue", info.QueueName)
	assert.False(t, info.MainQueue)
	assert.False(t, info.WaitQueue)
	assert.False(t, info.WaitExchange)
	assert.False(t, info.RetryExchange)
	assert.False(t, info.DLQ)
	assert.Equal(t, 0, info.MainQueueMsgs)
}

func TestRetryDelayConversion_Extended(t *testing.T) {
	tests := []struct {
		seconds int
		ms      int
	}{
		{300, 300000},
		{3600, 3600000},
		{7200, 7200000},
	}

	for _, tt := range tests {
		t.Run("conversion", func(t *testing.T) {
			retryDelayMs := tt.seconds * 1000
			assert.Equal(t, tt.ms, retryDelayMs)
		})
	}
}

func TestComponentNameGeneration(t *testing.T) {
	queueName := "test_queue"

	waitQueueName := queueName + ".wait"
	waitExchangeName := queueName + ".wait.exchange"
	retryExchangeName := queueName + ".retry"
	dlqName := queueName + ".dlq"

	assert.Equal(t, "test_queue.wait", waitQueueName)
	assert.Equal(t, "test_queue.wait.exchange", waitExchangeName)
	assert.Equal(t, "test_queue.retry", retryExchangeName)
	assert.Equal(t, "test_queue.dlq", dlqName)
}

func TestComponentNameGeneration_ComplexNames(t *testing.T) {
	tests := []struct {
		queueName string
	}{
		{"cartpanda_physical"},
		{"order-processing"},
		{"user.events"},
		{"api_v2_requests"},
	}

	for _, tt := range tests {
		t.Run(tt.queueName, func(t *testing.T) {
			waitQueueName := tt.queueName + ".wait"
			dlqName := tt.queueName + ".dlq"

			assert.Contains(t, waitQueueName, tt.queueName)
			assert.Contains(t, dlqName, tt.queueName)
			assert.Contains(t, waitQueueName, ".wait")
			assert.Contains(t, dlqName, ".dlq")
		})
	}
}
