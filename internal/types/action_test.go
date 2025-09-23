package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction_String(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected string
	}{
		{
			name:     "Unknown action",
			action:   Unknown,
			expected: "unknown",
		},
		{
			name:     "BasicHash action",
			action:   BasicHash,
			expected: "basic hash",
		},
		{
			name:     "SecureHash action",
			action:   SecureHash,
			expected: "secure hash",
		},
		{
			name:     "Encrypt action",
			action:   Encrypt,
			expected: "encrypt",
		},
		{
			name:     "Decrypt action",
			action:   Decrypt,
			expected: "decrypt",
		},
		{
			name:     "Invalid action",
			action:   Action(127), // Valid int8 value but invalid action
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestActionConstants(t *testing.T) {
	// Test that the constants have expected values
	assert.Equal(t, Action(0), Unknown)
	assert.Equal(t, Action(1), BasicHash)
	assert.Equal(t, Action(2), SecureHash)
	assert.Equal(t, Action(3), Encrypt)
	assert.Equal(t, Action(4), Decrypt)
}