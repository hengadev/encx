package encx

import (
	"fmt"
	"reflect"
)

// Struct validation operations

// validateObjectForProcessing checks if the provided object is a non-nil pointer to a struct.
// It returns an error if the object is nil, not a pointer, or not pointing to a struct,
// or if the pointer is not settable.
func validateObjectForProcessing(object any) error {
	if object == nil {
		return fmt.Errorf("%w: ProcessStruct requires a non-nil object. "+
			"Usage: crypto.ProcessStruct(ctx, &myStruct)", ErrNilPointer)
	}
	v := reflect.ValueOf(object)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("%w: ProcessStruct requires a pointer to a struct, got %T. "+
			"Usage: crypto.ProcessStruct(ctx, &myStruct) not crypto.ProcessStruct(ctx, myStruct)", 
			ErrInvalidFieldType, object)
	}
	if v.IsNil() { // Check for nil pointer after getting Value
		return fmt.Errorf("%w: ProcessStruct requires a non-nil pointer to a struct. "+
			"Your pointer is nil", ErrNilPointer)
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("%w: ProcessStruct requires a pointer to a struct, got pointer to %s. "+
			"Usage: crypto.ProcessStruct(ctx, &myStruct)", ErrInvalidFieldType, elem.Type())
	}
	if !elem.CanSet() {
		return fmt.Errorf("%w: struct fields must be settable. "+
			"Make sure your struct fields are exported (start with uppercase)", ErrInvalidFieldType)
	}
	return nil
}

// validateDEKField checks the DEK field and generates or retrieves the DEK
func (c *Crypto) validateDEKField(object any) ([]byte, error) {
	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	// Find the DEK field
	dekField, exists := t.FieldByName(FieldDEK)
	if !exists {
		return nil, fmt.Errorf("%w: %s field is required for struct processing", ErrMissingField, FieldDEK)
	}

	dekValue := v.FieldByName(FieldDEK)
	if !dekValue.IsValid() {
		return nil, fmt.Errorf("%w: %s field is not accessible", ErrInvalidFieldType, FieldDEK)
	}

	// Check if DEK field is of correct type
	if dekField.Type.Kind() != reflect.Slice || dekField.Type.Elem().Kind() != reflect.Uint8 {
		return nil, fmt.Errorf("%w: %s field must be of type []byte", ErrInvalidFieldType, FieldDEK)
	}

	// Check if DEK is nil or empty
	var dek []byte
	if dekValue.IsNil() || dekValue.Len() == 0 {
		// Generate a new DEK
		var err error
		dek, err = c.GenerateDEK()
		if err != nil {
			return nil, fmt.Errorf("failed to generate DEK: %w", err)
		}
		// Set the DEK in the struct
		dekValue.SetBytes(dek)
	} else {
		dek = dekValue.Bytes()
	}

	// Validate DEK length
	if len(dek) != 32 { // AES-256 requires 32-byte keys
		return nil, fmt.Errorf("DEK length: %w: expected 32, got %d", ErrInvalidFieldType, len(dek))
	}

	return dek, nil
}