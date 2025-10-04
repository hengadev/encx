package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

func TestArgon2Params_GetMethods(t *testing.T) {
	params := &Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	assert.Equal(t, uint32(64*1024), params.GetMemory())
	assert.Equal(t, uint32(3), params.GetIterations())
	assert.Equal(t, uint8(4), params.GetParallelism())
	assert.Equal(t, uint32(16), params.GetSaltLength())
	assert.Equal(t, uint32(32), params.GetKeyLength())
}

func TestArgon2Params_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  *Argon2Params
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid params",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: false,
		},
		{
			name: "memory too low",
			params: &Argon2Params{
				Memory:      4096, // Below 8192 minimum
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "memory",
		},
		{
			name: "iterations too low",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  1, // Below 2 minimum
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "iterations",
		},
		{
			name: "parallelism zero",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 0, // Below 1 minimum
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "parallelism",
		},
		{
			name: "salt length too short",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  8, // Below 16 minimum
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "saltLength",
		},
		{
			name: "key length too short",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   16, // Below 32 minimum
			},
			wantErr: true,
			errMsg:  "keyLength",
		},
		{
			name: "multiple validation failures",
			params: &Argon2Params{
				Memory:      4096, // Too low
				Iterations:  1,    // Too low
				Parallelism: 0,    // Too low
				SaltLength:  8,    // Too low
				KeyLength:   16,   // Too low
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidateConfig(t *testing.T) {
	// Create temp directory for database tests
	tempDir, err := os.MkdirTemp("", "encx_test_validation_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mockKMS := &MockKeyManagementService{}
	validParams := &Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid complete config",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "test-key",
				Pepper:       []byte("valid-pepper-16-bytes"),
				Argon2Params: validParams,
				DBPath:       tempDir,
				DBFilename:   "test.db",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name: "nil KMS service",
			config: &Config{
				KMSService:   nil,
				KEKAlias:     "test-key",
				Pepper:       []byte("valid-pepper-16-bytes"),
				Argon2Params: validParams,
				DBPath:       tempDir,
				DBFilename:   "test.db",
			},
			wantErr: true,
			errMsg:  "KMS service",
		},
		{
			name: "empty KEK alias",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "",
				Pepper:       []byte("valid-pepper-16-bytes"),
				Argon2Params: validParams,
				DBPath:       tempDir,
				DBFilename:   "test.db",
			},
			wantErr: true,
			errMsg:  "KEK alias",
		},
		{
			name: "nil Argon2 params",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "test-key",
				Pepper:       []byte("valid-pepper-16-bytes"),
				Argon2Params: nil,
				DBPath:       tempDir,
				DBFilename:   "test.db",
			},
			wantErr: true,
			errMsg:  "Argon2 parameters",
		},
		{
			name: "invalid Argon2 params",
			config: &Config{
				KMSService: mockKMS,
				KEKAlias:   "test-key",
				Pepper:     []byte("valid-pepper-16-bytes"),
				Argon2Params: &Argon2Params{
					Memory:      1024, // Too low
					Iterations:  3,
					Parallelism: 4,
					SaltLength:  16,
					KeyLength:   32,
				},
				DBPath:     tempDir,
				DBFilename: "test.db",
			},
			wantErr: true,
			errMsg:  "Argon2 parameters",
		},
		{
			name: "empty database path",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "test-key",
				Pepper:       []byte("valid-pepper-16-bytes"),
				Argon2Params: validParams,
				DBPath:       "",
				DBFilename:   "test.db",
			},
			wantErr: true,
			errMsg:  "database",
		},
		{
			name: "empty database filename",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "test-key",
				Pepper:       []byte("valid-pepper-16-bytes"),
				Argon2Params: validParams,
				DBPath:       tempDir,
				DBFilename:   "",
			},
			wantErr: true,
			errMsg:  "database",
		},
		{
			name: "no pepper provided",
			config: &Config{
				KMSService:       mockKMS,
				KEKAlias:         "test-key",
				Pepper:           nil,
				PepperSecretPath: "",
				Argon2Params:     validParams,
				DBPath:           tempDir,
				DBFilename:       "test.db",
			},
			wantErr: true,
			errMsg:  "pepper",
		},
		{
			name: "pepper too short",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "test-key",
				Pepper:       []byte("short"),
				Argon2Params: validParams,
				DBPath:       tempDir,
				DBFilename:   "test.db",
			},
			wantErr: true,
			errMsg:  "pepper",
		},
		{
			name: "pepper too long",
			config: &Config{
				KMSService:   mockKMS,
				KEKAlias:     "test-key",
				Pepper:       make([]byte, 300), // Over 256 limit
				Argon2Params: validParams,
				DBPath:       tempDir,
				DBFilename:   "test.db",
			},
			wantErr: true,
			errMsg:  "pepper",
		},
		{
			name: "pepper secret path provided instead of pepper bytes",
			config: &Config{
				KMSService:       mockKMS,
				KEKAlias:         "test-key",
				Pepper:           nil,
				PepperSecretPath: "/secrets/pepper",
				Argon2Params:     validParams,
				DBPath:           tempDir,
				DBFilename:       "test.db",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()
			err := validator.ValidateConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_validateKMSService(t *testing.T) {
	validator := NewValidator()

	t.Run("valid KMS service", func(t *testing.T) {
		mockKMS := &MockKeyManagementService{}
		err := validator.validateKMSService(mockKMS)
		assert.NoError(t, err)
	})

	t.Run("nil KMS service", func(t *testing.T) {
		err := validator.validateKMSService(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "KMS service cannot be nil")
	})
}

func TestValidator_validateKEKAlias(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		alias   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid alias",
			alias:   "test-key",
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
			alias:   string(make([]byte, 300)),
			wantErr: true,
			errMsg:  "KEK alias too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateKEKAlias(tt.alias)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_validateArgon2Params(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		params  *Argon2Params
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid params",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: false,
		},
		{
			name:    "nil params",
			params:  nil,
			wantErr: true,
			errMsg:  "Argon2 parameters cannot be nil",
		},
		{
			name: "memory too low",
			params: &Argon2Params{
				Memory:      2048,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "memory parameter too low",
		},
		{
			name: "memory too high",
			params: &Argon2Params{
				Memory:      2 * 1024 * 1024, // 2GB, over 1GB limit
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "memory parameter too high",
		},
		{
			name: "iterations zero",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  0,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "iterations parameter too low",
		},
		{
			name: "iterations too high",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  150,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "iterations parameter too high",
		},
		{
			name: "parallelism zero",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 0,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "parallelism parameter too low",
		},
		{
			name: "salt length too low",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  4,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "salt length too low",
		},
		{
			name: "salt length too high",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  128,
				KeyLength:   32,
			},
			wantErr: true,
			errMsg:  "salt length too high",
		},
		{
			name: "key length too low",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   8,
			},
			wantErr: true,
			errMsg:  "key length too low",
		},
		{
			name: "key length too high",
			params: &Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 4,
				SaltLength:  16,
				KeyLength:   256,
			},
			wantErr: true,
			errMsg:  "key length too high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateArgon2Params(tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_validateDatabaseConfig(t *testing.T) {
	validator := NewValidator()

	// Create temp directory for tests
	tempDir, err := os.MkdirTemp("", "encx_test_db_validation_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name       string
		dbPath     string
		dbFilename string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid database config",
			dbPath:     tempDir,
			dbFilename: "test.db",
			wantErr:    false,
		},
		{
			name:       "empty database path",
			dbPath:     "",
			dbFilename: "test.db",
			wantErr:    true,
			errMsg:     "database path cannot be empty",
		},
		{
			name:       "whitespace only database path",
			dbPath:     "   ",
			dbFilename: "test.db",
			wantErr:    true,
			errMsg:     "database path cannot be empty",
		},
		{
			name:       "empty database filename",
			dbPath:     tempDir,
			dbFilename: "",
			wantErr:    true,
			errMsg:     "database filename cannot be empty",
		},
		{
			name:       "whitespace only database filename",
			dbPath:     tempDir,
			dbFilename: "   ",
			wantErr:    true,
			errMsg:     "database filename cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateDatabaseConfig(tt.dbPath, tt.dbFilename)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_validatePepperConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name             string
		pepper           []byte
		pepperSecretPath string
		wantErr          bool
		errMsg           string
	}{
		{
			name:             "valid pepper bytes",
			pepper:           []byte("valid-pepper-16-bytes"),
			pepperSecretPath: "",
			wantErr:          false,
		},
		{
			name:             "valid pepper secret path",
			pepper:           nil,
			pepperSecretPath: "/secrets/pepper",
			wantErr:          false,
		},
		{
			name:             "both pepper and secret path provided",
			pepper:           []byte("valid-pepper-16-bytes"),
			pepperSecretPath: "/secrets/pepper",
			wantErr:          false,
		},
		{
			name:             "neither pepper nor secret path provided",
			pepper:           nil,
			pepperSecretPath: "",
			wantErr:          true,
			errMsg:           "either pepper bytes or pepper secret path must be provided",
		},
		{
			name:             "empty pepper and empty secret path",
			pepper:           []byte{},
			pepperSecretPath: "",
			wantErr:          true,
			errMsg:           "either pepper bytes or pepper secret path must be provided",
		},
		{
			name:             "pepper too short",
			pepper:           []byte("short"),
			pepperSecretPath: "",
			wantErr:          true,
			errMsg:           "pepper too short",
		},
		{
			name:             "pepper too long",
			pepper:           make([]byte, 300),
			pepperSecretPath: "",
			wantErr:          true,
			errMsg:           "pepper too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePepperConfig(tt.pepper, tt.pepperSecretPath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckDirectoryWritable(t *testing.T) {
	// Create temp directory for tests
	tempDir, err := os.MkdirTemp("", "encx_test_writable_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		dirPath string
		wantErr bool
		setup   func() string
		cleanup func(string)
	}{
		{
			name:    "writable existing directory",
			dirPath: tempDir,
			wantErr: false,
		},
		{
			name: "non-existent directory gets created",
			setup: func() string {
				return filepath.Join(tempDir, "new_directory")
			},
			wantErr: false,
			cleanup: func(path string) {
				os.RemoveAll(path)
			},
		},
		{
			name: "nested directory gets created",
			setup: func() string {
				return filepath.Join(tempDir, "nested", "deep", "directory")
			},
			wantErr: false,
			cleanup: func(path string) {
				os.RemoveAll(filepath.Join(tempDir, "nested"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath := tt.dirPath
			if tt.setup != nil {
				dirPath = tt.setup()
			}

			err := checkDirectoryWritable(dirPath)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify directory was created
				_, statErr := os.Stat(dirPath)
				assert.NoError(t, statErr)
			}

			if tt.cleanup != nil {
				tt.cleanup(dirPath)
			}
		})
	}
}

func TestValidator_validateSerializer(t *testing.T) {
	validator := NewValidator()

	// This function is deprecated and should always return nil
	err := validator.validateSerializer()
	assert.NoError(t, err)
}

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	assert.NotNil(t, validator)
}
