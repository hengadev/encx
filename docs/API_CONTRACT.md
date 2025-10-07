# ENCX API Contract v1.0

**Status**: STABLE
**Version**: 1.0.0
**Last Updated**: 2025-10-05

This document defines the stable public API contract for ENCX. All functions marked as STABLE in this document MUST remain backward compatible within the v1.x version range.

---

## Table of Contents

1. [Core Crypto API](#core-crypto-api) - **STABLE**
2. [Configuration Options](#configuration-options) - **STABLE**
3. [KMS Provider Interface](#kms-provider-interface) - **STABLE**
4. [Code Generation API](#code-generation-api) - **BETA**
5. [Testing Utilities](#testing-utilities) - **STABLE**
6. [Observability](#observability) - **STABLE**
7. [Deprecated APIs](#deprecated-apis)

---

## Core Crypto API

**Status**: ✅ **STABLE** - These APIs are guaranteed to remain backward compatible in v1.x

### Constructor

```go
// NewCrypto creates a new Crypto instance using functional options
//
// This is the primary constructor and will remain stable for v1.x
func NewCrypto(ctx context.Context, options ...Option) (*Crypto, error)
```

**Contract Guarantees**:
- Function signature will not change
- Options pattern allows extensibility without breaking changes
- Returns `*Crypto` and `error`
- Validates all required parameters before returning

**Example**:
```go
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsService),
    encx.WithKEKAlias("my-kek"),
    encx.WithPepper(pepper),
)
```

---

### Data Encryption Key (DEK) Operations

```go
// GenerateDEK generates a new 32-byte Data Encryption Key
func (c *Crypto) GenerateDEK() ([]byte, error)

// EncryptDEK encrypts a DEK using the KEK from KMS
func (c *Crypto) EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error)

// DecryptDEKWithVersion decrypts a DEK using a specific KEK version
func (c *Crypto) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error)
```

**Contract Guarantees**:
- `GenerateDEK()` always returns 32 bytes (AES-256)
- DEK operations use crypto/rand for secure randomness
- KEK versions are monotonically increasing integers
- Thread-safe for concurrent use

---

### Data Encryption Operations

```go
// EncryptData encrypts plaintext using AES-256-GCM with the provided DEK
func (c *Crypto) EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)

// DecryptData decrypts ciphertext using AES-256-GCM with the provided DEK
func (c *Crypto) DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
```

**Contract Guarantees**:
- Uses AES-256-GCM (authenticated encryption)
- DEK must be exactly 32 bytes
- Empty plaintext is allowed
- Output includes authentication tag
- Thread-safe

**Error Conditions**:
- Returns error if DEK is not 32 bytes
- Returns error if decryption authentication fails
- Returns error if ciphertext is corrupted

---

### Stream Encryption Operations

```go
// EncryptStream encrypts data from reader to writer using the provided DEK
func (c *Crypto) EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error

// DecryptStream decrypts data from reader to writer using the provided DEK
func (c *Crypto) DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
```

**Contract Guarantees**:
- Suitable for large files (doesn't load entire file into memory)
- Uses chunked processing
- Context can be used for cancellation
- Thread-safe

---

### Hashing Operations

```go
// HashBasic creates a fast, non-cryptographic hash for indexing
func (c *Crypto) HashBasic(ctx context.Context, value []byte) string

// HashSecure creates a cryptographically secure hash using Argon2id
func (c *Crypto) HashSecure(ctx context.Context, value []byte) (string, error)

// CompareBasicHashAndValue compares a value against a basic hash
func (c *Crypto) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)

// CompareSecureHashAndValue compares a value against a secure hash (constant-time)
func (c *Crypto) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
```

**Contract Guarantees**:
- `HashBasic`: Fast, deterministic, suitable for database indexing
- `HashSecure`: Uses Argon2id with pepper, slow by design, timing-attack resistant
- Comparison functions use constant-time comparison for secure hashes
- Thread-safe

**Security Notes**:
- `HashBasic` is NOT suitable for passwords - use `HashSecure`
- `HashSecure` includes pepper for additional security
- Argon2 parameters are configurable via `WithArgon2Params`

---

### Key Rotation

```go
// RotateKEK initiates a KEK rotation process
func (c *Crypto) RotateKEK(ctx context.Context) error
```

**Contract Guarantees**:
- Creates new KEK version in KMS
- Maintains backward compatibility with old KEK versions
- Atomic operation (either succeeds completely or fails)
- Safe to call concurrently

---

### Utility Methods

```go
// GetPepper returns the pepper value (for testing/debugging)
func (c *Crypto) GetPepper() []byte

// GetArgon2Params returns the Argon2 parameters
func (c *Crypto) GetArgon2Params() *Argon2Params

// GetAlias returns the KEK alias
func (c *Crypto) GetAlias() string

// GetCurrentKEKVersion retrieves the current KEK version
func (c *Crypto) GetCurrentKEKVersion(ctx context.Context, alias string) (int, error)
```

---

## Configuration Options

**Status**: ✅ **STABLE** - All option functions are guaranteed stable

### Required Options

```go
// WithKMSService sets the Key Management Service provider
func WithKMSService(kms KeyManagementService) Option

// WithKEKAlias sets the Key Encryption Key alias
func WithKEKAlias(alias string) Option

// One of these MUST be provided:
func WithPepper(pepper []byte) Option                    // Direct pepper
func WithPepperSecretPath(secretPath string) Option      // Pepper from KMS secret
```

**Validation**:
- `WithKMSService`: KMS cannot be nil
- `WithKEKAlias`: Alias must be non-empty, max 256 chars, alphanumeric + `-_/`
- `WithPepper`: Must be exactly 32 bytes, cannot be all zeros
- `WithPepperSecretPath`: Path must be non-empty
- Cannot provide both `WithPepper` and `WithPepperSecretPath`

---

### Optional Configuration

```go
// WithArgon2Params configures Argon2id hashing parameters
func WithArgon2Params(params *Argon2Params) Option

// WithKeyMetadataDB provides a database connection for key metadata
func WithKeyMetadataDB(db *sql.DB) Option

// WithDBPath sets the database directory path
func WithDBPath(path string) Option

// WithDBFilename sets the database filename
func WithDBFilename(filename string) Option

// WithKeyMetadataDBPath sets the full database path
func WithKeyMetadataDBPath(path string) Option

// WithKeyMetadataDBFilename sets the DB filename in default directory
func WithKeyMetadataDBFilename(filename string) Option

// WithMetricsCollector sets the metrics collector
func WithMetricsCollector(collector MetricsCollector) Option

// WithObservabilityHook sets the observability hook
func WithObservabilityHook(hook ObservabilityHook) Option
```

**Default Values**:
- Argon2Params: Memory=65536 (64MB), Iterations=3, Parallelism=4, SaltLength=16, KeyLength=32
- DBPath: `.encx`
- DBFilename: `metadata.db`

---

## KMS Provider Interface

**Status**: ✅ **STABLE** - This interface will not change in v1.x

```go
type KeyManagementService interface {
    // GetKeyID resolves a key alias to a key ID
    GetKeyID(ctx context.Context, alias string) (string, error)

    // CreateKey creates a new encryption key
    CreateKey(ctx context.Context, description string) (string, error)

    // EncryptDEK encrypts a Data Encryption Key
    EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)

    // DecryptDEK decrypts a Data Encryption Key
    DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}
```

**Contract Guarantees**:
- All methods accept `context.Context` for cancellation/timeouts
- Thread-safe implementation required
- Error handling must be consistent with ENCX error types

**Supported Implementations**:
- ✅ HashiCorp Vault (`providers/hashicorpvault`)
- ✅ S3-based KMS (`providers/s3`)
- 🚧 AWS KMS (`providers/awskms`) - In development

---

## Code Generation API

**Status**: 🟡 **BETA** - Pattern is stable, implementation details may evolve

### Generated Function Pattern

For a struct with encx tags:

```go
type User struct {
    Email    string `encx:"encrypt,hash_basic"`
    Password string `encx:"hash_secure"`
}
```

The generator creates:

```go
// Process{StructName}Encx encrypts and hashes tagged fields
func ProcessUserEncx(ctx context.Context, crypto *encx.Crypto, source *User) (*UserEncx, error)

// Decrypt{StructName}Encx decrypts the encrypted struct
func DecryptUserEncx(ctx context.Context, crypto *encx.Crypto, source *UserEncx) (*User, error)
```

**Contract Guarantees**:
- Function naming pattern: `Process{StructName}Encx` and `Decrypt{StructName}Encx`
- Always takes `context.Context` as first parameter
- Always takes `*encx.Crypto` as second parameter
- Thread-safe generated code
- Uses `errsx.Map` for collecting multiple errors

**Beta Notice**: Field naming conventions and struct metadata may evolve based on user feedback.

---

## Testing Utilities

**Status**: ✅ **STABLE**

```go
// NewTestCrypto creates a Crypto instance suitable for testing
func NewTestCrypto(t interface{}) (*Crypto, error)

// NewSimpleTestKMS creates a simple in-memory KMS for testing
func NewSimpleTestKMS() KeyManagementService
```

**Contract Guarantees**:
- `NewTestCrypto`: Returns fully functional Crypto with sensible test defaults
- `NewSimpleTestKMS`: In-memory implementation, no external dependencies
- Both suitable for unit tests

---

## Observability

**Status**: ✅ **STABLE**

### Metrics Collector Interface

```go
type MetricsCollector interface {
    RecordOperation(ctx context.Context, action Action, duration time.Duration, err error)
    RecordKMSCall(ctx context.Context, operation string, duration time.Duration, err error)
}
```

### Observability Hook Interface

```go
type ObservabilityHook interface {
    BeforeOperation(ctx context.Context, action Action, metadata map[string]interface{}) context.Context
    AfterOperation(ctx context.Context, action Action, err error, metadata map[string]interface{})
    OnError(ctx context.Context, action Action, err error)
}
```

**Contract Guarantees**:
- All hooks are optional (nil-safe)
- Hooks execute synchronously
- Hook errors don't affect operation outcomes
- Thread-safe

---

## Deprecated APIs

**Status**: ⚠️ **DEPRECATED** - Will be removed in v2.0.0

### Deprecated Constructors

```go
// Deprecated: Use NewCrypto with functional options
func New(ctx context.Context, kmsService KeyManagementService, kekAlias string,
         pepperSecretPath string, options ...Option) (*Crypto, error)

// Deprecated: Use NewCrypto with functional options
func NewCryptoLegacy(ctx context.Context, kmsService KeyManagementService,
                     kekAlias string, pepperSecretPath string, options ...Option) (*Crypto, error)
```

**Migration Path**:
```go
// OLD:
crypto, err := encx.New(ctx, kms, "my-kek", "secret/pepper")

// NEW:
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kms),
    encx.WithKEKAlias("my-kek"),
    encx.WithPepperSecretPath("secret/pepper"),
)
```

**Deprecation Timeline**:
- v1.0.0 - v1.x: Marked as deprecated, fully supported
- v2.0.0: Removed

---

## Versioning Policy

ENCX follows Semantic Versioning 2.0.0:

- **Major version** (v2.0.0): Breaking changes to STABLE APIs
- **Minor version** (v1.1.0): New features, no breaking changes
- **Patch version** (v1.0.1): Bug fixes, no API changes

### BETA APIs

APIs marked as BETA may change in minor versions but will be clearly documented in release notes.

---

## Error Handling

All ENCX APIs follow consistent error patterns:

```go
// Sentinel errors for programmatic handling
var (
    ErrKMSUnavailable       error // KMS service unavailable
    ErrAuthenticationFailed error // Authentication failed
    ErrInvalidConfiguration error // Invalid configuration
    ErrEncryptionFailed     error // Encryption operation failed
    ErrDecryptionFailed     error // Decryption operation failed
    ErrDatabaseUnavailable  error // Database unavailable
)
```

**Error Guarantees**:
- All errors include context (operation, field name, etc.)
- Errors can be tested with `errors.Is()` and `errors.As()`
- Multi-errors use `errsx.Map` for structured error collection

---

## Thread Safety

**All public APIs are thread-safe unless explicitly documented otherwise.**

Concurrent use is safe for:
- All `Crypto` methods
- All configuration options
- KMS provider implementations
- Generated code

---

## Performance Guarantees

While specific performance will vary by environment:

- `HashBasic`: O(n) where n is input size, <1ms for typical inputs
- `HashSecure`: Intentionally slow (100-500ms), configurable via Argon2Params
- `EncryptData`/`DecryptData`: O(n), ~1-10ms for KB-sized inputs
- `EncryptStream`/`DecryptStream`: Constant memory usage regardless of file size

---

## Support and Maintenance

- **v1.x branch**: Active development, new features, bug fixes
- **Security updates**: Backported to all supported versions
- **API questions**: GitHub Discussions
- **Bug reports**: GitHub Issues

---

## License

This API contract is part of the ENCX project and subject to the same license terms.
