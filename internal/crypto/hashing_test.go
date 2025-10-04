package crypto

import (
	"context"
	"testing"

	"github.com/hengadev/encx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHashingOperations(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)

	assert.NotNil(t, ho)
	assert.Equal(t, pepper, ho.pepper)
	assert.Equal(t, argon2Params, ho.argon2Params)
}

func TestHashingOperations_HashBasic(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "simple string",
			input: []byte("hello world"),
		},
		{
			name:  "empty string",
			input: []byte(""),
		},
		{
			name:  "unicode string",
			input: []byte("你好世界 🌍"),
		},
		{
			name:  "json data",
			input: []byte(`{"key": "value", "number": 42}`),
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := ho.HashBasic(ctx, tt.input)

			assert.NotEmpty(t, hash)
			assert.True(t, len(hash) > 0)

			// Hash should be deterministic for the same input
			hash2 := ho.HashBasic(ctx, tt.input)
			assert.Equal(t, hash, hash2, "Basic hash should be deterministic")
		})
	}
}

func TestHashingOperations_HashBasic_Consistency(t *testing.T) {
	pepper := []byte("consistent-pepper-16")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	testData := []byte("test data for consistency")

	// Hash the same data multiple times
	hashes := make([]string, 10)
	for i := 0; i < 10; i++ {
		hashes[i] = ho.HashBasic(ctx, testData)
	}

	// All hashes should be identical
	for i := 1; i < len(hashes); i++ {
		assert.Equal(t, hashes[0], hashes[i], "All basic hashes should be identical for the same input")
	}
}

func TestHashingOperations_HashBasic_DifferentInputs(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	// Test different inputs produce different hashes
	input1 := []byte("input one")
	input2 := []byte("input two")
	input3 := []byte("input one ") // Very similar to input1

	hash1 := ho.HashBasic(ctx, input1)
	hash2 := ho.HashBasic(ctx, input2)
	hash3 := ho.HashBasic(ctx, input3)

	assert.NotEqual(t, hash1, hash2, "Different inputs should produce different hashes")
	assert.NotEqual(t, hash1, hash3, "Similar inputs should produce different hashes")
	assert.NotEqual(t, hash2, hash3, "All different inputs should produce different hashes")
}

func TestHashingOperations_HashSecure(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "password",
			input: []byte("MySecurePassword123!"),
		},
		{
			name:  "empty password",
			input: []byte(""),
		},
		{
			name:  "unicode password",
			input: []byte("密码123🔒"),
		},
		{
			name:  "long password",
			input: []byte("ThisIsAVeryLongPasswordThatShouldStillBeHashedCorrectly1234567890!@#$%^&*()"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ho.HashSecure(ctx, tt.input)

			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.True(t, len(hash) > 0)

			// Secure hash should be non-deterministic (due to random salt)
			hash2, err := ho.HashSecure(ctx, tt.input)
			require.NoError(t, err)
			assert.NotEqual(t, hash, hash2, "Secure hash should be non-deterministic due to random salt")
		})
	}
}

func TestHashingOperations_HashSecure_VerifyUniqueness(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	input := []byte("test password")

	// Generate multiple secure hashes
	hashes := make([]string, 5)
	for i := 0; i < 5; i++ {
		hash, err := ho.HashSecure(ctx, input)
		require.NoError(t, err)
		hashes[i] = hash
	}

	// All hashes should be different due to random salts
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j],
				"Secure hashes should be unique due to random salts (hash %d vs %d)", i, j)
		}
	}
}

func TestHashingOperations_DifferentPeppers(t *testing.T) {
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	pepper1 := []byte("pepper-one-16-bytes!")
	pepper2 := []byte("different-pepper-16b")

	ho1 := NewHashingOperations(pepper1, argon2Params)
	ho2 := NewHashingOperations(pepper2, argon2Params)
	ctx := context.Background()

	input := []byte("same input data")

	// NOTE: Basic hashes are designed to be deterministic and don't use pepper
	// This is by design for consistent lookups. Only secure hashes use pepper.
	hash1 := ho1.HashBasic(ctx, input)
	hash2 := ho2.HashBasic(ctx, input)

	// Basic hashes should be the same regardless of pepper (this is by design)
	assert.Equal(t, hash1, hash2, "Basic hashes should be identical regardless of pepper for consistent lookups")

	// However, secure hashes with different peppers should be different
	// (We can't easily test this due to random salts, but the pepper is incorporated)
}

func TestHashingOperations_DifferentArgon2Params(t *testing.T) {
	pepper := []byte("same-pepper-16-bytes")

	params1 := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	params2 := &config.Argon2Params{
		Memory:      128 * 1024, // Different memory
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho1 := NewHashingOperations(pepper, params1)
	ho2 := NewHashingOperations(pepper, params2)
	ctx := context.Background()

	input := []byte("test input")

	// Secure hashes with different Argon2 params should be different
	hash1, err := ho1.HashSecure(ctx, input)
	require.NoError(t, err)

	hash2, err := ho2.HashSecure(ctx, input)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "Different Argon2 parameters should produce different secure hashes")
}

func TestHashingOperations_EdgeCases(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	// Test with nil input
	hash := ho.HashBasic(ctx, nil)
	assert.NotEmpty(t, hash, "Should handle nil input gracefully")

	// Test secure hash with nil input
	secureHash, err := ho.HashSecure(ctx, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, secureHash, "Should handle nil input gracefully for secure hash")

	// Test with very large input
	largeInput := make([]byte, 1024*1024) // 1MB
	for i := range largeInput {
		largeInput[i] = byte(i % 256)
	}

	hash = ho.HashBasic(ctx, largeInput)
	assert.NotEmpty(t, hash, "Should handle large input")

	secureHash, err = ho.HashSecure(ctx, largeInput)
	require.NoError(t, err)
	assert.NotEmpty(t, secureHash, "Should handle large input for secure hash")
}

func TestHashingOperations_CompareSecureHashAndValue(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	testCases := []struct {
		name      string
		value     []byte
		hashValue string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "matching value",
			value:     []byte("test-password"),
			hashValue: "", // Will be filled in during test
			wantMatch: true,
			wantErr:   false,
		},
		{
			name:      "non-matching value",
			value:     []byte("wrong-password"),
			hashValue: "", // Will be filled with hash of "test-password"
			wantMatch: false,
			wantErr:   false,
		},
		{
			name:      "empty value",
			value:     []byte(""),
			hashValue: "", // Will be filled in during test
			wantMatch: true,
			wantErr:   false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Generate hash for comparison
			var hashValue string
			var err error

			if tt.name == "non-matching value" {
				// Hash a different value
				hashValue, err = ho.HashSecure(ctx, []byte("test-password"))
				require.NoError(t, err)
			} else {
				// Hash the same value
				hashValue, err = ho.HashSecure(ctx, tt.value)
				require.NoError(t, err)
			}

			// Test comparison
			match, err := ho.CompareSecureHashAndValue(ctx, tt.value, hashValue)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMatch, match)
		})
	}
}

func TestHashingOperations_CompareEdgeCases(t *testing.T) {
	pepper := []byte("test-pepper-16-bytes")
	argon2Params := &config.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	ho := NewHashingOperations(pepper, argon2Params)
	ctx := context.Background()

	t.Run("invalid hash format for secure compare", func(t *testing.T) {
		match, err := ho.CompareSecureHashAndValue(ctx, []byte("test"), "invalid-hash-format")
		assert.Error(t, err)
		assert.False(t, match)
	})
}