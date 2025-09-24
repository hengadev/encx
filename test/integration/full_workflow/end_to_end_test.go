package full_workflow_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/hengadev/encx"
	"github.com/hengadev/encx/internal/codegen"
	"github.com/hengadev/encx/internal/config"
)

// EndToEndWorkflowTestSuite tests complete ENCX workflows from configuration to operation
type EndToEndWorkflowTestSuite struct {
	suite.Suite
	tempDir string
	crypto  *encx.Crypto
	ctx     context.Context
}

// SetupSuite creates test environment
func (suite *EndToEndWorkflowTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "encx-e2e-workflow-*")
	require.NoError(suite.T(), err)
	suite.tempDir = tempDir

	// Create crypto instance with test KMS
	suite.crypto, err = encx.NewCrypto(suite.ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("e2e-workflow-key"),
		encx.WithPepper([]byte("e2e-test-pepper-32-bytes-exactly")),
	)
	require.NoError(suite.T(), err)
}

// TearDownSuite cleans up
func (suite *EndToEndWorkflowTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// TestCompleteUserDataWorkflow tests a realistic user data encryption scenario
func (suite *EndToEndWorkflowTestSuite) TestCompleteUserDataWorkflow() {
	// Phase 1: Configuration and Code Generation
	configFile := filepath.Join(suite.tempDir, "encx.yaml")
	configContent := `version: "1.0"
generation_options:
  output_dir: "./generated"
  package_name: "models"
  include_validation: true
  include_json_tags: true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(suite.T(), err)

	// Phase 2: Create user model with encryption tags
	userModelFile := filepath.Join(suite.tempDir, "user.go")
	userModelContent := `package models

import "time"

// User represents a user account with sensitive data
type User struct {
	ID          int64     ` + "`json:\"id\" db:\"id\"`" + `
	Username    string    ` + "`json:\"username\" db:\"username\"`" + `
	Email       string    ` + "`json:\"email\" encx:\"encrypt,hash_basic\" db:\"email\"`" + `
	Phone       string    ` + "`json:\"phone\" encx:\"encrypt\" db:\"phone\"`" + `
	SSN         string    ` + "`json:\"ssn\" encx:\"hash_secure\" db:\"ssn\"`" + `
	CreditCard  string    ` + "`json:\"-\" encx:\"encrypt\" db:\"credit_card\"`" + `
	Salary      int64     ` + "`json:\"salary\" encx:\"encrypt\" db:\"salary\"`" + `
	DateOfBirth time.Time ` + "`json:\"dob\" encx:\"encrypt\" db:\"date_of_birth\"`" + `
	IsActive    bool      ` + "`json:\"is_active\" db:\"is_active\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `

	// Encrypted companion fields
	EmailEncrypted      []byte ` + "`json:\"email_encrypted,omitempty\" db:\"email_encrypted\"`" + `
	EmailHash           string ` + "`json:\"email_hash,omitempty\" db:\"email_hash\"`" + `
	PhoneEncrypted      []byte ` + "`json:\"phone_encrypted,omitempty\" db:\"phone_encrypted\"`" + `
	SSNHashSecure       string ` + "`json:\"ssn_hash_secure,omitempty\" db:\"ssn_hash_secure\"`" + `
	CreditCardEncrypted []byte ` + "`json:\"-\" db:\"credit_card_encrypted\"`" + `
	SalaryEncrypted     []byte ` + "`json:\"salary_encrypted,omitempty\" db:\"salary_encrypted\"`" + `
	DateOfBirthEncrypted []byte ` + "`json:\"dob_encrypted,omitempty\" db:\"dob_encrypted\"`" + `
}
`

	err = os.WriteFile(userModelFile, []byte(userModelContent), 0644)
	require.NoError(suite.T(), err)

	// Phase 3: Run code generation
	generator := codegen.NewGenerator()
	err = generator.ProcessDirectory(suite.tempDir, configFile)
	require.NoError(suite.T(), err, "Code generation should succeed")

	// Phase 4: Test generated encryption methods exist
	generatedFile := filepath.Join(suite.tempDir, "generated", "user_encx.go")
	assert.FileExists(suite.T(), generatedFile, "Generated encryption file should exist")

	// Phase 5: Create test user and test encryption workflow
	testUser := createTestUser()

	// Phase 6: Test field-level encryption operations
	suite.testFieldEncryption(testUser)

	// Phase 7: Test searchable hashing operations
	suite.testHashingOperations(testUser)

	// Phase 8: Test bulk operations
	suite.testBulkOperations()

	// Phase 9: Test concurrent operations
	suite.testConcurrentOperations()
}

// TestHealthCheckIntegration tests integration with health check systems
func (suite *EndToEndWorkflowTestSuite) TestHealthCheckIntegration() {
	// Test that crypto operations work with health monitoring
	ctx := context.WithValue(suite.ctx, "health_check", true)

	testData := []byte("health check test data")

	// Generate DEK
	dek, err := suite.crypto.GenerateDEK(ctx)
	require.NoError(suite.T(), err)

	// Encrypt data
	encrypted, err := suite.crypto.EncryptData(ctx, testData, dek)
	require.NoError(suite.T(), err)

	// Decrypt data
	decrypted, err := suite.crypto.DecryptData(ctx, encrypted, dek)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), testData, decrypted)
}

