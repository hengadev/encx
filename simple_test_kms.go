package encx

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// SimpleTestKMS is a simple in-memory KMS implementation for testing
// This avoids the complexity of mocking and provides consistent encryption/decryption
type SimpleTestKMS struct {
	mu       sync.RWMutex
	secrets  map[string][]byte
	keys     map[string]string
	nextKeyID int
	dekStore map[string][]byte // Store DEKs to ensure consistent encryption/decryption
}

func NewSimpleTestKMS() *SimpleTestKMS {
	kms := &SimpleTestKMS{
		secrets:  make(map[string][]byte),
		keys:     make(map[string]string),
		dekStore: make(map[string][]byte),
	}
	
	// Set up default test pepper
	kms.secrets["secret/data/pepper"] = []byte("test-pepper-32-chars-for-testing")
	
	return kms
}

func (s *SimpleTestKMS) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// For testing, just return a fixed key
	return []byte("test-key-32-bytes-for-testing!!"), nil
}

func (s *SimpleTestKMS) GetKeyID(ctx context.Context, alias string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if keyID, exists := s.keys[alias]; exists {
		return keyID, nil
	}
	return "", fmt.Errorf("key not found for alias: %s", alias)
}

func (s *SimpleTestKMS) CreateKey(ctx context.Context, description string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.nextKeyID++
	keyID := fmt.Sprintf("test-key-%d", s.nextKeyID)
	s.keys[description] = keyID
	
	return keyID, nil
}

func (s *SimpleTestKMS) RotateKey(ctx context.Context, keyID string) error {
	// For testing, rotation is always successful
	return nil
}

func (s *SimpleTestKMS) GetSecret(ctx context.Context, path string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if secret, exists := s.secrets[path]; exists {
		return secret, nil
	}
	return nil, fmt.Errorf("secret not found at path: %s", path)
}

func (s *SimpleTestKMS) SetSecret(ctx context.Context, path string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.secrets[path] = value
	return nil
}

func (s *SimpleTestKMS) EncryptDEK(ctx context.Context, keyID string, plaintextDEK []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// For testing, we simulate encryption by storing the DEK and returning a "ciphertext"
	ciphertext := fmt.Sprintf("encrypted-dek-%x", plaintextDEK[:8])
	s.dekStore[ciphertext] = plaintextDEK
	
	return []byte(ciphertext), nil
}

func (s *SimpleTestKMS) DecryptDEK(ctx context.Context, keyID string, ciphertextDEK []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	ciphertext := string(ciphertextDEK)
	if dek, exists := s.dekStore[ciphertext]; exists {
		return dek, nil
	}
	
	return nil, fmt.Errorf("failed to decrypt DEK: %s", ciphertext)
}

func (s *SimpleTestKMS) EncryptDEKWithVersion(ctx context.Context, plaintextDEK []byte, version int) ([]byte, error) {
	// For testing, version doesn't matter
	return s.EncryptDEK(ctx, "test-key", plaintextDEK)
}

func (s *SimpleTestKMS) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, version int) ([]byte, error) {
	// For testing, version doesn't matter
	return s.DecryptDEK(ctx, "test-key", ciphertextDEK)
}

// NewTestCryptoWithSimpleKMS creates a test crypto instance using the simple test KMS
// This provides fully functional encryption/decryption without mocking complexity
func NewTestCryptoWithSimpleKMS(t testing.TB) (*Crypto, *SimpleTestKMS) {
	t.Helper()
	
	ctx := context.Background()
	simpleKMS := NewSimpleTestKMS()
	
	// Create temporary database file for testing
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test_metadata.db"
	
	// Use the regular New constructor with the simple KMS
	crypto, err := New(
		ctx,
		simpleKMS,
		"test-key-alias",
		"secret/data/pepper",
		WithKeyMetadataDBPath(dbPath), // Use file-based database
	)
	if err != nil {
		t.Fatalf("Failed to create test crypto with simple KMS: %v", err)
	}
	
	// Add cleanup
	t.Cleanup(func() {
		// The crypto instance will clean up the database connection
	})
	
	return crypto, simpleKMS
}