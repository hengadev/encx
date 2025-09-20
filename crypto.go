package encx

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/hengadev/encx/internal/config"
	"github.com/hengadev/encx/internal/crypto"
	"github.com/hengadev/encx/internal/monitoring"
	"github.com/hengadev/encx/internal/performance"
	"github.com/hengadev/encx/internal/processor"
	"github.com/hengadev/encx/internal/serialization"
	"github.com/hengadev/encx/internal/types"
	"github.com/hengadev/errsx"

	_ "github.com/mattn/go-sqlite3"
)

// Type aliases for backward compatibility
type (
	MetricsCollector    = monitoring.MetricsCollector
	ObservabilityHook   = monitoring.ObservabilityHook
	Config              = config.Config
	Option              = config.Option
	Action              = types.Action
	BatchProcessOptions = performance.BatchProcessOptions
	BatchProcessResult  = performance.BatchProcessResult
	BatchError          = performance.BatchError
)

// Action constants for backward compatibility
const (
	Unknown    = types.Unknown
	BasicHash  = types.BasicHash
	SecureHash = types.SecureHash
	Encrypt    = types.Encrypt
	Decrypt    = types.Decrypt
)

// Interface aliases for backward compatibility
type (
	KeyManagementService = config.KeyManagementService
	Serializer           = serialization.Serializer
)

type CryptoService interface {
	GetPepper() []byte
	GetArgon2Params() *Argon2Params
	GetAlias() string
	GenerateDEK() ([]byte, error)
	EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)
	DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
	ProcessStruct(ctx context.Context, object any) error
	DecryptStruct(ctx context.Context, object any) error
	EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error)
	DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error)
	RotateKEK(ctx context.Context) error
	HashBasic(ctx context.Context, value []byte) string
	HashSecure(ctx context.Context, value []byte) (string, error)
	CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
	CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
	EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
	DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
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
	dekOps          *crypto.DEKOperations
	dataEncryption  *crypto.DataEncryption
	hashingOps      *crypto.HashingOperations
	keyRotationOps  *crypto.KeyRotationOperations
	structProcessor *processor.StructProcessor
}

// New creates a new Crypto instance using the legacy constructor signature.
// Deprecated: Use NewCrypto with options instead for better validation and flexibility.
//
// Example migration:
//
//	// Old way:
//	crypto, err := encx.New(ctx, kmsService, "my-kek", "secret/pepper")
//
//	// New way:
//	crypto, err := encx.NewCrypto(ctx,
//	    encx.WithKMSService(kmsService),
//	    encx.WithKEKAlias("my-kek"),
//	    encx.WithPepperSecretPath("secret/pepper"),
//	)
func New(
	ctx context.Context,
	kmsService KeyManagementService,
	kekAlias string,
	pepperSecretPath string,
	options ...Option,
) (*Crypto, error) {
	return NewCryptoLegacy(ctx, kmsService, kekAlias, pepperSecretPath, options...)
}

// NewCryptoLegacy creates a new Crypto instance using the legacy parameter style
func NewCryptoLegacy(
	ctx context.Context,
	kmsService KeyManagementService,
	kekAlias string,
	pepperSecretPath string,
	options ...Option,
) (*Crypto, error) {
	// Convert legacy parameters to options
	opts := []Option{
		WithKMSService(kmsService),
		WithKEKAlias(kekAlias),
		WithPepperSecretPath(pepperSecretPath),
	}

	// Append additional options
	opts = append(opts, options...)

	return NewCrypto(ctx, opts...)
}

// NewCrypto creates a new Crypto instance using the options pattern
func NewCrypto(ctx context.Context, options ...Option) (*Crypto, error) {
	cfg := config.DefaultConfig()

	// Apply all options
	if err := config.ApplyOptions(cfg, options); err != nil {
		return nil, fmt.Errorf("failed to apply options: %w", err)
	}

	// Validate configuration
	validator := config.NewValidator()
	if err := validator.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
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
	cryptoInstance.dekOps = crypto.NewDEKOperations(cfg.KMSService, cfg.KEKAlias)
	cryptoInstance.dataEncryption = crypto.NewDataEncryption()
	cryptoInstance.hashingOps = crypto.NewHashingOperations(cfg.Pepper, cryptoInstance.argon2Params, nil)
	cryptoInstance.keyRotationOps = crypto.NewKeyRotationOperations(cfg.KMSService, cfg.KEKAlias, cfg.KeyMetadataDB, cfg.ObservabilityHook)

	// Initialize struct processor components
	fieldProcessor := processor.NewFieldProcessor(cryptoInstance.dataEncryption, cryptoInstance.hashingOps, nil)
	processorValidator := processor.NewValidator(cryptoInstance.dekOps)
	cryptoInstance.structProcessor = processor.NewStructProcessor(fieldProcessor, processorValidator, cfg.ObservabilityHook, cryptoInstance)

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

func (c *Crypto) ProcessStruct(ctx context.Context, object any) error {
	errorCollector := &errorCollectorAdapter{}
	return c.structProcessor.ProcessStruct(ctx, object, errorCollector)
}

func (c *Crypto) DecryptStruct(ctx context.Context, object any) error {
	errorCollector := &errorCollectorAdapter{}
	return c.structProcessor.DecryptStruct(ctx, object, errorCollector)
}

// ProcessStructsBatch processes multiple structs concurrently with optimized batching
func (c *Crypto) ProcessStructsBatch(ctx context.Context, structs []any, options ...*BatchProcessOptions) (*BatchProcessResult, error) {
	return performance.ProcessStructsBatch(ctx, c, structs, options...)
}

// DecryptStructsBatch decrypts multiple structs concurrently with optimized batching
func (c *Crypto) DecryptStructsBatch(ctx context.Context, structs []any, options ...*BatchProcessOptions) (*BatchProcessResult, error) {
	return performance.DecryptStructsBatch(ctx, c, structs, options...)
}

// GetProcessingStats returns performance statistics
func (c *Crypto) GetProcessingStats() map[string]interface{} {
	return performance.GetProcessingStats()
}

// Getter methods for legacy compatibility
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
