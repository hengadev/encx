package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hengadev/encx/internal/monitoring"
	"github.com/hengadev/errsx"
)

// Config holds the complete configuration for a Crypto instance
type Config struct {
	KMSService        KeyManagementService
	KEKAlias          string
	Pepper            []byte
	PepperSecretPath  string
	Argon2Params      *Argon2Params
	KeyMetadataDB     *sql.DB
	DBPath            string
	DBFilename        string
	MetricsCollector  MetricsCollector
	ObservabilityHook ObservabilityHook
}

// KeyManagementService defines the interface for KMS operations
type KeyManagementService interface {
	GetKeyID(ctx context.Context, alias string) (string, error)
	CreateKey(ctx context.Context, description string) (string, error)
	EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
	DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}

// Argon2Params holds parameters for Argon2 hashing (internal config version)
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// Interface methods for crypto package compatibility
func (a *Argon2Params) GetMemory() uint32     { return a.Memory }
func (a *Argon2Params) GetIterations() uint32 { return a.Iterations }
func (a *Argon2Params) GetParallelism() uint8 { return a.Parallelism }
func (a *Argon2Params) GetSaltLength() uint32 { return a.SaltLength }
func (a *Argon2Params) GetKeyLength() uint32  { return a.KeyLength }

// Validate checks if the Argon2 parameters are within acceptable ranges
func (a *Argon2Params) Validate() error {
	errs := errsx.Map{}

	// Memory should be at least 8KB (8192 KiB)
	if a.Memory < 8192 {
		errs.Set("memory", fmt.Errorf("memory must be at least 8192 KiB, got %d", a.Memory))
	}

	// Iterations should be at least 2
	if a.Iterations < 2 {
		errs.Set("iterations", fmt.Errorf("iterations must be at least 2, got %d", a.Iterations))
	}

	// Parallelism should be at least 1
	if a.Parallelism < 1 {
		errs.Set("parallelism", fmt.Errorf("parallelism must be at least 1, got %d", a.Parallelism))
	}

	// Salt length should be at least 16 bytes
	if a.SaltLength < 16 {
		errs.Set("saltLength", fmt.Errorf("salt length must be at least 16 bytes, got %d", a.SaltLength))
	}

	// Key length should be at least 32 bytes
	if a.KeyLength < 32 {
		errs.Set("keyLength", fmt.Errorf("key length must be at least 32 bytes, got %d", a.KeyLength))
	}

	return errs.AsError()
}

// Type aliases for interfaces from monitoring package
type (
	MetricsCollector  = monitoring.MetricsCollector
	ObservabilityHook = monitoring.ObservabilityHook
)

// Validator handles configuration validation
type Validator struct{}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateConfig validates the complete configuration
func (v *Validator) ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate KMS service
	if err := v.validateKMSService(config.KMSService); err != nil {
		return fmt.Errorf("KMS service validation failed: %w", err)
	}

	// Validate KEK alias
	if err := v.validateKEKAlias(config.KEKAlias); err != nil {
		return fmt.Errorf("KEK alias validation failed: %w", err)
	}

	// Validate Argon2 parameters
	if err := v.validateArgon2Params(config.Argon2Params); err != nil {
		return fmt.Errorf("Argon2 parameters validation failed: %w", err)
	}

	// Validate database configuration
	if err := v.validateDatabaseConfig(config.DBPath, config.DBFilename); err != nil {
		return fmt.Errorf("database configuration validation failed: %w", err)
	}

	// TODO: Serializer validation removed - handled in generated code

	// Validate pepper configuration
	if err := v.validatePepperConfig(config.Pepper, config.PepperSecretPath); err != nil {
		return fmt.Errorf("pepper configuration validation failed: %w", err)
	}

	return nil
}

// validateKMSService validates the KMS service
func (v *Validator) validateKMSService(kms KeyManagementService) error {
	if kms == nil {
		return fmt.Errorf("KMS service is required")
	}
	return nil
}

