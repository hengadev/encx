package encx

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"
)

// CryptoServiceMock is a mock implementation of the CryptoService interface
// for testing purposes. It uses testify/mock for easy setup and verification.
type CryptoServiceMock struct {
	mock.Mock
}

func NewCryptoServiceMock() *CryptoServiceMock {
	return &CryptoServiceMock{}
}

func (m *CryptoServiceMock) GetPepper() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *CryptoServiceMock) GetArgon2Params() *Argon2Params {
	args := m.Called()
	return args.Get(0).(*Argon2Params)
}

func (m *CryptoServiceMock) GetAlias() string {
	args := m.Called()
	return args.String(0)
}

func (m *CryptoServiceMock) GenerateDEK() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *CryptoServiceMock) EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error) {
	args := m.Called(ctx, plaintext, dek)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *CryptoServiceMock) DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error) {
	args := m.Called(ctx, ciphertext, dek)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *CryptoServiceMock) ProcessStruct(ctx context.Context, object any) error {
	args := m.Called(ctx, object)
	return args.Error(0)
}

func (m *CryptoServiceMock) DecryptStruct(ctx context.Context, object any) error {
	args := m.Called(ctx, object)
	return args.Error(0)
}

func (m *CryptoServiceMock) EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error) {
	args := m.Called(ctx, plaintextDEK)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *CryptoServiceMock) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error) {
	args := m.Called(ctx, ciphertextDEK, kekVersion)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *CryptoServiceMock) RotateKEK(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *CryptoServiceMock) HashBasic(ctx context.Context, value []byte) string {
	args := m.Called(ctx, value)
	return args.String(0)
}

func (m *CryptoServiceMock) HashSecure(ctx context.Context, value []byte) (string, error) {
	args := m.Called(ctx, value)
	return args.String(0), args.Error(1)
}

func (m *CryptoServiceMock) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	args := m.Called(ctx, value, hashValue)
	return args.Bool(0), args.Error(1)
}

func (m *CryptoServiceMock) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	args := m.Called(ctx, value, hashValue)
	return args.Bool(0), args.Error(1)
}

func (m *CryptoServiceMock) EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error {
	args := m.Called(ctx, reader, writer, dek)
	return args.Error(0)
}

func (m *CryptoServiceMock) DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error {
	args := m.Called(ctx, reader, writer, dek)
	return args.Error(0)
}
