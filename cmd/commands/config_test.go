package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{
			name:     "empty password",
			password: "",
			expected: "",
		},
		{
			name:     "short password",
			password: "ab",
			expected: "**",
		},
		{
			name:     "single character",
			password: "a",
			expected: "**",
		},
		{
			name:     "normal password",
			password: "mypassword123",
			expected: "m***3",
		},
		{
			name:     "long password",
			password: "verylongpassword123456",
			expected: "v***6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPassword(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}
