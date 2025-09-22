package serialization

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializerType_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		serializer SerializerType
		expected   bool
	}{
		{"JSON is valid", JSON, true},
		{"GOB is valid", GOB, true},
		{"Basic is valid", Basic, true},
		{"Empty string is invalid", SerializerType(""), false},
		{"Unknown serializer is invalid", SerializerType("unknown"), false},
		{"Case sensitive - json lowercase", SerializerType("json"), true}, // Direct comparison with constant
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.serializer.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSerializerType_CreateSerializer(t *testing.T) {
	tests := []struct {
		name       string
		serializer SerializerType
		expected   interface{}
	}{
		{"JSON creates JSONSerializer", JSON, &JSONSerializer{}},
		{"GOB creates GOBSerializer", GOB, &GOBSerializer{}},
		{"Basic creates BasicTypeSerializer", Basic, &BasicTypeSerializer{}},
		{"Invalid returns nil", SerializerType("invalid"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.serializer.CreateSerializer()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.IsType(t, tt.expected, result)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSerializerType_RequiredImport(t *testing.T) {
	expected := "github.com/hengadev/encx/internal/serialization"

	tests := []struct {
		name       string
		serializer SerializerType
		expected   string
	}{
		{"JSON import", JSON, expected},
		{"GOB import", GOB, expected},
		{"Basic import", Basic, expected},
		{"Invalid import", SerializerType("invalid"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.serializer.RequiredImport()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSerializerType_SerializerFactory(t *testing.T) {
	tests := []struct {
		name       string
		serializer SerializerType
		expected   string
	}{
		{"JSON factory", JSON, "&serialization.JSONSerializer{}"},
		{"GOB factory", GOB, "&serialization.GOBSerializer{}"},
		{"Basic factory", Basic, "&serialization.BasicTypeSerializer{}"},
		{"Invalid factory", SerializerType("invalid"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.serializer.SerializerFactory()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSerializerType_String(t *testing.T) {
	tests := []struct {
		name       string
		serializer SerializerType
		expected   string
	}{
		{"JSON string", JSON, "json"},
		{"GOB string", GOB, "gob"},
		{"Basic string", Basic, "basic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.serializer.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSerializerType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    SerializerType
		expectError bool
	}{
		{"Valid JSON", "json", JSON, false},
		{"Valid GOB", "gob", GOB, false},
		{"Valid Basic", "basic", Basic, false},
		{"Case insensitive JSON", "JSON", JSON, false},
		{"Case insensitive GOB", "GOB", GOB, false},
		{"Case insensitive Basic", "BASIC", Basic, false},
		{"Whitespace trimmed", "  json  ", JSON, false},
		{"Mixed case", "Json", JSON, false},
		{"Invalid serializer", "invalid", SerializerType(""), true},
		{"Empty string", "", SerializerType(""), true},
		{"Only whitespace", "   ", SerializerType(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSerializerType(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, SerializerType(""), result)
				assert.Contains(t, err.Error(), "invalid serializer type")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAllSerializerTypes(t *testing.T) {
	result := AllSerializerTypes()

	expected := []SerializerType{JSON, GOB, Basic}
	assert.Equal(t, expected, result)
	assert.Len(t, result, 3)

	// Ensure all returned types are valid
	for _, serializer := range result {
		assert.True(t, serializer.IsValid(), "serializer %s should be valid", serializer)
	}
}

func TestSerializerType_EndToEnd(t *testing.T) {
	// Test the complete workflow
	input := "gob"

	// Parse from string
	serializerType, err := ParseSerializerType(input)
	require.NoError(t, err)
	assert.Equal(t, GOB, serializerType)

	// Verify it's valid
	assert.True(t, serializerType.IsValid())

	// Create serializer instance
	serializer := serializerType.CreateSerializer()
	assert.IsType(t, &GOBSerializer{}, serializer)

	// Get factory string
	factory := serializerType.SerializerFactory()
	assert.Equal(t, "&serialization.GOBSerializer{}", factory)

	// Get import
	importPath := serializerType.RequiredImport()
	assert.Equal(t, "github.com/hengadev/encx/internal/serialization", importPath)

	// Convert back to string
	stringForm := serializerType.String()
	assert.Equal(t, "gob", stringForm)
}