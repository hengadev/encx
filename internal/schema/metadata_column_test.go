package schema

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetadataColumn(t *testing.T) {
	mc := NewMetadataColumn()

	assert.NotNil(t, mc)
	assert.NotNil(t, mc.data)
	assert.True(t, mc.IsEmpty())
}

func TestMetadataColumn_SetAndGet(t *testing.T) {
	mc := NewMetadataColumn()

	// Test setting and getting string value
	mc.Set("key1", "value1")
	val, exists := mc.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	// Test setting and getting int value
	mc.Set("key2", 42)
	val, exists = mc.Get("key2")
	assert.True(t, exists)
	assert.Equal(t, 42, val)

	// Test getting non-existent key
	val, exists = mc.Get("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestMetadataColumn_GetString(t *testing.T) {
	mc := NewMetadataColumn()

	// Test getting string value
	mc.Set("string_key", "test_value")
	val, ok := mc.GetString("string_key")
	assert.True(t, ok)
	assert.Equal(t, "test_value", val)

	// Test getting non-string value
	mc.Set("int_key", 42)
	val, ok = mc.GetString("int_key")
	assert.False(t, ok)
	assert.Equal(t, "", val)

	// Test getting non-existent key
	val, ok = mc.GetString("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestMetadataColumn_GetInt(t *testing.T) {
	mc := NewMetadataColumn()

	// Test getting int value
	mc.Set("int_key", 42)
	val, ok := mc.GetInt("int_key")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	// Test getting float64 value (JSON unmarshaling often produces float64)
	mc.Set("float_key", 123.0)
	val, ok = mc.GetInt("float_key")
	assert.True(t, ok)
	assert.Equal(t, 123, val)

	// Test getting int64 value
	mc.Set("int64_key", int64(456))
	val, ok = mc.GetInt("int64_key")
	assert.True(t, ok)
	assert.Equal(t, 456, val)

	// Test getting non-numeric value
	mc.Set("string_key", "not_a_number")
	val, ok = mc.GetInt("string_key")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// Test getting non-existent key
	val, ok = mc.GetInt("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestMetadataColumn_Delete(t *testing.T) {
	mc := NewMetadataColumn()

	// Set a value then delete it
	mc.Set("key1", "value1")
	assert.False(t, mc.IsEmpty())

	mc.Delete("key1")
	_, exists := mc.Get("key1")
	assert.False(t, exists)
	assert.True(t, mc.IsEmpty())

	// Test deleting non-existent key (should not panic)
	mc.Delete("nonexistent")

	// Test deleting from nil data
	mc.data = nil
	mc.Delete("key1") // Should not panic
}

func TestMetadataColumn_Keys(t *testing.T) {
	mc := NewMetadataColumn()

	// Test empty metadata (initialized but empty)
	keys := mc.Keys()
	assert.Empty(t, keys)

	// Test with data
	mc.Set("key1", "value1")
	mc.Set("key2", "value2")
	mc.Set("key3", "value3")

	keys = mc.Keys()
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.Contains(t, keys, "key3")

	// Test with nil data
	mc.data = nil
	keys = mc.Keys()
	assert.Nil(t, keys)
}

func TestMetadataColumn_IsEmpty(t *testing.T) {
	mc := NewMetadataColumn()

	// Initially empty
	assert.True(t, mc.IsEmpty())

	// Add data
	mc.Set("key1", "value1")
	assert.False(t, mc.IsEmpty())

	// Delete data
	mc.Delete("key1")
	assert.True(t, mc.IsEmpty())

	// Test with nil data
	mc.data = nil
	assert.True(t, mc.IsEmpty())
}

func TestMetadataColumn_Clear(t *testing.T) {
	mc := NewMetadataColumn()

	// Add data
	mc.Set("key1", "value1")
	mc.Set("key2", "value2")
	assert.False(t, mc.IsEmpty())

	// Clear data
	mc.Clear()
	assert.True(t, mc.IsEmpty())
	assert.NotNil(t, mc.data) // Should be empty map, not nil
}

func TestMetadataColumn_Scan(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
		expectedLen int
		checkData   func(t *testing.T, mc *MetadataColumn)
	}{
		{
			name:        "scan nil value",
			input:       nil,
			expectError: false,
			expectedLen: 0,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				assert.True(t, mc.IsEmpty())
			},
		},
		{
			name:        "scan empty string",
			input:       "",
			expectError: false,
			expectedLen: 0,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				assert.True(t, mc.IsEmpty())
			},
		},
		{
			name:        "scan empty bytes",
			input:       []byte{},
			expectError: false,
			expectedLen: 0,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				assert.True(t, mc.IsEmpty())
			},
		},
		{
			name:        "scan valid JSON string",
			input:       `{"key1":"value1","key2":42}`,
			expectError: false,
			expectedLen: 2,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				val, ok := mc.GetString("key1")
				assert.True(t, ok)
				assert.Equal(t, "value1", val)

				val2, ok := mc.GetInt("key2")
				assert.True(t, ok)
				assert.Equal(t, 42, val2)
			},
		},
		{
			name:        "scan valid JSON bytes",
			input:       []byte(`{"test":"data","num":123}`),
			expectError: false,
			expectedLen: 2,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				val, ok := mc.GetString("test")
				assert.True(t, ok)
				assert.Equal(t, "data", val)
			},
		},
		{
			name:        "scan invalid JSON",
			input:       `{"invalid": json}`,
			expectError: true,
			expectedLen: 0,
			checkData:   nil,
		},
		{
			name:        "scan unsupported type",
			input:       123,
			expectError: true,
			expectedLen: 0,
			checkData:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMetadataColumn()
			err := mc.Scan(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, mc.data, tt.expectedLen)
				if tt.checkData != nil {
					tt.checkData(t, mc)
				}
			}
		})
	}
}

