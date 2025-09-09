package encx

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// NewCrypto creates a new Crypto instance with comprehensive configuration validation.
// This is the recommended constructor for production use.
//
// Example usage:
//
//	crypto, err := encx.NewCrypto(ctx,
//	    encx.WithKMSService(kmsService),
//	    encx.WithKEKAlias("my-app-kek"),
//	    encx.WithPepperSecretPath("secret/my-app/pepper"),
//	    encx.WithDatabasePath("/var/lib/myapp/encx.db"),
//	)
func NewCrypto(ctx context.Context, options ...Option) (*Crypto, error) {
	// Initialize configuration
	config := &Config{}

	// Apply all options
	for i, opt := range options {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("invalid option %d: %w", i+1, err)
		}
	}

	// Set defaults for unspecified options
	if err := setDefaults(config); err != nil {
		return nil, fmt.Errorf("failed to set default values: %w", err)
	}

	// Comprehensive validation
	if err := validateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("%w: configuration validation failed: %w", ErrInvalidConfiguration, err)
	}

	// Set up database if not provided
	if config.KeyMetadataDB == nil {
		db, dbPath, err := setupDatabase(config)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to setup database: %w", ErrDatabaseUnavailable, err)
		}
		config.KeyMetadataDB = db
		config.DBPath = dbPath
	}

	// Create the Crypto instance
	crypto := &Crypto{
		kmsService:        config.KMSService,
		kekAlias:          config.KEKAlias,
		pepper:            config.Pepper,
		argon2Params:      config.Argon2Params,
		serializer:        config.Serializer,
		keyMetadataDB:     config.KeyMetadataDB,
		metricsCollector:  config.MetricsCollector,
		observabilityHook: config.ObservabilityHook,
	}

	// Initialize database schema
	if err := initializeDatabase(crypto.keyMetadataDB, config.DBPath); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	// Ensure initial KEK exists
	if err := crypto.ensureInitialKEK(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure initial KEK for alias '%s': %w", config.KEKAlias, err)
	}

	// Final validation of the created instance
	if err := validateCryptoInstance(crypto); err != nil {
		return nil, fmt.Errorf("final instance validation failed: %w", err)
	}

	return crypto, nil
}

// setupDatabase creates and configures the key metadata database
func setupDatabase(config *Config) (*sql.DB, string, error) {
	var dbPath string

	// Determine database path
	if config.DBPath != "" {
		dbPath = config.DBPath
	} else if config.DBFilename != "" {
		// Use custom filename in default directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		defaultDataDir := filepath.Join(cwd, defaultDBDirName)
		if err := os.MkdirAll(defaultDataDir, 0700); err != nil {
			return nil, "", fmt.Errorf("failed to create database directory '%s': %w", defaultDataDir, err)
		}

		dbPath = filepath.Join(defaultDataDir, config.DBFilename)
	} else {
		// Use default path and filename
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
		}

		defaultDataDir := filepath.Join(cwd, defaultDBDirName)
		if err := os.MkdirAll(defaultDataDir, 0700); err != nil {
			return nil, "", fmt.Errorf("failed to create default database directory '%s': %w", defaultDataDir, err)
		}

		dbPath = filepath.Join(defaultDataDir, defaultDBFileName)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open database at '%s': %w", dbPath, err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("database connection test failed for '%s': %w", dbPath, err)
	}

	return db, dbPath, nil
}

// initializeDatabase creates the required database schema
func initializeDatabase(db *sql.DB, dbPath string) error {
	schema := `
		CREATE TABLE IF NOT EXISTS kek_versions (
			alias TEXT NOT NULL,
			version INTEGER NOT NULL,
			creation_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_deprecated BOOLEAN DEFAULT FALSE,
			kms_key_id TEXT NOT NULL,
			PRIMARY KEY (alias, version)
		);
		
		CREATE INDEX IF NOT EXISTS idx_kek_versions_alias ON kek_versions(alias);
		CREATE INDEX IF NOT EXISTS idx_kek_versions_active ON kek_versions(alias, is_deprecated);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create database schema in '%s': %w", dbPath, err)
	}

	return nil
}

// validateCryptoInstance performs final validation on the created Crypto instance
func validateCryptoInstance(crypto *Crypto) error {
	if crypto == nil {
		return fmt.Errorf("crypto instance is nil")
	}

	if crypto.kmsService == nil {
		return fmt.Errorf("KMS service is nil in crypto instance")
	}

	if crypto.kekAlias == "" {
		return fmt.Errorf("KEK alias is empty in crypto instance")
	}

	if len(crypto.pepper) != 32 {
		return fmt.Errorf("pepper has invalid length in crypto instance: expected 32, got %d", len(crypto.pepper))
	}

	if crypto.argon2Params == nil {
		return fmt.Errorf("Argon2 parameters are nil in crypto instance")
	}

	if crypto.serializer == nil {
		return fmt.Errorf("serializer is nil in crypto instance")
	}

	if crypto.keyMetadataDB == nil {
		return fmt.Errorf("key metadata database is nil in crypto instance")
	}

	// Test database connectivity
	if err := crypto.keyMetadataDB.Ping(); err != nil {
		return fmt.Errorf("database connection test failed in crypto instance: %w", err)
	}

	return nil
}

// NewCryptoLegacy provides the old constructor signature for backward compatibility.
// Deprecated: Use NewCrypto with options instead for better validation and flexibility.
func NewCryptoLegacy(
	ctx context.Context,
	kmsService KeyManagementService,
	kekAlias string,
	pepperSecretPath string,
	options ...CryptoOption,
) (*Crypto, error) {
	// Convert old options to new options
	var newOptions []Option

	newOptions = append(newOptions, WithKMSService(kmsService))
	newOptions = append(newOptions, WithKEKAlias(kekAlias))
	newOptions = append(newOptions, WithPepperSecretPath(pepperSecretPath))

	// Convert legacy options
	for _, oldOpt := range options {
		// Create a temporary Crypto instance to apply the old option
		tempCrypto := &Crypto{
			argon2Params: DefaultArgon2Params,
			serializer:   &JSONSerializer{},
		}

		if err := oldOpt(tempCrypto); err != nil {
			return nil, fmt.Errorf("failed to convert legacy option: %w", err)
		}

		// Extract the applied configuration and convert to new options
		if tempCrypto.argon2Params != nil && tempCrypto.argon2Params != DefaultArgon2Params {
			newOptions = append(newOptions, WithArgon2ParamsV2(tempCrypto.argon2Params))
		}

		if tempCrypto.keyMetadataDB != nil {
			newOptions = append(newOptions, WithDatabase(tempCrypto.keyMetadataDB))
		}

		// Note: Serializer comparison is more complex, so we'll just add it if it's not the default
		if tempCrypto.serializer != nil {
			newOptions = append(newOptions, WithSerializerV2(tempCrypto.serializer))
		}
	}

	return NewCrypto(ctx, newOptions...)
}

