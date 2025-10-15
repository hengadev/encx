package serialization

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"time"
)

// Serialize converts a value to bytes using a compact binary format.
// This is optimized for deterministic output and performance.
func Serialize(value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		// [4-byte length][UTF-8 bytes]
		data := []byte(v)
		result := make([]byte, 4+len(data))
		binary.LittleEndian.PutUint32(result[:4], uint32(len(data)))
		copy(result[4:], data)
		return result, nil

	case int64:
		// [8 bytes little-endian]
		result := make([]byte, 8)
		binary.LittleEndian.PutUint64(result, uint64(v))
		return result, nil

	case int32:
		// [4 bytes little-endian]
		result := make([]byte, 4)
		binary.LittleEndian.PutUint32(result, uint32(v))
		return result, nil

	case int:
		// [8 bytes little-endian] (treat as int64)
		result := make([]byte, 8)
		binary.LittleEndian.PutUint64(result, uint64(v))
		return result, nil

	case uint64:
		// [8 bytes little-endian]
		result := make([]byte, 8)
		binary.LittleEndian.PutUint64(result, v)
		return result, nil

	case uint32:
		// [4 bytes little-endian]
		result := make([]byte, 4)
		binary.LittleEndian.PutUint32(result, v)
		return result, nil

	case uint:
		// [8 bytes little-endian] (treat as uint64)
		result := make([]byte, 8)
		binary.LittleEndian.PutUint64(result, uint64(v))
		return result, nil

	case bool:
		// [1 byte: 0x00=false, 0x01=true]
		if v {
			return []byte{0x01}, nil
		}
		return []byte{0x00}, nil

	case time.Time:
		// [8 bytes Unix nano little-endian]
		result := make([]byte, 8)
		binary.LittleEndian.PutUint64(result, uint64(v.UnixNano()))
		return result, nil

	case []byte:
		// [4-byte length][raw bytes]
		result := make([]byte, 4+len(v))
		binary.LittleEndian.PutUint32(result[:4], uint32(len(v)))
		copy(result[4:], v)
		return result, nil

	case float64:
		// [8 bytes little-endian]
		result := make([]byte, 8)
		binary.LittleEndian.PutUint64(result, math.Float64bits(v))
		return result, nil

	case float32:
		// [4 bytes little-endian]
		result := make([]byte, 4)
		binary.LittleEndian.PutUint32(result, math.Float32bits(v))
		return result, nil

	default:
		// Fall back to reflection for custom types and type aliases
		return serializeWithReflection(value)
	}
}

// serializeWithReflection handles serialization of custom types and type aliases
// by examining their underlying kind using reflection.
func serializeWithReflection(value any) ([]byte, error) {
	rv := reflect.ValueOf(value)

	switch rv.Kind() {
	case reflect.Ptr:
		// Handle pointer types
		if rv.IsNil() {
			// Serialize nil pointer as a single byte marker (0x00) followed by nothing
			return []byte{0x00}, nil
		}
		// Dereference and serialize the underlying value
		// Prefix with 0x01 to indicate non-nil pointer
		underlying := rv.Elem().Interface()
		serialized, err := Serialize(underlying)
		if err != nil {
			return nil, err
		}
		result := make([]byte, 1+len(serialized))
		result[0] = 0x01 // Non-nil marker
		copy(result[1:], serialized)
		return result, nil

	case reflect.String:
		// Handle string-based type aliases (e.g., type Role string)
		return Serialize(rv.String())

	case reflect.Int64:
		return Serialize(rv.Int())

	case reflect.Int32:
		return Serialize(int32(rv.Int()))

	case reflect.Int16:
		return Serialize(int32(rv.Int()))

	case reflect.Int8:
		return Serialize(int32(rv.Int()))

	case reflect.Int:
		return Serialize(int(rv.Int()))

	case reflect.Uint64:
		return Serialize(rv.Uint())

	case reflect.Uint32:
		return Serialize(uint32(rv.Uint()))

	case reflect.Uint16:
		return Serialize(uint32(rv.Uint()))

	case reflect.Uint8:
		return Serialize(uint32(rv.Uint()))

	case reflect.Uint:
		return Serialize(uint(rv.Uint()))

	case reflect.Bool:
		return Serialize(rv.Bool())

	case reflect.Float64:
		return Serialize(rv.Float())

	case reflect.Float32:
		return Serialize(float32(rv.Float()))

	case reflect.Array:
		// Handle array types (e.g., UUID which is [16]byte)
		// Convert to []byte slice
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			slice := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				slice[i] = uint8(rv.Index(i).Uint())
			}
			return Serialize(slice)
		}
		return nil, fmt.Errorf("unsupported type for compact serialization: %T (array of non-byte)", value)

	case reflect.Slice:
		// Handle custom slice types
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return Serialize(rv.Bytes())
		}
		return nil, fmt.Errorf("unsupported type for compact serialization: %T (slice of non-byte)", value)

	default:
		return nil, fmt.Errorf("unsupported type for compact serialization: %T", value)
	}
}

