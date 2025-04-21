package encx

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// Serializer defines an interface for converting Go data types to and from byte arrays.
// Implementations of this interface handle the encoding of struct fields before
// encryption or hashing, and potentially the decoding after decryption.
type Serializer interface {
	// Serialize takes a reflect.Value representing a field and returns its byte
	// representation and an error if serialization fails. Different implementations
	// offer varying trade-offs in terms of performance, size, and interoperability.
	Serialize(v reflect.Value) ([]byte, error)

	// Deserialize takes a byte array and a reflect.Value (pointer to the field)
	// and populates the field with the deserialized data. This method is optional
	// if the package user handles deserialization outside of the core processing.
	Deserialize(data []byte, v reflect.Value) error
}

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
	// Implement JSON deserialization logic if needed
	return json.Unmarshal(data, v.Addr().Interface())
}

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

// GOBSerializer implements the Serializer interface using the encoding/gob package.
// It offers efficient binary encoding specifically for Go data types, often resulting
// in smaller sizes and faster performance than JSON. However, it has limited
// interoperability with non-Go systems. Choose this if performance and Go-specific
// handling are primary concerns and cross-language compatibility is not required.
type GOBSerializer struct{}

func (g GOBSerializer) Serialize(v reflect.Value) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v.Interface()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GOBSerializer) Deserialize(data []byte, v reflect.Value) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(v.Addr().Interface())
}
