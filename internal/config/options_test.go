package config

import (
	"context"
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

// TestWithKMSService removed - WithKMSService option removed in v0.6.0
// KMS service is now a required parameter in NewCrypto() function
func TestWithKMSService(t *testing.T) {
	t.Skip("WithKMSService option removed in v0.6.0 - KMS is now a required parameter")
}

// TestWithKEKAlias removed - WithKEKAlias option removed in v0.6.0
// KEK alias is now set via ENCX_KEK_ALIAS environment variable
func TestWithKEKAlias(t *testing.T) {
	t.Skip("WithKEKAlias option removed in v0.6.0 - use ENCX_KEK_ALIAS environment variable")
}

// TestWithPepper removed - WithPepper option removed in v0.6.0
// Pepper is now automatically generated and persisted
func TestWithPepper(t *testing.T) {
	t.Skip("WithPepper option removed in v0.6.0 - pepper is now automatically generated")
}

// TestWithKeyMetadataDB removed - WithKeyMetadataDB option removed in v0.6.0
// Database is now automatically managed by NewCrypto
func TestWithKeyMetadataDB(t *testing.T) {
	t.Skip("WithKeyMetadataDB option removed in v0.6.0 - database is now automatically managed")
}

// TestWithKeyMetadataDBPath - deprecated, database is now auto-managed
func TestWithKeyMetadataDBPath(t *testing.T) {
	t.Skip("WithKeyMetadataDBPath is deprecated - database is auto-managed in v0.6.0+")
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

// TestApplyOptions removed - tests removed options
func TestApplyOptions(t *testing.T) {
	t.Skip("Test references removed options - WithKMSService, WithKEKAlias, WithPepper removed in v0.6.0")
}

// TestApplyOptions_WithErrors removed - tests removed options
func TestApplyOptions_WithErrors(t *testing.T) {
	t.Skip("Test references removed options - WithKMSService, WithKEKAlias removed in v0.6.0")
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

// TestWithPepperSecretPath removed - WithPepperSecretPath option removed in v0.6.0
// Pepper path is now set via ENCX_PEPPER_SECRET_PATH environment variable
func TestWithPepperSecretPath(t *testing.T) {
	t.Skip("WithPepperSecretPath option removed in v0.6.0 - use ENCX_PEPPER_SECRET_PATH environment variable")
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

// TestWithKeyMetadataDBFilename - deprecated, database is now auto-managed
func TestWithKeyMetadataDBFilename(t *testing.T) {
	t.Skip("WithKeyMetadataDBFilename is deprecated - database is auto-managed in v0.6.0+")
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