package processor

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hengadev/errsx"
)

// StructProcessor handles struct processing operations
type StructProcessor struct {
	fieldProcessor *FieldProcessor
	validator      *Validator
	observability  ObservabilityHook
	dekManager     DEKManager
}

// ObservabilityHook defines observability operations for struct processing
type ObservabilityHook interface {
	OnProcessStart(ctx context.Context, operation string, metadata map[string]interface{})
	OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]interface{})
	OnError(ctx context.Context, operation string, err error, metadata map[string]interface{})
}

// ErrorCollector defines interface for collecting multiple errors
type ErrorCollector interface {
	Set(key string, err error)
	AsError() error
	IsEmpty() bool
}

// DEKManager defines interface for DEK operations
type DEKManager interface {
	DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, version int) ([]byte, error)
}

// NewStructProcessor creates a new StructProcessor instance
func NewStructProcessor(fieldProcessor *FieldProcessor, validator *Validator, observability ObservabilityHook, dekManager DEKManager) *StructProcessor {
	return &StructProcessor{
		fieldProcessor: fieldProcessor,
		validator:      validator,
		observability:  observability,
		dekManager:     dekManager,
	}
}

// dekContextKey is used as a context key for DEK values
// Note: This is defined here to be shared with field.go
var dekContextKey = struct{}{}

// ProcessStruct encrypts, hashes, and processes fields in a struct based on `encx` tags.
//
// Supported tags:
//   - encrypt: AES-GCM encryption, requires companion *Encrypted field
//   - hash_secure: Argon2id hashing with pepper, requires companion *Hash field
//   - hash_basic: SHA-256 hashing, requires companion *Hash field
//   - Combined tags: comma-separated for multiple operations, e.g. "encrypt,hash_basic"
//
// Required struct fields:
//   - DEK []byte: Data Encryption Key (auto-generated if nil)
//   - DEKEncrypted []byte: Encrypted DEK (set automatically)
//   - KeyVersion int: KEK version used (set automatically)
func (sp *StructProcessor) ProcessStruct(ctx context.Context, object any, errorCollector ErrorCollector) error {
	// Monitoring: Start processing
	start := time.Now()
	metadata := map[string]interface{}{
		"operation_type": "struct_processing",
		"struct_type":    reflect.TypeOf(object).String(),
	}
	sp.observability.OnProcessStart(ctx, "ProcessStruct", metadata)

	if err := sp.validator.ValidateObjectForProcessing(object); err != nil {
		errorCollector.Set("validate object for struct encryption", err)
	}

	dek, err := sp.validator.ValidateDEKField(object)
	if err != nil {
		errorCollector.Set("validate DEK related field for struct encryption", err)
	}

	if !errorCollector.IsEmpty() {
		finalErr := errorCollector.AsError()
		sp.observability.OnError(ctx, "ProcessStruct", finalErr, metadata)
		sp.observability.OnProcessComplete(ctx, "ProcessStruct", time.Since(start), finalErr, metadata)
		return finalErr
	}

	// Create a new context with the DEK value
	ctxWithDEK := context.WithValue(ctx, dekContextKey, dek)

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip fields that cannot be processed
		if sp.shouldSkipField(field.Name) {
			continue
		}

		// Skip unexported fields that cannot be set
		if !fieldValue.CanSet() {
			continue
		}

		if tag := field.Tag.Get("encx"); tag != "" {
			if err := sp.fieldProcessor.ProcessField(ctxWithDEK, v, field, tag); err != nil {
				errorCollector.Set(fmt.Sprintf("processing field '%s' with tag '%s' in struct type %s",
					field.Name, tag, t.String()), err)
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			// Recursively call ProcessStruct (or a similar function) passing the context
			if err := sp.processEmbeddedStruct(ctxWithDEK, embeddedVal, embeddedType); err != nil {
				errorCollector.Set(fmt.Sprintf("processing embedded struct '%s' of type %s in struct type %s",
					field.Name, embeddedType.String(), t.String()), err)
			}
		}
	}

	if err := sp.setEncryptedDEK(ctxWithDEK, v); err != nil {
		errorCollector.Set(fmt.Sprintf("setting encrypted DEK field in struct type %s", t.String()), err)
	}

	if err := sp.setKeyVersion(ctxWithDEK, v); err != nil {
		errorCollector.Set(fmt.Sprintf("setting key version field in struct type %s", t.String()), err)
	}

	finalErr := errorCollector.AsError()
	// Monitoring: Record completion (success or failure)
	if finalErr != nil {
		sp.observability.OnError(ctx, "ProcessStruct", finalErr, metadata)
	}
	sp.observability.OnProcessComplete(ctx, "ProcessStruct", time.Since(start), finalErr, metadata)

	return finalErr
}

