package encx

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hengadev/encx/internal/config"
	"github.com/hengadev/encx/internal/crypto"
	"github.com/hengadev/encx/internal/monitoring"
	"github.com/hengadev/encx/internal/types"
	"github.com/hengadev/errsx"

	_ "github.com/mattn/go-sqlite3"
)

// Type aliases
type (
	MetricsCollector  = monitoring.MetricsCollector
	ObservabilityHook = monitoring.ObservabilityHook
	Config            = config.Config
	Option            = config.Option
	Action            = types.Action
)

// Action constants
const (
	Unknown    = types.Unknown
	BasicHash  = types.BasicHash
	SecureHash = types.SecureHash
	Encrypt    = types.Encrypt
	Decrypt    = types.Decrypt
)

// Interface aliases
type (
	KeyManagementService = config.KeyManagementService
)

type CryptoService interface {
	GetPepper() []byte
	GetArgon2Params() *Argon2Params
	GetAlias() string
	GenerateDEK() ([]byte, error)
	EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)
	DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
	EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error)
	DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error)
	RotateKEK(ctx context.Context) error
	HashBasic(ctx context.Context, value []byte) string
	HashSecure(ctx context.Context, value []byte) (string, error)
	CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
	CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
	EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
	DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
	GetCurrentKEKVersion(ctx context.Context, alias string) (int, error)
	GetKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error)
}

type Crypto struct {
	kmsService        KeyManagementService
	kekAlias          string
	pepper            []byte
	argon2Params      *Argon2Params
	keyMetadataDB     *sql.DB
	metricsCollector  MetricsCollector
	observabilityHook ObservabilityHook

	// Internal components
	dekOps         *crypto.DEKOperations
	dataEncryption *crypto.DataEncryption
	hashingOps     *crypto.HashingOperations
	keyRotationOps *crypto.KeyRotationOperations
}

// ValidateEnvironment validates the required environment variables and configuration
// before creating a Crypto instance. This allows for fail-fast validation at application
// startup rather than discovering configuration errors during runtime.
//
// This function checks:
// - ENCX_KEK_ALIAS is set and valid (max 256 characters)
// - ENCX_PEPPER_SECRET_PATH is set (or ENCX_ALLOW_IN_MEMORY_PEPPER=true for testing)
// - Pepper storage directory exists and is writable (if path is provided)
//
// Example usage in main():
//
//	func main() {
//	    // Validate environment at startup
//	    if err := encx.ValidateEnvironment(); err != nil {
//	        log.Fatal("Invalid ENCX configuration:", err)
//	    }
//
//	    // Now start your application
//	    startServer()
//	}
//
// Returns nil if validation succeeds, or an error describing what's wrong.
func ValidateEnvironment() error {
	// Check ENCX_KEK_ALIAS
	kekAlias := os.Getenv("ENCX_KEK_ALIAS")
	if kekAlias == "" {
		return fmt.Errorf("ENCX_KEK_ALIAS environment variable is required")
	}
	if len(kekAlias) > 256 {
		return fmt.Errorf("ENCX_KEK_ALIAS must be 256 characters or less, got %d", len(kekAlias))
	}

	// Check ENCX_PEPPER_SECRET_PATH (required unless explicitly opted out)
	pepperPath := os.Getenv("ENCX_PEPPER_SECRET_PATH")
	allowInMemory := os.Getenv("ENCX_ALLOW_IN_MEMORY_PEPPER") == "true"

	if pepperPath == "" && !allowInMemory {
		return fmt.Errorf("ENCX_PEPPER_SECRET_PATH is required for production use. " +
			"Set ENCX_ALLOW_IN_MEMORY_PEPPER=true only for testing (data will be lost on restart)")
	}

	// Validate pepper path if provided
	if pepperPath != "" {
		if err := validatePepperPath(pepperPath); err != nil {
			return fmt.Errorf("pepper path validation failed: %w", err)
		}
	}

	return nil
}

// validatePepperPath checks that the pepper storage directory exists and is writable
func validatePepperPath(pepperPath string) error {
	dir := filepath.Dir(pepperPath)

	// Check if directory exists, if not try to create it
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Try to create the directory with secure permissions
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("pepper directory '%s' does not exist and cannot be created: %w", dir, err)
		}
	}

	// Check if directory is writable by attempting to create a test file
	testFile := filepath.Join(dir, ".encx_validate_test")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("pepper directory '%s' is not writable: %w", dir, err)
	}
	f.Close()
	os.Remove(testFile) // Clean up test file

	return nil
}

