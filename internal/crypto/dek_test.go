package crypto

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKMSService is a mock implementation of KeyManagementService
type MockKMSService struct {
	mock.Mock
}

func (m *MockKMSService) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	args := m.Called(ctx, keyID, plaintext)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKMSService) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	args := m.Called(ctx, keyID, ciphertext)
	return args.Get(0).([]byte), args.Error(1)
}

// MockVersionManager is a mock implementation of KMSVersionManager
type MockVersionManager struct {
	mock.Mock
}

func (m *MockVersionManager) GetCurrentKEKVersion(ctx context.Context, alias string) (int, error) {
	args := m.Called(ctx, alias)
	return args.Int(0), args.Error(1)
}

func (m *MockVersionManager) GetKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error) {
	args := m.Called(ctx, alias, version)
	return args.String(0), args.Error(1)
}

func TestNewDEKOperations(t *testing.T) {
	mockKMS := &MockKMSService{}
	kekAlias := "test-alias"

	dekOps := NewDEKOperations(mockKMS, kekAlias)

	assert.NotNil(t, dekOps)
	assert.Equal(t, mockKMS, dekOps.kmsService)
	assert.Equal(t, kekAlias, dekOps.kekAlias)
}

func TestDEKOperations_GenerateDEK(t *testing.T) {
	mockKMS := &MockKMSService{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	dek, err := dekOps.GenerateDEK()

	require.NoError(t, err)
	assert.Len(t, dek, 32) // Should be 32 bytes for AES-256

	// Generate multiple DEKs to ensure they're different (randomness test)
	dek2, err := dekOps.GenerateDEK()
	require.NoError(t, err)
	assert.NotEqual(t, dek, dek2, "Generated DEKs should be different")
}

func TestDEKOperations_EncryptDEK_Success(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	plaintextDEK := []byte("test-dek-32-bytes-for-encryption!")
	expectedCiphertext := []byte("encrypted-dek-data")

	// Set up mock expectations
	mockVersionManager.On("GetCurrentKEKVersion", ctx, "test-alias").Return(1, nil)
	mockVersionManager.On("GetKMSKeyIDForVersion", ctx, "test-alias", 1).Return("kms-key-id", nil)
	mockKMS.On("EncryptDEK", ctx, "kms-key-id", plaintextDEK).Return(expectedCiphertext, nil)

	// Execute
	ciphertext, err := dekOps.EncryptDEK(ctx, plaintextDEK, mockVersionManager)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedCiphertext, ciphertext)

	// Verify all expectations were met
	mockKMS.AssertExpectations(t)
	mockVersionManager.AssertExpectations(t)
}

func TestDEKOperations_EncryptDEK_GetVersionError(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	plaintextDEK := []byte("test-dek-32-bytes-for-encryption!")
	expectedError := errors.New("version error")

	// Set up mock expectations
	mockVersionManager.On("GetCurrentKEKVersion", ctx, "test-alias").Return(0, expectedError)

	// Execute
	ciphertext, err := dekOps.EncryptDEK(ctx, plaintextDEK, mockVersionManager)

	// Verify
	assert.Nil(t, ciphertext)
	assert.Equal(t, expectedError, err)

	mockVersionManager.AssertExpectations(t)
}

func TestDEKOperations_EncryptDEK_GetKeyIDError(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	plaintextDEK := []byte("test-dek-32-bytes-for-encryption!")
	expectedError := errors.New("key ID error")

	// Set up mock expectations
	mockVersionManager.On("GetCurrentKEKVersion", ctx, "test-alias").Return(1, nil)
	mockVersionManager.On("GetKMSKeyIDForVersion", ctx, "test-alias", 1).Return("", expectedError)

	// Execute
	ciphertext, err := dekOps.EncryptDEK(ctx, plaintextDEK, mockVersionManager)

	// Verify
	assert.Nil(t, ciphertext)
	assert.Equal(t, expectedError, err)

	mockVersionManager.AssertExpectations(t)
}

func TestDEKOperations_EncryptDEK_KMSError(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	plaintextDEK := []byte("test-dek-32-bytes-for-encryption!")
	kmsError := errors.New("KMS encryption failed")

	// Set up mock expectations
	mockVersionManager.On("GetCurrentKEKVersion", ctx, "test-alias").Return(1, nil)
	mockVersionManager.On("GetKMSKeyIDForVersion", ctx, "test-alias", 1).Return("kms-key-id", nil)
	mockKMS.On("EncryptDEK", ctx, "kms-key-id", plaintextDEK).Return([]byte(nil), kmsError)

	// Execute
	ciphertext, err := dekOps.EncryptDEK(ctx, plaintextDEK, mockVersionManager)

	// Verify
	assert.Nil(t, ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encrypt DEK with KMS (version 1)")
	assert.Contains(t, err.Error(), kmsError.Error())

	// Verify all expectations were met
	mockKMS.AssertExpectations(t)
	mockVersionManager.AssertExpectations(t)
}

func TestDEKOperations_DecryptDEKWithVersion_Success(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	ciphertextDEK := []byte("encrypted-dek-data")
	expectedPlaintext := []byte("test-dek-32-bytes-for-encryption!")
	kekVersion := 1

	// Set up mock expectations
	mockVersionManager.On("GetKMSKeyIDForVersion", ctx, "test-alias", kekVersion).Return("kms-key-id", nil)
	mockKMS.On("DecryptDEK", ctx, "kms-key-id", ciphertextDEK).Return(expectedPlaintext, nil)

	// Execute
	plaintext, err := dekOps.DecryptDEKWithVersion(ctx, ciphertextDEK, kekVersion, mockVersionManager)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedPlaintext, plaintext)

	// Verify all expectations were met
	mockKMS.AssertExpectations(t)
	mockVersionManager.AssertExpectations(t)
}

func TestDEKOperations_DecryptDEKWithVersion_GetKeyIDError(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	ciphertextDEK := []byte("encrypted-dek-data")
	kekVersion := 1
	expectedError := errors.New("key ID error")

	// Set up mock expectations
	mockVersionManager.On("GetKMSKeyIDForVersion", ctx, "test-alias", kekVersion).Return("", expectedError)

	// Execute
	plaintext, err := dekOps.DecryptDEKWithVersion(ctx, ciphertextDEK, kekVersion, mockVersionManager)

	// Verify
	assert.Nil(t, plaintext)
	assert.Equal(t, expectedError, err)

	mockVersionManager.AssertExpectations(t)
}

func TestDEKOperations_DecryptDEKWithVersion_KMSError(t *testing.T) {
	mockKMS := &MockKMSService{}
	mockVersionManager := &MockVersionManager{}
	dekOps := NewDEKOperations(mockKMS, "test-alias")

	ctx := context.Background()
	ciphertextDEK := []byte("encrypted-dek-data")
	kekVersion := 1
	kmsError := errors.New("KMS decryption failed")

	// Set up mock expectations
	mockVersionManager.On("GetKMSKeyIDForVersion", ctx, "test-alias", kekVersion).Return("kms-key-id", nil)
	mockKMS.On("DecryptDEK", ctx, "kms-key-id", ciphertextDEK).Return([]byte(nil), kmsError)

	// Execute
	plaintext, err := dekOps.DecryptDEKWithVersion(ctx, ciphertextDEK, kekVersion, mockVersionManager)

	// Verify
	assert.Nil(t, plaintext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt DEK with KMS (version 1)")
	assert.Contains(t, err.Error(), kmsError.Error())

	// Verify all expectations were met
	mockKMS.AssertExpectations(t)
	mockVersionManager.AssertExpectations(t)
}