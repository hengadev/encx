package encx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/hengadev/errsx"
)

// encryptWithGCM encrypts a string using the provided GCM cipher
func encryptWithGCM(gcm cipher.AEAD, data string) (string, error) {
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// encryptString encrypts a string using AES-GCM
func encryptString(key []byte, data string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	return encryptWithGCM(gcm, data)
}

// handleEncryption encrypts a field based on its type
func handleEncryption(field reflect.StructField, fieldValue reflect.Value, structValue reflect.Value, gcm cipher.AEAD, errs *errsx.Map) {
	var encrypted string
	var err error

	// Store the original field type for type checking
	originalFieldType := field.Type
	originalFieldValue := fieldValue

	// handle dataencryptionkey field
	if field.Name == DEK_FIELD {
		encrypted, err = encryptWithGCM(gcm, originalFieldValue.String())
		fieldValue.SetString(encrypted)
		return
	}

	targetFieldName := field.Name + ENCRYPTED_FIELD_SUFFIX
	targetField := structValue.FieldByName(targetFieldName)

	if !targetField.IsValid() || !targetField.CanSet() {
		errs.Set(fmt.Sprintf("missing field for '%s' encryption", field.Name), fmt.Sprintf("%s field is required for encrypting %s", targetFieldName, field.Name))
		return
	}

	// Check if the target field is a string
	if targetField.Kind() != reflect.String {
		errs.Set(fmt.Sprintf("invalid target field type for '%s' in encryption", targetFieldName), fmt.Errorf("%s must be of type string to store an encrypt value", targetFieldName))
		return
	}

	// Encrypt based on ORIGINAL field type (not the target field type)
	switch originalFieldType.Kind() {
	case reflect.String:
		encrypted, err = encryptWithGCM(gcm, originalFieldValue.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue := strconv.FormatInt(originalFieldValue.Int(), 10)
		encrypted, err = encryptWithGCM(gcm, strValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		strValue := strconv.FormatUint(originalFieldValue.Uint(), 10)
		encrypted, err = encryptWithGCM(gcm, strValue)
	default:
		// Handle time.Time specifically
		if originalFieldType == reflect.TypeOf(time.Time{}) {
			timeValue, ok := originalFieldValue.Interface().(time.Time)
			if !ok {
				errs.Set(fmt.Sprintf("type conversion for '%s'", field.Name), "failed to convert to time.Time")
			}
			if timeValue.IsZero() {
				// Skip encryption for zero time
				return
			}
			dateStr := timeValue.Format(time.RFC3339)
			encrypted, err = encryptWithGCM(gcm, dateStr)
		} else {
			errs.Set(fmt.Sprintf("unsupported type '%s'", field.Name), fmt.Sprintf("field %s has unsupported type %s", field.Name, originalFieldType.String()))
		}
	}

	if err != nil {
		errs.Set(fmt.Sprintf("encrypt '%s'", field.Name), fmt.Sprintf("failed to encrypt %s: %v", field.Name, err))
	}

	// Set encrypted value to target field
	targetField.SetString(encrypted)
}