// generateRandomPepper creates a cryptographically secure random 32-byte pepper
func generateRandomPepper() ([]byte, error) {
	pepper := make([]byte, 32)
	if _, err := rand.Read(pepper); err != nil {
		return nil, fmt.Errorf("failed to generate random pepper: %w", err)
	}
	return pepper, nil
}

// loadOrGeneratePepper loads pepper from storage or generates a new one
func loadOrGeneratePepper(ctx context.Context, kmsService KeyManagementService, kekAlias, pepperPath string) ([]byte, error) {
	// Check if pepper already exists at the specified path
	if pepperPath != "" {
		if pepper, exists := checkPepperExists(pepperPath); exists {
			return pepper, nil
		}
	}

	// Generate new pepper
	pepper, err := generateRandomPepper()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pepper: %w", err)
	}

	// Store the pepper using KMS if path is provided
	if pepperPath != "" {
		if err := storePepperWithKMS(ctx, kmsService, kekAlias, pepperPath, pepper); err != nil {
			return nil, fmt.Errorf("failed to store pepper: %w", err)
		}
	}

	return pepper, nil
}

// checkPepperExists checks if pepper exists at the specified path
// This is a simplified implementation - in production you'd integrate with your secret store
func checkPepperExists(pepperPath string) ([]byte, bool) {
	// For now, implement file-based storage
	// In production, this could be KMS, Vault, etc.
	if _, err := os.Stat(pepperPath); os.IsNotExist(err) {
		return nil, false
	}

	pepper, err := os.ReadFile(pepperPath)
	if err != nil {
		return nil, false
	}

	if len(pepper) != 32 {
		return nil, false
	}

	return pepper, true
}

// storePepperWithKMS stores pepper using KMS encryption
// This is a simplified implementation - in production you'd integrate with your secret store
func storePepperWithKMS(ctx context.Context, kmsService KeyManagementService, kekAlias, pepperPath string, pepper []byte) error {
	// For now, implement file-based storage with basic permissions
	// In production, this should use KMS to encrypt the pepper before storing
	dir := filepath.Dir(pepperPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create pepper directory: %w", err)
	}

	if err := os.WriteFile(pepperPath, pepper, 0600); err != nil {
		return fmt.Errorf("failed to write pepper file: %w", err)
	}

	return nil
}

