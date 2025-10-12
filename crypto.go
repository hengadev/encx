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

// Crypto provides cryptographic operations with Key Management Service (KMS) integration
// and secret storage capabilities.
//
// This struct holds both:
//   - KeyManagementService: for cryptographic operations (encrypting/decrypting DEKs)
//   - SecretManagementService: for secure storage of secrets (like peppers)
type Crypto struct {
	kmsService        KeyManagementService
	secretStore       SecretManagementService
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

// generateRandomPepper creates a cryptographically secure random 32-byte pepper
func generateRandomPepper() ([]byte, error) {
	pepper := make([]byte, 32)
	if _, err := rand.Read(pepper); err != nil {
		return nil, fmt.Errorf("failed to generate random pepper: %w", err)
	}
	return pepper, nil
}

// loadOrGeneratePepperFromSecretStore loads an existing pepper from the SecretManagementService
// or generates and stores a new one if it doesn't exist.
//
// This function implements automatic pepper lifecycle management:
// 1. Check if pepper exists using PepperExists()
// 2. If yes, load it using GetPepper()
// 3. If no, generate a new random 32-byte pepper
// 4. Store the new pepper using StorePepper()
// 5. Return the pepper
//
// The pepper is stored at a path determined by the SecretManagementService implementation
// based on the pepperAlias (see GetStoragePath()).
func loadOrGeneratePepperFromSecretStore(ctx context.Context, secrets SecretManagementService, pepperAlias string) ([]byte, error) {
	// Check if pepper already exists
	exists, err := secrets.PepperExists(ctx, pepperAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to check if pepper exists: %w", err)
	}

	// If pepper exists, load and return it
	if exists {
		pepper, err := secrets.GetPepper(ctx, pepperAlias)
		if err != nil {
			return nil, fmt.Errorf("failed to load existing pepper: %w", err)
		}
		return pepper, nil
	}

	// Generate new random pepper
	pepper, err := generateRandomPepper()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new pepper: %w", err)
	}

	// Store the new pepper
	if err := secrets.StorePepper(ctx, pepperAlias, pepper); err != nil {
		return nil, fmt.Errorf("failed to store new pepper: %w", err)
	}

	return pepper, nil
}

// NewCrypto creates a new Crypto instance with explicit configuration and dependencies.
//
// This is the primary constructor for production use. It provides explicit dependency injection
// for both the KeyManagementService (cryptographic operations) and SecretManagementService
// (pepper storage), along with explicit configuration.
//
// Architecture:
//   - Separates cryptographic operations (KMS) from secret storage (SecretManagementService)
//   - Automatic pepper lifecycle management (load or generate)
//   - Validates configuration and applies defaults
//   - Initializes key metadata database with proper schema
//   - Sets up all internal cryptographic components
//
// **Parameters:**
//
//	ctx: Context for initialization operations (pepper loading, database setup, etc.)
//	kms: KeyManagementService for encrypting/decrypting DEKs (required)
//	secrets: SecretManagementService for pepper storage (required)
//	cfg: Configuration struct with KEKAlias, PepperAlias, database settings (required)
//	options: Optional runtime configuration (metrics, observability, custom database, etc.)
//
// **Returns:**
//
//	*Crypto: Fully initialized crypto instance
//	error: Initialization error
//
// **Example Usage (AWS):**
//
//	import (
//	    "github.com/hengadev/encx"
//	    "github.com/hengadev/encx/providers/aws"
//	)
//
//	// Create AWS providers
//	kms, err := aws.NewKMSService()
//	secrets, err := aws.NewSecretsManagerStore()
//
//	// Configure
//	cfg := encx.Config{
//	    KEKAlias:    "alias/my-service-kek",
//	    PepperAlias: "my-service",
//	}
//
//	// Create crypto instance
//	crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
//
// **Example Usage (HashiCorp Vault):**
//
//	import (
//	    "github.com/hengadev/encx"
//	    "github.com/hengadev/encx/providers/hashicorp"
//	)
//
//	// Create Vault providers
//	transit, err := hashicorp.NewTransitService()
//	kv, err := hashicorp.NewKVStore()
//
//	// Configure
//	cfg := encx.Config{
//	    KEKAlias:    "my-service-kek",
//	    PepperAlias: "my-service",
//	}
//
//	// Create crypto instance
//	crypto, err := encx.NewCrypto(ctx, transit, kv, cfg)
//
// **Example Usage (Testing):**
//
//	// Use in-memory implementations for testing
//	kms := encx.NewSimpleTestKMS()
//	secrets := encx.NewInMemorySecretStore()
//
//	cfg := encx.Config{
//	    KEKAlias:    "test-key",
//	    PepperAlias: "test-service",
//	}
//
//	crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
//
// **Configuration Loading:**
//
// For 12-factor apps, use LoadConfigFromEnvironment():
//
//	cfg, err := encx.LoadConfigFromEnvironment()
//	crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
//
// Or for a convenience function that does both, use NewCryptoFromEnv():
//
//	crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
func NewCrypto(
	ctx context.Context,
	kms KeyManagementService,
	secrets SecretManagementService,
	cfg Config,
	options ...Option,
) (*Crypto, error) {
	// Validate required dependencies
	if kms == nil {
		return nil, fmt.Errorf("KeyManagementService is required")
	}
	if secrets == nil {
		return nil, fmt.Errorf("SecretManagementService is required")
	}

	// Validate and apply defaults to configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Load or generate pepper from SecretManagementService
	pepper, err := loadOrGeneratePepperFromSecretStore(ctx, secrets, cfg.PepperAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate pepper: %w", err)
	}

	// Create internal config for compatibility with existing code
	internalCfg := config.DefaultConfig()
	internalCfg.KMSService = kms
	internalCfg.KEKAlias = cfg.KEKAlias
	internalCfg.Pepper = pepper
	internalCfg.DBPath = cfg.DBPath
	internalCfg.DBFilename = cfg.DBFilename

	// Apply optional configuration
	if err := config.ApplyOptions(internalCfg, options); err != nil {
		return nil, fmt.Errorf("failed to apply options: %w", err)
	}

	// Initialize KeyMetadataDB if not provided via options
	if internalCfg.KeyMetadataDB == nil {
		dbPath := filepath.Join(cfg.DBPath, cfg.DBFilename)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open key metadata database at '%s': %w", dbPath, err)
		}

		// Initialize database schema if needed
		if err := initializeDatabaseSchema(ctx, db); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to initialize database schema: %w", err)
		}

		internalCfg.KeyMetadataDB = db
	}

	// Set defaults for optional components
	if internalCfg.MetricsCollector == nil {
		internalCfg.MetricsCollector = &monitoring.NoOpMetricsCollector{}
	}
	if internalCfg.ObservabilityHook == nil {
		internalCfg.ObservabilityHook = &monitoring.NoOpObservabilityHook{}
	}

	// Create Crypto instance with both services
	cryptoInstance := &Crypto{
		kmsService:        kms,
		secretStore:       secrets,
		kekAlias:          cfg.KEKAlias,
		pepper:            pepper,
		argon2Params:      convertArgon2Params(internalCfg.Argon2Params),
		keyMetadataDB:     internalCfg.KeyMetadataDB,
		metricsCollector:  internalCfg.MetricsCollector,
		observabilityHook: internalCfg.ObservabilityHook,
	}

	// Initialize internal components
	dekOps, err := crypto.NewDEKOperations(kms, cfg.KEKAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to create DEK operations: %w", err)
	}
	cryptoInstance.dekOps = dekOps

	cryptoInstance.dataEncryption = crypto.NewDataEncryption()

	hashingOps, err := crypto.NewHashingOperations(pepper, cryptoInstance.argon2Params)
	if err != nil {
		return nil, fmt.Errorf("failed to create hashing operations: %w", err)
	}
	cryptoInstance.hashingOps = hashingOps

	keyRotationOps, err := crypto.NewKeyRotationOperations(kms, cfg.KEKAlias, internalCfg.KeyMetadataDB, internalCfg.ObservabilityHook)
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