func TestMetadataColumn_Value(t *testing.T) {
	tests := []struct {
		name        string
		setupData   func(mc *MetadataColumn)
		expectedJSON string
	}{
		{
			name:        "empty metadata",
			setupData:   func(mc *MetadataColumn) {},
			expectedJSON: "{}",
		},
		{
			name: "metadata with data",
			setupData: func(mc *MetadataColumn) {
				mc.Set("key1", "value1")
				mc.Set("key2", 42)
			},
			expectedJSON: `{"key1":"value1","key2":42}`,
		},
		{
			name: "metadata with nil data",
			setupData: func(mc *MetadataColumn) {
				mc.data = nil
			},
			expectedJSON: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMetadataColumn()
			tt.setupData(mc)

			value, err := mc.Value()
			require.NoError(t, err)

			bytes, ok := value.([]byte)
			require.True(t, ok)

			// For non-empty data, parse and compare the structure since JSON field order isn't guaranteed
			if tt.expectedJSON == "{}" {
				assert.Equal(t, tt.expectedJSON, string(bytes))
			} else {
				// Verify it's valid JSON and contains expected data
				testMC := NewMetadataColumn()
				err := testMC.Scan(bytes)
				require.NoError(t, err)

				// Check specific values instead of exact JSON string
				val, ok := testMC.GetString("key1")
				assert.True(t, ok)
				assert.Equal(t, "value1", val)

				val2, ok := testMC.GetInt("key2")
				assert.True(t, ok)
				assert.Equal(t, 42, val2)
			}
		})
	}
}

func TestMetadataColumn_ToJSON(t *testing.T) {
	mc := NewMetadataColumn()

	// Test empty metadata
	jsonBytes, err := mc.ToJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte("{}"), jsonBytes)

	// Test with data
	mc.Set("key1", "value1")
	mc.Set("key2", 42)

	jsonBytes, err = mc.ToJSON()
	assert.NoError(t, err)

	// Parse back to verify
	testMC := NewMetadataColumn()
	err = testMC.Scan(jsonBytes)
	require.NoError(t, err)

	val, ok := testMC.GetString("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test with nil data
	mc.data = nil
	jsonBytes, err = mc.ToJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte("{}"), jsonBytes)
}

func TestMetadataColumn_FromJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
		checkData   func(t *testing.T, mc *MetadataColumn)
	}{
		{
			name:        "valid JSON",
			input:       []byte(`{"key1":"value1","key2":42}`),
			expectError: false,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				val, ok := mc.GetString("key1")
				assert.True(t, ok)
				assert.Equal(t, "value1", val)

				val2, ok := mc.GetInt("key2")
				assert.True(t, ok)
				assert.Equal(t, 42, val2)
			},
		},
		{
			name:        "empty JSON",
			input:       []byte(`{}`),
			expectError: false,
			checkData: func(t *testing.T, mc *MetadataColumn) {
				assert.True(t, mc.IsEmpty())
			},
		},
		{
			name:        "invalid JSON",
			input:       []byte(`{"invalid": json}`),
			expectError: true,
			checkData:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMetadataColumn()
			err := mc.FromJSON(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkData != nil {
					tt.checkData(t, mc)
				}
			}
		})
	}
}

func TestMetadataColumn_FromJSON_NilData(t *testing.T) {
	mc := &MetadataColumn{data: nil}

	err := mc.FromJSON([]byte(`{"key":"value"}`))
	assert.NoError(t, err)
	assert.NotNil(t, mc.data)

	val, ok := mc.GetString("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestMetadataColumn_DriverValuer(t *testing.T) {
	mc := NewMetadataColumn()
	mc.Set("test", "value")

	// Test that MetadataColumn implements driver.Valuer
	var valuer driver.Valuer = mc
	value, err := valuer.Value()
	assert.NoError(t, err)
	assert.NotNil(t, value)

	// Verify the value is JSON bytes
	bytes, ok := value.([]byte)
	assert.True(t, ok)

	// Verify it can be scanned back
	newMC := NewMetadataColumn()
	err = newMC.Scan(bytes)
	assert.NoError(t, err)

	val, exists := newMC.Get("test")
	assert.True(t, exists)
	assert.Equal(t, "value", val)
}