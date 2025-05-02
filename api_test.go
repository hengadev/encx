package encx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashSecure(t *testing.T) {
	ctx := context.Background()
	t.Run("successful hash generation", func(t *testing.T) {
		c := createTestCrypto(t)
		input := []byte("password123")

		hash, err := c.HashSecure(ctx, input)
		require.NoError(t, err)

		// Basic validation of the output format
		parts := strings.Split(hash, "$")
		assert.Equal(t, 6, len(parts))

		// Validate base64 encoding of salt and hash
		_, err = base64.RawStdEncoding.DecodeString(parts[4])
		assert.Nil(t, err)

		_, err = base64.RawStdEncoding.DecodeString(parts[5])
		assert.Nil(t, err)
	})
	t.Run("empty pepper should error", func(t *testing.T) {
		c := createTestCrypto(t)
		c.pepper = make([]byte, 0)

		_, err := c.HashSecure(ctx, []byte("password"))
		assert.ErrorIs(t, err, ErrUninitializedPepper)
	})
	t.Run("nil input should work", func(t *testing.T) {
		c := createTestCrypto(t)

		hash, err := c.HashSecure(ctx, nil)
		require.NoError(t, err)

		parts := strings.Split(hash, "$")
		assert.Equal(t, 6, len(parts))
	})
	t.Run("empty input should work", func(t *testing.T) {
		c := createTestCrypto(t)

		hash, err := c.HashSecure(ctx, []byte{})
		require.NoError(t, err)

		parts := strings.Split(hash, "$")
		assert.Equal(t, 6, len(parts))
	})
}

func TestHashSecure_Randomness(t *testing.T) {
	ctx := context.Background()
	c := createTestCrypto(t)
	input := []byte("same-password")

	// Test that multiple calls with same input produce different hashes (due to random salt)
	hash1, err := c.HashSecure(ctx, input)
	require.NoError(t, err)

	hash2, err := c.HashSecure(ctx, input)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
}

func TestHashSecure_FormatVerification(t *testing.T) {
	ctx := context.Background()
	c := createTestCrypto(t)
	input := []byte("password")

	hash, err := c.HashSecure(ctx, input)
	require.NoError(t, err)

	parts := strings.Split(hash, "$")
	require.NoError(t, err)

	// Verify each part of the format
	assert.Equal(t, "", parts[0])
	assert.Equal(t, "argon2id", parts[1])
	assert.True(t, strings.HasPrefix(parts[2], "v="))
	assert.True(t, strings.HasPrefix(parts[3], "m="))
}

func TestHashBasic(t *testing.T) {
	tests := []struct {
		name     string
		value    []byte
		expected string
	}{
		{
			name:     "empty input",
			value:    []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			value:    []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "special characters",
			value:    []byte("123!@#$%^&*()"),
			expected: "1080903e0276494a19e02ca357d828892c8445ece2b597edea509d6ff690b477",
		},
		{
			name:     "unicode characters",
			value:    []byte("世界"),
			expected: "33650a369521ec29f2e26c43d25967535bcb26436755f536735d1ef6e84a1ec5",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}
	c := &Crypto{}
	ctx := context.Background()
	for _, tt := range tests {
		actual := c.HashBasic(ctx, tt.value)
		assert.Equal(t, tt.expected, actual)
	}
}

func TestHashBasic_LargeInput(t *testing.T) {
	c := &Crypto{}
	ctx := context.Background()

	// For large value, we need to compute the expected value separately
	value := bytes.Repeat([]byte("a"), 10000)
	hash := sha256.Sum256(value)
	expected := hex.EncodeToString(hash[:])

	actual := c.HashBasic(ctx, value)
	assert.Equal(t, expected, actual)
}

func TestIsZeroPepper(t *testing.T) {
	tests := []struct {
		name   string
		pepper []byte
		want   bool
	}{
		{
			name:   "all zeros",
			pepper: []byte{0, 0, 0, 0},
			want:   true,
		},
		{
			name:   "single zero byte",
			pepper: []byte{0},
			want:   true,
		},
		{
			name:   "empty slice",
			pepper: []byte{},
			want:   true,
		},
		{
			name:   "non-zero at beginning",
			pepper: []byte{1, 0, 0, 0},
			want:   false,
		},
		{
			name:   "non-zero at middle",
			pepper: []byte{0, 0, 1, 0},
			want:   false,
		},
		{
			name:   "non-zero at end",
			pepper: []byte{0, 0, 0, 1},
			want:   false,
		},
		{
			name:   "all non-zero",
			pepper: []byte{1, 2, 3, 4},
			want:   false,
		},
		{
			name:   "nil slice",
			pepper: nil,
			want:   true,
		},
		{
			name:   "large zero slice",
			pepper: make([]byte, 1024), // all zeros
			want:   true,
		},
		{
			name:   "large non-zero slice",
			pepper: bytes.Repeat([]byte{1}, 1024),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isZeroPepper(tt.pepper))
		})
	}
}

func createTestCrypto(t *testing.T) *Crypto {
	t.Helper()
	return &Crypto{
		pepper:       []byte("test-pepper-1234567890"),
		argon2Params: DefaultArgon2Params,
	}
}
