package encx

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	_ "github.com/mattn/go-sqlite3"
)

// TestCryptoOptions provides configuration for creating test crypto instances
type TestCryptoOptions struct {
	UseRealDatabase bool              // If false, uses in-memory database
	CustomPepper    []byte            // If nil, uses default test pepper
	CustomKMSMock   KeyManagementService // If nil, creates default mock
	DBPath          string            // Custom database path (only used if UseRealDatabase is true)
}

// NewTestCrypto creates a Crypto instance configured for testing.
// This bypasses the complex New() constructor and creates a minimal instance
// suitable for unit testing without external dependencies.
func NewTestCrypto(t testing.TB, options ...*TestCryptoOptions) (*Crypto, *KeyManagementServiceMock) {
	t.Helper()
	
	var opts *TestCryptoOptions
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &TestCryptoOptions{}
	}

	// Set up pepper
	pepper := opts.CustomPepper
	if pepper == nil {
		pepper = []byte("test-pepper-32-chars-for-testing") // Exactly 32 chars
	}
	if len(pepper) != 32 {
		t.Fatalf("Test pepper must be exactly 32 bytes, got %d", len(pepper))
	}

	// Set up KMS mock
	var kmsMock *KeyManagementServiceMock
	if opts.CustomKMSMock != nil {
		// Type assertion to get the mock if it's provided
		if mock, ok := opts.CustomKMSMock.(*KeyManagementServiceMock); ok {
			kmsMock = mock
		} else {
			t.Fatal("CustomKMSMock must be a *KeyManagementServiceMock")
		}
	} else {
		kmsMock = NewKeyManagementServiceMock()
		// Set up default mock expectations for basic functionality only
		setupDefaultKMSMockExpectations(kmsMock)
	}

	// Set up database
	var db *sql.DB
	var err error
	
	if opts.UseRealDatabase {
		dbPath := opts.DBPath
		if dbPath == "" {
			// Create temporary directory for test database
			tempDir := t.TempDir()
			dbPath = filepath.Join(tempDir, "test_metadata.db")
		}
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
	} else {
		// Use in-memory database
		db, err = sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("Failed to create in-memory test database: %v", err)
		}
	}

	// Create the crypto instance
	crypto := &Crypto{
		kmsService:    kmsMock,
		kekAlias:      "test-key-alias",
		pepper:        pepper,
		argon2Params:  DefaultArgon2Params,
		serializer:    JSONSerializer{},
		keyMetadataDB: db,
	}

	// Initialize the database tables
	if err := initializeTestDatabase(db); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Add cleanup for the database connection
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("Warning: Failed to close test database: %v", err)
		}
	})

	return crypto, kmsMock
}

// NewTestCryptoWithMockKMS creates a test crypto instance with a pre-configured KMS mock
func NewTestCryptoWithMockKMS(t testing.TB, kmsMock *KeyManagementServiceMock) *Crypto {
	t.Helper()
	
	crypto, _ := NewTestCrypto(t, &TestCryptoOptions{
		CustomKMSMock: kmsMock,
	})
	
	return crypto
}

// setupDefaultKMSMockExpectations configures the KMS mock with reasonable defaults for testing
func setupDefaultKMSMockExpectations(kmsMock *KeyManagementServiceMock) {
	// Default pepper retrieval
	kmsMock.On("GetSecret", mock.Anything, "secret/data/pepper").
		Return([]byte("test-pepper-32-chars-for-testing"), nil).
		Maybe()

	// Default key operations
	kmsMock.On("GetKeyID", mock.Anything, "test-key-alias").
		Return("test-kms-key-id", nil).
		Maybe()

	kmsMock.On("CreateKey", mock.Anything, "test-key-alias").
		Return("test-kms-key-id", nil).
		Maybe()

	// For DEK operations, we need consistent encryption/decryption
	// Instead of complex mocking, just use passthrough - let the real crypto work
	// but with simple mock KMS operations
	
	// Create a fixed 32-byte DEK for consistency
	testDEK := make([]byte, 32)
	copy(testDEK, []byte("test-dek-32-bytes-for-aes256-key"))
	
	kmsMock.On("EncryptDEK", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("mock-encrypted-dek-data"), nil).
		Maybe()

	kmsMock.On("DecryptDEK", mock.Anything, mock.Anything, mock.Anything).
		Return(testDEK, nil). // Always return the same 32-byte DEK
		Maybe()
}

// initializeTestDatabase sets up the required database tables for testing
func initializeTestDatabase(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kek_versions (
			alias TEXT NOT NULL,
			version INTEGER NOT NULL,
			creation_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_deprecated BOOLEAN DEFAULT FALSE,
			kms_key_id TEXT NOT NULL,
			PRIMARY KEY (alias, version)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create kek_versions table: %w", err)
	}

	// Insert initial test KEK version
	_, err = db.Exec(`
		INSERT OR IGNORE INTO kek_versions (alias, version, kms_key_id) 
		VALUES ('test-key-alias', 1, 'test-kms-key-id')
	`)
	if err != nil {
		return fmt.Errorf("failed to insert initial test KEK: %w", err)
	}

	return nil
}

// NewKeyManagementServiceMock creates a new KMS mock instance
func NewKeyManagementServiceMock() *KeyManagementServiceMock {
	return &KeyManagementServiceMock{}
}

// TestDataFactory provides utilities for creating predictable test data
type TestDataFactory struct {
	crypto CryptoService
}

// NewTestDataFactory creates a new test data factory with the given crypto service
func NewTestDataFactory(crypto CryptoService) *TestDataFactory {
	return &TestDataFactory{crypto: crypto}
}

// CreatePredictableEncryptedData creates encrypted data with a fixed DEK for testing
// This allows tests to have predictable encrypted values that can be compared
func (f *TestDataFactory) CreatePredictableEncryptedData(ctx context.Context, plaintext string) ([]byte, []byte, error) {
	// Use a fixed DEK for predictable results in tests
	fixedDEK := []byte("test-dek-32-chars-for-predictabl") // Exactly 32 chars for AES-256
	
	encrypted, err := f.crypto.EncryptData(ctx, []byte(plaintext), fixedDEK)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt test data: %w", err)
	}
	
	return encrypted, fixedDEK, nil
}

// CreateTestStruct creates a test struct with encrypted fields for testing
func (f *TestDataFactory) CreateTestStruct(ctx context.Context, plainValue string) (*TestStructExample, error) {
	testStruct := &TestStructExample{
		PlainField:    "plain-" + plainValue,
		EncryptedField: plainValue,
	}
	
	err := f.crypto.ProcessStruct(ctx, testStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to process test struct: %w", err)
	}
	
	return testStruct, nil
}

// TestStructExample is an example struct for testing encryption functionality
type TestStructExample struct {
	PlainField            string `json:"plain_field"`
	EncryptedField        string `encx:"encrypt" json:"encrypted_field"`
	EncryptedFieldEncrypted []byte `json:"encrypted_field_encrypted"`
	DEK                   []byte `json:"-"` // DEK field required by encx
	DEKEncrypted          []byte `json:"dek_encrypted"`
	KeyVersion            int    `json:"key_version"`
}