// DecryptStruct decrypts fields in a struct based on `encx` tags.
//
// Required struct fields:
//   - DEK []byte: Data Encryption Key (will be populated from DEKEncrypted)
//   - DEKEncrypted []byte: Encrypted DEK
//   - KeyVersion int: KEK version used for DEK decryption
func (sp *StructProcessor) DecryptStruct(ctx context.Context, object any, errorCollector ErrorCollector) error {
	// Monitoring: Start processing
	start := time.Now()
	metadata := map[string]interface{}{
		"operation_type": "struct_decryption",
		"object_type":    reflect.TypeOf(object).String(),
	}
	sp.observability.OnProcessStart(ctx, "DecryptStruct", metadata)

	// Validate input
	if err := validateObjectForProcessing(object); err != nil {
		errorCollector.Set("validate object for struct decryption", err)
	}

	// Get key version
	keyVersion, err := sp.getKeyVersion(object)
	if err != nil {
		errorCollector.Set("get key version", err)
	}

	// Get DEK
	dek, err := sp.getDEK(ctx, object, keyVersion)
	if err != nil {
		errorCollector.Set("get DEK", err)
	}

	if !errorCollector.IsEmpty() {
		finalErr := errorCollector.AsError()
		sp.observability.OnError(ctx, "DecryptStruct", finalErr, metadata)
		sp.observability.OnProcessComplete(ctx, "DecryptStruct", time.Since(start), finalErr, metadata)
		return finalErr
	}

	// Create a new context with the DEK value
	ctxWithDEK := context.WithValue(ctx, dekContextKey, dek)

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	// Process all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get(StructTag); tag != "" {
			fieldVal := v.FieldByName(field.Name)
			operations := strings.Split(tag, ",")
			for _, op := range operations {
				op = strings.TrimSpace(op)
				if op == TagEncrypt {
					if err := sp.decryptField(ctxWithDEK, field, v, fieldVal, dek); err != nil {
						errorCollector.Set(fmt.Sprintf("decrypt field '%s'", field.Name), err)
					}
				}
				// Future: handle other operations like hash verification
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			if err := sp.decryptEmbeddedStruct(ctxWithDEK, embeddedVal, embeddedType); err != nil {
				errorCollector.Set(fmt.Sprintf("decrypt embedded field '%s'", field.Name), err)
			}
		}
	}

	finalErr := errorCollector.AsError()
	// Monitoring: Record completion (success or failure)
	if finalErr != nil {
		sp.observability.OnError(ctx, "DecryptStruct", finalErr, metadata)
	}
	sp.observability.OnProcessComplete(ctx, "DecryptStruct", time.Since(start), finalErr, metadata)

	return finalErr
}

// shouldSkipField determines if a field should be skipped during processing
func (sp *StructProcessor) shouldSkipField(fieldName string) bool {
	fieldsToSkip := []string{"DEK", "DEKEncrypted", "KeyVersion"}
	for _, skipField := range fieldsToSkip {
		if fieldName == skipField {
			return true
		}
	}
	return false
}

// processEmbeddedStruct handles processing of embedded structs
func (sp *StructProcessor) processEmbeddedStruct(ctx context.Context, embeddedVal reflect.Value, embeddedType reflect.Type) error {
	// Implementation would recursively process embedded structs
	// This is a placeholder for the embedded struct processing logic
	return nil
}

// setEncryptedDEK sets the encrypted DEK field in the struct
func (sp *StructProcessor) setEncryptedDEK(ctx context.Context, v reflect.Value) error {
	// Implementation would set the DEKEncrypted field
	// This is a placeholder for the DEK encryption logic
	return nil
}

// setKeyVersion sets the key version field in the struct
func (sp *StructProcessor) setKeyVersion(ctx context.Context, v reflect.Value) error {
	// Implementation would set the KeyVersion field
	// This is a placeholder for the key version setting logic
	return nil
}