// TestMultipleDataTypes tests encryption of various Go data types
func (suite *EndToEndWorkflowTestSuite) TestMultipleDataTypes() {
	testCases := []struct {
		name string
		data interface{}
	}{
		{"String", "sensitive string data"},
		{"Integer", int64(123456789)},
		{"Float", 99.99},
		{"Boolean", true},
		{"Time", time.Now()},
		{"Bytes", []byte("binary data")},
		{"LargeString", string(make([]byte, 10000))}, // 10KB string
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Convert to bytes for encryption
			var dataBytes []byte
			var err error

			switch v := tc.data.(type) {
			case string:
				dataBytes = []byte(v)
			case int64:
				dataBytes = []byte(fmt.Sprintf("%d", v))
			case float64:
				dataBytes = []byte(fmt.Sprintf("%f", v))
			case bool:
				if v {
					dataBytes = []byte("true")
				} else {
					dataBytes = []byte("false")
				}
			case time.Time:
				dataBytes = []byte(v.Format(time.RFC3339))
			case []byte:
				dataBytes = v
			}

			// Generate DEK
			dek, err := suite.crypto.GenerateDEK(suite.ctx)
			require.NoError(t, err)

			// Encrypt
			encrypted, err := suite.crypto.EncryptData(suite.ctx, dataBytes, dek)
			require.NoError(t, err)
			assert.NotEqual(t, dataBytes, encrypted)

			// Decrypt
			decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
			require.NoError(t, err)
			assert.Equal(t, dataBytes, decrypted)
		})
	}
}

// testFieldEncryption tests individual field encryption
func (suite *EndToEndWorkflowTestSuite) testFieldEncryption(user TestUser) {
	// Test email encryption
	dek, err := suite.crypto.GenerateDEK(suite.ctx)
	require.NoError(suite.T(), err)

	emailBytes := []byte(user.Email)
	encryptedEmail, err := suite.crypto.EncryptData(suite.ctx, emailBytes, dek)
	require.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), emailBytes, encryptedEmail)

	// Test decryption
	decryptedEmail, err := suite.crypto.DecryptData(suite.ctx, encryptedEmail, dek)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), emailBytes, decryptedEmail)
}

// testHashingOperations tests searchable hashing
func (suite *EndToEndWorkflowTestSuite) testHashingOperations(user TestUser) {
	// Test basic hash (for searching)
	emailHash, err := suite.crypto.HashForSearch(suite.ctx, user.Email)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), emailHash)

	// Test secure hash (for authentication)
	ssnHash, err := suite.crypto.HashSecure(suite.ctx, user.SSN)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), ssnHash)

	// Ensure different data produces different hashes
	differentEmailHash, err := suite.crypto.HashForSearch(suite.ctx, "different@example.com")
	require.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), emailHash, differentEmailHash)
}

