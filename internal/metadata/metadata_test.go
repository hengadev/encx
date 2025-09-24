package metadata

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptionMetadata(t *testing.T) {
	kekAlias := "test-alias"
	generatorVersion := "v1.0.0"
	pepperVersion := 1

	// Record time before creating metadata
	beforeTime := time.Now().Unix()

	metadata := NewEncryptionMetadata(kekAlias, generatorVersion, pepperVersion)

	// Record time after creating metadata
	afterTime := time.Now().Unix()

	// Verify all fields are set correctly
	assert.Equal(t, pepperVersion, metadata.PepperVersion)
	assert.Equal(t, kekAlias, metadata.KEKAlias)
	assert.Equal(t, generatorVersion, metadata.GeneratorVersion)

	// Verify encryption time is within reasonable bounds
	assert.GreaterOrEqual(t, metadata.EncryptionTime, beforeTime)
	assert.LessOrEqual(t, metadata.EncryptionTime, afterTime)
}

func TestEncryptionMetadata_ToJSON(t *testing.T) {
	metadata := &EncryptionMetadata{
		PepperVersion:    1,
		KEKAlias:         "test-alias",
		EncryptionTime:   1640995200, // 2022-01-01 00:00:00 UTC
		GeneratorVersion: "v1.0.0",
	}

	jsonData, err := metadata.ToJSON()
	require.NoError(t, err)

	// Verify the JSON contains expected fields
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, float64(1), parsed["pepper_version"])
	assert.Equal(t, "test-alias", parsed["kek_alias"])
	assert.Equal(t, float64(1640995200), parsed["encryption_time"])
	assert.Equal(t, "v1.0.0", parsed["generator_version"])
}

func TestEncryptionMetadata_FromJSON(t *testing.T) {
	jsonData := `{
		"pepper_version": 2,
		"kek_alias": "production-key",
		"encryption_time": 1640995200,
		"generator_version": "v2.0.0"
	}`

	var metadata EncryptionMetadata
	err := metadata.FromJSON([]byte(jsonData))
	require.NoError(t, err)

	assert.Equal(t, 2, metadata.PepperVersion)
	assert.Equal(t, "production-key", metadata.KEKAlias)
	assert.Equal(t, int64(1640995200), metadata.EncryptionTime)
	assert.Equal(t, "v2.0.0", metadata.GeneratorVersion)
}

func TestEncryptionMetadata_FromJSON_InvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": json}`

	var metadata EncryptionMetadata
	err := metadata.FromJSON([]byte(invalidJSON))
	assert.Error(t, err)
}

func TestEncryptionMetadata_Validate(t *testing.T) {
	tests := []struct {
		name      string
		metadata  *EncryptionMetadata
		expectErr error
	}{
		{
			name: "valid metadata",
			metadata: &EncryptionMetadata{
				PepperVersion:    1,
				KEKAlias:         "test-alias",
				EncryptionTime:   time.Now().Unix(),
				GeneratorVersion: "v1.0.0",
			},
			expectErr: nil,
		},
		{
			name: "missing KEK alias",
			metadata: &EncryptionMetadata{
				PepperVersion:    1,
				KEKAlias:         "",
				EncryptionTime:   time.Now().Unix(),
				GeneratorVersion: "v1.0.0",
			},
			expectErr: ErrMissingKEKAlias,
		},
		{
			name: "missing generator version",
			metadata: &EncryptionMetadata{
				PepperVersion:    1,
				KEKAlias:         "test-alias",
				EncryptionTime:   time.Now().Unix(),
				GeneratorVersion: "",
			},
			expectErr: ErrMissingGeneratorVersion,
		},
		{
			name: "both missing",
			metadata: &EncryptionMetadata{
				PepperVersion:    1,
				KEKAlias:         "",
				EncryptionTime:   time.Now().Unix(),
				GeneratorVersion: "",
			},
			expectErr: ErrMissingKEKAlias, // Should return first error encountered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.expectErr != nil {
				assert.Equal(t, tt.expectErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncryptionMetadata_JSONRoundTrip(t *testing.T) {
	original := NewEncryptionMetadata("test-alias", "v1.0.0", 1)

	// Serialize to JSON
	jsonData, err := original.ToJSON()
	require.NoError(t, err)

	// Deserialize from JSON
	var deserialized EncryptionMetadata
	err = deserialized.FromJSON(jsonData)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.PepperVersion, deserialized.PepperVersion)
	assert.Equal(t, original.KEKAlias, deserialized.KEKAlias)
	assert.Equal(t, original.EncryptionTime, deserialized.EncryptionTime)
	assert.Equal(t, original.GeneratorVersion, deserialized.GeneratorVersion)
}

func TestErrorConstants(t *testing.T) {
	// Verify error constants are defined and have meaningful messages
	assert.Equal(t, "KEK alias is required", ErrMissingKEKAlias.Error())
	assert.Equal(t, "generator version is required", ErrMissingGeneratorVersion.Error())
	assert.Equal(t, "invalid metadata format", ErrInvalidMetadataFormat.Error())
}

