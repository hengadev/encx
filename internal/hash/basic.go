package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hengadev/encx/internal/encxerr"
)

func HandleBasic(
	field reflect.StructField,
	fieldValue reflect.Value,
	structValue reflect.Value,
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

	// Skip hashing if field has zero value and set empty hash
	if fieldValue.IsZero() {
		targetField.SetString("")
		return nil
	}

	// Hash based on field type
	switch originalType.Kind() {
	case reflect.String:
		targetField.SetString(Basic(fieldValue.String()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		targetField.SetString(Basic(strconv.FormatInt(fieldValue.Int(), 10)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		targetField.SetString(Basic(strconv.FormatUint(fieldValue.Uint(), 10)))
	case reflect.Float32, reflect.Float64:
		targetField.SetString(Basic(strconv.FormatFloat(fieldValue.Float(), 'f', -1, 64)))
	default:
		if field.Type == reflect.TypeOf(time.Time{}) {
			timeValue, ok := fieldValue.Interface().(time.Time)
			if !ok {
				return encxerr.NewTypeConversionError(field.Name, "time.Time", encxerr.BasicHash)
			}
			targetField.SetString(Basic(timeValue.Format(time.RFC3339)))
		} else {
			return encxerr.NewUnsupportedTypeError(field.Name, field.Type.String(), encxerr.BasicHash)

		}
	}

	return nil
}

func Basic(value string) string {
	valueHash := sha256.Sum256([]byte(strings.ToLower(value)))
	return hex.EncodeToString(valueHash[:])
}
