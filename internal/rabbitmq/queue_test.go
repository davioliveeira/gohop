package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateQueueOptions(t *testing.T) {
	opts := CreateQueueOptions{
		Name:       "test_queue",
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Type:       "quorum",
		Arguments:  make(map[string]interface{}),
	}

	assert.Equal(t, "test_queue", opts.Name)
	assert.True(t, opts.Durable)
	assert.False(t, opts.AutoDelete)
	assert.Equal(t, "quorum", opts.Type)
}

func TestQueueInfo(t *testing.T) {
	info := QueueInfo{
		Name:       "test_queue",
		Messages:   10,
		Consumers:  2,
		Unacked:    3,
		Type:       "quorum",
		Durable:    true,
		AutoDelete: false,
		VHost:      "/",
	}

	assert.Equal(t, "test_queue", info.Name)
	assert.Equal(t, 10, info.Messages)
	assert.Equal(t, 2, info.Consumers)
	assert.Equal(t, 3, info.Unacked)
	assert.Equal(t, "quorum", info.Type)
}
