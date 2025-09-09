package encx

import (
	"errors"
	"fmt"
)

var (
	// High-level service errors
	ErrKMSUnavailable       = errors.New("KMS service unavailable")
	ErrKeyRotationRequired  = errors.New("key rotation required")
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrEncryptionFailed     = errors.New("encryption failed")
	ErrDecryptionFailed     = errors.New("decryption failed")
	ErrDatabaseUnavailable  = errors.New("database unavailable")

	// Crypto errors
	ErrUninitializedPepper = errors.New("pepper value appears to be uninitialized (all zeros)")

	// Field errors
	ErrMissingField       = errors.New("missing required field")
	ErrMissingTargetField = errors.New("missing required target field")
	ErrInvalidFieldType   = errors.New("invalid field type")
	ErrUnsupportedType    = errors.New("unsupported type")

	// Conversion errors
	ErrTypeConversion = errors.New("type conversion failed")
	ErrNilPointer     = errors.New("nil pointer encountered")

	// Operation errors
	ErrOperationFailed = errors.New("operation failed")
	ErrInvalidFormat   = errors.New("invalid format")
)

func NewUninitalizedPepperError() error {
	return ErrUninitializedPepper
}

func NewMissingFieldError(fieldName string, action Action) error {
	return fmt.Errorf("%w: '%s' is required to %s", ErrMissingTargetField, fieldName, action)
}

func NewMissingTargetFieldError(fieldName string, targetFieldName string, action Action) error {
	return fmt.Errorf("%w: '%s' is required to %s %s", ErrMissingTargetField, targetFieldName, action, fieldName)
}

func NewInvalidFieldTypeError(fieldName string, expectedType, actualType string, action Action) error {
	return fmt.Errorf("%w: '%s' must be of type %s to %s, got %s",
		ErrInvalidFieldType, fieldName, expectedType, action, actualType)
}

func NewUnsupportedTypeError(fieldName string, typeName string, action Action) error {
	return fmt.Errorf("%w: field '%s' has unsupported type %s for %s operation",
		ErrUnsupportedType, fieldName, typeName, action)
}

func NewTypeConversionError(fieldName string, typeName string, action Action) error {
	return fmt.Errorf("%w: failed to convert field '%s' to %s for %s operation",
		ErrTypeConversion, fieldName, typeName, action)
}

func NewNilPointerError(fieldName string, action Action) error {
	return fmt.Errorf("%w: field %s is a nil pointer and cannot be processed for %s operation", ErrNilPointer, fieldName, action)
}

func NewOperationFailedError(fieldName string, action Action, details string) error {
	if details != "" {
		return fmt.Errorf("%w: %s operation failed for field '%s': %s",
			ErrOperationFailed, action, fieldName, details)
	}
	return fmt.Errorf("%w: %s operation failed for field '%s'",
		ErrOperationFailed, action, fieldName)
}

func NewInvalidFormatError(fieldName string, formatName string, action Action) error {
	return fmt.Errorf("%w: field '%s' has invalid format for %s operation, expected %s format",
		ErrInvalidFormat, fieldName, action, formatName)
}

// IsRetryableError returns true if the error represents a transient failure that might succeed on retry.
func IsRetryableError(err error) bool {
	return errors.Is(err, ErrKMSUnavailable) ||
		errors.Is(err, ErrDatabaseUnavailable)
}

// IsConfigurationError returns true if the error represents a configuration problem.
func IsConfigurationError(err error) bool {
	return errors.Is(err, ErrInvalidConfiguration) ||
		errors.Is(err, ErrUninitializedPepper) ||
		errors.Is(err, ErrMissingField) ||
		errors.Is(err, ErrMissingTargetField) ||
		errors.Is(err, ErrInvalidFieldType) ||
		errors.Is(err, ErrUnsupportedType)
}

// IsAuthError returns true if the error represents an authentication problem.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthenticationFailed)
}

// IsOperationError returns true if the error represents a failure during encryption/decryption operations.
func IsOperationError(err error) bool {
	return errors.Is(err, ErrEncryptionFailed) ||
		errors.Is(err, ErrDecryptionFailed) ||
		errors.Is(err, ErrOperationFailed)
}

// IsValidationError returns true if the error represents a data validation problem.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidFormat) ||
		errors.Is(err, ErrTypeConversion) ||
		errors.Is(err, ErrNilPointer)
}