// NewCrypto creates a new Crypto instance with automatic pepper management
//
// This function creates a new Crypto instance with the following automatic initialization:
// 1. Reads required environment variables (ENCX_KEK_ALIAS, ENCX_PEPPER_SECRET_PATH)
// 2. Generates or loads pepper using KMS integration
// 3. Creates and initializes the SQLite database with required schema
// 4. Sets up internal cryptographic components
// 5. Ensures an initial KEK exists in the database
//
// **Environment Variables:**
//
//	ENCX_KEK_ALIAS: KMS key alias for this service (required)
//	ENCX_PEPPER_SECRET_PATH: Path where pepper is stored/loaded (required for production)
//	ENCX_ALLOW_IN_MEMORY_PEPPER: Set to "true" for testing only (allows empty pepper path)
//
// **Parameters:**
//
//	ctx: Context for the initialization process
//	kmsService: KMS service instance (required)
//	options: Optional configuration (database settings, monitoring, etc.)
//
// **Returns:**
//
//	*Crypto: Fully initialized crypto instance
//	error: Initialization error
//
// **Production Example:**
//
//	// Set environment variables:
//	// export ENCX_KEK_ALIAS="my-service-prod"
//	// export ENCX_PEPPER_SECRET_PATH="/etc/encx/pepper"
//
//	crypto, err := encx.NewCrypto(ctx, kmsService)
//	if err != nil {
//	    panic(err)
//	}
//	// Ready to use immediately!
//
// **Testing Example:**
//
//	// For testing only - data will be lost on restart
//	os.Setenv("ENCX_KEK_ALIAS", "test-key")
//	os.Setenv("ENCX_ALLOW_IN_MEMORY_PEPPER", "true")
//
//	crypto, err := encx.NewCrypto(ctx, testKMS)
//	// ...
func NewCrypto(ctx context.Context, kmsService KeyManagementService, options ...Option) (*Crypto, error) {
	// Validate required KMS service
	if kmsService == nil {
		return nil, fmt.Errorf("KMS service is required")
	}

	// Read required environment variables
	kekAlias := os.Getenv("ENCX_KEK_ALIAS")
	pepperSecretPath := os.Getenv("ENCX_PEPPER_SECRET_PATH")
	allowInMemory := os.Getenv("ENCX_ALLOW_IN_MEMORY_PEPPER") == "true"

	if kekAlias == "" {
		return nil, fmt.Errorf("ENCX_KEK_ALIAS environment variable is required")
	}

	// Enforce pepper persistence requirement (with testing opt-out)
	if pepperSecretPath == "" && !allowInMemory {
		return nil, fmt.Errorf("ENCX_PEPPER_SECRET_PATH is required for production use. " +
			"Set ENCX_ALLOW_IN_MEMORY_PEPPER=true only for testing (data will be lost on restart)")
	}

	// Start with default configuration
	cfg := config.DefaultConfig()

	// Set required configuration from environment and parameters
	cfg.KMSService = kmsService
	cfg.KEKAlias = kekAlias

	// Apply optional configuration
	if err := config.ApplyOptions(cfg, options); err != nil {
		return nil, fmt.Errorf("failed to apply options: %w", err)
	}

	// Generate or load pepper automatically
	pepper, err := loadOrGeneratePepper(ctx, kmsService, kekAlias, pepperSecretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate pepper: %w", err)
	}
	cfg.Pepper = pepper

	// Validate configuration (skip pepper validation since it's auto-generated)
	validator := config.NewValidator()
	if err := validator.ValidateConfigForEnvironment(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Initialize KeyMetadataDB if not provided via options
	if cfg.KeyMetadataDB == nil {
		dbPath := filepath.Join(cfg.DBPath, cfg.DBFilename)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open key metadata database at '%s': %w", dbPath, err)
		}

		// Initialize database schema if needed
		if err := initializeDatabaseSchema(ctx, db); err != nil {
			return nil, fmt.Errorf("failed to initialize database schema: %w", err)
		}

		cfg.KeyMetadataDB = db
	}

	// Set defaults for optional components
	if cfg.MetricsCollector == nil {
		cfg.MetricsCollector = &monitoring.NoOpMetricsCollector{}
	}
	if cfg.ObservabilityHook == nil {
		cfg.ObservabilityHook = &monitoring.NoOpObservabilityHook{}
	}

	// Create Crypto instance
	cryptoInstance := &Crypto{
		kmsService:        cfg.KMSService,
		kekAlias:          cfg.KEKAlias,
		pepper:            cfg.Pepper,
		argon2Params:      convertArgon2Params(cfg.Argon2Params),
		keyMetadataDB:     cfg.KeyMetadataDB,
		metricsCollector:  cfg.MetricsCollector,
		observabilityHook: cfg.ObservabilityHook,
	}

	// Initialize internal components
	dekOps, err := crypto.NewDEKOperations(cfg.KMSService, cfg.KEKAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to create DEK operations: %w", err)
	}
	cryptoInstance.dekOps = dekOps

	cryptoInstance.dataEncryption = crypto.NewDataEncryption()

	hashingOps, err := crypto.NewHashingOperations(cfg.Pepper, cryptoInstance.argon2Params)
	if err != nil {
		return nil, fmt.Errorf("failed to create hashing operations: %w", err)
	}
	cryptoInstance.hashingOps = hashingOps

	keyRotationOps, err := crypto.NewKeyRotationOperations(cfg.KMSService, cfg.KEKAlias, cfg.KeyMetadataDB, cfg.ObservabilityHook)
	if err != nil {
		return nil, fmt.Errorf("failed to create key rotation operations: %w", err)
	}
	cryptoInstance.keyRotationOps = keyRotationOps

	// Ensure initial KEK exists (create if needed)
	if err := cryptoInstance.keyRotationOps.EnsureInitialKEK(ctx, cryptoInstance); err != nil {
		return nil, fmt.Errorf("failed to ensure initial KEK: %w", err)
	}

	return cryptoInstance, nil
}

// convertArgon2Params converts config.Argon2Params to root Argon2Params
func convertArgon2Params(configParams *config.Argon2Params) *Argon2Params {
	if configParams == nil {
		return nil
	}
	return &Argon2Params{
		Memory:      configParams.Memory,
		Iterations:  configParams.Iterations,
		Parallelism: configParams.Parallelism,
		SaltLength:  configParams.SaltLength,
		KeyLength:   configParams.KeyLength,
	}
}

// Public API methods that delegate to internal components

func (c *Crypto) GenerateDEK() ([]byte, error) {
	return c.dekOps.GenerateDEK()
}

func (c *Crypto) EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error) {
	return c.dataEncryption.EncryptData(ctx, plaintext, dek)
}

func (c *Crypto) DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error) {
	return c.dataEncryption.DecryptData(ctx, ciphertext, dek)
}

func (c *Crypto) EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error) {
	return c.dekOps.EncryptDEK(ctx, plaintextDEK, c)
}

