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
	awskms "github.com/hengadev/encx/providers/keys/aws"
)

// AWSKMSIntegrationTestSuite contains integration tests for AWS KMS provider
type AWSKMSIntegrationTestSuite struct {
	suite.Suite
	awsKMS encx.KeyManagementService
	crypto *encx.Crypto
	ctx    context.Context
	keyID  string
}

// SetupSuite runs once before all tests in the suite
func (suite *AWSKMSIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Skip if AWS environment variables are not set
	if os.Getenv("AWS_REGION") == "" || os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		suite.T().Skip("Skipping AWS KMS integration tests: AWS credentials not configured")
	}

	// Create AWS KMS service
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	kmsService, err := awskms.NewKMSService(suite.ctx, awskms.Config{
		Region: region,
	})
	require.NoError(suite.T(), err, "Failed to create AWS KMS service")
	suite.awsKMS = kmsService

	// Use existing key or create a test key
	keyAlias := os.Getenv("AWS_KMS_KEY_ALIAS")
	if keyAlias == "" {
		// Create a new test key
		keyID, err := suite.awsKMS.CreateKey(suite.ctx, "encx-integration-test")
		require.NoError(suite.T(), err, "Failed to create test key")
		suite.keyID = keyID
		suite.T().Logf("Created test KMS key: %s", keyID)
	} else {
		// Use existing key alias
		keyID, err := suite.awsKMS.GetKeyID(suite.ctx, keyAlias)
		require.NoError(suite.T(), err, "Failed to get KMS key ID for alias: %s", keyAlias)
		suite.keyID = keyID
		suite.T().Logf("Using existing KMS key: %s (alias: %s)", keyID, keyAlias)
	}

	// Create in-memory secret store for integration testing
	// (In production, use aws.NewSecretsManagerStore)
	secretStore := encx.NewInMemorySecretStore()

	// Create crypto instance with AWS KMS and new API
	cfg := encx.Config{
		KEKAlias:    suite.keyID,
		PepperAlias: "integration-test",
	}

	suite.crypto, err = encx.NewCrypto(suite.ctx, suite.awsKMS, secretStore, cfg)
	require.NoError(suite.T(), err, "Failed to create crypto instance")
}

// TestAWSKMSBasicOperations tests basic AWS KMS operations
func (suite *AWSKMSIntegrationTestSuite) TestAWSKMSBasicOperations() {
	// Test DEK encryption/decryption
	testDEK := make([]byte, 32)
	for i := range testDEK {
		testDEK[i] = byte(i % 256)
	}

	// Test EncryptDEK
	encryptedDEK, err := suite.awsKMS.EncryptDEK(suite.ctx, suite.keyID, testDEK)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), encryptedDEK)

	// Test DecryptDEK
	decryptedDEK, err := suite.awsKMS.DecryptDEK(suite.ctx, suite.keyID, encryptedDEK)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testDEK, decryptedDEK)
}

// TestAWSKMSEncryptionRoundtrip tests full encryption/decryption roundtrip
func (suite *AWSKMSIntegrationTestSuite) TestAWSKMSEncryptionRoundtrip() {
	testDEK := make([]byte, 32)
	for i := range testDEK {
		testDEK[i] = byte(i % 256)
	}

	// Multiple rounds to test consistency
	for i := 0; i < 5; i++ {
		encrypted, err := suite.awsKMS.EncryptDEK(suite.ctx, suite.keyID, testDEK)
		assert.NoError(suite.T(), err)

		decrypted, err := suite.awsKMS.DecryptDEK(suite.ctx, suite.keyID, encrypted)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), testDEK, decrypted)
	}
}

// TestAWSKMSWithCryptoOperations tests crypto operations using AWS KMS
func (suite *AWSKMSIntegrationTestSuite) TestAWSKMSWithCryptoOperations() {
	// Test data encryption with AWS KMS
	testData := []byte("sensitive user data for AWS KMS integration test")

	// Generate DEK
	dek, err := suite.crypto.GenerateDEK()
	assert.NoError(suite.T(), err)

	// Encrypt data
	encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
	assert.NoError(suite.T(), err)

	// Decrypt data
	decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), testData, decrypted)
}

// TestAWSKMSConcurrentOperations tests concurrent access
func (suite *AWSKMSIntegrationTestSuite) TestAWSKMSConcurrentOperations() {
	const numGoroutines = 10
	const operationsPerGoroutine = 3

	testDEK := make([]byte, 32)
	for i := range testDEK {
		testDEK[i] = byte(i % 256)
	}

	results := make(chan error, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < operationsPerGoroutine; j++ {
				encrypted, err := suite.awsKMS.EncryptDEK(suite.ctx, suite.keyID, testDEK)
				if err != nil {
					results <- err
					continue
				}

				decrypted, err := suite.awsKMS.DecryptDEK(suite.ctx, suite.keyID, encrypted)
				if err != nil {
					results <- err
					continue
				}

				if !assert.ObjectsAreEqual(testDEK, decrypted) {
					results <- assert.AnError
					continue
				}

				results <- nil
			}
		}()
	}

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
func TestAWSKMSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(AWSKMSIntegrationTestSuite))
}

// TestAWSEnvironmentSetup tests that AWS environment is properly configured
func TestAWSEnvironmentSetup(t *testing.T) {
	if os.Getenv("AWS_REGION") == "" {
		t.Skip("Skipping AWS environment test: AWS_REGION not set")
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("Skipping AWS environment test: AWS_ACCESS_KEY_ID not set")
	}

	ctx := context.Background()
	region := os.Getenv("AWS_REGION")

	// Test AWS KMS service creation
	kmsService, err := awskms.NewKMSService(ctx, awskms.Config{
		Region: region,
	})
	require.NoError(t, err, "Failed to create AWS KMS service")
	assert.NotNil(t, kmsService)

	t.Logf("Successfully created AWS KMS service for region: %s", region)
}