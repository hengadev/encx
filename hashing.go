package encx

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"

	"golang.org/x/crypto/argon2"
)

// Hashing operations - basic and secure

// HashBasic performs a basic SHA256 hash on the byte representation of the input.
func (c *Crypto) HashBasic(ctx context.Context, value []byte) string {
	valueHash := sha256.Sum256(value)
	return hex.EncodeToString(valueHash[:])
}

// HashSecure performs a secure Argon2id hash on the byte representation of the input,
// incorporating the configured Argon2 parameters and pepper.
func (c *Crypto) HashSecure(ctx context.Context, value []byte) (string, error) {
	if isZeroPepper(c.pepper) {
		return "", NewUninitalizedPepperError()
	}

	// Generate random salt
	salt := make([]byte, c.argon2Params.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Combine value with pepper
	peppered := append(value, c.pepper[:]...)

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		peppered,
		salt,
		c.argon2Params.Iterations,
		c.argon2Params.Memory,
		c.argon2Params.Parallelism,
		c.argon2Params.KeyLength,
	)

	// Encode params, salt, and hash into a string
	params := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		c.argon2Params.Memory,
		c.argon2Params.Iterations,
		c.argon2Params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return params, nil
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

// CompareSecureHashAndValue compares a secure hash with a value
func (c *Crypto) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%w: value cannot be nil", ErrNilPointer)
	}
	v, err := c.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	valueHashed, err := c.HashSecure(ctx, v)
	if err != nil {
		return false, fmt.Errorf("secure hashing failed for value : %w", err)
	}
	return valueHashed == hashValue, nil
}

// CompareBasicHashAndValue compares a basic hash with a value
func (c *Crypto) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%w: value cannot be nil", ErrNilPointer)
	}
	v, err := c.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	return c.HashBasic(ctx, v) == hashValue, nil
}
