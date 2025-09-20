package serialization

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// BasicTypeSerializer implements the Serializer interface by directly converting
// basic types (string, numbers, bool) to their ASCII string representations and
// using encoding/json as a fallback for more complex types (struct, slice, map).
// It can be more efficient for simple fields but might lead to inconsistent
// serialization of complex structures. Use this if you primarily deal with flat
// data structures and need optimal performance for basic types.
type BasicTypeSerializer struct{}

func (d *BasicTypeSerializer) Serialize(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return []byte(rv.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(rv.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(rv.Uint(), 10)), nil
	case reflect.Float32:
		return []byte(strconv.FormatFloat(rv.Float(), 'f', -1, 32)), nil
	case reflect.Float64:
		return []byte(strconv.FormatFloat(rv.Float(), 'f', -1, 64)), nil
	case reflect.Bool:
		return []byte(strconv.FormatBool(rv.Bool())), nil
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(time.Time{}) {
			return v.(time.Time).MarshalBinary()
		} else {
			// Fallback to JSON for other structs
			return json.Marshal(v)
		}
	case reflect.Slice, reflect.Map:
		// Handle byte slices directly, otherwise use JSON
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return v.([]byte), nil
		} else {
			return json.Marshal(v)
		}
	default:
		return nil, fmt.Errorf("unsupported type for serialization: %v", rv.Type())
	}
}

func (d *BasicTypeSerializer) Deserialize(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer, got %v", rv.Kind())
	}
	elem := rv.Elem()

	switch elem.Kind() {
	case reflect.String:
		elem.SetString(string(data))
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse int: %w", err)
		}
		elem.SetInt(val)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse uint: %w", err)
		}
		elem.SetUint(val)
		return nil
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return fmt.Errorf("failed to parse float: %w", err)
		}
		elem.SetFloat(val)
		return nil
	case reflect.Bool:
		val, err := strconv.ParseBool(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse bool: %w", err)
		}
		elem.SetBool(val)
		return nil
	case reflect.Struct:
		if elem.Type() == reflect.TypeOf(time.Time{}) {
			var t time.Time
			if err := t.UnmarshalBinary(data); err != nil {
				return fmt.Errorf("failed to unmarshal time: %w", err)
			}
			elem.Set(reflect.ValueOf(t))
			return nil
		} else {
			// Fallback to JSON for other structs
			return json.Unmarshal(data, v)
		}
	case reflect.Slice:
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			elem.SetBytes(data)
			return nil
		} else {
			return json.Unmarshal(data, v)
		}
	case reflect.Map:
		return json.Unmarshal(data, v)
	default:
		return fmt.Errorf("unsupported type for deserialization: %v", elem.Type())
	}
}

