package encx

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/hengadev/errsx"
	"golang.org/x/crypto/argon2"
)

// TODO: remove the deps on encryptor and usse the Argon2Params thing
func hashSecure(value string, argon2Params *Argon2Params, pepper []byte) (string, error) {
	// Generate random salt
	salt := make([]byte, argon2Params.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Combine value with pepper
	peppered := append([]byte(value), pepper...)

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		peppered,
		salt,
		argon2Params.Iterations,
		argon2Params.Memory,
		argon2Params.Parallelism,
		argon2Params.KeyLength,
	)

	// Encode params, salt, and hash into a string
	params := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Params.Memory,
		argon2Params.Iterations,
		argon2Params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return params, nil
}
func handleSecureHashing(
	field reflect.StructField,
	fieldValue reflect.Value,
	structValue reflect.Value,
	argon2Params *Argon2Params,
	pepper []byte,
	errs *errsx.Map,
) {
	var hashedValue string
	var err error

	// Store original field value and type
	originalFieldValue := fieldValue

	// Find target hashed field
	targetFieldName := field.Name + HASHED_FIELD_SUFFIX
	fmt.Println("target field name:", targetFieldName)
	targetField := structValue.FieldByName(targetFieldName)

	if !targetField.IsValid() || !targetField.CanSet() {
		errs.Set(fmt.Sprintf("missing field for '%s' secure hashing", field.Name), fmt.Errorf("%s field is required for hashing %s", targetFieldName, field.Name))
		return
	}

	// Check if the target field is a string
	if targetField.Kind() != reflect.String {
		errs.Set(fmt.Sprintf("invalid target field type for '%s' in secure hash", targetFieldName), fmt.Errorf("%s must be of type string to store a secure hash value", targetFieldName))
		return
	}

	// Hash based on field type
	switch field.Type.Kind() {
	case reflect.String:
		hashedValue, err = hashSecure(originalFieldValue.String(), argon2Params, pepper)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue := strconv.FormatInt(originalFieldValue.Int(), 10)
		hashedValue, err = hashSecure(strValue, argon2Params, pepper)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		strValue := strconv.FormatUint(originalFieldValue.Uint(), 10)
		hashedValue, err = hashSecure(strValue, argon2Params, pepper)
	case reflect.Float32, reflect.Float64:
		strValue := strconv.FormatFloat(originalFieldValue.Float(), 'f', -1, 64)
		hashedValue, err = hashSecure(strValue, argon2Params, pepper)
	default:
		if field.Type == reflect.TypeOf(time.Time{}) {
			timeValue, ok := originalFieldValue.Interface().(time.Time)
			if !ok {
				errs.Set(fmt.Sprintf("type conversion for '%s'", field.Name), "failed to convert to time.Time")
			}
			dateStr := timeValue.Format(time.RFC3339)
			hashedValue, err = hashSecure(dateStr, argon2Params, pepper)
		} else {
			errs.Set(fmt.Sprintf("unsupported type '%s'", field.Name), fmt.Sprintf("field %s has unsupported type %s", field.Name, field.Type.String()))
		}
	}

	if err != nil {
		errs.Set(fmt.Sprintf("hash_secure '%s'", field.Name), fmt.Sprintf("failed to hash %s: %v", field.Name, err))
	}

	// Set the hashed value
	targetField.SetString(hashedValue)
}