// getKeyVersion retrieves the key version from a struct
func (sp *StructProcessor) getKeyVersion(object any) (int, error) {
	var errs errsx.Map
	v := reflect.ValueOf(object).Elem()
	keyVersionValue := v.FieldByName(FieldKeyVersion)
	if !keyVersionValue.IsValid() {
		errs.Set(fmt.Sprintf("'%s' value not valid", FieldKeyVersion), fmt.Errorf("field '%s' not found in struct", FieldKeyVersion))
	}
	if keyVersionValue.IsValid() && keyVersionValue.Kind() != reflect.Int {
		errs.Set(fmt.Sprintf("'%s' kind not valid", FieldKeyVersion), fmt.Errorf("field '%s' must be of type int, got %s", FieldKeyVersion, keyVersionValue.Type().String()))
	}
	if !errs.IsEmpty() {
		return 0, errs.AsError()
	}
	keyVersion := int(keyVersionValue.Int())
	return keyVersion, nil
}

// getDEK retrieves and decrypts the DEK from a struct
func (sp *StructProcessor) getDEK(ctx context.Context, object any, keyVersion int) ([]byte, error) {
	var errs errsx.Map
	v := reflect.ValueOf(object).Elem()

	encryptedDEKFieldValue := v.FieldByName(FieldDEKEncrypted)
	if !encryptedDEKFieldValue.IsValid() {
		errs.Set(fmt.Sprintf("'%s' value not valid", FieldDEKEncrypted), fmt.Errorf("field '%s' not found in struct", FieldDEKEncrypted))
	}
	if encryptedDEKFieldValue.IsValid() && (encryptedDEKFieldValue.Kind() != reflect.Slice || encryptedDEKFieldValue.Type().Elem().Kind() != reflect.Uint8) {
		errs.Set(fmt.Sprintf("'%s' kind not valid", FieldDEKEncrypted), fmt.Errorf("field '%s' must be of type []byte, got %s", FieldDEKEncrypted, encryptedDEKFieldValue.Type().String()))
	}
	encryptedDEKBytes := encryptedDEKFieldValue.Bytes()

	dek, err := sp.dekManager.DecryptDEKWithVersion(ctx, encryptedDEKBytes, keyVersion)
	if err != nil {
		errs.Set("decrypt DEK", err)
	}
	if len(dek) != 32 {
		errs.Set("DEK length", fmt.Errorf("decrypted DEK has incorrect length: expected 32, got %d", len(dek)))
	}

	return dek, errs.AsError()
}

// decryptField decrypts a single field using the provided DEK
func (sp *StructProcessor) decryptField(ctx context.Context, field reflect.StructField, v, fieldVal reflect.Value, dek []byte) error {
	// Skip special fields
	if sp.shouldSkipField(field.Name) {
		return nil
	}

	encryptedFieldName := field.Name + SuffixEncrypted
	encryptedField := v.FieldByName(encryptedFieldName)
	if encryptedField.IsValid() && encryptedField.Kind() == reflect.Slice && fieldVal.CanSet() {
		ciphertext := encryptedField.Bytes()
		plaintextBytes, err := sp.fieldProcessor.encryptor.DecryptData(ctx, ciphertext, dek)
		if err != nil {
			return fmt.Errorf("decryption failed for field '%s': %w", field.Name, err)
		}

		// TODO: Deserialization will be handled in generated code
		return fmt.Errorf("struct processor deserialization is deprecated - use generated code instead")
	}
	return nil
}

// decryptEmbeddedStruct recursively decrypts fields in embedded structs
func (sp *StructProcessor) decryptEmbeddedStruct(ctx context.Context, v reflect.Value, t reflect.Type) error {
	var decryptErrs errsx.Map
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get(StructTag); tag != "" {
			fieldVal := v.FieldByName(field.Name)
			operations := strings.Split(tag, ",")
			for _, op := range operations {
				op = strings.TrimSpace(op)
				if op == TagEncrypt {
					dek, ok := ctx.Value(dekContextKey).([]byte)
					if !ok {
						return fmt.Errorf("DEK not found in context for field '%s'", field.Name)
					}
					if len(dek) != 32 {
						return fmt.Errorf("DEK has invalid length: expected 32 bytes, got %d bytes", len(dek))
					}
					if err := sp.decryptField(ctx, field, v, fieldVal, dek); err != nil {
						decryptErrs.Set(fmt.Sprintf("decrypt embedded field '%s'", field.Name), err)
					}
				}
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			if err := sp.decryptEmbeddedStruct(ctx, embeddedVal, embeddedType); err != nil {
				decryptErrs.Set(fmt.Sprintf("decrypt deeply embedded field '%s'", field.Name), err)
			}
		}
	}
	return decryptErrs.AsError()
}
