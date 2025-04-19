package hash

import (
	"reflect"

	"github.com/hengadev/encx/internal/encxerr"
)

func validateAndPrepareField(
	field reflect.StructField,
	fieldValue reflect.Value,
	structValue reflect.Value,
	hashedFieldSuffix string,
) (reflect.Value, reflect.Value, reflect.Type, error) {
	// Find target hashed field
	targetFieldName := field.Name + hashedFieldSuffix
	targetField := structValue.FieldByName(targetFieldName)

	if !targetField.IsValid() || !targetField.CanSet() {
		return reflect.Value{}, reflect.Value{}, nil, encxerr.NewMissingFieldError(field.Name, targetFieldName, encxerr.SecureHash)
	}

	// Check if the target field is a string
	if targetField.Kind() != reflect.String {
		return reflect.Value{}, reflect.Value{}, nil, encxerr.NewInvalidFieldTypeError(
			targetFieldName,
			"string",
			targetField.Type().String(),
			encxerr.SecureHash,
		)
	}

	// Handle pointer types
	originalType := field.Type
	if field.Type.Kind() == reflect.Ptr {
		// Check for nil pointer
		if fieldValue.IsNil() {
			return reflect.Value{}, reflect.Value{}, nil, encxerr.NewNilPointerError(field.Name, encxerr.SecureHash)
		}
		// Dereference the pointer to get the actual value
		fieldValue = fieldValue.Elem()
		originalType = originalType.Elem() // Get the type the pointer points to
	}

	return fieldValue, targetField, originalType, nil
}
