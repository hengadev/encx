package encx

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hengadev/errsx"
)

// Field processing operations

// processField processes a single field based on its encx tag(s)
// Supports both single tags and comma-separated combined tags like "encrypt,hash_basic"
func (c *Crypto) processField(ctx context.Context, v reflect.Value, field reflect.StructField, tag string) error {
	fieldValue := v.FieldByName(field.Name)
	if !fieldValue.IsValid() {
		return fmt.Errorf("field '%s' is not accessible in struct of type %s", field.Name, v.Type())
	}

	// Parse comma-separated tags
	tags := strings.Split(strings.TrimSpace(tag), ",")
	for i, t := range tags {
		tags[i] = strings.TrimSpace(t)
	}

	// Validate field value before processing
	if err := c.validateFieldForProcessing(field, fieldValue, tags); err != nil {
		return fmt.Errorf("validation failed for field '%s' with tag '%s': %w", field.Name, tag, err)
	}

	// Process each tag operation
	for _, singleTag := range tags {
		switch singleTag {
		case TagEncrypt:
			if err := c.processEncryptField(ctx, v, field, fieldValue); err != nil {
				return fmt.Errorf("encryption failed for field '%s': %w", field.Name, err)
			}
		case TagHashSecure:
			if err := c.processHashField(ctx, v, field, fieldValue, true); err != nil {
				return fmt.Errorf("secure hashing failed for field '%s': %w", field.Name, err)
			}
		case TagHashBasic:
			if err := c.processHashField(ctx, v, field, fieldValue, false); err != nil {
				return fmt.Errorf("basic hashing failed for field '%s': %w", field.Name, err)
			}
		default:
			return fmt.Errorf("unsupported encx tag '%s' for field '%s'. Supported tags: %s, %s, %s",
				singleTag, field.Name, TagEncrypt, TagHashSecure, TagHashBasic)
		}
	}

	return nil
}

// validateFieldForProcessing performs pre-processing validation on a field
func (c *Crypto) validateFieldForProcessing(field reflect.StructField, fieldValue reflect.Value, tags []string) error {
	// Check if field can be set
	if !fieldValue.CanSet() {
		return fmt.Errorf("field '%s' cannot be modified (not settable)", field.Name)
	}

	// Validate field types for all operations
	for _, tag := range tags {
		switch tag {
		case TagEncrypt:
			// For encryption, we need to be able to serialize the value
			if !isSerializableType(fieldValue.Type()) {
				return fmt.Errorf("field type %s is not serializable for encryption", fieldValue.Type())
			}
		case TagHashSecure, TagHashBasic:
			// For hashing, we need to be able to serialize the value
			if !isSerializableType(fieldValue.Type()) {
				return fmt.Errorf("field type %s is not serializable for hashing", fieldValue.Type())
			}
		}
	}

	return nil
}

// isSerializableType checks if a type can be serialized by our serializer
func isSerializableType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool,
		reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Ptr:
		return true
	case reflect.Interface:
		// Interfaces are serializable if they contain serializable values
		return true
	default:
		return false
	}
}

// processEncryptField handles encryption of a field
func (c *Crypto) processEncryptField(ctx context.Context, structValue reflect.Value, field reflect.StructField, fieldValue reflect.Value) error {
	// Find the companion encrypted field
	encryptedFieldName := field.Name + SuffixEncrypted
	encryptedField := structValue.FieldByName(encryptedFieldName)
	if !encryptedField.IsValid() {
		return fmt.Errorf("encryption requires companion field: '%s' field must exist to store encrypted data for field '%s'. "+
			"Add '%s []byte' to your struct", encryptedFieldName, field.Name, encryptedFieldName)
	}

	// Ensure the encrypted field is of type []byte
	if encryptedField.Type().Kind() != reflect.Slice || encryptedField.Type().Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("encryption target field type error: field '%s' must be of type []byte to store encrypted data, "+
			"got %s. Change to: %s []byte", encryptedFieldName, encryptedField.Type(), encryptedFieldName)
	}

	// Serialize the field value
	serializedValue, err := c.serializer.Serialize(fieldValue)
	if err != nil {
		return fmt.Errorf("serialization failed for field '%s' of type %s during encryption: %w. "+
			"Ensure the field contains serializable data", field.Name, fieldValue.Type(), err)
	}

	// Get DEK from context
	dek, ok := ctx.Value(dekContextKey{}).([]byte)
	if !ok || len(dek) == 0 {
		return fmt.Errorf("DEK not found in context for field '%s'", field.Name)
	}

	// Encrypt the serialized value
	encryptedValue, err := c.EncryptData(ctx, serializedValue, dek)
	if err != nil {
		return fmt.Errorf("encryption operation failed for field '%s' using AES-GCM: %w. "+
			"This could indicate invalid DEK or encryption service issues", field.Name, err)
	}

	// Set the encrypted value
	encryptedField.SetBytes(encryptedValue)

	// Clear the original field by setting it to zero value
	fieldValue.Set(reflect.Zero(field.Type))

	return nil
}

