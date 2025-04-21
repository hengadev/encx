package encxerr

import (
	"errors"
	"fmt"
)

var (
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
