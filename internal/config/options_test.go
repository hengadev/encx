package config

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "github.com/mattn/go-sqlite3"
)

// MockKeyManagementService for testing
type MockKeyManagementService struct {
	mock.Mock
}

func (m *MockKeyManagementService) GetKeyID(ctx context.Context, alias string) (string, error) {
	args := m.Called(ctx, alias)
	return args.String(0), args.Error(1)
}

func (m *MockKeyManagementService) CreateKey(ctx context.Context, description string) (string, error) {
	args := m.Called(ctx, description)
	return args.String(0), args.Error(1)
}

func (m *MockKeyManagementService) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	args := m.Called(ctx, keyID, plaintext)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKeyManagementService) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	args := m.Called(ctx, keyID, ciphertext)
	return args.Get(0).([]byte), args.Error(1)
}

func TestWithKMSService(t *testing.T) {
	tests := []struct {
		name    string
		kms     KeyManagementService
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid KMS service",
			kms:     &MockKeyManagementService{},
			wantErr: false,
		},
		{
			name:    "nil KMS service",
			kms:     nil,
			wantErr: true,
			errMsg:  "KMS service cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithKMSService(tt.kms)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.kms, config.KMSService)
			}
		})
	}
}

func TestWithKEKAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid alias",
			alias:   "test-alias",
			wantErr: false,
		},
		{
			name:    "empty alias",
			alias:   "",
			wantErr: true,
			errMsg:  "KEK alias cannot be empty",
		},
		{
			name:    "whitespace only alias",
			alias:   "   ",
			wantErr: true,
			errMsg:  "KEK alias cannot be empty",
		},
		{
			name:    "alias too long",
			alias:   string(make([]byte, 257)), // 257 characters
			wantErr: true,
			errMsg:  "KEK alias too long",
		},
		{
			name:    "alias at max length",
			alias:   string(make([]byte, 256)), // 256 characters
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithKEKAlias(tt.alias)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.alias, config.KEKAlias)
			}
		})
	}
}

func TestWithPepper(t *testing.T) {
	tests := []struct {
		name    string
		pepper  []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid pepper",
			pepper:  []byte("valid-pepper-16-bytes"),
			wantErr: false,
		},
		{
			name:    "empty pepper",
			pepper:  []byte{},
			wantErr: true,
			errMsg:  "pepper cannot be empty",
		},
		{
			name:    "nil pepper",
			pepper:  nil,
			wantErr: true,
			errMsg:  "pepper cannot be empty",
		},
		{
			name:    "pepper too short",
			pepper:  []byte("short"),
			wantErr: true,
			errMsg:  "pepper too short",
		},
		{
			name:    "pepper too long",
			pepper:  make([]byte, 257), // 257 bytes
			wantErr: true,
			errMsg:  "pepper too long",
		},
		{
			name:    "pepper at min length",
			pepper:  make([]byte, 16), // 16 bytes
			wantErr: false,
		},
		{
			name:    "pepper at max length",
			pepper:  make([]byte, 256), // 256 bytes
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithPepper(tt.pepper)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.pepper, config.Pepper)
			}
		})
	}
}

func TestWithKeyMetadataDB(t *testing.T) {
	// Create a test database
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	tests := []struct {
		name    string
		db      *sql.DB
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid database",
			db:      db,
			wantErr: false,
		},
		{
			name:    "nil database",
			db:      nil,
			wantErr: true,
			errMsg:  "database connection cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithKeyMetadataDB(tt.db)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.db, config.KeyMetadataDB)
			}
		})
	}
}

func TestWithKeyMetadataDBPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "encx_test_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "memory database",
			path:    ":memory:",
			wantErr: false,
		},
		{
			name:    "valid file path",
			path:    filepath.Join(tempDir, "test.db"),
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "database path cannot be empty",
		},
		{
			name:    "whitespace only path",
			path:    "   ",
			wantErr: true,
			errMsg:  "database path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithKeyMetadataDBPath(tt.path)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config.KeyMetadataDB)
				// Clean up the database connection
				if config.KeyMetadataDB != nil {
					config.KeyMetadataDB.Close()
				}
			}
		})
	}
}

func TestWithArgon2Params(t *testing.T) {
	validParams := &Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	tests := []struct {
		name    string
		params  *Argon2Params
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid params",
			params:  validParams,
			wantErr: false,
		},
		{
			name:    "nil params",
			params:  nil,
			wantErr: true,
			errMsg:  "Argon2 parameters cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithArgon2Params(tt.params)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.params, config.Argon2Params)
			}
		})
	}
}

func TestApplyOptions(t *testing.T) {
	mockKMS := &MockKeyManagementService{}
	pepper := []byte("test-pepper-16-bytes")

	config := &Config{}
	options := []Option{
		WithKMSService(mockKMS),
		WithKEKAlias("test-alias"),
		WithPepper(pepper),
	}

	err := ApplyOptions(config, options)

	assert.NoError(t, err)
	assert.Equal(t, mockKMS, config.KMSService)
	assert.Equal(t, "test-alias", config.KEKAlias)
	assert.Equal(t, pepper, config.Pepper)
}

func TestApplyOptions_WithErrors(t *testing.T) {
	config := &Config{}
	options := []Option{
		WithKMSService(nil), // This will cause an error
		WithKEKAlias("valid-alias"),
	}

	err := ApplyOptions(config, options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "KMS service cannot be nil")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Argon2Params)
	assert.Equal(t, uint32(65536), config.Argon2Params.Memory)
	assert.Equal(t, uint32(3), config.Argon2Params.Iterations)
	assert.Equal(t, uint8(4), config.Argon2Params.Parallelism)
	assert.Equal(t, uint32(16), config.Argon2Params.SaltLength)
	assert.Equal(t, uint32(32), config.Argon2Params.KeyLength)
}