package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagValidator(t *testing.T) {
	validator := NewTagValidator()

	tests := []struct {
		name      string
		fieldName string
		tags      []string
		expectErr bool
	}{
		{
			name:      "Valid single encrypt tag",
			fieldName: "Email",
			tags:      []string{"encrypt"},
			expectErr: false,
		},
		{
			name:      "Valid encrypt with hash_basic",
			fieldName: "Email",
			tags:      []string{"encrypt", "hash_basic"},
			expectErr: false,
		},
		{
			name:      "Invalid hash_basic with hash_secure",
			fieldName: "Email",
			tags:      []string{"hash_basic", "hash_secure"},
			expectErr: true,
		},
		{
			name:      "Invalid unknown tag",
			fieldName: "Email",
			tags:      []string{"unknown"},
			expectErr: true,
		},
		{
			name:      "Valid duplicate tags filtered",
			fieldName: "Email",
			tags:      []string{"encrypt", "encrypt"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateFieldTags(tt.fieldName, tt.tags)
			hasErrors := len(errors) > 0
			assert.Equal(t, tt.expectErr, hasErrors, "Expected hasErrors=%v, got hasErrors=%v, errors=%v", tt.expectErr, hasErrors, errors)
		})
	}
}

