package crypto

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

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
}

// NewHashingOperations creates a new HashingOperations instance
func NewHashingOperations(pepper []byte, argon2Params Argon2ParamsInterface) (*HashingOperations, error) {
	if argon2Params == nil {
		return nil, fmt.Errorf("argon2 parameters cannot be nil")
	}
	return &HashingOperations{
		pepper:       pepper,
		argon2Params: argon2Params,
	}, nil
}

// HashBasic performs a basic SHA256 hash on the byte representation of the input.
// The input value should be serialized bytes. For comparing hashed values, use CompareBasicHashAndValue
// which handles serialization internally.
func HashBasic(ctx context.Context, value []byte) string {
	valueHash := sha256.Sum256(value)
	return hex.EncodeToString(valueHash[:])
}

// HashBasic performs a basic SHA256 hash on the byte representation of the input.
// The input value should be serialized bytes. For comparing hashed values, use CompareBasicHashAndValue
// which handles serialization internally.
func (h *HashingOperations) HashBasic(ctx context.Context, value []byte) string {
	return HashBasic(ctx, value)
}

// HashSecure performs a secure Argon2id hash on the byte representation of the input,
// incorporating the configured Argon2 parameters and pepper.
// The input value should be serialized bytes. For comparing hashed values, use CompareSecureHashAndValue
// which handles serialization internally.
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

// CompareSecureHashAndValue compares a secure hash with a value.
// The value parameter can be of any type and will be serialized internally using the compact serializer.
// This serialization must match the serialization used when generating the hash with HashSecure.
func (h *HashingOperations) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("value cannot be nil")
	}

	// Parse the stored hash to extract parameters, salt, and hash
	parts := strings.Split(hashValue, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid hash format")
	}

	// Parse version
	versionPart := parts[2]
	if !strings.HasPrefix(versionPart, "v=") {
		return false, fmt.Errorf("invalid version format")
	}
	version, err := strconv.Atoi(versionPart[2:])
	if err != nil {
		return false, fmt.Errorf("invalid version number: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("unsupported Argon2 version")
	}

	// Parse parameters (m=memory,t=iterations,p=parallelism)
	paramsPart := parts[3]
	paramPairs := strings.Split(paramsPart, ",")
	if len(paramPairs) != 3 {
		return false, fmt.Errorf("invalid parameters format")
	}

	var memory, iterations uint32
	var parallelism uint8

	for _, pair := range paramPairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) != 2 {
			return false, fmt.Errorf("invalid parameter format")
		}
		value, err := strconv.ParseUint(keyValue[1], 10, 32)
		if err != nil {
			return false, fmt.Errorf("invalid parameter value: %w", err)
		}
		switch keyValue[0] {
		case "m":
			memory = uint32(value)
		case "t":
			iterations = uint32(value)
		case "p":
			parallelism = uint8(value)
		default:
			return false, fmt.Errorf("unknown parameter: %s", keyValue[0])
		}
	}

	// Decode salt and stored hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Serialize the value using compact serializer
	serializedValue, err := serialization.Serialize(value)
	if err != nil {
		return false, fmt.Errorf("failed to serialize value: %w", err)
	}

	// Combine value with pepper
	peppered := append(serializedValue, h.pepper[:]...)

	// Generate hash using the extracted salt and parameters
	computedHash := argon2.IDKey(
		peppered,
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(storedHash)),
	)

	// CRITICAL: Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(computedHash, storedHash) == 1 {
		return true, nil
	}
	return false, nil
}

// CompareBasicHashAndValue compares a basic hash with a value.
// The value parameter can be of any type and will be serialized internally using the compact serializer.
// This serialization must match the serialization used when generating the hash with HashBasic.
func (h *HashingOperations) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("value cannot be nil")
	}

	// Serialize the value using compact serializer
	serializedValue, err := serialization.Serialize(value)
	if err != nil {
		return false, fmt.Errorf("failed to serialize value: %w", err)
	}

	computedHash := h.HashBasic(ctx, serializedValue)
	return computedHash == hashValue, nil
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
