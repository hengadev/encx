package encx

import (
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/hengadev/errsx"
)

// decryptWithGCM decrypts a base64 encoded string using the provided GCM cipher
func decryptWithGCM(gcm cipher.AEAD, encryptedData string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// handleDecryption decrypts a field based on its type
func handleDecryption(field reflect.StructField, fieldValue reflect.Value, structValue reflect.Value, gcm cipher.AEAD, errs *errsx.Map) {
	// handle DataEncryptionKey field separately
	if field.Name == DEK_FIELD {
		encryptedValue := fieldValue.String()
		if encryptedValue == "" {
			return // nothing to decrypt
		}

		decrypted, err := decryptWithGCM(gcm, encryptedValue)
		if err != nil {
			errs.Set(fmt.Sprintf("decrypt '%s'", field.Name), fmt.Sprintf("failed to decrypt %s: %v", field.Name, err))
			return
		}
		fieldValue.SetString(decrypted)
		return
	}

	// For other fields, look for the corresponding encrypted field
	encryptedFieldName := field.Name + ENCRYPTED_FIELD_SUFFIX
	encryptedField := structValue.FieldByName(encryptedFieldName)

	if !encryptedField.IsValid() || encryptedField.Kind() != reflect.String {
		// Skip if no encrypted field exists or it's not a string
		return
	}

	encryptedValue := encryptedField.String()
	if encryptedValue == "" {
		return // nothing to decrypt
	}
	// Decrypt value
	decrypted, err := decryptWithGCM(gcm, encryptedValue)
	if err != nil {
		errs.Set(fmt.Sprintf("decrypt '%s'", field.Name), fmt.Sprintf("failed to decrypt %s: %v", field.Name, err))
		return
	}

	// Set the decrypted value based on field type
	switch field.Type.Kind() {
	case reflect.String:
		fieldValue.SetString(decrypted)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(decrypted, 10, 64)
		if err != nil {
			errs.Set(fmt.Sprintf("convert '%s'", field.Name), fmt.Sprintf("failed to convert decrypted value to int: %v", err))
			return
		}
		fieldValue.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(decrypted, 10, 64)
		if err != nil {
			errs.Set(fmt.Sprintf("convert '%s'", field.Name), fmt.Sprintf("failed to convert decrypted value to uint: %v", err))
			return
		}
		fieldValue.SetUint(uintVal)
	default:
		// Handle time.Time specifically
		if field.Type == reflect.TypeOf(time.Time{}) {
			timeValue, err := time.Parse(time.RFC3339, decrypted)
			if err != nil {
				errs.Set(fmt.Sprintf("convert '%s'", field.Name), fmt.Sprintf("failed to parse time: %v", err))
				return
			}
			fieldValue.Set(reflect.ValueOf(timeValue))
		} else {
			errs.Set(fmt.Sprintf("unsupported type '%s'", field.Name), fmt.Sprintf("field %s has unsupported type %s", field.Name, field.Type.String()))
		}
	}
}
