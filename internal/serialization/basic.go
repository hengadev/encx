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

func (d *BasicTypeSerializer) Serialize(v reflect.Value) ([]byte, error) {
	switch v.Kind() {
	case reflect.String:
		return []byte(v.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 32)), nil
	case reflect.Float64:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), nil
	case reflect.Bool:
		return []byte(strconv.FormatBool(v.Bool())), nil
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface().(time.Time).MarshalBinary() // Or use a string format like ISO 8601
		} else {
			// Fallback to JSON for other structs
			return json.Marshal(v.Interface())
		}
	case reflect.Slice, reflect.Map:
		// Handle byte slices directly, otherwise use JSON
		if v.Type().Elem().Kind() == reflect.Uint8 {
			if v.CanAddr() {
				return v.Slice(0, v.Len()).Bytes(), nil
			}
			// Create a copy if not addressable
			b := make([]byte, v.Len())
			reflect.Copy(reflect.ValueOf(b), v)
			return b, nil
		} else {
			return json.Marshal(v.Interface())
		}
	default:
		return nil, fmt.Errorf("unsupported type for serialization: %v", v.Type())
	}
}

func (d *BasicTypeSerializer) Deserialize(data []byte, v reflect.Value) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(string(data))
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse int: %w", err)
		}
		v.SetInt(val)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse uint: %w", err)
		}
		v.SetUint(val)
		return nil
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return fmt.Errorf("failed to parse float: %w", err)
		}
		v.SetFloat(val)
		return nil
	case reflect.Bool:
		val, err := strconv.ParseBool(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse bool: %w", err)
		}
		v.SetBool(val)
		return nil
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			var t time.Time
			if err := t.UnmarshalBinary(data); err != nil {
				return fmt.Errorf("failed to unmarshal time: %w", err)
			}
			v.Set(reflect.ValueOf(t))
			return nil
		} else {
			// Fallback to JSON for other structs
			return json.Unmarshal(data, v.Addr().Interface())
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes(data)
			return nil
		} else {
			return json.Unmarshal(data, v.Addr().Interface())
		}
	case reflect.Map:
		return json.Unmarshal(data, v.Addr().Interface())
	default:
		return fmt.Errorf("unsupported type for deserialization: %v", v.Type())
	}
}

