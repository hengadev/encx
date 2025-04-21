package encx

import (
	"fmt"
	"reflect"
	"github.com/hengadev/errsx"
)
// validateObjectForProcessing checks if the provided object is a non-nil pointer to a struct.
// It returns an error if the object is nil, not a pointer, or not pointing to a struct,
// or if the pointer is not settable.
func validateObjectForProcessing(object any) error {
	if object == nil {
		return fmt.Errorf("nil object encountered: the object can not be processed for encryption")
	}
	v := reflect.ValueOf(object)
	if v.Kind() != reflect.Ptr {
		return NewInvalidKindError("Must be a pointer to a struct.")
	}
	if v.IsNil() { // Check for nil pointer after getting Value
		return fmt.Errorf("nil pointer to struct encountered: the object cannot be processed")
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return NewInvalidKindError("Must be a pointer to a struct.")
	}
	if !v.CanSet() { // Check if the pointer's value can be modified
		return fmt.Errorf("cannot set value on the provided pointer")
	}
	return nil
}

// validateDEKField checks if the provided object has a valid DEK field ([]byte).
// If the DEK field is not set (nil or zero value), it generates a default DEK.
// It also ensures the presence of the DEKEncrypted field ([]byte).
func (c *Crypto) validateDEKField(object any) ([]byte, error) {
	v := reflect.ValueOf(object).Elem()

	var errs errsx.Map

	dekFieldValue := v.FieldByName(DEK_FIELD)
	if !dekFieldValue.IsValid() {
		return nil, fmt.Errorf("field '%s' is required", DEK_FIELD)
	}

	var dek []byte

	if dekFieldValue.Kind() == reflect.Slice && dekFieldValue.Type().Elem().Kind() == reflect.Uint8 && dekFieldValue.Len() == 32 {
		dek = dekFieldValue.Interface().([]byte)
	} else if dekFieldValue.IsNil() || dekFieldValue.Len() == 0 {
		// Generate default DEK
		defaultDEK, err := c.GenerateDEK()
		if err != nil {
			return nil, fmt.Errorf("failed to generate default DEK: %w", err)
		}
		dek = defaultDEK
		dekFieldValue.Set(reflect.ValueOf(dek)) // Set the generated DEK back to the struct
	} else {
		return nil, fmt.Errorf("field '%s' must be a 32-byte []byte or nil/empty for default generation", DEK_FIELD)
	}

	if !dekFieldValue.CanSet() {
		return nil, fmt.Errorf("field '%s' is not settable", DEK_FIELD)
	}

	encryptedDEKField := v.FieldByName(DEK_ENCRYPTED_FIELD)
	if !encryptedDEKField.IsValid() {
		return nil, fmt.Errorf("field '%s' is required for storing the encrypted DEK", DEK_ENCRYPTED_FIELD)
	}
	if encryptedDEKField.Kind() != reflect.Slice || encryptedDEKField.Type().Elem().Kind() != reflect.Uint8 {
		return nil, fmt.Errorf("field '%s' must be of type []byte", DEK_ENCRYPTED_FIELD)
	}
	if !encryptedDEKField.CanSet() {
		return nil, fmt.Errorf("field '%s' is not settable", DEK_ENCRYPTED_FIELD)
	}

	return dek, errs.AsError()
}


// setEncryptedDEK encrypts the provided DEK using the KMS and sets the resulting ciphertext in the DEKEncrypted field of the given reflect.Value.
func (c *Crypto) setEncryptedDEK(ctx context.Context, v reflect.Value, dek []byte) error {
	encryptedDEK, err := c.EncryptDEK(ctx, dek)
	if err != nil {
		return fmt.Errorf("failed to encrypt DEK: %w", err)
	}
	encryptedDEKField := v.FieldByName(DEK_ENCRYPTED_FIELD)
	encryptedDEKField.SetBytes(encryptedDEK)
	return nil
}
