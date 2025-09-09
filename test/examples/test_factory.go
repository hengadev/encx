package encx_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/hengadev/encx"
)

// TestCryptoOptions provides configuration for creating test crypto instances
type TestCryptoOptions struct {
	UseRealDatabase bool                      // If false, uses in-memory database
	CustomPepper    []byte                    // If nil, uses default test pepper
	CustomKMSMock   encx.KeyManagementService // If nil, creates default mock KMS
	DBPath          string                    // Custom database path (only used if UseRealDatabase is true)
}

// NewTestCrypto creates a Crypto instance configured for testing.
// This bypasses the complex New() constructor and creates a minimal instance
// suitable for unit testing without external dependencies.
func NewTestCrypto(t testing.TB, options ...*TestCryptoOptions) (*encx.Crypto, encx.KeyManagementService) {
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

	// Set up KMS implementation
	var kmsService encx.KeyManagementService
	if opts.CustomKMSMock != nil {
		kmsService = opts.CustomKMSMock
	} else {
		// Use the simple test KMS for reliable testing
		kmsService = NewSimpleTestKMS()
	}

	// KMS should have the pepper already set up
	if simpleKMS, ok := kmsService.(*SimpleTestKMS); ok {
		ctx := context.Background()
		err := simpleKMS.SetSecret(ctx, "secret/data/pepper", pepper)
		if err != nil {
			t.Fatalf("Failed to set test pepper in KMS: %v", err)
		}
	}

	// Create the crypto instance using the proper constructor
	ctx := context.Background()
	var crypto *encx.Crypto
	var err error

	if opts.UseRealDatabase {
		dbPath := opts.DBPath
		if dbPath == "" {
			// Create temporary directory for test database
			tempDir := t.TempDir()
			dbPath = filepath.Join(tempDir, "test_metadata.db")
		}
		crypto, err = encx.New(
			ctx,
			kmsService,
			"test-key-alias",
			"secret/data/pepper", // Use KMS path for pepper
			encx.WithKeyMetadataDBPath(dbPath),
		)
	} else {
		// Use in-memory database
		crypto, err = encx.New(
			ctx,
			kmsService,
			"test-key-alias",
			"secret/data/pepper", // Use KMS path for pepper
			// No database path will use in-memory by default
		)
	}

	if err != nil {
		t.Fatalf("Failed to create test crypto instance: %v", err)
	}

	return crypto, kmsService
}

// NewTestCryptoWithKMS creates a test crypto instance with a specific KMS implementation
func NewTestCryptoWithKMS(t testing.TB, kms encx.KeyManagementService) *encx.Crypto {
	t.Helper()

	crypto, _ := NewTestCrypto(t, &TestCryptoOptions{
		CustomKMSMock: kms,
	})

	return crypto
}

// TestDataFactory provides utilities for creating predictable test data
type TestDataFactory struct {
	crypto encx.CryptoService
}

// NewTestDataFactory creates a new test data factory with the given crypto service
func NewTestDataFactory(crypto encx.CryptoService) *TestDataFactory {
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
		PlainField:     "plain-" + plainValue,
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
	PlainField              string `json:"plain_field"`
	EncryptedField          string `encx:"encrypt" json:"encrypted_field"`
	EncryptedFieldEncrypted []byte `json:"encrypted_field_encrypted"`
	DEK                     []byte `json:"-"` // DEK field required by encx
	DEKEncrypted            []byte `json:"dek_encrypted"`
	KeyVersion              int    `json:"key_version"`
}
