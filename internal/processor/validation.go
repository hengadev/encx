package processor

import (
	"fmt"
	"reflect"
)

// Validator handles input validation for struct processing
type Validator struct {
	dekGenerator DEKGenerator
}

// DEKGenerator defines interface for DEK generation
type DEKGenerator interface {
	GenerateDEK() ([]byte, error)
}

// NewValidator creates a new Validator instance
func NewValidator(dekGenerator DEKGenerator) *Validator {
	return &Validator{
		dekGenerator: dekGenerator,
	}
}

// ValidateObjectForProcessing validates an object before processing
func (v *Validator) ValidateObjectForProcessing(object any) error {
	if object == nil {
		return fmt.Errorf("object cannot be nil")
	}

	// Check if the object is a pointer
	objValue := reflect.ValueOf(object)
	if objValue.Kind() != reflect.Ptr {
		return fmt.Errorf("object must be a pointer to a struct, got %T", object)
	}

	// Check if the pointer points to a struct
	objValue = objValue.Elem()
	if objValue.Kind() != reflect.Struct {
		return fmt.Errorf("object must be a pointer to a struct, got pointer to %s", objValue.Kind())
	}

	// Check if the object can be modified
	if !objValue.CanSet() {
		return fmt.Errorf("object cannot be modified (not settable)")
	}

	return nil
}

// ValidateDEKField validates and prepares the DEK field in a struct
func (v *Validator) ValidateDEKField(object any) ([]byte, error) {
	objValue := reflect.ValueOf(object).Elem()
	objType := objValue.Type()

	// Check for required DEK field
	dekField := objValue.FieldByName("DEK")
	if !dekField.IsValid() {
		return nil, fmt.Errorf("struct %s must have a DEK field of type []byte", objType.Name())
	}

	if dekField.Type().Kind() != reflect.Slice || dekField.Type().Elem().Kind() != reflect.Uint8 {
		return nil, fmt.Errorf("DEK field in struct %s must be of type []byte", objType.Name())
	}

	// Check for DEKEncrypted field
	dekEncryptedField := objValue.FieldByName("DEKEncrypted")
	if !dekEncryptedField.IsValid() {
		return nil, fmt.Errorf("struct %s must have a DEKEncrypted field of type []byte", objType.Name())
	}

	if dekEncryptedField.Type().Kind() != reflect.Slice || dekEncryptedField.Type().Elem().Kind() != reflect.Uint8 {
		return nil, fmt.Errorf("DEKEncrypted field in struct %s must be of type []byte", objType.Name())
	}

	// Check for KeyVersion field
	keyVersionField := objValue.FieldByName("KeyVersion")
	if !keyVersionField.IsValid() {
		return nil, fmt.Errorf("struct %s must have a KeyVersion field of type int", objType.Name())
	}

	if keyVersionField.Type().Kind() != reflect.Int {
		return nil, fmt.Errorf("KeyVersion field in struct %s must be of type int", objType.Name())
	}

	// Get or generate DEK
	var dek []byte
	if dekField.IsNil() || dekField.Len() == 0 {
		// Generate a new DEK
		generatedDEK, err := v.dekGenerator.GenerateDEK()
		if err != nil {
			return nil, fmt.Errorf("failed to generate DEK for struct %s: %w", objType.Name(), err)
		}
		dek = generatedDEK

		// Set the DEK in the struct
		dekField.SetBytes(dek)
	} else {
		// Use existing DEK
		dek = dekField.Bytes()
	}

	return dek, nil
}

// ValidateStructTags validates struct tags for correctness
func (v *Validator) ValidateStructTags(object any) error {
	objValue := reflect.ValueOf(object).Elem()
	objType := objValue.Type()

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		tag := field.Tag.Get("encx")

		if tag == "" {
			continue
		}

		// Validate tag format and companion fields
		if err := v.validateFieldTag(objType, field, tag); err != nil {
			return fmt.Errorf("validation failed for field '%s' in struct %s: %w", field.Name, objType.Name(), err)
		}
	}

	return nil
}

// validateFieldTag validates a single field's tag and its requirements
func (v *Validator) validateFieldTag(structType reflect.Type, field reflect.StructField, tag string) error {
	// Parse tags (support comma-separated)
	tags := parseTagString(tag)

	for _, singleTag := range tags {
		switch singleTag {
		case "encrypt":
			// Check for companion encrypted field
			encryptedFieldName := field.Name + "Encrypted"
			if _, found := structType.FieldByName(encryptedFieldName); !found {
				return fmt.Errorf("encrypt tag requires companion field '%s' of type []byte", encryptedFieldName)
			}
		case "hash_secure", "hash_basic":
			// Check for companion hash field
			hashFieldName := field.Name + "Hash"
			if _, found := structType.FieldByName(hashFieldName); !found {
				return fmt.Errorf("%s tag requires companion field '%s' of type string", singleTag, hashFieldName)
			}
		default:
			return fmt.Errorf("unsupported tag '%s'", singleTag)
		}
	}

	return nil
}

// parseTagString parses a comma-separated tag string
func parseTagString(tag string) []string {
	if tag == "" {
		return nil
	}

	// Split by comma and trim whitespace
	parts := make([]string, 0)
	for _, part := range splitAndTrim(tag, ",") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}

// splitAndTrim splits a string by separator and trims whitespace
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range split(s, sep) {
		trimmed := trim(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// split splits a string by separator
func split(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	result := make([]string, 0)
	start := 0

	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}

	result = append(result, s[start:])
	return result
}

// trim removes leading and trailing whitespace
func trim(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isWhitespace(s[start]) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isWhitespace checks if a character is whitespace
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

