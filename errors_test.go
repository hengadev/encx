package encx

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{"KMS Unavailable", ErrKMSUnavailable, ErrKMSUnavailable},
		{"Authentication Failed", ErrAuthenticationFailed, ErrAuthenticationFailed},
		{"Invalid Configuration", ErrInvalidConfiguration, ErrInvalidConfiguration},
		{"Encryption Failed", ErrEncryptionFailed, ErrEncryptionFailed},
		{"Decryption Failed", ErrDecryptionFailed, ErrDecryptionFailed},
		{"Database Unavailable", ErrDatabaseUnavailable, ErrDatabaseUnavailable},
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
		name           string
		err            error
		isRetryable    bool
		isConfig       bool
		isAuth         bool
		isOperation    bool
		isValidation   bool
	}{
		{
			name:        "KMS Unavailable",
			err:         fmt.Errorf("test: %w", ErrKMSUnavailable),
			isRetryable: true,
		},
		{
			name:        "Database Unavailable",
			err:         fmt.Errorf("test: %w", ErrDatabaseUnavailable),
			isRetryable: true,
		},
		{
			name:     "Authentication Failed",
			err:      fmt.Errorf("test: %w", ErrAuthenticationFailed),
			isAuth:   true,
		},
		{
			name:     "Invalid Configuration",
			err:      fmt.Errorf("test: %w", ErrInvalidConfiguration),
			isConfig: true,
		},
		{
			name:        "Encryption Failed",
			err:         fmt.Errorf("test: %w", ErrEncryptionFailed),
			isOperation: true,
		},
		{
			name:        "Decryption Failed",
			err:         fmt.Errorf("test: %w", ErrDecryptionFailed),
			isOperation: true,
		},
		{
			name:         "Type Conversion Error",
			err:          fmt.Errorf("test: %w", ErrTypeConversion),
			isValidation: true,
		},
		{
			name:     "Missing Field Error",
			err:      fmt.Errorf("test: %w", ErrMissingField),
			isConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.isRetryable {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.isRetryable)
			}
			if got := IsConfigurationError(tt.err); got != tt.isConfig {
				t.Errorf("IsConfigurationError() = %v, want %v", got, tt.isConfig)
			}
			if got := IsAuthError(tt.err); got != tt.isAuth {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.isAuth)
			}
			if got := IsOperationError(tt.err); got != tt.isOperation {
				t.Errorf("IsOperationError() = %v, want %v", got, tt.isOperation)
			}
			if got := IsValidationError(tt.err); got != tt.isValidation {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.isValidation)
			}
		})
	}
}

func TestErrorClassificationMutualExclusivity(t *testing.T) {
	// Test that errors are classified into only one category
	testErrors := []error{
		ErrKMSUnavailable,
		ErrAuthenticationFailed,
		ErrInvalidConfiguration,
		ErrEncryptionFailed,
		ErrTypeConversion,
	}

	for _, err := range testErrors {
		wrapped := fmt.Errorf("test: %w", err)
		
		classifications := []bool{
			IsRetryableError(wrapped),
			IsConfigurationError(wrapped),
			IsAuthError(wrapped),
			IsOperationError(wrapped),
			IsValidationError(wrapped),
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