// processHashField handles hashing of a field
func (c *Crypto) processHashField(ctx context.Context, structValue reflect.Value, field reflect.StructField, fieldValue reflect.Value, secure bool) error {
	// Find the companion hash field
	hashFieldName := field.Name + SuffixHashed
	hashField := structValue.FieldByName(hashFieldName)
	if !hashField.IsValid() {
		hashType := "hash_basic"
		if secure {
			hashType = "hash_secure"
		}
		return fmt.Errorf("hashing requires companion field: '%s' field must exist to store hashed data for field '%s' with tag '%s'. "+
			"Add '%s string' to your struct", hashFieldName, field.Name, hashType, hashFieldName)
	}

	// Ensure the hash field is of type string
	if hashField.Type().Kind() != reflect.String {
		return fmt.Errorf("hash target field type error: field '%s' must be of type string to store hashed data, "+
			"got %s. Change to: %s string", hashFieldName, hashField.Type(), hashFieldName)
	}

	// Serialize the field value
	serializedValue, err := c.serializer.Serialize(fieldValue)
	if err != nil {
		return fmt.Errorf("serialization failed for field '%s' of type %s during hashing: %w. "+
			"Ensure the field contains serializable data", field.Name, fieldValue.Type(), err)
	}

	// Hash the serialized value
	var hashedValue string
	if secure {
		hashedValue, err = c.HashSecure(ctx, serializedValue)
		if err != nil {
			return fmt.Errorf("secure hashing operation failed for field '%s' using Argon2id: %w. "+
				"This could indicate insufficient memory or invalid pepper configuration", field.Name, err)
		}
	} else {
		hashedValue = c.HashBasic(ctx, serializedValue)
	}

	// Set the hashed value
	hashField.SetString(hashedValue)

	return nil
}

// processEmbeddedStruct processes embedded structs recursively.
// This function takes a context, a reflect.Value representing the embedded struct,
// and a reflect.Type representing the type of the embedded struct.
// It processes each field of the embedded struct based on the 'encx' tag,
// and recursively processes any further embedded structs within it.
// It returns an error if any processing fails.
func (c *Crypto) processEmbeddedStruct(ctx context.Context, v reflect.Value, t reflect.Type) error {
	var errs errsx.Map
	for i := range t.NumField() {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields that cannot be processed
		if !fieldValue.CanSet() {
			continue
		}

		if tag := field.Tag.Get(StructTag); tag != "" {
			if err := c.processField(ctx, v, field, tag); err != nil {
				errs.Set(fmt.Sprintf("processing embedded field '%s.%s' with tag '%s' in struct type %s",
					t.Name(), field.Name, tag, t.String()), err)
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			if err := c.processEmbeddedStruct(ctx, embeddedVal, embeddedType); err != nil {
				errs.Set(fmt.Sprintf("processing deeply embedded struct '%s.%s' of type %s",
					t.Name(), field.Name, embeddedType.String()), err)
			}
		}
	}
	return errs.AsError()
}

// setEncryptedDEK encrypts the DEK and stores it in the DEKEncrypted field
func (c *Crypto) setEncryptedDEK(ctx context.Context, v reflect.Value) error {
	// Get DEK from context
	dek, ok := ctx.Value(dekContextKey{}).([]byte)
	if !ok || len(dek) == 0 {
		return fmt.Errorf("DEK not found in processing context. This indicates an internal error in DEK management")
	}

	// Encrypt the DEK
	encryptedDEK, err := c.EncryptDEK(ctx, dek)
	if err != nil {
		return fmt.Errorf("DEK encryption failed using KEK: %w. This could indicate KMS connectivity issues or invalid key configuration", err)
	}

	// Set the encrypted DEK in the struct
	dekEncryptedField := v.FieldByName(FieldDEKEncrypted)
	if !dekEncryptedField.IsValid() {
		return fmt.Errorf("struct processing requires field: '%s' field must exist to store encrypted DEK. "+
			"Add '%s []byte' to your struct", FieldDEKEncrypted, FieldDEKEncrypted)
	}

	// Verify field is correct type
	if dekEncryptedField.Type().Kind() != reflect.Slice || dekEncryptedField.Type().Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("field type error: '%s' field must be of type []byte to store encrypted DEK, "+
			"got %s. Change to: %s []byte", FieldDEKEncrypted, dekEncryptedField.Type(), FieldDEKEncrypted)
	}

	dekEncryptedField.SetBytes(encryptedDEK)

	return nil
}

// setKeyVersion sets the current key version in the struct
func (c *Crypto) setKeyVersion(ctx context.Context, v reflect.Value) error {
	// Get current key version
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return fmt.Errorf("failed to retrieve current KEK version from key metadata: %w. "+
			"This could indicate database connectivity issues or missing key configuration", err)
	}

	// Set the key version in the struct
	keyVersionField := v.FieldByName(FieldKeyVersion)
	if !keyVersionField.IsValid() {
		return fmt.Errorf("struct processing requires field: '%s' field must exist to track key version. "+
			"Add '%s int' to your struct", FieldKeyVersion, FieldKeyVersion)
	}

	// Verify field is correct type
	if keyVersionField.Type().Kind() != reflect.Int {
		return fmt.Errorf("field type error: '%s' field must be of type int to store key version, "+
			"got %s. Change to: %s int", FieldKeyVersion, keyVersionField.Type(), FieldKeyVersion)
	}

	keyVersionField.SetInt(int64(currentVersion))

	return nil
}

// shouldSkipField checks if a field should be skipped during processing
func shouldSkipField(fieldName string) bool {
	for _, skipField := range fieldsToSkip {
		if fieldName == skipField {
			return true
		}
	}
	// Skip any field ending with "Encrypted" or "Hash" suffixes
	return strings.HasSuffix(fieldName, SuffixEncrypted) || strings.HasSuffix(fieldName, SuffixHashed)
}
