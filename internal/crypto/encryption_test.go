package crypto

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDataEncryption(t *testing.T) {
	de := NewDataEncryption()

	assert.NotNil(t, de)
}

func TestDataEncryption_EncryptDecryptData(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK (32 bytes for AES-256)
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
		dek       []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid data encryption",
			plaintext: []byte("Hello, World!"),
			dek:       dek,
			wantErr:   false,
		},
		{
			name:      "empty data encryption",
			plaintext: []byte(""),
			dek:       dek,
			wantErr:   false,
		},
		{
			name:      "large data encryption",
			plaintext: make([]byte, 10000), // 10KB of data
			dek:       dek,
			wantErr:   false,
		},
		{
			name:      "invalid DEK length",
			plaintext: []byte("test"),
			dek:       []byte("short"), // Invalid DEK length
			wantErr:   true,
			errMsg:    "invalid key size",
		},
		{
			name:      "nil DEK",
			plaintext: []byte("test"),
			dek:       nil,
			wantErr:   true,
			errMsg:    "invalid key size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			ciphertext, err := de.EncryptData(ctx, tt.plaintext, tt.dek)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, ciphertext)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			// Test decryption
			decrypted, err := de.DecryptData(ctx, ciphertext, tt.dek)
			require.NoError(t, err)

			// Handle nil vs empty slice difference
			if len(tt.plaintext) == 0 && len(decrypted) == 0 {
				// Both are effectively empty, this is fine
			} else {
				assert.Equal(t, tt.plaintext, decrypted)
			}
		})
	}
}

func TestDataEncryption_DecryptData_InvalidInputs(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext []byte
		dek        []byte
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "too short ciphertext",
			ciphertext: []byte("short"),
			dek:        dek,
			wantErr:    true,
			errMsg:     "invalid ciphertext size",
		},
		{
			name:       "invalid DEK for decryption",
			ciphertext: make([]byte, 32), // Valid length but random data
			dek:        []byte("invalid_key"),
			wantErr:    true,
			errMsg:     "invalid key size",
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: make([]byte, 32), // Valid length but corrupted data
			dek:        dek,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := de.DecryptData(ctx, tt.ciphertext, tt.dek)
			assert.Error(t, err)
			if tt.errMsg != "" {
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestDataEncryption_EncryptDecryptData_Roundtrip(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	testData := [][]byte{
		[]byte("Simple text"),
		[]byte("{\"json\": \"data\", \"number\": 42}"),
		[]byte("Unicode: 你好世界 🌍"),
		make([]byte, 1000), // 1KB of zeros
	}

	for i, data := range testData {
		t.Run(fmt.Sprintf("roundtrip_%d", i), func(t *testing.T) {
			// Encrypt
			ciphertext, err := de.EncryptData(ctx, data, dek)
			require.NoError(t, err)

			// Decrypt
			decrypted, err := de.DecryptData(ctx, ciphertext, dek)
			require.NoError(t, err)

			// Verify - handle nil vs empty slice difference
			if len(data) == 0 && len(decrypted) == 0 {
				// Both are effectively empty, this is fine
			} else {
				assert.Equal(t, data, decrypted)
			}
		})
	}
}

func TestDataEncryption_EncryptData_Deterministic(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	plaintext := []byte("test data for determinism")

	// Encrypt the same data multiple times
	ciphertext1, err := de.EncryptData(ctx, plaintext, dek)
	require.NoError(t, err)

	ciphertext2, err := de.EncryptData(ctx, plaintext, dek)
	require.NoError(t, err)

	// Ciphertexts should be different due to random IV
	assert.NotEqual(t, ciphertext1, ciphertext2, "Encryption should not be deterministic due to random IV")

	// But both should decrypt to the same plaintext
	decrypted1, err := de.DecryptData(ctx, ciphertext1, dek)
	require.NoError(t, err)

	decrypted2, err := de.DecryptData(ctx, ciphertext2, dek)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}