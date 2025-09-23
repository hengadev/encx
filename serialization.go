package encx

import (
	"fmt"

	"github.com/hengadev/encx/internal/serialization"
)

// SerializeValue converts a value to bytes using ENCX's compact binary format.
// This is the same serialization used internally by ENCX for field encryption.
// Applications can use this function to create consistent hash values for database queries.
//
// Supported types: string, int, int32, int64, uint, uint32, uint64, bool, time.Time, []byte, float32, float64
//
// Example:
//   data, err := encx.SerializeValue("hello world")
//   if err != nil {
//       // handle error
//   }
func SerializeValue(value any) ([]byte, error) {
	if value == nil {
		return nil, ErrNilPointer
	}

	data, err := serialization.Serialize(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}

	return data, nil
}

// DeserializeValue converts bytes back to a value using ENCX's compact binary format.
// The target parameter must be a pointer to the type you want to deserialize into.
//
// Example:
//   var result string
//   err := encx.DeserializeValue(data, &result)
//   if err != nil {
//       // handle error
//   }
func DeserializeValue(data []byte, target any) error {
	if target == nil {
		return ErrNilPointer
	}

	err := serialization.Deserialize(data, target)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	return nil
}