// Deserialize converts bytes back to a value of the specified type.
// The caller must specify the expected type.
func Deserialize(data []byte, target any) error {
	switch t := target.(type) {
	case *string:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for string length")
		}
		length := binary.LittleEndian.Uint32(data[:4])
		if len(data) < int(4+length) {
			return fmt.Errorf("insufficient data for string content")
		}
		*t = string(data[4 : 4+length])
		return nil

	case *int64:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for int64")
		}
		*t = int64(binary.LittleEndian.Uint64(data[:8]))
		return nil

	case *int32:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for int32")
		}
		*t = int32(binary.LittleEndian.Uint32(data[:4]))
		return nil

	case *int:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for int")
		}
		*t = int(binary.LittleEndian.Uint64(data[:8]))
		return nil

	case *uint64:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for uint64")
		}
		*t = binary.LittleEndian.Uint64(data[:8])
		return nil

	case *uint32:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for uint32")
		}
		*t = binary.LittleEndian.Uint32(data[:4])
		return nil

	case *uint:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for uint")
		}
		*t = uint(binary.LittleEndian.Uint64(data[:8]))
		return nil

	case *bool:
		if len(data) < 1 {
			return fmt.Errorf("insufficient data for bool")
		}
		*t = data[0] != 0x00
		return nil

	case *time.Time:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for time.Time")
		}
		nanos := int64(binary.LittleEndian.Uint64(data[:8]))
		*t = time.Unix(0, nanos)
		return nil

	case *[]byte:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for []byte length")
		}
		length := binary.LittleEndian.Uint32(data[:4])
		if len(data) < int(4+length) {
			return fmt.Errorf("insufficient data for []byte content")
		}
		*t = make([]byte, length)
		copy(*t, data[4:4+length])
		return nil

	case *float64:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for float64")
		}
		bits := binary.LittleEndian.Uint64(data[:8])
		*t = math.Float64frombits(bits)
		return nil

	case *float32:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for float32")
		}
		bits := binary.LittleEndian.Uint32(data[:4])
		*t = math.Float32frombits(bits)
		return nil

	default:
		// Fall back to reflection for custom types and type aliases
		return deserializeWithReflection(data, target)
	}
}

