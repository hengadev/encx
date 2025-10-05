package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Option represents a configuration option for creating a Crypto instance
type Option func(*Config) error

// WithKMSService sets the Key Management Service provider
func WithKMSService(kms KeyManagementService) Option {
	return func(c *Config) error {
		if kms == nil {
			return fmt.Errorf("KMS service cannot be nil")
		}
		c.KMSService = kms
		return nil
	}
}

// WithKEKAlias sets the Key Encryption Key alias
func WithKEKAlias(alias string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("KEK alias cannot be empty or whitespace only")
		}
		if len(alias) > 256 {
			return fmt.Errorf("KEK alias too long: maximum 256 characters, got %d", len(alias))
		}

		// Validate characters: alphanumeric, hyphens, underscores, and forward slashes
		for _, ch := range alias {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
				 (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '/') {
				return fmt.Errorf("KEK alias contains invalid character '%c': only alphanumeric, hyphens, underscores, and forward slashes allowed", ch)
			}
		}

		c.KEKAlias = strings.TrimSpace(alias)
		return nil
	}
}

// WithPepper sets the pepper directly as bytes
func WithPepper(pepper []byte) Option {
	return func(c *Config) error {
		if len(pepper) == 0 {
			return fmt.Errorf("pepper cannot be empty")
		}
		if len(pepper) != 32 {
			return fmt.Errorf("pepper must be exactly 32 bytes, got %d", len(pepper))
		}

		// Check if pepper is all zeros (uninitialized)
		allZeros := true
		for _, b := range pepper {
			if b != 0 {
				allZeros = false
				break
			}
		}
		if allZeros {
			return fmt.Errorf("pepper is uninitialized (all zeros)")
		}

		c.Pepper = pepper
		return nil
	}
}

// WithPepperSecretPath sets the path to retrieve pepper from KMS
func WithPepperSecretPath(secretPath string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(secretPath) == "" {
			return fmt.Errorf("pepper secret path cannot be empty")
		}
		c.PepperSecretPath = secretPath
		return nil
	}
}

// WithArgon2Params sets the Argon2 hashing parameters
func WithArgon2Params(params *Argon2Params) Option {
	return func(c *Config) error {
		if params == nil {
			return fmt.Errorf("Argon2 parameters cannot be nil")
		}
		validator := NewValidator()
		if err := validator.validateArgon2Params(params); err != nil {
			return fmt.Errorf("invalid Argon2 parameters: %w", err)
		}
		c.Argon2Params = params
		return nil
	}
}


// WithKeyMetadataDB sets the database connection directly
func WithKeyMetadataDB(db *sql.DB) Option {
	return func(c *Config) error {
		// Check if database was already configured via path
		if c.KeyMetadataDB != nil {
			return fmt.Errorf("database cannot be configured both via connection and path")
		}

		if db == nil {
			return fmt.Errorf("database connection cannot be nil")
		}

		// Test the database connection
		if err := db.Ping(); err != nil {
			return fmt.Errorf("database connection test failed: %w", err)
		}

		c.KeyMetadataDB = db
		return nil
	}
}

// WithDBPath sets the database directory path
func WithDBPath(path string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("database path cannot be empty")
		}

		// Validate directory is writable
		if err := checkDirectoryWritable(path); err != nil {
			return fmt.Errorf("database path validation failed: %w", err)
		}

		c.DBPath = path
		return nil
	}
}

// WithDBFilename sets the database filename
func WithDBFilename(filename string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(filename) == "" {
			return fmt.Errorf("database filename cannot be empty")
		}
		c.DBFilename = filename
		return nil
	}
}

// WithKeyMetadataDBPath sets the full path to the key metadata database
func WithKeyMetadataDBPath(path string) Option {
	return func(c *Config) error {
		// Check if database was already configured via WithKeyMetadataDB
		if c.KeyMetadataDB != nil {
			return fmt.Errorf("database cannot be configured both via connection and path")
		}

		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("database path cannot be empty")
		}

		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return fmt.Errorf("failed to open key metadata database with path '%s': %w", path, err)
		}

		c.KeyMetadataDB = db
		c.DBPath = path
		return nil
	}
}

// WithKeyMetadataDBFilename sets the filename for the key metadata database within the default directory
func WithKeyMetadataDBFilename(filename string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(filename) == "" {
			return fmt.Errorf("database filename cannot be empty")
		}

		// Validate that filename doesn't contain path separators
		if strings.ContainsAny(filename, "/\\") {
			return fmt.Errorf("database filename cannot contain path separators")
		}

		// Validate filename length
		if len(filename) > 255 {
			return fmt.Errorf("database filename too long: maximum 255 characters, got %d", len(filename))
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory for default DB path: %w", err)
		}

		defaultDataDir := filepath.Join(cwd, ".encx")
		if err := os.MkdirAll(defaultDataDir, 0700); err != nil {
			return fmt.Errorf("failed to create default '.encx' directory: %w", err)
		}

		dbPath := filepath.Join(defaultDataDir, filename)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open key metadata database with filename '%s': %w", filename, err)
		}

		c.KeyMetadataDB = db
		c.DBPath = defaultDataDir
		c.DBFilename = filename
		return nil
	}
}

// WithMetricsCollector sets the metrics collector
func WithMetricsCollector(collector MetricsCollector) Option {
	return func(c *Config) error {
		c.MetricsCollector = collector
		return nil
	}
}

// WithObservabilityHook sets the observability hook
func WithObservabilityHook(hook ObservabilityHook) Option {
	return func(c *Config) error {
		c.ObservabilityHook = hook
		return nil
	}
}

// DefaultConfig creates a default configuration
func DefaultConfig() *Config {
	return &Config{
		Argon2Params: &Argon2Params{
			Memory:      65536, // 64MB
			Iterations:  3,     // 3 iterations
			Parallelism: 4,     // 4 threads
			SaltLength:  16,    // 16 bytes salt
			KeyLength:   32,    // 32 bytes key
		},
		DBPath:     ".encx",
		DBFilename: "metadata.db",
	}
}

// ApplyOptions applies all configuration options to a config
func ApplyOptions(config *Config, options []Option) error {
	for i, opt := range options {
		if err := opt(config); err != nil {
			return fmt.Errorf("option %d failed: %w", i, err)
		}
	}
	return nil
}
