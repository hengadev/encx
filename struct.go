package encx

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"github.com/hengadev/errsx"
)
// validateObjectForProcessing checks if the provided object is a non-nil pointer to a struct.
// It returns an error if the object is nil, not a pointer, or not pointing to a struct,
// or if the pointer is not settable.
func validateObjectForProcessing(object any) error {
	if object == nil {
		return fmt.Errorf("%w: object cannot be nil", ErrNilPointer)
	}
	v := reflect.ValueOf(object)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("%w: must be a pointer to a struct", ErrInvalidFieldType)

	}
	if v.IsNil() { // Check for nil pointer after getting Value
		return fmt.Errorf("%w: pointer to struct cannot be nil", ErrNilPointer)
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("%w: must be a pointer to a struct", ErrInvalidFieldType)
	}
	if !v.CanSet() { // Check if the pointer's value can be modified
		return fmt.Errorf("%w: cannot set value on the provided pointer", ErrOperationFailed)
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

// processField handles the encryption or hashing of a single field based on the 'encx' tag.
// It takes the reflect.Value of the struct, the reflect.StructField of the current field,
// and the Crypto service instance. It returns an error if processing fails.
func (c *Crypto) processField(ctx context.Context, v reflect.Value, field reflect.StructField) error {
	fieldVal := v.FieldByName(field.Name)
	tag := field.Tag.Get("encx")
	operations := strings.Split(tag, ",")
	for _, op := range operations {
		op = strings.TrimSpace(op)
		switch op {
		case "encrypt":
			encryptedFieldName := field.Name + ENCRYPTED_FIELD_SUFFIX
			encryptedField := v.FieldByName(encryptedFieldName)
			if encryptedField.IsValid() && encryptedField.CanSet() {
				plaintext, err := c.serializer.Serialize(fieldVal)
				if err != nil {
					return fmt.Errorf("failed to serialize field '%s': %w", field.Name, err)
				}
				dekField := v.FieldByName(DEK_FIELD)
				dek, ok := dekField.Interface().([]byte)
				if !ok || len(dek) != 32 {
					return fmt.Errorf("invalid DEK in field '%s'", DEK_FIELD)
				}
				ciphertext, err := c.EncryptData(plaintext, dek)
				if err != nil {
					return fmt.Errorf("encryption failed for field '%s': %w", field.Name, err)
				}
				// Set the encrypted value
				encryptedField.SetString(base64.StdEncoding.EncodeToString(ciphertext))
			}
		case "hash_secure":
			hashFieldName := field.Name + HASHED_FIELD_SUFFIX
			hashField := v.FieldByName(hashFieldName)
			if hashField.IsValid() && hashField.CanSet() {
				valueToHashBytes, err := c.serializer.Serialize(fieldVal)
				if err != nil {
					return fmt.Errorf("failed to serialize field '%s' for hashing: %w", field.Name, err)
				}
				hashedValue, err := c.HashSecure(valueToHashBytes)
				if err != nil {
					return fmt.Errorf("secure hashing failed for field '%s': %w", field.Name, err)
				}
				hashField.SetString(hashedValue)
			}
		case "hash_basic":
			hashFieldName := field.Name + HASHED_FIELD_SUFFIX
			hashField := v.FieldByName(hashFieldName)
			if hashField.IsValid() && hashField.CanSet() {
				valueToHash, err := c.serializer.Serialize(fieldVal)
				if err != nil {
					return fmt.Errorf("failed to serialize field '%s' for basic hashing: %w", field.Name, err)
				}
				hashedValue := c.HashBasic(valueToHash)
				hashField.SetString(hashedValue)
			}
		}
	}

	return nil
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