// testBulkOperations tests processing multiple records
func (suite *EndToEndWorkflowTestSuite) testBulkOperations() {
	users := []TestUser{
		createTestUser(),
		{
			ID:          2,
			Email:       "user2@example.com",
			Phone:       "+1-555-0002",
			SSN:         "987-65-4321",
			CreditCard:  "5555555555554444",
			Salary:      75000,
		},
		{
			ID:          3,
			Email:       "user3@example.com",
			Phone:       "+1-555-0003",
			SSN:         "111-22-3333",
			CreditCard:  "4111111111111111",
			Salary:      90000,
		},
	}

	// Process all users
	dek, err := suite.crypto.GenerateDEK(suite.ctx)
	require.NoError(suite.T(), err)

	for i, user := range users {
		// Encrypt sensitive fields
		encryptedEmail, err := suite.crypto.EncryptData(suite.ctx, []byte(user.Email), dek)
		require.NoError(suite.T(), err, "Failed to encrypt email for user %d", i+1)

		encryptedPhone, err := suite.crypto.EncryptData(suite.ctx, []byte(user.Phone), dek)
		require.NoError(suite.T(), err, "Failed to encrypt phone for user %d", i+1)

		// Hash for searching
		emailHash, err := suite.crypto.HashForSearch(suite.ctx, user.Email)
		require.NoError(suite.T(), err, "Failed to hash email for user %d", i+1)

		// Store encrypted versions
		users[i].EmailEncrypted = encryptedEmail
		users[i].PhoneEncrypted = encryptedPhone
		users[i].EmailHash = emailHash
	}

	// Verify all encryptions worked
	for i, user := range users {
		assert.NotEmpty(suite.T(), user.EmailEncrypted, "User %d should have encrypted email", i+1)
		assert.NotEmpty(suite.T(), user.PhoneEncrypted, "User %d should have encrypted phone", i+1)
		assert.NotEmpty(suite.T(), user.EmailHash, "User %d should have email hash", i+1)
	}
}

// testConcurrentOperations tests thread safety
func (suite *EndToEndWorkflowTestSuite) testConcurrentOperations() {
	const numGoroutines = 10
	const operationsPerGoroutine = 5

	results := make(chan error, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				testData := []byte(fmt.Sprintf("test-data-g%d-op%d", goroutineID, j))

				// Generate DEK
				dek, err := suite.crypto.GenerateDEK(suite.ctx)
				if err != nil {
					results <- err
					continue
				}

				// Encrypt
				encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
				if err != nil {
					results <- err
					continue
				}

				// Decrypt
				decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
				if err != nil {
					results <- err
					continue
				}

				// Verify
				if string(decrypted) != string(testData) {
					results <- fmt.Errorf("data mismatch in goroutine %d operation %d", goroutineID, j)
					continue
				}

				results <- nil
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines*operationsPerGoroutine; i++ {
		select {
		case err := <-results:
			require.NoError(suite.T(), err)
		case <-time.After(30 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// TestUser represents test user data
type TestUser struct {
	ID          int64
	Email       string
	Phone       string
	SSN         string
	CreditCard  string
	Salary      int64

	// Encrypted fields
	EmailEncrypted      []byte
	PhoneEncrypted      []byte
	CreditCardEncrypted []byte
	SalaryEncrypted     []byte

	// Hash fields
	EmailHash     string
	SSNHashSecure string
}

// createTestUser creates a test user with sample data
func createTestUser() TestUser {
	return TestUser{
		ID:         1,
		Email:      "test@example.com",
		Phone:      "+1-555-0001",
		SSN:        "123-45-6789",
		CreditCard: "4000000000000002",
		Salary:     85000,
	}
}

// TestSuite entry point
func TestEndToEndWorkflowSuite(t *testing.T) {
	suite.Run(t, new(EndToEndWorkflowTestSuite))
}