package metadata

import (
	"encoding/json"
	"time"
)

// EncryptionMetadata contains metadata about how a struct was encrypted
type EncryptionMetadata struct {
	SerializerType   string `json:"serializer_type"`
	PepperVersion    int    `json:"pepper_version"`
	KEKAlias         string `json:"kek_alias"`
	EncryptionTime   int64  `json:"encryption_time"`
	GeneratorVersion string `json:"generator_version"`
}

// ToJSON serializes the metadata to JSON bytes
func (em *EncryptionMetadata) ToJSON() ([]byte, error) {
	return json.Marshal(em)
}

// FromJSON deserializes metadata from JSON bytes
func (em *EncryptionMetadata) FromJSON(data []byte) error {
	return json.Unmarshal(data, em)
}

// NewEncryptionMetadata creates a new EncryptionMetadata instance
func NewEncryptionMetadata(serializerType, kekAlias, generatorVersion string, pepperVersion int) *EncryptionMetadata {
	return &EncryptionMetadata{
		SerializerType:   serializerType,
		PepperVersion:    pepperVersion,
		KEKAlias:         kekAlias,
		EncryptionTime:   time.Now().Unix(),
		GeneratorVersion: generatorVersion,
	}
}

// Validate checks if the metadata is valid
func (em *EncryptionMetadata) Validate() error {
	if em.SerializerType == "" {
		return ErrMissingSerializerType
	}
	if em.KEKAlias == "" {
		return ErrMissingKEKAlias
	}
	if em.GeneratorVersion == "" {
		return ErrMissingGeneratorVersion
	}
	return nil
}