package serialization

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// JSONSerializer implements the Serializer interface using the encoding/json package.
// It provides good compatibility with complex data structures and decent human readability
// (of the serialized form), but might have a slight performance overhead for basic types
// compared to more direct conversions. It is a good default choice for general use.
type JSONSerializer struct{}

func (j JSONSerializer) Serialize(v reflect.Value) ([]byte, error) {
	// Implement JSON serialization logic here (similar to your previous serializeField)
	switch v.Kind() {
	case reflect.String:
		return []byte(v.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	// ... (other basic types)
	default:
		return json.Marshal(v.Interface())
	}
}

func (j JSONSerializer) Deserialize(data []byte, v reflect.Value) error {
	if v.Kind() == reflect.String {
		v.SetString(string(data))
		return nil
	}
	if v.Kind() == reflect.Int || v.Kind() == reflect.Int64 || v.Kind() == reflect.Int32 || v.Kind() == reflect.Int16 || v.Kind() == reflect.Int8 {
		var val int64
		if err := json.Unmarshal(data, &val); err != nil {
			return fmt.Errorf("failed to unmarshal int: %w", err)
		}
		v.SetInt(val)
		return nil
	}
	if v.Kind() == reflect.Bool {
		var val bool
		if err := json.Unmarshal(data, &val); err != nil {
			return fmt.Errorf("failed to unmarshal bool: %w", err)
		}
		v.SetBool(val)
		return nil
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		v.SetBytes(data)
		return nil
	}
	// Handle other basic types as needed
	return json.Unmarshal(data, v.Addr().Interface())
}