// deserializeWithReflection handles deserialization of custom types and type aliases
// by examining their underlying kind using reflection.
func deserializeWithReflection(data []byte, target any) error {
	rv := reflect.ValueOf(target)

	// Target must be a pointer
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer, got %T", target)
	}

	// Get the element the pointer points to
	elem := rv.Elem()
	if !elem.CanSet() {
		return fmt.Errorf("target element cannot be set")
	}

	switch elem.Kind() {
	case reflect.Ptr:
		// Handle pointer types
		if len(data) < 1 {
			return fmt.Errorf("insufficient data for pointer marker")
		}
		marker := data[0]
		if marker == 0x00 {
			// Nil pointer - set to zero value (nil)
			elem.Set(reflect.Zero(elem.Type()))
			return nil
		} else if marker == 0x01 {
			// Non-nil pointer - deserialize the underlying value
			// Create a new value of the pointed-to type
			pointedType := elem.Type().Elem()
			newValue := reflect.New(pointedType)

			// Deserialize into the new value
			if err := Deserialize(data[1:], newValue.Interface()); err != nil {
				return err
			}

			// Set the pointer to point to the new value
			elem.Set(newValue)
			return nil
		}
		return fmt.Errorf("invalid pointer marker byte: 0x%02x", marker)

	case reflect.String:
		// Handle string-based type aliases (e.g., type Role string)
		var s string
		if err := Deserialize(data, &s); err != nil {
			return err
		}
		elem.SetString(s)
		return nil

	case reflect.Int64:
		var i int64
		if err := Deserialize(data, &i); err != nil {
			return err
		}
		elem.SetInt(i)
		return nil

	case reflect.Int32:
		var i int32
		if err := Deserialize(data, &i); err != nil {
			return err
		}
		elem.SetInt(int64(i))
		return nil

	case reflect.Int16:
		var i int32
		if err := Deserialize(data, &i); err != nil {
			return err
		}
		elem.SetInt(int64(i))
		return nil

	case reflect.Int8:
		var i int32
		if err := Deserialize(data, &i); err != nil {
			return err
		}
		elem.SetInt(int64(i))
		return nil

	case reflect.Int:
		var i int
		if err := Deserialize(data, &i); err != nil {
			return err
		}
		elem.SetInt(int64(i))
		return nil

	case reflect.Uint64:
		var u uint64
		if err := Deserialize(data, &u); err != nil {
			return err
		}
		elem.SetUint(u)
		return nil

	case reflect.Uint32:
		var u uint32
		if err := Deserialize(data, &u); err != nil {
			return err
		}
		elem.SetUint(uint64(u))
		return nil

	case reflect.Uint16:
		var u uint32
		if err := Deserialize(data, &u); err != nil {
			return err
		}
		elem.SetUint(uint64(u))
		return nil

	case reflect.Uint8:
		var u uint32
		if err := Deserialize(data, &u); err != nil {
			return err
		}
		elem.SetUint(uint64(u))
		return nil

	case reflect.Uint:
		var u uint
		if err := Deserialize(data, &u); err != nil {
			return err
		}
		elem.SetUint(uint64(u))
		return nil

	case reflect.Bool:
		var b bool
		if err := Deserialize(data, &b); err != nil {
			return err
		}
		elem.SetBool(b)
		return nil

	case reflect.Float64:
		var f float64
		if err := Deserialize(data, &f); err != nil {
			return err
		}
		elem.SetFloat(f)
		return nil

	case reflect.Float32:
		var f float32
		if err := Deserialize(data, &f); err != nil {
			return err
		}
		elem.SetFloat(float64(f))
		return nil

	case reflect.Array:
		// Handle array types (e.g., UUID which is [16]byte)
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			var slice []byte
			if err := Deserialize(data, &slice); err != nil {
				return err
			}
			if len(slice) != elem.Len() {
				return fmt.Errorf("array length mismatch: expected %d, got %d", elem.Len(), len(slice))
			}
			for i := 0; i < elem.Len(); i++ {
				elem.Index(i).SetUint(uint64(slice[i]))
			}
			return nil
		}
		return fmt.Errorf("unsupported target type for compact deserialization: %T (array of non-byte)", target)

	case reflect.Slice:
		// Handle custom slice types
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			var slice []byte
			if err := Deserialize(data, &slice); err != nil {
				return err
			}
			elem.SetBytes(slice)
			return nil
		}
		return fmt.Errorf("unsupported target type for compact deserialization: %T (slice of non-byte)", target)

	default:
		return fmt.Errorf("unsupported target type for compact deserialization: %T", target)
	}
}
