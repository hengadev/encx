package encx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTestCrypto demonstrates the basic usage of the test factory
func TestNewTestCrypto(t *testing.T) {
	ctx := context.Background()

	// Create a test crypto instance with default settings
	crypto, kmsMock := NewTestCrypto(t)
	assert.NotNil(t, crypto)
	assert.NotNil(t, kmsMock)

	// Test basic crypto operations
	dek, err := crypto.GenerateDEK()
	require.NoError(t, err)
	assert.Len(t, dek, 32) // Should be 32 bytes for AES-256

	// Test encryption/decryption
	plaintext := []byte("test data")
	encrypted, err := crypto.EncryptData(ctx, plaintext, dek)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := crypto.DecryptData(ctx, encrypted, dek)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TestCryptoServiceMock demonstrates how to use the CryptoService mock
func TestCryptoServiceMock(t *testing.T) {
	ctx := context.Background()

	// Create a mock crypto service
	mockCrypto := NewCryptoServiceMock()

	// Set up expectations
	mockCrypto.On("GenerateDEK").Return([]byte("fake-dek-32-chars-for-testing!!"), nil)
	mockCrypto.On("EncryptData", ctx, []byte("test"), []byte("fake-dek-32-chars-for-testing!!")).
		Return([]byte("encrypted-data"), nil)
	mockCrypto.On("GetPepper").Return([]byte("test-pepper-32-chars-for-testing"))

	// Use the mock
	dek, err := mockCrypto.GenerateDEK()
	require.NoError(t, err)
	assert.Equal(t, []byte("fake-dek-32-chars-for-testing!!"), dek)

	encrypted, err := mockCrypto.EncryptData(ctx, []byte("test"), dek)
	require.NoError(t, err)
	assert.Equal(t, []byte("encrypted-data"), encrypted)

	pepper := mockCrypto.GetPepper()
	assert.Equal(t, []byte("test-pepper-32-chars-for-testing"), pepper)

	// Verify all expectations were met
	mockCrypto.AssertExpectations(t)
}

// TestTestDataFactory demonstrates predictable encrypted data creation
func TestTestDataFactory(t *testing.T) {
	ctx := context.Background()

	// Create test crypto instance
	crypto, _ := NewTestCrypto(t)

	// Create test data factory
	factory := NewTestDataFactory(crypto)

	// Create predictable encrypted data
	encrypted1, dek1, err := factory.CreatePredictableEncryptedData(ctx, "test-value")
	require.NoError(t, err)

	encrypted2, dek2, err := factory.CreatePredictableEncryptedData(ctx, "test-value")
	require.NoError(t, err)

	// DEKs should be the same (predictable)
	assert.Equal(t, dek1, dek2)

	// But encrypted data should be different due to random nonces
	// (this is expected behavior for proper encryption)
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same value
	decrypted1, err := crypto.DecryptData(ctx, encrypted1, dek1)
	require.NoError(t, err)

	decrypted2, err := crypto.DecryptData(ctx, encrypted2, dek2)
	require.NoError(t, err)

	assert.Equal(t, []byte("test-value"), decrypted1)
	assert.Equal(t, []byte("test-value"), decrypted2)
	assert.Equal(t, decrypted1, decrypted2)
}

// TestStructEncryption demonstrates struct-level encryption testing
func TestStructEncryption(t *testing.T) {
	ctx := context.Background()

	// Create test crypto instance
	crypto, _ := NewTestCrypto(t)

	// Create test data factory
	factory := NewTestDataFactory(crypto)

	// Create and process test struct
	testStruct, err := factory.CreateTestStruct(ctx, "sensitive-phone-number")
	require.NoError(t, err)

	// Verify struct was processed correctly
	assert.Equal(t, "plain-sensitive-phone-number", testStruct.PlainField)
	assert.Empty(t, testStruct.EncryptedField)  // Should be empty after encryption
	assert.NotNil(t, testStruct.DEKEncrypted)   // Should have encrypted DEK
	assert.Greater(t, testStruct.KeyVersion, 0) // Should have key version

	// Test decryption
	err = crypto.DecryptStruct(ctx, testStruct)
	require.NoError(t, err)

	// Verify decryption worked
	assert.Equal(t, "plain-sensitive-phone-number", testStruct.PlainField)
	assert.Equal(t, "sensitive-phone-number", testStruct.EncryptedField) // Should be decrypted
}

// TestPhoneEncryptionScenario demonstrates a real-world phone number encryption scenario
func TestPhoneEncryptionScenario(t *testing.T) {
	ctx := context.Background()

	// This addresses the original comment about phone GET endpoint testing limitations

	// Create test crypto instance
	crypto, _ := NewTestCrypto(t)

	// Define a phone number struct similar to what might be used in an API
	type User struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		PhoneNumber string `encx:"encrypt" json:"phone_number"`
		Email       string `json:"email"`

		// Required fields for encx
		DEK          []byte `json:"-"`
		DEKEncrypted []byte `json:"dek_encrypted"`
		KeyVersion   int    `json:"key_version"`
	}

	// Test data
	user := &User{
		ID:          123,
		Name:        "John Doe",
		PhoneNumber: "+1-555-0123",
		Email:       "john@example.com",
	}

	// Encrypt the struct (simulate saving to database)
	err := crypto.ProcessStruct(ctx, user)
	require.NoError(t, err)

	// Verify phone number was encrypted (is now empty)
	assert.Equal(t, 123, user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Empty(t, user.PhoneNumber) // Should be encrypted and cleared
	assert.Equal(t, "john@example.com", user.Email)
	assert.NotNil(t, user.DEKEncrypted)
	assert.Greater(t, user.KeyVersion, 0)

	// Simulate reading from database and decrypting for API response
	userFromDB := &User{
		ID:           user.ID,
		Name:         user.Name,
		PhoneNumber:  "", // Empty from DB
		Email:        user.Email,
		DEKEncrypted: user.DEKEncrypted,
		KeyVersion:   user.KeyVersion,
	}

	// Decrypt the phone number
	err = crypto.DecryptStruct(ctx, userFromDB)
	require.NoError(t, err)

	// Verify phone number was decrypted correctly
	assert.Equal(t, "+1-555-0123", userFromDB.PhoneNumber)

	// This test can now be used reliably in integration testing environments
	// because it doesn't depend on external services or complex setup
}

// TestCustomTestCryptoOptions demonstrates advanced test crypto configuration
func TestCustomTestCryptoOptions(t *testing.T) {
	// Create custom KMS mock with specific behavior
	kmsMock := NewCryptoServiceMock()
	kmsMock.On("GetPepper").
		Return([]byte("custom-pepper-32-chars-for-test"))

	// Create crypto with custom options
	crypto, _ := NewTestCrypto(t, &TestCryptoOptions{
		UseRealDatabase: false, // Use in-memory DB
		CustomPepper:    []byte("custom-pepper-32-chars-for-test"),
	})

	// Test that custom configuration works
	pepper := crypto.GetPepper()
	assert.Equal(t, []byte("custom-pepper-32-chars-for-test"), pepper)

	alias := crypto.GetAlias()
	assert.Equal(t, "test-key-alias", alias) // Default test alias
}

// BenchmarkTestCryptoCreation benchmarks the test crypto creation performance
func BenchmarkTestCryptoCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		crypto, _ := NewTestCrypto(b)
		_ = crypto // Use the variable
	}
}

// ExampleNewTestCrypto shows how to use the test factory in documentation
func ExampleNewTestCrypto() {
	// This example demonstrates the simplest way to create a test crypto instance

	// In your test function:
	// crypto, kmsMock := NewTestCrypto(t)

	// Now you can use crypto for testing without any external dependencies
	// The kmsMock can be configured for specific test scenarios
}

// ExampleCryptoServiceMock shows how to use the CryptoService mock
func ExampleCryptoServiceMock() {
	// Create a mock crypto service
	mockCrypto := NewCryptoServiceMock()

	// Set up expectations
	ctx := context.Background()
	mockCrypto.On("GenerateDEK").Return([]byte("test-dek-32-chars-for-testing!!"), nil)
	mockCrypto.On("EncryptData", ctx, []byte("data"), []byte("test-dek-32-chars-for-testing!!")).
		Return([]byte("encrypted"), nil)

	// Use in your application code (dependency injection)
	// service := NewMyService(mockCrypto)
	// result := service.ProcessData("data")

	// Verify expectations were met
	// mockCrypto.AssertExpectations(t)
}

