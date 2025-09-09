package processor

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hengadev/encx/internal/serialization"
)

// FieldProcessor handles individual field processing operations
type FieldProcessor struct {
	encryptor  DataEncryptor
	hasher     DataHasher
	serializer serialization.Serializer
}

// DataEncryptor defines interface for data encryption operations
type DataEncryptor interface {
	EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)
	DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
}

// DataHasher defines interface for data hashing operations
type DataHasher interface {
	HashBasic(ctx context.Context, value []byte) string
	HashSecure(ctx context.Context, value []byte) (string, error)
}

// NewFieldProcessor creates a new FieldProcessor instance
func NewFieldProcessor(encryptor DataEncryptor, hasher DataHasher, serializer serialization.Serializer) *FieldProcessor {
	return &FieldProcessor{
		encryptor:  encryptor,
		hasher:     hasher,
		serializer: serializer,
	}
}

// ProcessField processes a single field based on its encx tag(s)
// Supports both single tags and comma-separated combined tags like "encrypt,hash_basic"
func (fp *FieldProcessor) ProcessField(ctx context.Context, v reflect.Value, field reflect.StructField, tag string) error {
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
	if err := fp.validateFieldForProcessing(field, fieldValue, tags); err != nil {
		return fmt.Errorf("validation failed for field '%s' with tag '%s': %w", field.Name, tag, err)
	}

	// Process each tag operation
	for _, singleTag := range tags {
		switch singleTag {
		case "encrypt":
			if err := fp.processEncryptField(ctx, v, field, fieldValue); err != nil {
				return fmt.Errorf("encryption failed for field '%s': %w", field.Name, err)
			}
		case "hash_secure":
			if err := fp.processHashField(ctx, v, field, fieldValue, true); err != nil {
				return fmt.Errorf("secure hashing failed for field '%s': %w", field.Name, err)
			}
		case "hash_basic":
			if err := fp.processHashField(ctx, v, field, fieldValue, false); err != nil {
				return fmt.Errorf("basic hashing failed for field '%s': %w", field.Name, err)
			}
		default:
			return fmt.Errorf("unsupported encx tag '%s' for field '%s'. Supported tags: encrypt, hash_secure, hash_basic",
				singleTag, field.Name)
		}
	}

	return nil
}

// validateFieldForProcessing performs pre-processing validation on a field
func (fp *FieldProcessor) validateFieldForProcessing(field reflect.StructField, fieldValue reflect.Value, tags []string) error {
	// Check if field can be set
	if !fieldValue.CanSet() {
		return fmt.Errorf("field '%s' cannot be modified (not settable)", field.Name)
	}

	// Validate field types for all operations
	for _, tag := range tags {
		switch tag {
		case "encrypt":
			// For encryption, we need to be able to serialize the value
			if !isSerializableType(fieldValue.Type()) {
				return fmt.Errorf("field type %s is not serializable for encryption", fieldValue.Type())
			}
		case "hash_secure", "hash_basic":
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
func (fp *FieldProcessor) processEncryptField(ctx context.Context, structValue reflect.Value, field reflect.StructField, fieldValue reflect.Value) error {
	// Extract DEK from context
	dek, ok := ctx.Value(dekContextKey).([]byte)
	if !ok {
		return fmt.Errorf("DEK not found in context")
	}

	// Serialize the field value
	serializedValue, err := fp.serializer.Serialize(fieldValue)
	if err != nil {
		return fmt.Errorf("failed to serialize field value for encryption: %w", err)
	}

	// Encrypt the serialized value
	encryptedValue, err := fp.encryptor.EncryptData(ctx, serializedValue, dek)
	if err != nil {
		return fmt.Errorf("failed to encrypt field value: %w", err)
	}

	// Find the corresponding encrypted field and set its value
	encryptedFieldName := field.Name + "Encrypted"
	encryptedField := structValue.FieldByName(encryptedFieldName)
	if !encryptedField.IsValid() {
		return fmt.Errorf("encrypted field '%s' not found for field '%s'", encryptedFieldName, field.Name)
	}

	if !encryptedField.CanSet() {
		return fmt.Errorf("encrypted field '%s' cannot be set", encryptedFieldName)
	}

	// Set the encrypted value
	if encryptedField.Type().Kind() == reflect.Slice && encryptedField.Type().Elem().Kind() == reflect.Uint8 {
		encryptedField.SetBytes(encryptedValue)
	} else {
		return fmt.Errorf("encrypted field '%s' must be of type []byte", encryptedFieldName)
	}

	return nil
}

// processHashField handles hashing of a field
func (fp *FieldProcessor) processHashField(ctx context.Context, structValue reflect.Value, field reflect.StructField, fieldValue reflect.Value, secure bool) error {
	// Serialize the field value
	serializedValue, err := fp.serializer.Serialize(fieldValue)
	if err != nil {
		return fmt.Errorf("failed to serialize field value for hashing: %w", err)
	}

	var hashValue string
	if secure {
		hashValue, err = fp.hasher.HashSecure(ctx, serializedValue)
		if err != nil {
			return fmt.Errorf("failed to perform secure hash: %w", err)
		}
	} else {
		hashValue = fp.hasher.HashBasic(ctx, serializedValue)
	}

	// Find the corresponding hash field and set its value
	hashFieldName := field.Name + "Hash"
	hashField := structValue.FieldByName(hashFieldName)
	if !hashField.IsValid() {
		return fmt.Errorf("hash field '%s' not found for field '%s'", hashFieldName, field.Name)
	}

	if !hashField.CanSet() {
		return fmt.Errorf("hash field '%s' cannot be set", hashFieldName)
	}

	// Set the hash value
	if hashField.Type().Kind() == reflect.String {
		hashField.SetString(hashValue)
	} else {
		return fmt.Errorf("hash field '%s' must be of type string", hashFieldName)
	}

	return nil
}

// dekContextKey is defined in struct.go and shared here
