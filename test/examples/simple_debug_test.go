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

	plaintext := []byte("test-data")
	encrypted, err := crypto.EncryptData(ctx, plaintext, dek)
	require.NoError(t, err)

	decrypted, err := crypto.DecryptData(ctx, encrypted, dek)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Test struct encryption
	type SimpleStruct struct {
		PlainField            string `json:"plain_field"`
		EncryptField          string `encx:"encrypt" json:"encrypt_field"`
		EncryptFieldEncrypted []byte `json:"encrypt_field_encrypted"`
		DEK                   []byte `json:"-"`
		DEKEncrypted          []byte `json:"dek_encrypted"`
		KeyVersion            int    `json:"key_version"`
	}

	testStruct := &SimpleStruct{
		PlainField:   "plain-value",
		EncryptField: "secret-value",
	}

	t.Logf("Before ProcessStruct: PlainField=%s, EncryptField=%s", testStruct.PlainField, testStruct.EncryptField)

	err = crypto.ProcessStruct(ctx, testStruct)
	require.NoError(t, err)

	t.Logf("After ProcessStruct: PlainField=%s, EncryptField=%s, DEKEncrypted len=%d, KeyVersion=%d",
		testStruct.PlainField, testStruct.EncryptField, len(testStruct.DEKEncrypted), testStruct.KeyVersion)

	// Verify encryption worked - field should be cleared
	assert.Equal(t, "plain-value", testStruct.PlainField)
	assert.Empty(t, testStruct.EncryptField) // Should be empty after encryption
	assert.NotEmpty(t, testStruct.DEKEncrypted)
	assert.Greater(t, testStruct.KeyVersion, 0)

	// Test decryption
	err = crypto.DecryptStruct(ctx, testStruct)
	require.NoError(t, err)

	t.Logf("After DecryptStruct: PlainField=%s, EncryptField=%s", testStruct.PlainField, testStruct.EncryptField)

	assert.Equal(t, "plain-value", testStruct.PlainField)
	assert.Equal(t, "secret-value", testStruct.EncryptField) // Should be restored
}

