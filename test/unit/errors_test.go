package encx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hengadev/encx"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{"KMS Unavailable", encx.ErrKMSUnavailable, encx.ErrKMSUnavailable},
		{"Authentication Failed", encx.ErrAuthenticationFailed, encx.ErrAuthenticationFailed},
		{"Invalid Configuration", encx.ErrInvalidConfiguration, encx.ErrInvalidConfiguration},
		{"Encryption Failed", encx.ErrEncryptionFailed, encx.ErrEncryptionFailed},
		{"Decryption Failed", encx.ErrDecryptionFailed, encx.ErrDecryptionFailed},
		{"Database Unavailable", encx.ErrDatabaseUnavailable, encx.ErrDatabaseUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("context: %w", tt.err)
			if !errors.Is(wrapped, tt.expected) {
				t.Errorf("Expected errors.Is(wrapped, %v) to be true", tt.expected)
			}
		})
	}
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		isRetryable  bool
		isConfig     bool
		isAuth       bool
		isOperation  bool
		isValidation bool
	}{
		{
			name:        "KMS Unavailable",
			err:         fmt.Errorf("test: %w", encx.ErrKMSUnavailable),
			isRetryable: true,
		},
		{
			name:        "Database Unavailable",
			err:         fmt.Errorf("test: %w", encx.ErrDatabaseUnavailable),
			isRetryable: true,
		},
		{
			name:   "Authentication Failed",
			err:    fmt.Errorf("test: %w", encx.ErrAuthenticationFailed),
			isAuth: true,
		},
		{
			name:     "Invalid Configuration",
			err:      fmt.Errorf("test: %w", encx.ErrInvalidConfiguration),
			isConfig: true,
		},
		{
			name:        "Encryption Failed",
			err:         fmt.Errorf("test: %w", encx.ErrEncryptionFailed),
			isOperation: true,
		},
		{
			name:        "Decryption Failed",
			err:         fmt.Errorf("test: %w", encx.ErrDecryptionFailed),
			isOperation: true,
		},
		{
			name:         "Type Conversion Error",
			err:          fmt.Errorf("test: %w", encx.ErrTypeConversion),
			isValidation: true,
		},
		{
			name:     "Missing Field Error",
			err:      fmt.Errorf("test: %w", encx.ErrMissingField),
			isConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encx.IsRetryableError(tt.err); got != tt.isRetryable {
				t.Errorf("encx.IsRetryableError() = %v, want %v", got, tt.isRetryable)
			}
			if got := encx.IsConfigurationError(tt.err); got != tt.isConfig {
				t.Errorf("encx.IsConfigurationError() = %v, want %v", got, tt.isConfig)
			}
			if got := encx.IsAuthError(tt.err); got != tt.isAuth {
				t.Errorf("encx.IsAuthError() = %v, want %v", got, tt.isAuth)
			}
			if got := encx.IsOperationError(tt.err); got != tt.isOperation {
				t.Errorf("encx.IsOperationError() = %v, want %v", got, tt.isOperation)
			}
			if got := encx.IsValidationError(tt.err); got != tt.isValidation {
				t.Errorf("encx.IsValidationError() = %v, want %v", got, tt.isValidation)
			}
		})
	}
}

func TestErrorClassificationMutualExclusivity(t *testing.T) {
	// Test that errors are classified into only one category
	testErrors := []error{
		encx.ErrKMSUnavailable,
		encx.ErrAuthenticationFailed,
		encx.ErrInvalidConfiguration,
		encx.ErrEncryptionFailed,
		encx.ErrTypeConversion,
	}

	for _, err := range testErrors {
		wrapped := fmt.Errorf("test: %w", err)

		classifications := []bool{
			encx.IsRetryableError(wrapped),
			encx.IsConfigurationError(wrapped),
			encx.IsAuthError(wrapped),
			encx.IsOperationError(wrapped),
			encx.IsValidationError(wrapped),
		}

		trueCount := 0
		for _, classification := range classifications {
			if classification {
				trueCount++
			}
		}

		if trueCount != 1 {
			t.Errorf("Error %v should be classified into exactly one category, got %d", err, trueCount)
		}
	}
}

