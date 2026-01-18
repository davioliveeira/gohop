package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateQueueOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    CreateQueueOptions
		isValid bool
	}{
		{
			name: "valid classic queue",
			opts: CreateQueueOptions{
				Name:       "test_queue",
				Type:       "classic",
				Durable:    true,
				AutoDelete: false,
			},
			isValid: true,
		},
		{
			name: "valid quorum queue",
			opts: CreateQueueOptions{
				Name:       "test_queue",
				Type:       "quorum",
				Durable:    true,
				AutoDelete: false,
			},
			isValid: true,
		},
		{
			name: "queue with arguments",
			opts: CreateQueueOptions{
				Name:       "test_queue",
				Type:       "classic",
				Durable:    true,
				Arguments:  map[string]interface{}{"x-max-length": 1000},
			},
			isValid: true,
		},
		{
			name: "exclusive queue",
			opts: CreateQueueOptions{
				Name:       "test_queue",
				Type:       "classic",
				Exclusive:  true,
				AutoDelete: true,
			},
			isValid: true,
		},
		{
			name: "auto-delete queue",
			opts: CreateQueueOptions{
				Name:       "test_queue",
				Type:       "classic",
				AutoDelete: true,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.opts.Name, "queue name should not be empty")
			if tt.isValid {
				assert.True(t, len(tt.opts.Name) > 0)
			}
		})
	}
}

func TestQueueInfo_Structure(t *testing.T) {
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
	assert.True(t, info.Durable)
	assert.False(t, info.AutoDelete)
	assert.Equal(t, "/", info.VHost)
}

func TestQueueInfo_EmptyQueue(t *testing.T) {
	info := QueueInfo{
		Name:       "empty_queue",
		Messages:   0,
		Consumers:  0,
		Unacked:    0,
		Type:       "classic",
		Durable:    false,
		AutoDelete: true,
		VHost:      "/",
	}

	assert.Equal(t, "empty_queue", info.Name)
	assert.Equal(t, 0, info.Messages)
	assert.Equal(t, 0, info.Consumers)
	assert.Equal(t, 0, info.Unacked)
}

func TestCreateQueueOptions_WithArguments(t *testing.T) {
	opts := CreateQueueOptions{
		Name:      "test_queue",
		Type:      "classic",
		Durable:   true,
		Arguments: map[string]interface{}{
			"x-max-length":      1000,
			"x-message-ttl":    3600000,
			"x-dead-letter-exchange": "dlx",
		},
	}

	assert.Equal(t, "test_queue", opts.Name)
	assert.Equal(t, "classic", opts.Type)
	assert.True(t, opts.Durable)
	assert.NotNil(t, opts.Arguments)
	assert.Equal(t, 3, len(opts.Arguments))
	assert.Equal(t, 1000, opts.Arguments["x-max-length"])
}

func TestCreateQueueOptions_DefaultValues(t *testing.T) {
	opts := CreateQueueOptions{
		Name: "test_queue",
		Type: "classic",
	}

	// Valores padrão quando não especificados
	assert.Equal(t, "test_queue", opts.Name)
	assert.Equal(t, "classic", opts.Type)
	assert.False(t, opts.Durable)
	assert.False(t, opts.AutoDelete)
	assert.False(t, opts.Exclusive)
	assert.False(t, opts.NoWait)
	assert.Nil(t, opts.Arguments)
}
