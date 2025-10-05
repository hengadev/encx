//go:build integration
// +build integration

package kms_providers_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/hengadev/encx"
	"github.com/hengadev/encx/providers/hashicorpvault"
)

// VaultIntegrationTestSuite contains integration tests for HashiCorp Vault KMS provider
type VaultIntegrationTestSuite struct {
	suite.Suite
	vault   *hashicorpvault.VaultService
	crypto  *encx.Crypto
	ctx     context.Context
	keyID   string
	testDEK []byte
}

// SetupSuite runs once before all tests in the suite
func (suite *VaultIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Skip if Vault environment variables are not set
	if os.Getenv("VAULT_ADDR") == "" {
		suite.T().Skip("Skipping Vault integration tests: VAULT_ADDR not set")
	}

	// Create Vault service
	vault, err := hashicorpvault.New()
	require.NoError(suite.T(), err, "Failed to create Vault service")
	suite.vault = vault

	// Create a test key
	keyName := "encx-integration-test-" + time.Now().Format("20060102-150405")
	keyID, err := vault.CreateKey(suite.ctx, keyName)
	require.NoError(suite.T(), err, "Failed to create test key")
	suite.keyID = keyID

	// Create crypto instance with Vault KMS
	suite.crypto, err = encx.NewCrypto(suite.ctx,
		encx.WithKMSService(vault),
		encx.WithKEKAlias(keyID),
		encx.WithPepperSecretPath("secret/encx/pepper"),
	)
	require.NoError(suite.T(), err, "Failed to create crypto instance")

	// Generate test DEK
	suite.testDEK = make([]byte, 32)
	for i := range suite.testDEK {
		suite.testDEK[i] = byte(i % 256)
	}
}

// TearDownSuite runs once after all tests in the suite
func (suite *VaultIntegrationTestSuite) TearDownSuite() {
	// Note: We don't delete the test key as Vault transit keys cannot be deleted
	// They can only be marked for deletion and will be cleaned up by Vault policies
}

// TestVaultKMSBasicOperations tests basic KMS operations
func (suite *VaultIntegrationTestSuite) TestVaultKMSBasicOperations() {
	// Test GetKeyID
	retrievedKeyID, err := suite.vault.GetKeyID(suite.ctx, suite.keyID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.keyID, retrievedKeyID)

	// Test EncryptDEK
	encryptedDEK, err := suite.vault.EncryptDEK(suite.ctx, suite.keyID, suite.testDEK)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), encryptedDEK)
	assert.NotEqual(suite.T(), suite.testDEK, encryptedDEK)

	// Test DecryptDEK
	decryptedDEK, err := suite.vault.DecryptDEK(suite.ctx, suite.keyID, encryptedDEK)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testDEK, decryptedDEK)
}

// TestVaultKMSEncryptionRoundtrip tests full encryption/decryption roundtrip
func (suite *VaultIntegrationTestSuite) TestVaultKMSEncryptionRoundtrip() {
	// Multiple rounds to test consistency
	for i := 0; i < 5; i++ {
		// Encrypt DEK
		encrypted, err := suite.vault.EncryptDEK(suite.ctx, suite.keyID, suite.testDEK)
		assert.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), encrypted)

		// Decrypt DEK
		decrypted, err := suite.vault.DecryptDEK(suite.ctx, suite.keyID, encrypted)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), suite.testDEK, decrypted)
	}
}

// TestVaultKMSSecretOperations tests secret storage operations
func (suite *VaultIntegrationTestSuite) TestVaultKMSSecretOperations() {
	secretPath := "secret/encx/integration-test-pepper"
	testPepper := []byte("test-pepper-value-for-integration")

	// Test SetSecret
	err := suite.vault.SetSecret(suite.ctx, secretPath, testPepper)
	assert.NoError(suite.T(), err)

	// Test GetSecret
	retrievedPepper, err := suite.vault.GetSecret(suite.ctx, secretPath)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testPepper, retrievedPepper)
}

