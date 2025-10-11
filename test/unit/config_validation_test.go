package encx_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hengadev/encx"
	"github.com/hengadev/encx/test/testutils"
)

func TestNewCrypto_ValidConfiguration(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Set required environment variables
	os.Setenv("ENCX_KEK_ALIAS", "test-kek")
	os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")

	// Create a test KMS service
	kms := testutils.NewSimpleTestKMS()

	crypto, err := encx.NewCrypto(ctx, kms,
		encx.WithDBPath(tempDir),
		encx.WithDBFilename("test.db"),
	)

	require.NoError(t, err)
	require.NotNil(t, crypto)

	// Verify crypto instance was created successfully
	assert.NotNil(t, crypto)
}

func TestNewCrypto_MissingRequiredFields(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setupEnv      func()
		kmsService    encx.KeyManagementService
		wantErr       string
	}{
		{
			name: "nil KMS service",
			setupEnv: func() {
				os.Setenv("ENCX_KEK_ALIAS", "test-kek")
				os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")
			},
			kmsService: nil,
			wantErr:    "KMS service is required",
		},
		{
			name: "missing KEK alias",
			setupEnv: func() {
				os.Unsetenv("ENCX_KEK_ALIAS")
				os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")
			},
			kmsService: testutils.NewSimpleTestKMS(),
			wantErr:    "ENCX_KEK_ALIAS environment variable is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment before and after test
			os.Unsetenv("ENCX_KEK_ALIAS")
			os.Unsetenv("ENCX_PEPPER_SECRET_PATH")

			tt.setupEnv()

			_, err := encx.NewCrypto(ctx, tt.kmsService)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestEnvironmentVariables_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setupEnv      func()
		wantErr       string
	}{
		{
			name: "valid environment variables",
			setupEnv: func() {
				os.Setenv("ENCX_KEK_ALIAS", "valid-alias")
				os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")
			},
			wantErr: "",
		},
		{
			name: "empty KEK alias",
			setupEnv: func() {
				os.Setenv("ENCX_KEK_ALIAS", "")
				os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")
			},
			wantErr: "ENCX_KEK_ALIAS environment variable is required",
		},
		{
			name: "missing KEK alias",
			setupEnv: func() {
				os.Unsetenv("ENCX_KEK_ALIAS")
				os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")
			},
			wantErr: "ENCX_KEK_ALIAS environment variable is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment before and after test
			os.Unsetenv("ENCX_KEK_ALIAS")
			os.Unsetenv("ENCX_PEPPER_SECRET_PATH")

			tt.setupEnv()

			_, err := encx.NewCrypto(ctx, testutils.NewSimpleTestKMS())

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestWithDBPath_Validation(t *testing.T) {
	ctx := context.Background()

	// Set required environment variables
	os.Setenv("ENCX_KEK_ALIAS", "test-kek")
	os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")

	t.Run("valid database path", func(t *testing.T) {
		tempDir := t.TempDir()

		crypto, err := encx.NewCrypto(ctx, testutils.NewSimpleTestKMS(),
			encx.WithDBPath(tempDir),
		)

		require.NoError(t, err)
		require.NotNil(t, crypto)
	})

	t.Run("valid database filename", func(t *testing.T) {
		crypto, err := encx.NewCrypto(ctx, testutils.NewSimpleTestKMS(),
			encx.WithDBFilename("custom-test.db"),
		)

		require.NoError(t, err)
		require.NotNil(t, crypto)
	})
}

func TestWithArgon2Params_Validation(t *testing.T) {
	ctx := context.Background()

	// Set required environment variables
	os.Setenv("ENCX_KEK_ALIAS", "test-kek")
	os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")

	t.Run("valid params", func(t *testing.T) {
		validParams := &encx.Argon2Params{
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
			SaltLength:  16,
			KeyLength:   32,
		}

		crypto, err := encx.NewCrypto(ctx, testutils.NewSimpleTestKMS(),
			encx.WithArgon2Params(validParams),
		)

		require.NoError(t, err)
		require.NotNil(t, crypto)
	})

	t.Run("nil params should use defaults", func(t *testing.T) {
		crypto, err := encx.NewCrypto(ctx, testutils.NewSimpleTestKMS())

		require.NoError(t, err)
		require.NotNil(t, crypto)
	})
}

func TestWithSerializer_Validation(t *testing.T) {
	// Serializer is internal API, not exposed publicly
	t.Skip("Serializer validation not exposed in public API")
}

func TestConfigurationConflicts(t *testing.T) {
	ctx := context.Background()

	// Set required environment variables
	os.Setenv("ENCX_KEK_ALIAS", "test-kek")
	os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")

	t.Run("database path and filename conflict", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test that both options can be used together (no conflict)
		crypto, err := encx.NewCrypto(ctx, testutils.NewSimpleTestKMS(),
			encx.WithDBPath(tempDir),
			encx.WithDBFilename("test.db"),
		)

		require.NoError(t, err)
		require.NotNil(t, crypto)
	})
}

func TestSetDefaults(t *testing.T) {
	// Default config is tested via NewCrypto
	t.Skip("SetDefaults is internal API, tested via NewCrypto")
}

// TestBackwardCompatibility removed - encx.New() no longer exists
// The API has been updated to use encx.NewCrypto() with options pattern

