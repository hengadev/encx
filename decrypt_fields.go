package encx

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/hengadev/errsx"
)

// DecryptFields decrypts all fields in a struct marked with the encrypt tag
func (s *Encryptor) DecryptFields(data any) error {
	// Create error collector
	var errs errsx.Map

	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("data must be a pointer to a struct")
	}

	// Dereference the pointer
	structValue := value.Elem()
	structType := structValue.Type()

	// Create master cipher for decryption
	masterBlock, err := aes.NewCipher(s.KeyEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create master cipher: %w", err)
	}

	masterGCM, err := cipher.NewGCM(masterBlock)
	if err != nil {
		return fmt.Errorf("failed to create master GCM: %w", err)
	}

	// Check if the struct has a DataEncryptionKey field and decrypt it first with master key
	var dekValue string
	dekField := structValue.FieldByName(DEK_FIELD)
	if dekField.IsValid() && dekField.Kind() == reflect.String {
		encryptedDEK := dekField.String()
		if encryptedDEK != "" {
			// Decrypt DEK with master key
			decryptedDEK, err := decryptWithGCM(masterGCM, encryptedDEK)
			if err != nil {
				return fmt.Errorf("failed to decrypt DataEncryptionKey: %w", err)
			}

			// Store the decrypted DEK value for decrypting other fields
			dekValue = decryptedDEK

			// Also update the field value with the decrypted DEK
			dekField.SetString(decryptedDEK)
		} else {
			return fmt.Errorf("DataEncryptionKey is empty")
		}
	} else {
		return fmt.Errorf("DataEncryptionKey field not found or not a string")
	}

	// Create a new cipher using the decrypted DEK for other fields
	encryptionKey, _ := hex.DecodeString(dekValue)
	dekBlock, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create DEK cipher: %w", err)
	}

	dekGCM, err := cipher.NewGCM(dekBlock)
	if err != nil {
		return fmt.Errorf("failed to create DEK GCM: %w", err)
	}

	// Iterate through all fields in the struct
	for i := range structType.NumField() {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Check if the field has the encx tag
		tag := field.Tag.Get(STRUCT_TAG)
		if tag == "" {
			continue
		}

		// Only process fields marked for encryption
		tagOptions := parseTag(tag)
		if !tagOptions["encrypt"] {
			continue
		}

		// Skip the DEK field as it was already handled
		if field.Name == DEK_FIELD {
			continue
		}

		// Handle decryption for this field using the DEK cipher
		handleDecryption(field, fieldValue, structValue, dekGCM, &errs)
	}

	if !errs.IsEmpty() {
		return errs.AsError()
	}

	return nil
}

// parseTag parses the encx tag into a map of options
func parseTag(tag string) map[string]bool {
	options := make(map[string]bool)

	// Split by comma
	parts := splitTag(tag)
	for _, part := range parts {
		options[part] = true
	}

	return options
}

// splitTag splits a tag by commas, handling whitespace
func splitTag(tag string) []string {
	var parts []string
	var currentPart string

	for _, char := range tag {
		if char == ',' {
			parts = append(parts, currentPart)
			currentPart = ""
			continue
		}

		if char != ' ' {
			currentPart += string(char)
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	return parts
}