// NewCryptoFromEnv is a convenience constructor that loads configuration from environment variables.
//
// This function combines LoadConfigFromEnvironment() and NewCrypto() into a single call,
// making it ideal for 12-factor applications that use environment-based configuration.
//
// **Environment Variables:**
//
//	ENCX_KEK_ALIAS: KMS key alias for this service (required)
//	ENCX_PEPPER_ALIAS: Service identifier for pepper storage (required)
//	ENCX_DB_PATH: Database directory (optional, default: .encx)
//	ENCX_DB_FILENAME: Database filename (optional, default: keys.db)
//
// **Parameters:**
//
//	ctx: Context for initialization operations
//	kms: KeyManagementService for encrypting/decrypting DEKs (required)
//	secrets: SecretManagementService for pepper storage (required)
//	options: Optional runtime configuration (metrics, observability, etc.)
//
// **Returns:**
//
//	*Crypto: Fully initialized crypto instance
//	error: Initialization or configuration error
//
// **Example Usage:**
//
//	// Set environment variables
//	// export ENCX_KEK_ALIAS="alias/my-service-kek"
//	// export ENCX_PEPPER_ALIAS="my-service"
//
//	kms, err := aws.NewKMSService()
//	secrets, err := aws.NewSecretsManagerStore()
//
//	crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
//
// This is equivalent to:
//
//	cfg, err := encx.LoadConfigFromEnvironment()
//	crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
func NewCryptoFromEnv(
	ctx context.Context,
	kms KeyManagementService,
	secrets SecretManagementService,
	options ...Option,
) (*Crypto, error) {
	// Load configuration from environment variables
	cfg, err := LoadConfigFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration from environment: %w", err)
	}

	// Create crypto instance with loaded configuration
	return NewCrypto(ctx, kms, secrets, cfg, options...)
}

// newCryptoLegacy creates a new Crypto instance with automatic pepper management (DEPRECATED)
//
// This is the legacy constructor that reads configuration from environment variables.
// It is kept temporarily for backward compatibility during migration.
//
// DEPRECATED: Use NewCrypto(ctx, kms, secrets, cfg, options...) instead.
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
func newCryptoLegacy(ctx context.Context, kmsService KeyManagementService, options ...Option) (*Crypto, error) {
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
