package serialization

import (
	"encoding/binary"
	"fmt"
	"math"
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
		return fmt.Errorf("unsupported target type for compact deserialization: %T", target)
	}
}
