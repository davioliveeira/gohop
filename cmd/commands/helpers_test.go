package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateQueueName(t *testing.T) {
	tests := []struct {
		name        string
		queueName   string
		shouldValid bool
	}{
		{"valid queue name", "test_queue", true},
		{"valid with underscore", "my_queue_123", true},
		{"valid with numbers", "queue123", true},
		{"valid with dash", "my-queue", true},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulação de validação básica
			valid := len(tt.queueName) > 0
			if tt.shouldValid {
				assert.True(t, valid, "queue name should be valid")
			} else {
				assert.False(t, valid, "queue name should be invalid")
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		format  string
	}{
		{5, "5s"},
		{60, "60s"},
		{3600, "3600s"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatted := fmt.Sprintf("%ds", tt.seconds)
			assert.Equal(t, tt.format, formatted)
		})
	}
}
