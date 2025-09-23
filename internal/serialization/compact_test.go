package serialization

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSerializeDeserialize(t *testing.T) {
	tests := []struct {
		name  string
		value any
		check func(t *testing.T, original, deserialized any)
	}{
		{
			name:  "string",
			value: "hello world",
			check: func(t *testing.T, original, deserialized any) {
				var result string
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(string) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "empty string",
			value: "",
			check: func(t *testing.T, original, deserialized any) {
				var result string
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(string) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "int64",
			value: int64(42),
			check: func(t *testing.T, original, deserialized any) {
				var result int64
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(int64) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "int32",
			value: int32(123),
			check: func(t *testing.T, original, deserialized any) {
				var result int32
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(int32) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "int",
			value: int(456),
			check: func(t *testing.T, original, deserialized any) {
				var result int
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(int) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "bool true",
			value: true,
			check: func(t *testing.T, original, deserialized any) {
				var result bool
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(bool) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "bool false",
			value: false,
			check: func(t *testing.T, original, deserialized any) {
				var result bool
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(bool) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "time.Time",
			value: time.Date(2024, 9, 23, 12, 30, 45, 123456789, time.UTC),
			check: func(t *testing.T, original, deserialized any) {
				var result time.Time
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if !result.Equal(original.(time.Time)) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "[]byte",
			value: []byte{1, 2, 3, 4, 5},
			check: func(t *testing.T, original, deserialized any) {
				var result []byte
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if !bytes.Equal(result, original.([]byte)) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "empty []byte",
			value: []byte{},
			check: func(t *testing.T, original, deserialized any) {
				var result []byte
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if !bytes.Equal(result, original.([]byte)) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "float64",
			value: float64(3.14159),
			check: func(t *testing.T, original, deserialized any) {
				var result float64
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(float64) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "float32",
			value: float32(2.718),
			check: func(t *testing.T, original, deserialized any) {
				var result float32
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if result != original.(float32) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.value, nil)
		})
	}
}

func TestSerializeUnsupportedType(t *testing.T) {
	type unsupported struct {
		Field string
	}

	_, err := Serialize(unsupported{Field: "test"})
	if err == nil {
		t.Error("Expected error for unsupported type, got nil")
	}

	expectedMsg := "unsupported type for compact serialization"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedMsg, err.Error())
	}
}

func TestDeserializeUnsupportedType(t *testing.T) {
	type unsupported struct {
		Field string
	}

	data := []byte{1, 2, 3, 4}
	var target unsupported

	err := Deserialize(data, &target)
	if err == nil {
		t.Error("Expected error for unsupported type, got nil")
	}

	expectedMsg := "unsupported target type for compact deserialization"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedMsg, err.Error())
	}
}

func TestDeserializeInsufficientData(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		target any
	}{
		{"string length", []byte{1, 2}, new(string)},
		{"int64", []byte{1, 2, 3}, new(int64)},
		{"int32", []byte{1, 2}, new(int32)},
		{"bool", []byte{}, new(bool)},
		{"time.Time", []byte{1, 2, 3}, new(time.Time)},
		{"[]byte length", []byte{1, 2}, new([]byte)},
		{"float64", []byte{1, 2, 3}, new(float64)},
		{"float32", []byte{1, 2}, new(float32)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Deserialize(tt.data, tt.target)
			if err == nil {
				t.Error("Expected error for insufficient data, got nil")
			}
			if !contains(err.Error(), "insufficient data") {
				t.Errorf("Expected error message to contain 'insufficient data', got %q", err.Error())
			}
		})
	}
}

func TestDeterministicSerialization(t *testing.T) {
	// Test that the same value always produces the same bytes
	value := "test string"

	data1, err1 := Serialize(value)
	if err1 != nil {
		t.Fatalf("First serialization failed: %v", err1)
	}

	data2, err2 := Serialize(value)
	if err2 != nil {
		t.Fatalf("Second serialization failed: %v", err2)
	}

	if !bytes.Equal(data1, data2) {
		t.Error("Serialization is not deterministic")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}