package encx

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Config holds the complete configuration for a Crypto instance
type Config struct {
	KMSService       KeyManagementService
	KEKAlias         string
	Pepper           []byte
	PepperSecretPath string
	Argon2Params     *Argon2Params
	Serializer       Serializer
	KeyMetadataDB    *sql.DB
	DBPath           string
	DBFilename       string
	MetricsCollector MetricsCollector
	ObservabilityHook ObservabilityHook
}

// Option represents a configuration option for creating a Crypto instance
type Option func(*Config) error

// WithKMSService sets the Key Management Service provider
func WithKMSService(kms KeyManagementService) Option {
	return func(c *Config) error {
		if kms == nil {
			return fmt.Errorf("%w: KMS service cannot be nil", ErrInvalidConfiguration)
		}
		c.KMSService = kms
		return nil
	}
}

// WithKEKAlias sets the Key Encryption Key alias
func WithKEKAlias(alias string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("%w: KEK alias cannot be empty or whitespace only", ErrInvalidConfiguration)
		}
		if len(alias) > 256 {
			return fmt.Errorf("%w: KEK alias too long: maximum 256 characters, got %d", ErrInvalidConfiguration, len(alias))
		}
		// Basic validation for safe alias characters
		for _, char := range alias {
			if !isValidAliasChar(char) {
				return fmt.Errorf("%w: KEK alias contains invalid character '%c': only alphanumeric, hyphens, underscores allowed", ErrInvalidConfiguration, char)
			}
		}
		c.KEKAlias = strings.TrimSpace(alias)
		return nil
	}
}

// WithPepper sets the pepper value directly (for testing or when pepper is managed externally)
func WithPepper(pepper []byte) Option {
	return func(c *Config) error {
		if len(pepper) == 0 {
			return fmt.Errorf("pepper cannot be empty")
		}
		if len(pepper) != 32 {
			return fmt.Errorf("pepper must be exactly 32 bytes, got %d", len(pepper))
		}
		if isZeroPepper(pepper) {
			return ErrUninitializedPepper
		}
		c.Pepper = make([]byte, len(pepper))
		copy(c.Pepper, pepper)
		return nil
	}
}

// WithPepperSecretPath sets the path to retrieve pepper from KMS
func WithPepperSecretPath(path string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("pepper secret path cannot be empty or whitespace only")
		}
		c.PepperSecretPath = strings.TrimSpace(path)
		return nil
	}
}

// WithDatabase sets a pre-configured database connection
func WithDatabase(db *sql.DB) Option {
	return func(c *Config) error {
		if db == nil {
			return fmt.Errorf("database connection cannot be nil")
		}
		// Test the database connection
		if err := db.Ping(); err != nil {
			return fmt.Errorf("%w: database connection test failed: %w", ErrDatabaseUnavailable, err)
		}
		c.KeyMetadataDB = db
		return nil
	}
}

// WithDatabasePath sets the full path to the key metadata database file
func WithDatabasePath(path string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("database path cannot be empty or whitespace only")
		}
		
		// Validate the directory exists or can be created
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create database directory '%s': %w", dir, err)
		}
		
		// Check if we can write to the directory
		if err := checkWritePermissions(dir); err != nil {
			return fmt.Errorf("insufficient permissions for database directory '%s': %w", dir, err)
		}
		
		c.DBPath = path
		return nil
	}
}

// WithDatabaseFilename sets just the filename for the database (uses default directory)
func WithDatabaseFilename(filename string) Option {
	return func(c *Config) error {
		if strings.TrimSpace(filename) == "" {
			return fmt.Errorf("database filename cannot be empty or whitespace only")
		}
		
		// Validate filename doesn't contain path separators
		if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
			return fmt.Errorf("database filename cannot contain path separators, use WithDatabasePath for full paths")
		}
		
		// Basic filename validation
		if len(filename) > 255 {
			return fmt.Errorf("database filename too long: maximum 255 characters, got %d", len(filename))
		}
		
		c.DBFilename = filename
		return nil
	}
}

// WithArgon2ParamsV2 sets custom Argon2id parameters (new validation system)
func WithArgon2ParamsV2(params *Argon2Params) Option {
	return func(c *Config) error {
		if params == nil {
			return fmt.Errorf("Argon2 parameters cannot be nil")
		}
		if err := params.Validate(); err != nil {
			return fmt.Errorf("invalid Argon2 parameters: %w", err)
		}
		c.Argon2Params = params
		return nil
	}
}

// WithSerializerV2 sets a custom serializer for field values (new validation system)
func WithSerializerV2(serializer Serializer) Option {
	return func(c *Config) error {
		if serializer == nil {
			return fmt.Errorf("serializer cannot be nil")
		}
		c.Serializer = serializer
		return nil
	}
}

