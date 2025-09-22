package serialization

import (
	"fmt"
	"strings"
)

// SerializerType represents the type of serializer to use for data conversion
type SerializerType string

const (
	// JSON uses the standard encoding/json package for serialization
	JSON SerializerType = "json"
	// GOB uses the encoding/gob package for Go-specific binary serialization
	GOB SerializerType = "gob"
	// Basic uses direct conversion for primitive types with JSON fallback
	Basic SerializerType = "basic"
)

// IsValid checks if the serializer type is supported
func (s SerializerType) IsValid() bool {
	switch s {
	case JSON, GOB, Basic:
		return true
	default:
		return false
	}
}

// CreateSerializer creates a new instance of the serializer
func (s SerializerType) CreateSerializer() Serializer {
	switch s {
	case JSON:
		return &JSONSerializer{}
	case GOB:
		return &GOBSerializer{}
	case Basic:
		return &BasicTypeSerializer{}
	default:
		return nil
	}
}

// RequiredImport returns the import path required for this serializer type
func (s SerializerType) RequiredImport() string {
	switch s {
	case JSON, GOB, Basic:
		return "github.com/hengadev/encx/internal/serialization"
	default:
		return ""
	}
}

// String returns the string representation of the serializer type
func (s SerializerType) String() string {
	return string(s)
}

// SerializerFactory returns the factory string for code generation
func (s SerializerType) SerializerFactory() string {
	switch s {
	case JSON:
		return "&serialization.JSONSerializer{}"
	case GOB:
		return "&serialization.GOBSerializer{}"
	case Basic:
		return "&serialization.BasicTypeSerializer{}"
	default:
		return ""
	}
}

// ParseSerializerType parses a string into a SerializerType and validates it
func ParseSerializerType(s string) (SerializerType, error) {
	serializerType := SerializerType(strings.ToLower(strings.TrimSpace(s)))

	if !serializerType.IsValid() {
		return "", fmt.Errorf("invalid serializer type '%s': must be one of [%s, %s, %s]",
			s, JSON, GOB, Basic)
	}

	return serializerType, nil
}

// AllSerializerTypes returns all supported serializer types
func AllSerializerTypes() []SerializerType {
	return []SerializerType{JSON, GOB, Basic}
}