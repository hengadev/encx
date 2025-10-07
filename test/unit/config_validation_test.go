package encx_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hengadev/encx"
	"github.com/hengadev/encx/test/testutils"
)

func TestNewCrypto_ValidConfiguration(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a test KMS service
	kms := testutils.NewSimpleTestKMS()
	pepper := []byte("test-pepper-exactly-32-bytes-OK!")

	crypto, err := encx.NewCrypto(ctx,
		encx.WithKMSService(kms),
		encx.WithKEKAlias("test-kek"),
		encx.WithPepper(pepper),
		encx.WithKeyMetadataDBPath(filepath.Join(tempDir, "test.db")),
	)

	require.NoError(t, err)
	require.NotNil(t, crypto)

	// Verify crypto instance was created successfully
	// (Cannot access unexported fields from external test package)
	assert.NotNil(t, crypto)
}

func TestNewCrypto_MissingRequiredFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		options []encx.Option
		wantErr string
	}{
		{
			name: "missing KMS service",
			options: []encx.Option{
				encx.WithKEKAlias("test-kek"),
				encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
			},
			wantErr: "KMS service is required",
		},
		{
			name: "missing KEK alias",
			options: []encx.Option{
				encx.WithKMSService(testutils.NewSimpleTestKMS()),
				encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
			},
			wantErr: "KEK alias is required",
		},
		{
			name: "missing pepper",
			options: []encx.Option{
				encx.WithKMSService(testutils.NewSimpleTestKMS()),
				encx.WithKEKAlias("test-kek"),
			},
			wantErr: "pepper must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encx.NewCrypto(ctx, tt.options...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestWithKEKAlias_Validation(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr string
	}{
		{
			name:    "valid alias",
			alias:   "valid-alias_123",
			wantErr: "",
		},
		{
			name:    "empty alias",
			alias:   "",
			wantErr: "cannot be empty",
		},
		{
			name:    "whitespace only",
			alias:   "   ",
			wantErr: "cannot be empty",
		},
		{
			name:    "too long",
			alias:   strings.Repeat("a", 257),
			wantErr: "too long",
		},
		{
			name:    "invalid characters",
			alias:   "invalid@alias",
			wantErr: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &encx.Config{}
			err := encx.WithKEKAlias(tt.alias)(config)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tt.alias), config.KEKAlias)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestWithPepper_Validation(t *testing.T) {
	tests := []struct {
		name    string
		pepper  []byte
		wantErr string
	}{
		{
			name:    "valid pepper",
			pepper:  []byte("test-pepper-exactly-32-bytes-OK!"),
			wantErr: "",
		},
		{
			name:    "empty pepper",
			pepper:  []byte{},
			wantErr: "cannot be empty",
		},
		{
			name:    "wrong length",
			pepper:  []byte("short"),
			wantErr: "must be exactly 32 bytes",
		},
		{
			name:    "zero pepper",
			pepper:  make([]byte, 32),
			wantErr: "uninitialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &encx.Config{}
			err := encx.WithPepper(tt.pepper)(config)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.pepper, config.Pepper)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestWithDatabase_Validation(t *testing.T) {
	t.Run("nil database", func(t *testing.T) {
		config := &encx.Config{}
		err := encx.WithKeyMetadataDB(nil)(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("invalid database connection", func(t *testing.T) {
		// Create a database and then close it to make it invalid
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		db.Close()

		config := &encx.Config{}
		err = encx.WithKeyMetadataDB(db)(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection test failed")
	})

	t.Run("valid database", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		config := &encx.Config{}
		err = encx.WithKeyMetadataDB(db)(config)
		assert.NoError(t, err)
		assert.Equal(t, db, config.KeyMetadataDB)
	})
}

func TestWithDatabasePath_Validation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: "cannot be empty",
		},
		{
			name:    "whitespace only",
			path:    "   ",
			wantErr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &encx.Config{}
			err := encx.WithKeyMetadataDBPath(tt.path)(config)

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}

	t.Run("valid path", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		config := &encx.Config{}
		err := encx.WithKeyMetadataDBPath(dbPath)(config)
		assert.NoError(t, err)
		assert.Equal(t, dbPath, config.DBPath)
	})
}

func TestWithDatabaseFilename_Validation(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  string
	}{
		{
			name:     "valid filename",
			filename: "test.db",
			wantErr:  "",
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  "cannot be empty",
		},
		{
			name:     "contains path separator",
			filename: "path/test.db",
			wantErr:  "cannot contain path separators",
		},
		{
			name:     "too long",
			filename: strings.Repeat("a", 256) + ".db",
			wantErr:  "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &encx.Config{}
			err := encx.WithKeyMetadataDBFilename(tt.filename)(config)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.filename, config.DBFilename)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestWithArgon2Params_Validation(t *testing.T) {
	t.Run("nil params", func(t *testing.T) {
		config := &encx.Config{}
		err := encx.WithArgon2Params(nil)(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("invalid params", func(t *testing.T) {
		invalidParams := &encx.Argon2Params{
			Memory: 1, // Too low
		}

		config := &encx.Config{}
		err := encx.WithArgon2Params(invalidParams)(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid Argon2 parameters")
	})

	t.Run("valid params", func(t *testing.T) {
		validParams := &encx.Argon2Params{
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
			SaltLength:  16,
			KeyLength:   32,
		}

		config := &encx.Config{}
		err := encx.WithArgon2Params(validParams)(config)
		assert.NoError(t, err)
		assert.Equal(t, validParams, config.Argon2Params)
	})
}

func TestWithSerializer_Validation(t *testing.T) {
	// Serializer is internal API, not exposed publicly
	t.Skip("Serializer validation not exposed in public API")
}

func TestConfigurationConflicts(t *testing.T) {
	ctx := context.Background()
	pepper := []byte("test-pepper-exactly-32-bytes-OK!")

	t.Run("pepper from both direct and secret path", func(t *testing.T) {
		_, err := encx.NewCrypto(ctx,
			encx.WithKMSService(testutils.NewSimpleTestKMS()),
			encx.WithKEKAlias("test-kek"),
			encx.WithPepper(pepper),
			encx.WithPepperSecretPath("secret/path"),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be provided both directly and via secret path")
	})

	t.Run("database from both connection and path", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		tempDir := t.TempDir()

		_, err = encx.NewCrypto(ctx,
			encx.WithKMSService(testutils.NewSimpleTestKMS()),
			encx.WithKEKAlias("test-kek"),
			encx.WithPepper(pepper),
			encx.WithKeyMetadataDB(db),
			encx.WithKeyMetadataDBPath(filepath.Join(tempDir, "test.db")),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be configured both via connection and path")
	})
}

func TestSetDefaults(t *testing.T) {
	// Default config is tested via NewCrypto
	t.Skip("SetDefaults is internal API, tested via NewCrypto")
}

// TestBackwardCompatibility removed - encx.New() no longer exists
// The API has been updated to use encx.NewCrypto() with options pattern