// validateKEKAlias validates the KEK alias
func (v *Validator) validateKEKAlias(alias string) error {
	if strings.TrimSpace(alias) == "" {
		return fmt.Errorf("KEK alias is required")
	}
	if len(alias) > 256 {
		return fmt.Errorf("KEK alias too long: maximum 256 characters, got %d", len(alias))
	}
	return nil
}

// validateArgon2Params validates Argon2 parameters
func (v *Validator) validateArgon2Params(params *Argon2Params) error {
	if params == nil {
		return fmt.Errorf("Argon2 parameters cannot be nil")
	}

	if params.Memory < 4096 {
		return fmt.Errorf("Argon2 memory parameter too low: minimum 4096 KiB, got %d", params.Memory)
	}
	if params.Memory > 1048576 { // 1GB limit
		return fmt.Errorf("Argon2 memory parameter too high: maximum 1048576 KiB, got %d", params.Memory)
	}

	if params.Iterations < 1 {
		return fmt.Errorf("Argon2 iterations parameter too low: minimum 1, got %d", params.Iterations)
	}
	if params.Iterations > 100 {
		return fmt.Errorf("Argon2 iterations parameter too high: maximum 100, got %d", params.Iterations)
	}

	if params.Parallelism < 1 {
		return fmt.Errorf("Argon2 parallelism parameter too low: minimum 1, got %d", params.Parallelism)
	}
	if params.Parallelism > 255 {
		return fmt.Errorf("Argon2 parallelism parameter too high: maximum 255, got %d", params.Parallelism)
	}

	if params.SaltLength < 8 {
		return fmt.Errorf("Argon2 salt length too low: minimum 8 bytes, got %d", params.SaltLength)
	}
	if params.SaltLength > 64 {
		return fmt.Errorf("Argon2 salt length too high: maximum 64 bytes, got %d", params.SaltLength)
	}

	if params.KeyLength < 16 {
		return fmt.Errorf("Argon2 key length too low: minimum 16 bytes, got %d", params.KeyLength)
	}
	if params.KeyLength > 128 {
		return fmt.Errorf("Argon2 key length too high: maximum 128 bytes, got %d", params.KeyLength)
	}

	return nil
}

// validateDatabaseConfig validates database configuration
func (v *Validator) validateDatabaseConfig(dbPath, dbFilename string) error {
	if strings.TrimSpace(dbPath) == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if strings.TrimSpace(dbFilename) == "" {
		return fmt.Errorf("database filename cannot be empty")
	}

	// Check if the directory is writable
	if err := checkDirectoryWritable(dbPath); err != nil {
		return fmt.Errorf("database directory validation failed: %w", err)
	}

	return nil
}

// validateSerializer validates the serializer (deprecated - now using compact serializer)
func (v *Validator) validateSerializer() error {
	// No validation needed for compact serializer - it's built-in
	return nil
}

// validatePepperConfig validates pepper configuration
func (v *Validator) validatePepperConfig(pepper []byte, pepperSecretPath string) error {
	// Check for conflict: both pepper and pepperSecretPath provided
	if len(pepper) > 0 && strings.TrimSpace(pepperSecretPath) != "" {
		return fmt.Errorf("pepper cannot be provided both directly and via secret path")
	}

	if len(pepper) == 0 && strings.TrimSpace(pepperSecretPath) == "" {
		return fmt.Errorf("pepper must be provided")
	}

	if len(pepper) > 0 {
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
	}

	return nil
}

// checkDirectoryWritable checks if a directory is writable
func checkDirectoryWritable(dirPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", dirPath, err)
	}

	// Test write permissions by creating a temporary file
	testFile := filepath.Join(dirPath, ".encx_write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory '%s' is not writable: %w", dirPath, err)
	}
	file.Close()

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log warning but don't fail validation
		// logger.Warn("Failed to remove test file", "file", testFile, "error", err)
	}

	return nil
}
