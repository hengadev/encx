package encx

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type KeyManagementServiceMock struct {
	mock.Mock
}

func (m *KeyManagementServiceMock) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	args := m.Called(ctx, keyID)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *KeyManagementServiceMock) GetKeyID(ctx context.Context, alias string) (string, error) {
	args := m.Called(ctx, alias)
	return args.String(0), args.Error(1)
}

func (m *KeyManagementServiceMock) CreateKey(ctx context.Context, description string) (string, error) {
	args := m.Called(ctx, description)
	return args.String(0), args.Error(1)
}

func (m *KeyManagementServiceMock) RotateKey(ctx context.Context, keyID string) error {
	args := m.Called(ctx, keyID)
	return args.Error(0)
}

func (m *KeyManagementServiceMock) GetSecret(ctx context.Context, path string) ([]byte, error) {
	args := m.Called(ctx, path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *KeyManagementServiceMock) SetSecret(ctx context.Context, path string, value []byte) error {
	args := m.Called(ctx, path, value)
	return args.Error(0)
}

func (m *KeyManagementServiceMock) EncryptDEK(ctx context.Context, keyID string, plaintextDEK []byte) ([]byte, error) {
	args := m.Called(ctx, keyID, plaintextDEK)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *KeyManagementServiceMock) DecryptDEK(ctx context.Context, keyID string, ciphertextDEK []byte) ([]byte, error) {
	args := m.Called(ctx, keyID, ciphertextDEK)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *KeyManagementServiceMock) EncryptDEKWithVersion(ctx context.Context, plaintextDEK []byte, version int) ([]byte, error) {
	args := m.Called(ctx, plaintextDEK, version)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *KeyManagementServiceMock) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, version int) ([]byte, error) {
	args := m.Called(ctx, ciphertextDEK, version)
	return args.Get(0).([]byte), args.Error(1)
}
