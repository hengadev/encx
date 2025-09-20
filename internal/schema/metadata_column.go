package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// MetadataColumn provides cross-database JSON column support
// It implements sql.Scanner and driver.Valuer for seamless database integration
type MetadataColumn struct {
	data map[string]interface{}
}

// NewMetadataColumn creates a new MetadataColumn instance
func NewMetadataColumn() *MetadataColumn {
	return &MetadataColumn{
		data: make(map[string]interface{}),
	}
}

// Set stores a key-value pair in the metadata
func (mc *MetadataColumn) Set(key string, value interface{}) {
	if mc.data == nil {
		mc.data = make(map[string]interface{})
	}
	mc.data[key] = value
}

// Get retrieves a value by key from the metadata
func (mc *MetadataColumn) Get(key string) (interface{}, bool) {
	if mc.data == nil {
		return nil, false
	}
	val, exists := mc.data[key]
	return val, exists
}

// GetString retrieves a string value by key
func (mc *MetadataColumn) GetString(key string) (string, bool) {
	val, exists := mc.Get(key)
	if !exists {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt retrieves an int value by key
func (mc *MetadataColumn) GetInt(key string) (int, bool) {
	val, exists := mc.Get(key)
	if !exists {
		return 0, false
	}

	// Handle different numeric types that JSON might unmarshal to
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

// Delete removes a key from the metadata
func (mc *MetadataColumn) Delete(key string) {
	if mc.data != nil {
		delete(mc.data, key)
	}
}

// Keys returns all keys in the metadata
func (mc *MetadataColumn) Keys() []string {
	if mc.data == nil {
		return nil
	}
	keys := make([]string, 0, len(mc.data))
	for k := range mc.data {
		keys = append(keys, k)
	}
	return keys
}

// IsEmpty returns true if the metadata contains no data
func (mc *MetadataColumn) IsEmpty() bool {
	return mc.data == nil || len(mc.data) == 0
}

// Clear removes all data from the metadata
func (mc *MetadataColumn) Clear() {
	mc.data = make(map[string]interface{})
}

// Scan implements sql.Scanner for database/sql
// This handles reading JSON data from both PostgreSQL JSONB and SQLite TEXT columns
func (mc *MetadataColumn) Scan(value interface{}) error {
	if value == nil {
		mc.data = make(map[string]interface{})
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into MetadataColumn", value)
	}

	if len(bytes) == 0 {
		mc.data = make(map[string]interface{})
		return nil
	}

	return json.Unmarshal(bytes, &mc.data)
}

// Value implements driver.Valuer for database/sql
// This returns JSON bytes for storage in the database
func (mc MetadataColumn) Value() (driver.Value, error) {
	if mc.data == nil || len(mc.data) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(mc.data)
}

// ToJSON returns the JSON representation of the metadata
func (mc *MetadataColumn) ToJSON() ([]byte, error) {
	if mc.data == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(mc.data)
}

// FromJSON populates the metadata from JSON bytes
func (mc *MetadataColumn) FromJSON(data []byte) error {
	if mc.data == nil {
		mc.data = make(map[string]interface{})
	}
	return json.Unmarshal(data, &mc.data)
}