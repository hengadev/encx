package encx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hengadev/errsx"
)

func hashBasic(value string) string {
	valueHash := sha256.Sum256([]byte(strings.ToLower(value)))
	return hex.EncodeToString(valueHash[:])
}

func handleBasicHashing(
	field reflect.StructField,
	fieldValue reflect.Value,
	structValue reflect.Value,
	errs *errsx.Map,
) {
	var hashedValue string
	var hasError bool

	// Store original field value and type
	originalFieldValue := fieldValue

	// Find target hashed field
	targetFieldName := field.Name + HASHED_FIELD_SUFFIX
	targetField := structValue.FieldByName(targetFieldName)

	if !targetField.IsValid() || !targetField.CanSet() {
		errs.Set(fmt.Sprintf("missing field for '%s' basic hashing", field.Name), fmt.Errorf("%s field is required for hashing %s", targetFieldName, field.Name))
		return
	}

	// Check if the target field is a string
	if targetField.Kind() != reflect.String {
		errs.Set(fmt.Sprintf("invalid target field type for '%s' in basic hash", targetFieldName), fmt.Errorf("%s must be of type string to store a basic hash value", targetFieldName))
		return
	}

	// Hash based on field type
	switch field.Type.Kind() {
	case reflect.String:
		hashedValue = hashBasic(originalFieldValue.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue := strconv.FormatInt(originalFieldValue.Int(), 10)
		hashedValue = hashBasic(strValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		strValue := strconv.FormatUint(originalFieldValue.Uint(), 10)
		hashedValue = hashBasic(strValue)
	case reflect.Float32, reflect.Float64:
		strValue := strconv.FormatFloat(originalFieldValue.Float(), 'f', -1, 64)
		hashedValue = hashBasic(strValue)
	default:
		if field.Type == reflect.TypeOf(time.Time{}) {
			timeValue, ok := originalFieldValue.Interface().(time.Time)
			if !ok {
				errs.Set(fmt.Sprintf("type conversion for '%s'", field.Name), "failed to convert to time.Time")
				hasError = true
			}
			dateStr := timeValue.Format(time.RFC3339)
			hashedValue = hashBasic(dateStr)
		} else {
			errs.Set(fmt.Sprintf("unsupported type '%s'", field.Name), fmt.Sprintf("field %s has unsupported type %s", field.Name, field.Type.String()))
			hasError = true
		}
	}

	// TODO: how to check for that thing an error during that function execution
	if hasError {
		errs.Set(fmt.Sprintf("hash_basic '%s'", field.Name), fmt.Sprintf("failed to hash %s", field.Name))
	}

	// Set the hashed value
	targetField.SetString(hashedValue)
}
