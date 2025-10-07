package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"strings"
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
// TestDataEncryption_EncryptStream tests stream encryption
func TestDataEncryption_EncryptStream(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   string
		dek     []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "small text",
			input:   "Hello, World!",
			dek:     dek,
			wantErr: false,
		},
		{
			name:    "large text",
			input:   strings.Repeat("A", 100000), // 100KB
			dek:     dek,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			dek:     dek,
			wantErr: false,
		},
		{
			name:    "invalid DEK",
			input:   "test",
			dek:     []byte("short"),
			wantErr: true,
			errMsg:  "invalid key size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			var writer bytes.Buffer

			err := de.EncryptStream(ctx, reader, &writer, tt.dek)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)

			// Encrypted data should not be empty (unless input was empty)
			if len(tt.input) > 0 {
				assert.NotEmpty(t, writer.Bytes())
				assert.NotEqual(t, []byte(tt.input), writer.Bytes())
			}
		})
	}
}

// TestDataEncryption_DecryptStream tests stream decryption
func TestDataEncryption_DecryptStream(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
		dek       []byte
	}{
		{
			name:      "small text",
			plaintext: "Hello, World!",
			dek:       dek,
		},
		{
			name:      "large text",
			plaintext: strings.Repeat("Test data ", 10000), // ~100KB
			dek:       dek,
		},
		{
			name:      "unicode text",
			plaintext: "Unicode: 你好世界 🌍 مرحبا",
			dek:       dek,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First encrypt
			reader := strings.NewReader(tt.plaintext)
			var encryptedBuf bytes.Buffer

			err := de.EncryptStream(ctx, reader, &encryptedBuf, tt.dek)
			require.NoError(t, err)

			// Then decrypt
			var decryptedBuf bytes.Buffer
			err = de.DecryptStream(ctx, &encryptedBuf, &decryptedBuf, tt.dek)
			require.NoError(t, err)

			// Verify decrypted matches original
			assert.Equal(t, tt.plaintext, decryptedBuf.String())
		})
	}
}

// TestDataEncryption_DecryptStream_InvalidInputs tests error handling
func TestDataEncryption_DecryptStream_InvalidInputs(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	// Create some properly encrypted data for tests that need it
	var encryptedBuf bytes.Buffer
	err = de.EncryptStream(ctx, strings.NewReader("test data"), &encryptedBuf, dek)
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   []byte
		dek     []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "corrupted data",
			input:   []byte("corrupted data not encrypted"),
			dek:     dek,
			wantErr: true,
		},
		{
			name:    "invalid DEK",
			input:   encryptedBuf.Bytes(),
			dek:     []byte("short"),
			wantErr: true,
			errMsg:  "invalid key size",
		},
		{
			name:    "empty data",
			input:   []byte{},
			dek:     dek,
			wantErr: false, // Empty should work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			var writer bytes.Buffer

			err := de.DecryptStream(ctx, reader, &writer, tt.dek)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDataEncryption_EncryptDecryptStream_LargeFile tests streaming with large data
func TestDataEncryption_EncryptDecryptStream_LargeFile(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	// Create 1MB of test data
	largeData := make([]byte, 1024*1024)
	_, err = rand.Read(largeData)
	require.NoError(t, err)

	// Encrypt
	reader := bytes.NewReader(largeData)
	var encryptedBuf bytes.Buffer

	err = de.EncryptStream(ctx, reader, &encryptedBuf, dek)
	require.NoError(t, err)

	// Decrypt
	var decryptedBuf bytes.Buffer
	err = de.DecryptStream(ctx, &encryptedBuf, &decryptedBuf, dek)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, largeData, decryptedBuf.Bytes())
}

// TestDecryptStream_MaliciousChunkSize tests protection against memory exhaustion attacks
func TestDecryptStream_MaliciousChunkSize(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	tests := []struct {
		name        string
		chunkSize   uint32
		wantErr     bool
		errContains string
	}{
		{
			name:        "chunk size exceeds maximum (4GB)",
			chunkSize:   0xFFFFFFFF, // 4GB - malicious
			wantErr:     true,
			errContains: "exceeds maximum allowed size",
		},
		{
			name:        "chunk size exceeds maximum (100MB)",
			chunkSize:   100 * 1024 * 1024, // 100MB - malicious
			wantErr:     true,
			errContains: "exceeds maximum allowed size",
		},
		{
			name:        "chunk size at maximum boundary (10MB)",
			chunkSize:   10*1024*1024 - 28, // Just under 10MB (accounting for GCM overhead)
			wantErr:     false,
			errContains: "",
		},
		{
			name:        "zero chunk size",
			chunkSize:   0,
			wantErr:     true,
			errContains: "invalid chunk size: 0",
		},
		{
			name:        "normal chunk size (1KB)",
			chunkSize:   1024,
			wantErr:     false,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create malicious stream with crafted chunk length
			var maliciousStream bytes.Buffer

			// Write malicious chunk length header
			lengthBytes := []byte{
				byte(tt.chunkSize >> 24),
				byte(tt.chunkSize >> 16),
				byte(tt.chunkSize >> 8),
				byte(tt.chunkSize),
			}
			maliciousStream.Write(lengthBytes)

			// For valid chunk sizes, write actual encrypted data
			if !tt.wantErr {
				// Create properly encrypted chunk of the specified size
				plaintext := make([]byte, tt.chunkSize-28) // Account for GCM overhead (12-byte nonce + 16-byte tag)
				ciphertext, err := de.EncryptData(ctx, plaintext, dek)
				require.NoError(t, err)
				maliciousStream.Write(ciphertext)
			} else {
				// For invalid sizes, write garbage data (won't be read due to validation)
				maliciousStream.Write([]byte("garbage data"))
			}

			// Attempt to decrypt the malicious stream
			var output bytes.Buffer
			err := de.DecryptStream(ctx, &maliciousStream, &output, dek)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDecryptStream_IncompleteChunk tests handling of incomplete chunks
func TestDecryptStream_IncompleteChunk(t *testing.T) {
	de := NewDataEncryption()
	ctx := context.Background()

	// Generate a valid DEK
	dek := make([]byte, 32)
	_, err := rand.Read(dek)
	require.NoError(t, err)

	tests := []struct {
		name        string
		setupStream func() *bytes.Buffer
		wantErr     bool
		errContains string
	}{
		{
			name: "incomplete chunk length header",
			setupStream: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				// Only write 2 bytes instead of 4
				buf.Write([]byte{0x00, 0x01})
				return buf
			},
			wantErr:     false, // Should treat as EOF
			errContains: "",
		},
		{
			name: "missing chunk data after length header",
			setupStream: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				// Write valid chunk length header for 100 bytes
				buf.Write([]byte{0x00, 0x00, 0x00, 0x64})
				// But don't write the chunk data
				return buf
			},
			wantErr:     true,
			errContains: "failed to read encrypted chunk",
		},
		{
			name: "partial chunk data",
			setupStream: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				// Write chunk length header for 100 bytes
				buf.Write([]byte{0x00, 0x00, 0x00, 0x64})
				// But only write 50 bytes
				buf.Write(make([]byte, 50))
				return buf
			},
			wantErr:     true,
			errContains: "failed to read encrypted chunk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := tt.setupStream()
			var output bytes.Buffer

			err := de.DecryptStream(ctx, stream, &output, dek)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
