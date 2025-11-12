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
			name:  "[]string",
			value: []string{"hello", "world", "test"},
			check: func(t *testing.T, original, deserialized any) {
				var result []string
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				orig := original.([]string)
				if len(result) != len(orig) {
					t.Fatalf("Expected length %d, got %d", len(orig), len(result))
				}
				for i := range orig {
					if result[i] != orig[i] {
						t.Errorf("Expected %v at index %d, got %v", orig[i], i, result[i])
					}
				}
			},
		},
		{
			name:  "empty []string",
			value: []string{},
			check: func(t *testing.T, original, deserialized any) {
				var result []string
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if len(result) != 0 {
					t.Errorf("Expected empty slice, got %v", result)
				}
			},
		},
		{
			name:  "[]string with empty strings",
			value: []string{"", "hello", "", "world", ""},
			check: func(t *testing.T, original, deserialized any) {
				var result []string
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				orig := original.([]string)
				if len(result) != len(orig) {
					t.Fatalf("Expected length %d, got %d", len(orig), len(result))
				}
				for i := range orig {
					if result[i] != orig[i] {
						t.Errorf("Expected %v at index %d, got %v", orig[i], i, result[i])
					}
				}
			},
		},
		{
			name:  "[]string with unicode",
			value: []string{"hello", "世界", "🔥", "test"},
			check: func(t *testing.T, original, deserialized any) {
				var result []string
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				orig := original.([]string)
				if len(result) != len(orig) {
					t.Fatalf("Expected length %d, got %d", len(orig), len(result))
				}
				for i := range orig {
					if result[i] != orig[i] {
						t.Errorf("Expected %v at index %d, got %v", orig[i], i, result[i])
					}
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

// Test type aliases (like enum types)
type Role string
type SessionState string
type UserID int64
type Priority int8
type Port uint16
type Tags []string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

const (
	SessionStateActive   SessionState = "active"
	SessionStateInactive SessionState = "inactive"
)

func TestSerializeDeserializeTypeAliases(t *testing.T) {
	tests := []struct {
		name  string
		value any
		check func(t *testing.T, original any)
	}{
		{
			name:  "Role type alias",
			value: RoleAdmin,
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result Role
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.(Role) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "SessionState type alias",
			value: SessionStateActive,
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result SessionState
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.(SessionState) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "UserID type alias (int64)",
			value: UserID(12345),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result UserID
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.(UserID) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "Priority type alias (int8)",
			value: Priority(5),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result Priority
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.(Priority) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "Port type alias (uint16)",
			value: Port(8080),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result Port
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.(Port) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "Tags type alias ([]string)",
			value: Tags{"admin", "user", "moderator"},
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result Tags
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				orig := original.(Tags)
				if len(result) != len(orig) {
					t.Fatalf("Expected length %d, got %d", len(orig), len(result))
				}
				for i := range orig {
					if result[i] != orig[i] {
						t.Errorf("Expected %v at index %d, got %v", orig[i], i, result[i])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.value)
		})
	}
}

// TestTypeAliasCompatibility ensures that type aliases serialize to the same
// bytes as their underlying type
func TestTypeAliasCompatibility(t *testing.T) {
	role := RoleAdmin
	str := string(role)

	roleData, err := Serialize(role)
	if err != nil {
		t.Fatalf("Failed to serialize role: %v", err)
	}

	strData, err := Serialize(str)
	if err != nil {
		t.Fatalf("Failed to serialize string: %v", err)
	}

	if !bytes.Equal(roleData, strData) {
		t.Errorf("Type alias should serialize to same bytes as underlying type")
	}
}

// Test UUID-like types (byte arrays)
type UUID [16]byte

func TestSerializeDeserializeUUID(t *testing.T) {
	// Create a sample UUID
	uuid := UUID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10}

	// Serialize
	data, err := Serialize(uuid)
	if err != nil {
		t.Fatalf("Failed to serialize UUID: %v", err)
	}

	// Deserialize
	var result UUID
	err = Deserialize(data, &result)
	if err != nil {
		t.Fatalf("Failed to deserialize UUID: %v", err)
	}

	// Verify
	if result != uuid {
		t.Errorf("Expected %v, got %v", uuid, result)
	}
}

func TestUUIDSerializesToByteSlice(t *testing.T) {
	// UUID should serialize to the same format as []byte
	uuid := UUID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	slice := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	uuidData, err := Serialize(uuid)
	if err != nil {
		t.Fatalf("Failed to serialize UUID: %v", err)
	}

	sliceData, err := Serialize(slice)
	if err != nil {
		t.Fatalf("Failed to serialize []byte: %v", err)
	}

	if !bytes.Equal(uuidData, sliceData) {
		t.Errorf("UUID should serialize to same bytes as []byte")
		t.Errorf("UUID data:  %v", uuidData)
		t.Errorf("Slice data: %v", sliceData)
	}
}

func TestByteArrays(t *testing.T) {
	tests := []struct {
		name  string
		value any
		check func(t *testing.T, original any)
	}{
		{
			name:  "[4]byte array",
			value: [4]byte{0xde, 0xad, 0xbe, 0xef},
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result [4]byte
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.([4]byte) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
		{
			name:  "[32]byte array",
			value: [32]byte{0x01, 0x02, 0x03, 0x04, 0x05},
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result [32]byte
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != original.([32]byte) {
					t.Errorf("Expected %v, got %v", original, result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.value)
		})
	}
}

// TestPointerTypes tests serialization and deserialization of pointer types
func TestPointerTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		check func(t *testing.T, original any)
	}{
		{
			name: "*time.Time non-nil",
			value: func() *time.Time {
				tm := time.Date(2024, 9, 23, 12, 30, 45, 123456789, time.UTC)
				return &tm
			}(),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *time.Time
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				origPtr := original.(*time.Time)
				if result == nil {
					t.Fatal("Expected non-nil pointer, got nil")
				}
				if !result.Equal(*origPtr) {
					t.Errorf("Expected %v, got %v", *origPtr, *result)
				}
			},
		},
		{
			name:  "*time.Time nil",
			value: (*time.Time)(nil),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *time.Time
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != nil {
					t.Errorf("Expected nil pointer, got %v", result)
				}
			},
		},
		{
			name: "*string non-nil",
			value: func() *string {
				s := "hello world"
				return &s
			}(),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *string
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				origPtr := original.(*string)
				if result == nil {
					t.Fatal("Expected non-nil pointer, got nil")
				}
				if *result != *origPtr {
					t.Errorf("Expected %v, got %v", *origPtr, *result)
				}
			},
		},
		{
			name:  "*string nil",
			value: (*string)(nil),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *string
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != nil {
					t.Errorf("Expected nil pointer, got %v", result)
				}
			},
		},
		{
			name: "*int64 non-nil",
			value: func() *int64 {
				i := int64(42)
				return &i
			}(),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *int64
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				origPtr := original.(*int64)
				if result == nil {
					t.Fatal("Expected non-nil pointer, got nil")
				}
				if *result != *origPtr {
					t.Errorf("Expected %v, got %v", *origPtr, *result)
				}
			},
		},
		{
			name:  "*int64 nil",
			value: (*int64)(nil),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *int64
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != nil {
					t.Errorf("Expected nil pointer, got %v", result)
				}
			},
		},
		{
			name: "*bool non-nil",
			value: func() *bool {
				b := true
				return &b
			}(),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *bool
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				origPtr := original.(*bool)
				if result == nil {
					t.Fatal("Expected non-nil pointer, got nil")
				}
				if *result != *origPtr {
					t.Errorf("Expected %v, got %v", *origPtr, *result)
				}
			},
		},
		{
			name: "*float64 non-nil",
			value: func() *float64 {
				f := 3.14159
				return &f
			}(),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *float64
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				origPtr := original.(*float64)
				if result == nil {
					t.Fatal("Expected non-nil pointer, got nil")
				}
				if *result != *origPtr {
					t.Errorf("Expected %v, got %v", *origPtr, *result)
				}
			},
		},
		{
			name: "*Role type alias non-nil",
			value: func() *Role {
				r := RoleAdmin
				return &r
			}(),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *Role
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				origPtr := original.(*Role)
				if result == nil {
					t.Fatal("Expected non-nil pointer, got nil")
				}
				if *result != *origPtr {
					t.Errorf("Expected %v, got %v", *origPtr, *result)
				}
			},
		},
		{
			name:  "*Role type alias nil",
			value: (*Role)(nil),
			check: func(t *testing.T, original any) {
				data, err := Serialize(original)
				if err != nil {
					t.Fatalf("Serialize failed: %v", err)
				}

				var result *Role
				err = Deserialize(data, &result)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				if result != nil {
					t.Errorf("Expected nil pointer, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.value)
		})
	}
}
