package hash

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/hengadev/encx/internal/encxerr"
	"github.com/hengadev/encx/internal/types"

	"golang.org/x/crypto/argon2"
)

func HandleSecure(
	field reflect.StructField,
	fieldValue reflect.Value,
	structValue reflect.Value,
	argon2Params *types.Argon2Params,
	pepper [16]byte,
) error {
	fieldValue, targetField, originalType, err := validateAndPrepareField(field, fieldValue, structValue, hashedFieldSuffix)
	if err != nil {
		return err
	}

	// Skip hashing if field has zero value
	if fieldValue.IsZero() {
		targetField.SetString("")
		return nil
	}

	var strValue string

	// Hash based on field type
	switch originalType.Kind() {
	case reflect.String:
		strValue = fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue = strconv.FormatInt(fieldValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		strValue = strconv.FormatUint(fieldValue.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		strValue = strconv.FormatFloat(fieldValue.Float(), 'f', -1, 64)
	default:
		if field.Type == reflect.TypeOf(time.Time{}) {
			timeValue, ok := fieldValue.Interface().(time.Time)
			if !ok {
				return encxerr.NewTypeConversionError(field.Name, "time.Time", encxerr.SecureHash)
			}
			strValue = timeValue.Format(time.RFC3339)
		} else {
			return encxerr.NewUnsupportedTypeError(field.Name, field.Type.String(), encxerr.SecureHash)
		}
	}

	hashedValue, err := encodeSecureHash(strValue, argon2Params, pepper)
	if err != nil {
		return encxerr.NewOperationFailedError(field.Name, encxerr.SecureHash, err.Error())
	}

	targetField.SetString(hashedValue)
	return nil
}

func encodeSecureHash(value string, argon2Params *types.Argon2Params, pepper [16]byte) (string, error) {
	if err := argon2Params.Validate(); err != nil {
		return "", fmt.Errorf("validate Argon2Params: %w", err)
	}
	// Generate random salt
	salt := make([]byte, argon2Params.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	if isZeroPepper(pepper) {
		return "", NewUninitalizedPepperError()
	}

	// Combine value with pepper
	peppered := append([]byte(value), pepper[:]...)

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

func isZeroPepper(pepper [16]byte) bool {
	for _, b := range pepper {
		if b != 0 {
			return false
		}
	}
	return true
}
