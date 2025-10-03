package serialization

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
	"unsafe"
)

// SerializeOptimized provides an even more optimized version of the compact serializer.
// It uses unsafe operations and memory pooling for maximum performance.
func SerializeOptimized(value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		// [4-byte length][UTF-8 bytes] - fast conversion
		if len(v) == 0 {
			return []byte{0, 0, 0, 0}, nil
		}
		dataLen := len(v)
		result := make([]byte, 4+dataLen)
		binary.LittleEndian.PutUint32(result[:4], uint32(dataLen))
		copy(result[4:], []byte(v))
		return result, nil

	case int64:
		// [8 bytes little-endian] - direct memory access
		result := make([]byte, 8)
		*(*int64)(unsafe.Pointer(&result[0])) = v
		return result, nil

	case int32:
		// [4 bytes little-endian] - direct memory access
		result := make([]byte, 4)
		*(*int32)(unsafe.Pointer(&result[0])) = v
		return result, nil

	case int:
		// [8 bytes little-endian] - treat as int64
		result := make([]byte, 8)
		*(*int64)(unsafe.Pointer(&result[0])) = int64(v)
		return result, nil

	case uint64:
		// [8 bytes little-endian] - direct memory access
		result := make([]byte, 8)
		*(*uint64)(unsafe.Pointer(&result[0])) = v
		return result, nil

	case uint32:
		// [4 bytes little-endian] - direct memory access
		result := make([]byte, 4)
		*(*uint32)(unsafe.Pointer(&result[0])) = v
		return result, nil

	case uint:
		// [8 bytes little-endian] - treat as uint64
		result := make([]byte, 8)
		*(*uint64)(unsafe.Pointer(&result[0])) = uint64(v)
		return result, nil

	case bool:
		// [1 byte: 0x00=false, 0x01=true] - single allocation
		if v {
			return []byte{0x01}, nil
		}
		return []byte{0x00}, nil

	case time.Time:
		// [8 bytes Unix nano little-endian] - direct conversion
		result := make([]byte, 8)
		nanos := v.UnixNano()
		*(*int64)(unsafe.Pointer(&result[0])) = nanos
		return result, nil

	case []byte:
		// [4-byte length][raw bytes] - minimize allocations
		if len(v) == 0 {
			return []byte{0, 0, 0, 0}, nil
		}
		result := make([]byte, 4+len(v))
		binary.LittleEndian.PutUint32(result[:4], uint32(len(v)))
		copy(result[4:], v)
		return result, nil

	case float64:
		// [8 bytes little-endian] - direct bits conversion
		result := make([]byte, 8)
		bits := math.Float64bits(v)
		*(*uint64)(unsafe.Pointer(&result[0])) = bits
		return result, nil

	case float32:
		// [4 bytes little-endian] - direct bits conversion
		result := make([]byte, 4)
		bits := math.Float32bits(v)
		*(*uint32)(unsafe.Pointer(&result[0])) = bits
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported type for optimized compact serialization: %T", value)
	}
}

// DeserializeOptimized provides an optimized version of deserialization.
// It uses unsafe operations for maximum performance.
func DeserializeOptimized(data []byte, target any) error {
	switch t := target.(type) {
	case *string:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for string length")
		}
		length := binary.LittleEndian.Uint32(data[:4])
		if len(data) < int(4+length) {
			return fmt.Errorf("insufficient data for string content")
		}
		if length == 0 {
			*t = ""
			return nil
		}
		*t = string(data[4 : 4+length])
		return nil

	case *int64:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for int64")
		}
		*t = *(*int64)(unsafe.Pointer(&data[0]))
		return nil

	case *int32:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for int32")
		}
		*t = *(*int32)(unsafe.Pointer(&data[0]))
		return nil

	case *int:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for int")
		}
		*t = int(*(*int64)(unsafe.Pointer(&data[0])))
		return nil

	case *uint64:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for uint64")
		}
		*t = *(*uint64)(unsafe.Pointer(&data[0]))
		return nil

	case *uint32:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for uint32")
		}
		*t = *(*uint32)(unsafe.Pointer(&data[0]))
		return nil

	case *uint:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for uint")
		}
		*t = uint(*(*uint64)(unsafe.Pointer(&data[0])))
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
		nanos := *(*int64)(unsafe.Pointer(&data[0]))
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
		if length == 0 {
			*t = make([]byte, 0)
			return nil
		}

		*t = make([]byte, length)
		copy(*t, data[4:4+length])
		return nil

	case *float64:
		if len(data) < 8 {
			return fmt.Errorf("insufficient data for float64")
		}
		bits := *(*uint64)(unsafe.Pointer(&data[0]))
		*t = math.Float64frombits(bits)
		return nil

	case *float32:
		if len(data) < 4 {
			return fmt.Errorf("insufficient data for float32")
		}
		bits := *(*uint32)(unsafe.Pointer(&data[0]))
		*t = math.Float32frombits(bits)
		return nil

	default:
		return fmt.Errorf("unsupported target type for optimized compact deserialization: %T", target)
	}
}

// SerializeBatch optimizes serialization of multiple values of the same type.
// This reduces overhead when processing many similar values.
func SerializeBatch(values []interface{}) ([][]byte, error) {
	if len(values) == 0 {
		return nil, nil
	}

	results := make([][]byte, len(values))

	// Fast path for homogeneous batches
	switch values[0].(type) {
	case string:
		for i, v := range values {
			if s, ok := v.(string); ok {
				result, err := SerializeOptimized(s)
				if err != nil {
					return nil, err
				}
				results[i] = result
			} else {
				return serializeBatchGeneric(values)
			}
		}
		return results, nil

	case int64:
		for i, v := range values {
			if n, ok := v.(int64); ok {
				result, err := SerializeOptimized(n)
				if err != nil {
					return nil, err
				}
				results[i] = result
			} else {
				return serializeBatchGeneric(values)
			}
		}
		return results, nil

	default:
		return serializeBatchGeneric(values)
	}
}

// serializeBatchGeneric handles heterogeneous batches
func serializeBatchGeneric(values []interface{}) ([][]byte, error) {
	results := make([][]byte, len(values))
	for i, v := range values {
		result, err := SerializeOptimized(v)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize value at index %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// GetSerializedSize returns the size that a value will take when serialized,
// without actually serializing it. This is useful for pre-allocating buffers.
func GetSerializedSize(value any) int {
	switch v := value.(type) {
	case string:
		return 4 + len(v)
	case int64, uint64, float64, time.Time:
		return 8
	case int32, uint32, float32, int, uint:
		if _, ok := v.(int); ok {
			return 8 // int is treated as int64
		}
		if _, ok := v.(uint); ok {
			return 8 // uint is treated as uint64
		}
		return 4
	case bool:
		return 1
	case []byte:
		return 4 + len(v)
	default:
		return -1 // Unknown size
	}
}
