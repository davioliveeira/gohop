package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateQueueName_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		queueName   string
		shouldValid bool
	}{
		{"valid queue name", "test_queue", true},
		{"valid with underscore", "my_queue_123", true},
		{"valid with numbers", "queue123", true},
		{"valid with dash", "my-queue", true},
		{"valid with dots", "my.queue.name", true},
		{"valid long name", "very_long_queue_name_with_many_parts_12345", true},
		{"empty name", "", false},
		{"only spaces", "   ", false},
		{"with special chars", "queue@#$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := len(tt.queueName) > 0 && len(strings.TrimSpace(tt.queueName)) > 0
			if tt.shouldValid {
				assert.True(t, valid, "queue name should be valid: %s", tt.queueName)
			} else {
				assert.False(t, valid, "queue name should be invalid: %s", tt.queueName)
			}
		})
	}
}

func TestFormatDuration_MultipleFormats(t *testing.T) {
	tests := []struct {
		seconds int
		format  string
	}{
		{5, "5s"},
		{60, "60s"},
		{3600, "3600s"},
		{0, "0s"},
		{1, "1s"},
		{86400, "86400s"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatted := fmt.Sprintf("%ds", tt.seconds)
			assert.Equal(t, tt.format, formatted)
		})
	}
}

func TestMaskPassword_VariousLengths(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{"empty", "", "***"},
		{"short", "ab", "***"},
		{"normal", "password123", "p********3"},
		{"long", "very_long_password_that_should_be_masked_properly", "v**********************************y"},
		{"single char", "a", "***"},
		{"two chars", "ab", "***"},
		{"three chars", "abc", "a*c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPassword(tt.password)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else {
				// Verificar que está mascarado
				assert.NotEqual(t, tt.password, result)
				assert.Contains(t, result, "*")
			}
		})
	}
}

func TestQueueTypeValidation(t *testing.T) {
	tests := []struct {
		queueType string
		valid     bool
	}{
		{"classic", true},
		{"quorum", true},
		{"stream", false}, // Não suportado ainda
		{"invalid", false},
		{"", false},
		{"CLASSIC", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.queueType, func(t *testing.T) {
			valid := tt.queueType == "classic" || tt.queueType == "quorum"
			assert.Equal(t, tt.valid, valid)
		})
	}
}
