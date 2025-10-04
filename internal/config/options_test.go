package config

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestWithPepperSecretPath(t *testing.T) {
	tests := []struct {
		name       string
		secretPath string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid secret path",
			secretPath: "/secrets/pepper",
			wantErr:    false,
		},
		{
			name:       "empty secret path",
			secretPath: "",
			wantErr:    true,
			errMsg:     "pepper secret path cannot be empty",
		},
		{
			name:       "whitespace only secret path",
			secretPath: "   ",
			wantErr:    true,
			errMsg:     "pepper secret path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithPepperSecretPath(tt.secretPath)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.secretPath, config.PepperSecretPath)
			}
		})
	}
}

func TestWithDBPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "encx_test_dbpath_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
		setup   func() string
	}{
		{
			name:    "valid writable directory",
			path:    tempDir,
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
		{
			name: "non-existent directory gets created",
			setup: func() string {
				return filepath.Join(tempDir, "new_subdir")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if tt.setup != nil {
				path = tt.setup()
			}

			config := &Config{}
			option := WithDBPath(path)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, path, config.DBPath)
			}
		})
	}
}

func TestWithDBFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid filename",
			filename: "test.db",
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  true,
			errMsg:   "database filename cannot be empty",
		},
		{
			name:     "whitespace only filename",
			filename: "   ",
			wantErr:  true,
			errMsg:   "database filename cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithDBFilename(tt.filename)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.filename, config.DBFilename)
			}
		})
	}
}

func TestWithKeyMetadataDBFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid filename",
			filename: "metadata.db",
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  true,
			errMsg:   "database filename cannot be empty",
		},
		{
			name:     "whitespace only filename",
			filename: "   ",
			wantErr:  true,
			errMsg:   "database filename cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			option := WithKeyMetadataDBFilename(tt.filename)
			err := option(config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config.KeyMetadataDB)
				assert.Equal(t, tt.filename, config.DBFilename)
				// Clean up
				if config.KeyMetadataDB != nil {
					config.KeyMetadataDB.Close()
				}
				// Clean up created directory
				os.RemoveAll(config.DBPath)
			}
		})
	}
}

func TestWithMetricsCollector(t *testing.T) {
	// Create a mock metrics collector
	mockCollector := &MockMetricsCollector{}

	config := &Config{}
	option := WithMetricsCollector(mockCollector)
	err := option(config)

	assert.NoError(t, err)
	assert.Equal(t, mockCollector, config.MetricsCollector)
}

func TestWithMetricsCollector_Nil(t *testing.T) {
	config := &Config{}
	option := WithMetricsCollector(nil)
	err := option(config)

	assert.NoError(t, err)
	assert.Nil(t, config.MetricsCollector)
}

func TestWithObservabilityHook(t *testing.T) {
	// Create a mock observability hook
	mockHook := &MockObservabilityHook{}

	config := &Config{}
	option := WithObservabilityHook(mockHook)
	err := option(config)

	assert.NoError(t, err)
	assert.Equal(t, mockHook, config.ObservabilityHook)
}

func TestWithObservabilityHook_Nil(t *testing.T) {
	config := &Config{}
	option := WithObservabilityHook(nil)
	err := option(config)

	assert.NoError(t, err)
	assert.Nil(t, config.ObservabilityHook)
}

// Mock implementations for monitoring interfaces
type MockMetricsCollector struct {
	mock.Mock
}

func (m *MockMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	m.Called(name, tags)
}

func (m *MockMetricsCollector) IncrementCounterBy(name string, value int64, tags map[string]string) {
	m.Called(name, value, tags)
}

func (m *MockMetricsCollector) SetGauge(name string, value float64, tags map[string]string) {
	m.Called(name, value, tags)
}

func (m *MockMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	m.Called(name, duration, tags)
}

func (m *MockMetricsCollector) RecordValue(name string, value float64, tags map[string]string) {
	m.Called(name, value, tags)
}

func (m *MockMetricsCollector) Flush() error {
	args := m.Called()
	return args.Error(0)
}

type MockObservabilityHook struct {
	mock.Mock
}

func (m *MockObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
	m.Called(ctx, operation, metadata)
}

func (m *MockObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	m.Called(ctx, operation, duration, err, metadata)
}

func (m *MockObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
	m.Called(ctx, operation, err, metadata)
}

func (m *MockObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
	m.Called(ctx, operation, keyAlias, keyVersion, metadata)
}