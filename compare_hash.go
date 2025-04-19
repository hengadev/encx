package encx

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hengadev/encx/internal/hash"

	"golang.org/x/crypto/argon2"
)

func (s *Crypto) CompareSecureHashAndValue(value any, hashValue string) (bool, error) {
	// Handle nil value
	if value == nil {
		return false, fmt.Errorf("value cannot be nil for secure comparison")
	}

	// Get the value type using reflection
	valueType := reflect.TypeOf(value)
	valueVal := reflect.ValueOf(value)

	// Handle nil pointer
	if valueType.Kind() == reflect.Ptr && valueVal.IsNil() {
		return false, fmt.Errorf("value pointer cannot be nil for secure comparison")
	}

	// Dereference pointer if needed
	if valueType.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
		value = valueVal.Interface()
	}

	// Extract the hash parameters
	parts := strings.Split(hashValue, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Extract algorithm type to ensure it's argon2id
	if parts[1] != "argon2id" {
		return false, fmt.Errorf("unknown hashing algorithm: %s", parts[1])
	}

	// Extract version
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("invalid hash version: %v", err)
	}

	// Extract parameters
	var memory, iterations, parallelism uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("invalid hash parameters: %v", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid salt: %v", err)
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid hash: %v", err)
	}

	// Convert the input value to a string based on its type
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case int, int8, int16, int32, int64:
		strValue = fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		strValue = fmt.Sprintf("%d", v)
	case float32, float64:
		strValue = fmt.Sprintf("%g", v)
	case time.Time:
		strValue = v.Format(time.RFC3339)
	default:
		return false, fmt.Errorf("unsupported value type: %T", value)
	}

	// Combine value with pepper
	// peppered := append([]byte(strValue), s.pepper...)
	peppered := append([]byte(strValue), s.pepper[:]...)

	// Hash the input value with the same parameters
	computedHash := argon2.IDKey(
		peppered,
		salt,
		iterations,
		memory,
		uint8(parallelism),
		uint32(len(decodedHash)),
	)

	// Compare hashes (constant-time comparison)
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1, nil
}

func (s *Crypto) CompareBasicHashAndValue(value any, hashValue string) (bool, error) {
	// Handle nil value
	if value == nil {
		return hash.Basic("") == hashValue, nil
	}

	// Get the value type using reflection
	valueType := reflect.TypeOf(value)
	valueVal := reflect.ValueOf(value)

	// Dereference pointer if needed
	if valueType.Kind() == reflect.Ptr {
		// Handle nil pointer
		if valueVal.IsNil() {
			return hash.Basic("") == hashValue, nil
		}
		// Update to the dereferenced value
		valueVal = valueVal.Elem()
		valueType = valueType.Elem()
		value = valueVal.Interface()
	}

	// Convert the input value to a string based on its type
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case int, int8, int16, int32, int64:
		strValue = fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		strValue = fmt.Sprintf("%d", v)
	case float32, float64:
		strValue = fmt.Sprintf("%g", v)
	case time.Time:
		strValue = v.Format(time.RFC3339)
	case []byte:
		strValue = string(v)
	case bool:
		strValue = fmt.Sprintf("%t", v)
	default:
		return false, fmt.Errorf("unsupported value type: %T", value)
	}

	// Compare the computed hash with the provided hash
	return hash.Basic(strValue) == hashValue, nil
}
