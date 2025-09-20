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
	encryptor DataEncryptor
	hasher    DataHasher
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
func NewFieldProcessor(encryptor DataEncryptor, hasher DataHasher, _ serialization.Serializer) *FieldProcessor {
	return &FieldProcessor{
		encryptor: encryptor,
		hasher:    hasher,
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
	// TODO: Serialization will be handled in generated code, not in FieldProcessor
	return fmt.Errorf("FieldProcessor.processEncryptField is deprecated - use generated code instead")
}

// processHashField handles hashing of a field
func (fp *FieldProcessor) processHashField(ctx context.Context, structValue reflect.Value, field reflect.StructField, fieldValue reflect.Value, secure bool) error {
	// TODO: Serialization will be handled in generated code, not in FieldProcessor
	return fmt.Errorf("FieldProcessor.processHashField is deprecated - use generated code instead")
}

// dekContextKey is defined in struct.go and shared here