// WithMetricsCollector sets a custom metrics collector for monitoring
func WithMetricsCollector(collector MetricsCollector) Option {
	return func(c *Config) error {
		if collector == nil {
			return fmt.Errorf("metrics collector cannot be nil")
		}
		c.MetricsCollector = collector
		return nil
	}
}

// WithObservabilityHook sets a custom observability hook for monitoring
func WithObservabilityHook(hook ObservabilityHook) Option {
	return func(c *Config) error {
		if hook == nil {
			return fmt.Errorf("observability hook cannot be nil")
		}
		c.ObservabilityHook = hook
		return nil
	}
}

// WithStandardMonitoring enables standard monitoring with optional metrics collector
func WithStandardMonitoring(collector MetricsCollector) Option {
	return func(c *Config) error {
		if collector == nil {
			collector = NewInMemoryMetricsCollector()
		}
		c.MetricsCollector = collector
		c.ObservabilityHook = NewStandardObservabilityHook(collector)
		return nil
	}
}

// validateConfig performs comprehensive validation of the final configuration
func validateConfig(ctx context.Context, config *Config) error {
	var errors []string
	
	// Validate required fields
	if config.KMSService == nil {
		errors = append(errors, "KMS service is required")
	}
	
	if config.KEKAlias == "" {
		errors = append(errors, "KEK alias is required")
	}
	
	// Validate pepper configuration
	pepperFromDirect := len(config.Pepper) > 0
	pepperFromSecretPath := config.PepperSecretPath != ""
	
	if !pepperFromDirect && !pepperFromSecretPath {
		errors = append(errors, "pepper must be provided either directly via WithPepper() or via secret path with WithPepperSecretPath()")
	}
	
	if pepperFromDirect && pepperFromSecretPath {
		errors = append(errors, "pepper cannot be provided both directly and via secret path - choose one method")
	}
	
	// Validate KMS accessibility if possible (non-destructive test)
	if config.KMSService != nil && pepperFromSecretPath {
		// Test KMS connectivity by attempting to retrieve the pepper
		pepper, err := config.KMSService.GetSecret(ctx, config.PepperSecretPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to retrieve pepper from KMS at path '%s': %v", config.PepperSecretPath, err))
		} else {
			if len(pepper) != 32 {
				errors = append(errors, fmt.Sprintf("pepper from KMS has invalid length: expected 32 bytes, got %d", len(pepper)))
			} else if isZeroPepper(pepper) {
				errors = append(errors, "pepper from KMS appears to be uninitialized (all zeros)")
			} else {
				// Store the retrieved pepper for later use
				config.Pepper = pepper
			}
		}
	}
	
	// Validate database configuration
	dbFromConnection := config.KeyMetadataDB != nil
	dbFromPath := config.DBPath != ""
	dbFromFilename := config.DBFilename != ""
	
	if dbFromConnection && (dbFromPath || dbFromFilename) {
		errors = append(errors, "database cannot be configured both via connection and path - choose one method")
	}
	
	if dbFromPath && dbFromFilename {
		errors = append(errors, "database cannot be configured with both full path and filename - choose one method")
	}
	
	// Validate Argon2 parameters
	if config.Argon2Params != nil {
		if err := config.Argon2Params.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("Argon2 parameters validation failed: %v", err))
		}
	}
	
	// Validate serializer if provided
	if config.Serializer != nil {
		// Test serializer with a simple value using reflection
		testValue := "test-serialization"
		testReflectValue := reflect.ValueOf(testValue)
		data, err := config.Serializer.Serialize(testReflectValue)
		if err != nil {
			errors = append(errors, fmt.Sprintf("serializer test failed during serialization: %v", err))
		} else {
			var result string
			resultReflectValue := reflect.ValueOf(&result).Elem()
			if err := config.Serializer.Deserialize(data, resultReflectValue); err != nil {
				errors = append(errors, fmt.Sprintf("serializer test failed during deserialization: %v", err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}
	
	return nil
}

// setDefaults applies default values for unspecified configuration options
func setDefaults(config *Config) error {
	if config.Argon2Params == nil {
		config.Argon2Params = DefaultArgon2Params
	}
	
	if config.Serializer == nil {
		config.Serializer = &JSONSerializer{}
	}
	
	if config.MetricsCollector == nil {
		config.MetricsCollector = &NoOpMetricsCollector{}
	}
	
	if config.ObservabilityHook == nil {
		config.ObservabilityHook = &NoOpObservabilityHook{}
	}
	
	return nil
}

// isValidAliasChar checks if a character is valid for KEK alias
func isValidAliasChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		   (char >= 'A' && char <= 'Z') ||
		   (char >= '0' && char <= '9') ||
		   char == '-' || char == '_'
}

// checkWritePermissions tests if we can write to a directory
func checkWritePermissions(dir string) error {
	testFile := filepath.Join(dir, ".encx_write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot create test file: %w", err)
	}
	file.Close()
	
	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("cannot remove test file: %w", err)
	}
	
	return nil
}