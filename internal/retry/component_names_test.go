package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRetryComponentNames(t *testing.T) {
	tests := []struct {
		queueName      string
		expectedWait   string
		expectedWaitEx string
		expectedRetry  string
		expectedDLQ    string
	}{
		{
			queueName:      "test_queue",
			expectedWait:   "test_queue.wait",
			expectedWaitEx: "test_queue.wait.exchange",
			expectedRetry:  "test_queue.retry",
			expectedDLQ:    "test_queue.dlq",
		},
		{
			queueName:      "cartpanda_physical",
			expectedWait:   "cartpanda_physical.wait",
			expectedWaitEx: "cartpanda_physical.wait.exchange",
			expectedRetry:  "cartpanda_physical.retry",
			expectedDLQ:    "cartpanda_physical.dlq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.queueName, func(t *testing.T) {
			waitQueueName := tt.queueName + ".wait"
			waitExchangeName := tt.queueName + ".wait.exchange"
			retryExchangeName := tt.queueName + ".retry"
			dlqName := tt.queueName + ".dlq"

			assert.Equal(t, tt.expectedWait, waitQueueName)
			assert.Equal(t, tt.expectedWaitEx, waitExchangeName)
			assert.Equal(t, tt.expectedRetry, retryExchangeName)
			assert.Equal(t, tt.expectedDLQ, dlqName)
		})
	}
}

func TestRetryDelayConversion(t *testing.T) {
	tests := []struct {
		retryDelaySeconds int
		expectedMs        int
	}{
		{5, 5000},
		{10, 10000},
		{30, 30000},
		{60, 60000},
	}

	for _, tt := range tests {
		t.Run("conversion", func(t *testing.T) {
			retryDelayMs := tt.retryDelaySeconds * 1000
			assert.Equal(t, tt.expectedMs, retryDelayMs)
		})
	}
}
