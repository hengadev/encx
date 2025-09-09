package crypto

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"

	"github.com/hengadev/encx/internal/serialization"
	"golang.org/x/crypto/argon2"
)

// Argon2ParamsInterface defines the interface for Argon2 parameters
type Argon2ParamsInterface interface {
	GetMemory() uint32
	GetIterations() uint32
	GetParallelism() uint8
	GetSaltLength() uint32
	GetKeyLength() uint32
}

// HashingOperations handles basic and secure hashing operations
type HashingOperations struct {
	pepper       []byte
	argon2Params Argon2ParamsInterface
	serializer   serialization.Serializer
}

// NewHashingOperations creates a new HashingOperations instance
func NewHashingOperations(pepper []byte, argon2Params Argon2ParamsInterface, serializer serialization.Serializer) *HashingOperations {
	return &HashingOperations{
		pepper:       pepper,
		argon2Params: argon2Params,
		serializer:   serializer,
	}
}

// HashBasic performs a basic SHA256 hash on the byte representation of the input.
func HashBasic(ctx context.Context, value []byte) string {
	valueHash := sha256.Sum256(value)
	return hex.EncodeToString(valueHash[:])
}

// HashBasic performs a basic SHA256 hash on the byte representation of the input.
func (h *HashingOperations) HashBasic(ctx context.Context, value []byte) string {
	return HashBasic(ctx, value)
}

// HashSecure performs a secure Argon2id hash on the byte representation of the input,
// incorporating the configured Argon2 parameters and pepper.
func (h *HashingOperations) HashSecure(ctx context.Context, value []byte) (string, error) {
	if isZeroPepper(h.pepper) {
		return "", fmt.Errorf("pepper is uninitialized")
	}

	// Generate random salt
	salt := make([]byte, h.argon2Params.GetSaltLength())
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Combine value with pepper
	peppered := append(value, h.pepper[:]...)

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		peppered,
		salt,
		h.argon2Params.GetIterations(),
		h.argon2Params.GetMemory(),
		h.argon2Params.GetParallelism(),
		h.argon2Params.GetKeyLength(),
	)

	// Encode params, salt, and hash into a string
	params := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.argon2Params.GetMemory(),
		h.argon2Params.GetIterations(),
		h.argon2Params.GetParallelism(),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return params, nil
}

// CompareSecureHashAndValue compares a secure hash with a value
func (h *HashingOperations) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("value cannot be nil")
	}
	v, err := h.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	valueHashed, err := h.HashSecure(ctx, v)
	if err != nil {
		return false, fmt.Errorf("secure hashing failed for value : %w", err)
	}
	return valueHashed == hashValue, nil
}

// CompareBasicHashAndValue compares a basic hash with a value
func (h *HashingOperations) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("value cannot be nil")
	}
	v, err := h.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	return h.HashBasic(ctx, v) == hashValue, nil
}

// isZeroPepper checks if pepper is all zero bytes (uninitialized)
func isZeroPepper(pepper []byte) bool {
	for _, b := range pepper {
		if b != 0 {
			return false
		}
	}
	return true
}

