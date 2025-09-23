package encx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the simple approach without complex mocking
func TestSimpleCrypto(t *testing.T) {
	ctx := context.Background()
	crypto, _ := NewTestCryptoWithSimpleKMS(t)

	// Test basic operations
	dek, err := crypto.GenerateDEK()
	require.NoError(t, err)
	t.Logf("Generated DEK length: %d", len(dek))

	// Test encryption/decryption
	plaintext := []byte("Hello, World!")
	ciphertext, err := crypto.EncryptData(ctx, plaintext, dek)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := crypto.DecryptData(ctx, ciphertext, dek)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// Test hashing functions
func TestHashingOperations(t *testing.T) {
	ctx := context.Background()
	crypto, _ := NewTestCryptoWithSimpleKMS(t)

	value := []byte("test value")

	// Test basic hash
	basicHash := crypto.HashBasic(ctx, value)
	assert.NotEmpty(t, basicHash)
	t.Logf("Basic hash: %s", basicHash)

	// Test secure hash
	secureHash, err := crypto.HashSecure(ctx, value)
	require.NoError(t, err)
	assert.NotEmpty(t, secureHash)
	assert.NotEqual(t, basicHash, secureHash)
	t.Logf("Secure hash: %s", secureHash)
}