func (c *Crypto) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error) {
	return c.dekOps.DecryptDEKWithVersion(ctx, ciphertextDEK, kekVersion, c)
}

func (c *Crypto) HashBasic(ctx context.Context, value []byte) string {
	return c.hashingOps.HashBasic(ctx, value)
}

func (c *Crypto) HashSecure(ctx context.Context, value []byte) (string, error) {
	return c.hashingOps.HashSecure(ctx, value)
}

func (c *Crypto) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	return c.hashingOps.CompareSecureHashAndValue(ctx, value, hashValue)
}

func (c *Crypto) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error) {
	return c.hashingOps.CompareBasicHashAndValue(ctx, value, hashValue)
}

func (c *Crypto) EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error {
	return c.dataEncryption.EncryptStream(ctx, reader, writer, dek)
}

func (c *Crypto) DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error {
	return c.dataEncryption.DecryptStream(ctx, reader, writer, dek)
}

func (c *Crypto) RotateKEK(ctx context.Context) error {
	return c.keyRotationOps.RotateKEK(ctx, c)
}

// errorCollectorAdapter adapts errsx.Map to processor.ErrorCollector interface
type errorCollectorAdapter struct {
	errMap errsx.Map
}

func (a *errorCollectorAdapter) Set(key string, err error) {
	a.errMap.Set(key, err)
}

func (a *errorCollectorAdapter) AsError() error {
	return a.errMap.AsError()
}

func (a *errorCollectorAdapter) IsEmpty() bool {
	return a.errMap.IsEmpty()
}

// Getter methods
func (c *Crypto) GetPepper() []byte {
	return c.pepper
}

func (c *Crypto) GetArgon2Params() *Argon2Params {
	return c.argon2Params
}

func (c *Crypto) GetAlias() string {
	return c.kekAlias
}

// Internal interface implementations
func (c *Crypto) GetCurrentKEKVersion(ctx context.Context, alias string) (int, error) {
	return c.getCurrentKEKVersion(ctx, alias)
}

func (c *Crypto) GetKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error) {
	return c.getKMSKeyIDForVersion(ctx, alias, version)
}

func (c *Crypto) getDatabasePathFromDB() (string, error) {
	var path string
	err := c.keyMetadataDB.QueryRow("PRAGMA database_list;").Scan(nil, &path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get database path from connection: %w", err)
	}
	return path, nil
}

// getKMSKeyIDForVersion retrieves the KMS Key ID for a specific KEK version and alias.
func (c *Crypto) getKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error) {
	row := c.keyMetadataDB.QueryRowContext(ctx, `
		SELECT kms_key_id FROM kek_versions
		WHERE alias = ? AND version = ?
	`, alias, version)
	var kmsKeyID string
	err := row.Scan(&kmsKeyID)
	if err != nil {
		return "", fmt.Errorf("failed to get KMS Key ID for alias '%s' version %d: %w", alias, version, err)
	}
	return kmsKeyID, nil
}

// getCurrentKEKVersion retrieves the current active KEK version for a given alias.
func (c *Crypto) getCurrentKEKVersion(ctx context.Context, alias string) (int, error) {
	row := c.keyMetadataDB.QueryRowContext(ctx, `
		SELECT version FROM kek_versions
		WHERE alias = ? AND is_deprecated = FALSE
		ORDER BY version DESC
		LIMIT 1
	`, alias)
	var version int
	err := row.Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil // No active key found
	} else if err != nil {
		return 0, fmt.Errorf("failed to get current KEK version for alias '%s': %w", alias, err)
	}
	return version, nil
}

// initializeDatabaseSchema creates the necessary tables if they don't exist
func initializeDatabaseSchema(ctx context.Context, db *sql.DB) error {
	// Check if kek_versions table exists
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='kek_versions'
	`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if kek_versions table exists: %w", err)
	}

	// Create table if it doesn't exist
	if count == 0 {
		_, err = db.ExecContext(ctx, `
			CREATE TABLE kek_versions (
				alias TEXT NOT NULL,
				version INTEGER NOT NULL,
				kms_key_id TEXT NOT NULL,
				is_deprecated BOOLEAN DEFAULT FALSE,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (alias, version)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create kek_versions table: %w", err)
		}

		// Create index for efficient queries
		_, err = db.ExecContext(ctx, `
			CREATE INDEX idx_kek_versions_active
			ON kek_versions(alias, is_deprecated, version DESC)
		`)
		if err != nil {
			return fmt.Errorf("failed to create index on kek_versions table: %w", err)
		}
	}

	return nil
}