// TestVaultKMSKeyRotation tests key rotation functionality
func (suite *VaultIntegrationTestSuite) TestVaultKMSKeyRotation() {
	// Encrypt with current key version
	encrypted1, err := suite.vault.EncryptDEK(suite.ctx, suite.keyID, suite.testDEK)
	assert.NoError(suite.T(), err)

	// Rotate key
	err = suite.vault.RotateKey(suite.ctx, suite.keyID)
	assert.NoError(suite.T(), err)

	// Encrypt with new key version
	encrypted2, err := suite.vault.EncryptDEK(suite.ctx, suite.keyID, suite.testDEK)
	assert.NoError(suite.T(), err)

	// Both encrypted values should be different (different key versions)
	assert.NotEqual(suite.T(), encrypted1, encrypted2)

	// Both should decrypt to the same plaintext
	decrypted1, err := suite.vault.DecryptDEK(suite.ctx, suite.keyID, encrypted1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testDEK, decrypted1)

	decrypted2, err := suite.vault.DecryptDEK(suite.ctx, suite.keyID, encrypted2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testDEK, decrypted2)
}

// TestVaultIntegratedCryptoOperations tests crypto operations using Vault KMS
func (suite *VaultIntegrationTestSuite) TestVaultIntegratedCryptoOperations() {
	// Test data encryption
	testData := []byte("sensitive user data for vault integration test")

	// Generate DEK
	dek, err := suite.crypto.GenerateDEK()
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), dek, 32) // AES-256 key length

	// Encrypt data
	encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), testData, encrypted)

	// Decrypt data
	decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testData, decrypted)

	// Test DEK encryption with KMS
	encryptedDEK, err := suite.crypto.EncryptDEK(suite.ctx, dek)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), encryptedDEK)

	// Test DEK decryption with KMS
	decryptedDEK, err := suite.crypto.DecryptDEKWithVersion(suite.ctx, encryptedDEK, 1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), dek, decryptedDEK)
}

// TestVaultKMSErrorHandling tests error conditions
func (suite *VaultIntegrationTestSuite) TestVaultKMSErrorHandling() {
	// Test with invalid key ID
	_, err := suite.vault.EncryptDEK(suite.ctx, "non-existent-key", suite.testDEK)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to encrypt")

	// Test decryption with invalid ciphertext
	_, err = suite.vault.DecryptDEK(suite.ctx, suite.keyID, []byte("invalid-ciphertext"))
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to decrypt")

	// Test getting non-existent secret
	_, err = suite.vault.GetSecret(suite.ctx, "secret/non-existent/path")
	assert.Error(suite.T(), err)
}

// TestVaultKMSConcurrentOperations tests concurrent access
func (suite *VaultIntegrationTestSuite) TestVaultKMSConcurrentOperations() {
	const numGoroutines = 10
	const operationsPerGoroutine = 5

	// Channel to collect results
	results := make(chan error, numGoroutines*operationsPerGoroutine)

	// Launch concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				// Encrypt
				encrypted, err := suite.vault.EncryptDEK(suite.ctx, suite.keyID, suite.testDEK)
				if err != nil {
					results <- err
					continue
				}

				// Decrypt
				decrypted, err := suite.vault.DecryptDEK(suite.ctx, suite.keyID, encrypted)
				if err != nil {
					results <- err
					continue
				}

				// Verify
				if !assert.ObjectsAreEqual(suite.testDEK, decrypted) {
					results <- assert.AnError
					continue
				}

				results <- nil // Success
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines*operationsPerGoroutine; i++ {
		select {
		case err := <-results:
			assert.NoError(suite.T(), err)
		case <-time.After(30 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// TestSuite entry point
func TestVaultIntegrationSuite(t *testing.T) {
	suite.Run(t, new(VaultIntegrationTestSuite))
}

// TestVaultEnvironmentSetup tests that Vault environment is properly configured
func TestVaultEnvironmentSetup(t *testing.T) {
	if os.Getenv("VAULT_ADDR") == "" {
		t.Skip("Skipping Vault environment test: VAULT_ADDR not set")
	}

	vault, err := hashicorpvault.New()
	require.NoError(t, err, "Should be able to create Vault service")
	require.NotNil(t, vault, "Vault service should not be nil")

	// Test basic connectivity by attempting to get a key ID
	ctx := context.Background()
	_, err = vault.GetKeyID(ctx, "test-key")
	// This should not error for key ID retrieval (it just returns the alias)
	assert.NoError(t, err)
}