package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupOptions(t *testing.T) {
	opts := SetupOptions{
		QueueName:  "test_queue",
		MaxRetries: 3,
		RetryDelay: 5,
		DLQTTL:     604800000,
		Force:      false,
	}

	assert.Equal(t, "test_queue", opts.QueueName)
	assert.Equal(t, 3, opts.MaxRetries)
	assert.Equal(t, 5, opts.RetryDelay)
	assert.Equal(t, 604800000, opts.DLQTTL)
	assert.False(t, opts.Force)
}

func TestRetrySystemInfo(t *testing.T) {
	info := &RetrySystemInfo{
		QueueName:       "test_queue",
		MainQueue:       true,
		WaitQueue:       true,
		WaitExchange:    true,
		RetryExchange:   true,
		DLQ:             true,
		MaxRetries:      3,
		RetryDelay:      5,
		DLQTTL:          604800000,
		MainQueueMsgs:   10,
		WaitQueueMsgs:   3,
		DLQMsgs:         2,
	}

	assert.True(t, info.MainQueue)
	assert.True(t, info.WaitQueue)
	assert.True(t, info.DLQ)
	assert.Equal(t, 10, info.MainQueueMsgs)
	assert.Equal(t, 3, info.WaitQueueMsgs)
	assert.Equal(t, 2, info.DLQMsgs)
}

// Teste de nomes de componentes do retry
func TestRetryComponentNames(t *testing.T) {
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
