package encx

// This file provides test utilities that are re-exported from internal testutils
// for use in examples and external testing.

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"fmt"

	"github.com/hengadev/encx/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// SimpleTestKMS implements a basic in-memory KMS for testing and examples
type SimpleTestKMS struct {
	keys map[string][]byte // keyID -> key material
}

// NewSimpleTestKMS creates a new simple test KMS with a default key
func NewSimpleTestKMS() config.KeyManagementService {
	// Generate a random 32-byte key for AES-256
	key := make([]byte, 32)
	rand.Read(key)

	return &SimpleTestKMS{
		keys: map[string][]byte{
			"test-key-id": key,
		},
	}
}

// GetKeyID returns a test key ID for the given alias
func (s *SimpleTestKMS) GetKeyID(ctx context.Context, alias string) (string, error) {
	return "test-key-id", nil
}

// CreateKey creates a new test key and returns its ID
func (s *SimpleTestKMS) CreateKey(ctx context.Context, description string) (string, error) {
	keyID := fmt.Sprintf("test-key-%d", len(s.keys))
	key := make([]byte, 32)
	rand.Read(key)
	s.keys[keyID] = key
	return keyID, nil
}

// EncryptDEK encrypts the DEK using AES-GCM
func (s *SimpleTestKMS) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	key, exists := s.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptDEK decrypts the DEK using AES-GCM
func (s *SimpleTestKMS) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	key, exists := s.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// NewTestCrypto creates a simple Crypto instance for testing and examples
// If t is nil, creates a basic test crypto for examples/demos
func NewTestCrypto(t interface{}) (*Crypto, error) {
	ctx := context.Background()

	// Create an in-memory database with the necessary schema
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	// Initialize the database schema
	if err := initializeTestDatabase(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create crypto instance with test configuration
	crypto, err := NewCrypto(ctx,
		WithKMSService(NewSimpleTestKMS()),
		WithKEKAlias("test-kek-alias"),
		WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		WithKeyMetadataDB(db),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create test crypto: %w", err)
	}

	return crypto, nil
}

// initializeTestDatabase creates the necessary tables for testing
func initializeTestDatabase(ctx context.Context, db *sql.DB) error {
	// Create kek_versions table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE kek_versions (
			alias TEXT NOT NULL,
			version INTEGER NOT NULL,
			kms_key_id TEXT NOT NULL,
			is_deprecated BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (alias, version)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create kek_versions table: %w", err)
	}

	// Insert a default KEK version
	_, err = db.ExecContext(ctx, `
		INSERT INTO kek_versions (alias, version, kms_key_id, is_deprecated)
		VALUES ('test-kek-alias', 1, 'test-key-id', FALSE)
	`)
	if err != nil {
		return fmt.Errorf("failed to insert default KEK version: %w", err)
	}

	return nil
}
