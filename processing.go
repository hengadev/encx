package encx

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"github.com/hengadev/errsx"
)

func (c *Crypto) Process(ctx context.Context, object any) error {
	var validErrs errsx.Map
	if err := validateObjectForProcessing(object); err != nil {
		validErrs.Set("validate object for struct encryption", err)
	}

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	dek, err := c.validateDEKField(object)
	if err != nil {
		validErrs.Set("validate DEK related field for struct encryption", err)
	}

	if validErrs.IsEmpty() {
		return validErrs.AsError()
	}

	var processErrs errsx.Map
	for i := range t.NumField() {
		field := t.Field(i)
		if tag := field.Tag.Get(STRUCT_TAG); tag != "" {
			if err := c.processField(ctx, v, field); err != nil {
				processErrs.Set(fmt.Sprintf("processing field '%s'", field.Name), err)
			}
		}
	}

	if err := c.setEncryptedDEK(ctx, v, dek); err != nil {
		processErrs.Set("set encrypted DEK field", err)
	}

	if err := c.setKeyVersion(ctx, v); err != nil {
		processErrs.Set("set key version field", err)
	}

	return processErrs.AsError()
}

func (c *Crypto) setKeyVersion(ctx context.Context, v reflect.Value) error {
	// Get the current KEK version
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return fmt.Errorf("failed to get current KEK version: %w", err)
	}
	// Set the KeyVersion field in the struct
	keyVersionField := v.FieldByName(VERSION_FIELD)
	if keyVersionField.IsValid() && keyVersionField.CanSet() && keyVersionField.Kind() == reflect.Int {
		keyVersionField.SetInt(int64(currentVersion))
	} else {
		return fmt.Errorf("invalid or non-settable '%s' field in struct", VERSION_FIELD)
	}
	return nil
}

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
		return nil, NewMissingFieldError(DEK_FIELD, Encrypt)
	}

	var dek []byte

	if dekFieldValue.Kind() == reflect.Slice && dekFieldValue.Type().Elem().Kind() == reflect.Uint8 && dekFieldValue.Len() == 32 {
		dek = dekFieldValue.Interface().([]byte)
	} else if dekFieldValue.IsNil() || dekFieldValue.Len() == 0 {
		// Generate default DEK
		defaultDEK, err := c.GenerateDEK()
		if err != nil {
			return nil, fmt.Errorf("failed to generate default DEK: %w", err) // Keep standard error for internal failure
		}
		dek = defaultDEK
		dekFieldValue.Set(reflect.ValueOf(dek)) // Set the generated DEK back to the struct
	} else {
		return nil, NewInvalidFieldTypeError(DEK_FIELD, "[]byte (length 32) or empty for default generation", dekFieldValue.Type().String(), Encrypt)
	}

	if !dekFieldValue.CanSet() {
		return nil, NewOperationFailedError(DEK_FIELD, Encrypt, "field is not settable")
	}

	encryptedDEKField := v.FieldByName(DEK_ENCRYPTED_FIELD)
	if !encryptedDEKField.IsValid() {
		return nil, NewMissingTargetFieldError(DEK_FIELD, DEK_ENCRYPTED_FIELD, Encrypt)
	}
	if encryptedDEKField.Kind() != reflect.Slice || encryptedDEKField.Type().Elem().Kind() != reflect.Uint8 {
		return nil, NewInvalidFieldTypeError(DEK_ENCRYPTED_FIELD, "[]byte", encryptedDEKField.Type().String(), Encrypt)
	}
	if !encryptedDEKField.CanSet() {
		return nil, NewOperationFailedError(DEK_ENCRYPTED_FIELD, Encrypt, "field is not settable")
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
		case ENCRYPT:
			encryptedFieldName := field.Name + ENCRYPTED_FIELD_SUFFIX
			encryptedField := v.FieldByName(encryptedFieldName)
			if !encryptedField.IsValid() {
				return NewMissingTargetFieldError(field.Name, encryptedFieldName, Encrypt)
			}
			if !encryptedField.CanSet() {
				return NewOperationFailedError(encryptedFieldName, Encrypt, "field is not settable")
			}
			plaintext, err := c.serializer.Serialize(fieldVal)
			if err != nil {
				return fmt.Errorf("failed to serialize field '%s' for encryption: %w", field.Name, err) // Keep underlying error
			}
			dekField := v.FieldByName(DEK_FIELD)
			dek, ok := dekField.Interface().([]byte)
			if !ok || len(dek) != 32 {
				return NewInvalidFormatError(DEK_FIELD, "32-byte []byte", Encrypt)
			}
			ciphertext, err := c.EncryptData(plaintext, dek)
			if err != nil {
				return fmt.Errorf("encryption failed for field '%s': %w", field.Name, err) // Keep underlying error
			}
			// Set the encrypted value
			encryptedField.SetString(base64.StdEncoding.EncodeToString(ciphertext))
		case SECURE:
			hashFieldName := field.Name + HASHED_FIELD_SUFFIX
			hashField := v.FieldByName(hashFieldName)
			if !hashField.IsValid() {
				return NewMissingTargetFieldError(field.Name, hashFieldName, SecureHash)
			}
			if !hashField.CanSet() {
				return NewOperationFailedError(hashFieldName, SecureHash, "field is not settable")
			}
			valueToHashBytes, err := c.serializer.Serialize(fieldVal)
			if err != nil {
				return fmt.Errorf("failed to serialize field '%s' for secure hashing: %w", field.Name, err) // Keep underlying error
			}
			hashedValue, err := c.HashSecure(valueToHashBytes)
			if err != nil {
				return fmt.Errorf("secure hashing failed for field '%s': %w", field.Name, err) // Keep underlying error
			}
			hashField.SetString(hashedValue)
		case BASIC:
			hashFieldName := field.Name + HASHED_FIELD_SUFFIX
			hashField := v.FieldByName(hashFieldName)
			if !hashField.IsValid() {
				return NewMissingTargetFieldError(field.Name, hashFieldName, BasicHash)
			}
			if !hashField.CanSet() {
				return NewOperationFailedError(hashFieldName, BasicHash, "field is not settable")
			}
			valueToHash, err := c.serializer.Serialize(fieldVal)
			if err != nil {
				return fmt.Errorf("failed to serialize field '%s' for basic hashing: %w", field.Name, err)
			}
			hashedValue := c.HashBasic(valueToHash)
			hashField.SetString(hashedValue)
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
