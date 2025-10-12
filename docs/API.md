# ENCX API Reference

Complete API documentation for the ENCX library.

## Table of Contents

- [Core Types](#core-types)
- [Main Functions](#main-functions)
- [Crypto Methods](#crypto-methods)
- [Validation Functions](#validation-functions)
- [Testing Utilities](#testing-utilities)
- [Error Types](#error-types)
- [Configuration Options](#configuration-options)

## Core Types

### Crypto

The main struct that provides all cryptographic operations.

```go
type Crypto struct {
    // Internal fields - not directly accessible
}
```

**Thread Safety**: The `Crypto` struct is safe for concurrent use.

### CryptoService Interface

Interface for all cryptographic operations, useful for dependency injection and testing.

```go
type CryptoService interface {
    // Struct operations
    ProcessStruct(ctx context.Context, object any) error
    DecryptStruct(ctx context.Context, object any) error
    
    // Data operations
    GenerateDEK() ([]byte, error)
    EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)
    DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
    
    // DEK operations
    EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error)
    DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error)
    
    // Hashing operations
    HashBasic(ctx context.Context, value []byte) string
    HashSecure(ctx context.Context, value []byte) (string, error)
    CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
    CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
    
    // Key management
    RotateKEK(ctx context.Context) error
    
    // Stream operations
    EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
    DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
    
    // Configuration
    GetPepper() []byte
    GetArgon2Params() *Argon2Params
    GetAlias() string
}
```

### KeyManagementService Interface

Interface for cryptographic operations using cloud KMS providers (AWS KMS, HashiCorp Vault Transit Engine).

```go
type KeyManagementService interface {
    GetKeyID(ctx context.Context, alias string) (string, error)
    CreateKey(ctx context.Context, description string) (string, error)
    EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
    DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}
```

**Implementations**:
- `providers/aws.KMSService` - AWS KMS implementation
- `providers/hashicorp.TransitService` - HashiCorp Vault Transit Engine implementation

### SecretManagementService Interface

Interface for secret storage providers (AWS Secrets Manager, HashiCorp Vault KV).

```go
type SecretManagementService interface {
    StorePepper(ctx context.Context, alias string, pepper []byte) error
    GetPepper(ctx context.Context, alias string) ([]byte, error)
    PepperExists(ctx context.Context, alias string) (bool, error)
    GetStoragePath(alias string) string
}
```

**Implementations**:
- `providers/aws.SecretsManagerStore` - AWS Secrets Manager implementation
- `providers/hashicorp.KVStore` - HashiCorp Vault KV v2 implementation
- `InMemorySecretStore` - In-memory implementation for testing

**Storage Paths**:
- AWS: `encx/{PepperAlias}/pepper`
- Vault: `secret/data/encx/{PepperAlias}/pepper`
- In-memory: `memory://{PepperAlias}/pepper`

### Config Struct

Configuration struct for explicit dependency injection.

```go
type Config struct {
    KEKAlias    string  // Required: KMS key identifier
    PepperAlias string  // Required: Service identifier for pepper storage
    DBPath      string  // Optional: Database directory (default: .encx)
    DBFilename  string  // Optional: Database filename (default: keys.db)
}
```

**Validation**:
- `KEKAlias` must not be empty and must be ≤ 256 characters
- `PepperAlias` must not be empty
- `DBPath` defaults to `.encx` if empty
- `DBFilename` defaults to `keys.db` if empty

**Methods**:
```go
func (c *Config) Validate() error
```

### Argon2Params

Configuration for Argon2id hashing.

```go
type Argon2Params struct {
    Memory      uint32 // Memory in KB
    Iterations  uint32 // Number of iterations
    Parallelism uint8  // Degree of parallelism
    SaltLength  uint32 // Salt length in bytes
    KeyLength   uint32 // Generated key length in bytes
}
```

**Default Values**:
- Memory: 65536 KB (64 MB)
- Iterations: 3
- Parallelism: 4
- SaltLength: 16 bytes
- KeyLength: 32 bytes

## Main Functions

### NewCrypto

Creates a new Crypto instance with explicit dependency injection (low-level API).

```go
func NewCrypto(
    ctx context.Context,
    kms KeyManagementService,
    secrets SecretManagementService,
    cfg Config,
    options ...Option,
) (*Crypto, error)
```

**Parameters**:
- `ctx`: Context for initialization
- `kms`: Key Management Service for cryptographic operations (required)
- `secrets`: Secret Management Service for pepper storage (required)
- `cfg`: Configuration struct with KEKAlias, PepperAlias, etc. (required)
- `options`: Additional configuration options (optional)

**Returns**:
- `*Crypto`: Configured crypto instance
- `error`: Initialization error, if any

**Example**:
```go
// Initialize providers
kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})

// Create explicit configuration
cfg := encx.Config{
    KEKAlias:    "alias/my-encryption-key",
    PepperAlias: "my-app-service",
}

// Initialize crypto
crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
if err != nil {
    log.Fatal(err)
}
```

**Behavior**:
- Validates all required parameters
- Validates Config struct (calls cfg.Validate())
- Checks KMS connectivity by retrieving KEK
- Checks/generates pepper in SecretManagementService
- Initializes key metadata database
- Returns ready-to-use Crypto instance

### NewCryptoFromEnv

Creates a new Crypto instance using environment variables (convenience API for 12-factor apps).

```go
func NewCryptoFromEnv(
    ctx context.Context,
    kms KeyManagementService,
    secrets SecretManagementService,
    options ...Option,
) (*Crypto, error)
```

**Parameters**:
- `ctx`: Context for initialization
- `kms`: Key Management Service (required)
- `secrets`: Secret Management Service (required)
- `options`: Additional configuration options (optional)

**Returns**:
- `*Crypto`: Configured crypto instance
- `error`: Initialization error, if any

**Required Environment Variables**:
- `ENCX_KEK_ALIAS`: KMS key identifier
- `ENCX_PEPPER_ALIAS`: Service identifier for pepper storage

**Optional Environment Variables**:
- `ENCX_DB_PATH`: Database directory (default: `.encx`)
- `ENCX_DB_FILENAME`: Database filename (default: `keys.db`)

**Example**:
```go
// Set environment variables:
// export ENCX_KEK_ALIAS="alias/my-encryption-key"
// export ENCX_PEPPER_ALIAS="my-app-service"

kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})

crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
if err != nil {
    log.Fatal(err)
}
```

### LoadConfigFromEnvironment

Loads configuration from environment variables.

```go
func LoadConfigFromEnvironment() (Config, error)
```

**Returns**:
- `Config`: Loaded configuration
- `error`: Loading/validation error, if any

**Environment Variables**:
- `ENCX_KEK_ALIAS`: KMS key identifier (required)
- `ENCX_PEPPER_ALIAS`: Service identifier (required)
- `ENCX_DB_PATH`: Database directory (optional, default: `.encx`)
- `ENCX_DB_FILENAME`: Database filename (optional, default: `keys.db`)

**Example**:
```go
cfg, err := encx.LoadConfigFromEnvironment()
if err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}

crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
```

### NewTestCrypto

Creates a crypto instance optimized for testing with in-memory storage.

```go
func NewTestCrypto(t testing.TB) *Crypto
```

**Parameters**:
- `t`: Testing interface (can be nil for non-test usage)

**Returns**:
- `*Crypto`: Test crypto instance with mock KMS and in-memory secret store

**Example**:
```go
func TestMyFunction(t *testing.T) {
    crypto := encx.NewTestCrypto(t)

    // Use crypto in tests
    dek, _ := crypto.GenerateDEK()
    encrypted, _ := crypto.EncryptData(ctx, plaintext, dek)
}
```

**Features**:
- Uses `SimpleTestKMS` (mock KMS implementation)
- Uses `InMemorySecretStore` (automatic pepper generation)
- Pre-configured with test-friendly Argon2 parameters
- Automatic cleanup on test completion

## Crypto Methods

### Struct Operations

**ENCX uses code generation for struct processing.** Generated functions provide better performance and type safety.

For a struct named `User`, the code generator creates:

```go
// Generated by encx-gen
func ProcessUserEncx(ctx context.Context, crypto encx.CryptoService, source *User) (*UserEncx, error)
func DecryptUserEncx(ctx context.Context, crypto encx.CryptoService, source *UserEncx) (*User, error)
```

**Pattern**:
- Processing: `Process<StructName>Encx`
- Decryption: `Decrypt<StructName>Encx`

**Example**:
```go
//go:generate encx-gen generate .

type User struct {
    Name  string `encx:"encrypt"`
    Email string `encx:"hash_basic"`
}

// Use generated functions
user := &User{Name: "John", Email: "john@example.com"}
userEncx, err := ProcessUserEncx(ctx, crypto, user)
if err != nil {
    return err
}

// Store userEncx in database...

// Later decrypt
decrypted, err := DecryptUserEncx(ctx, crypto, userEncx)
// decrypted.Name is now available
```

**Recursive Package Discovery**:
When using `encx-gen generate .`, the tool automatically discovers all Go packages in subdirectories recursively, making it ideal for processing entire projects from a single command.

**See**: [Code Generation Guide](./CODE_GENERATION_GUIDE.md) for complete documentation.

### Data Operations

#### GenerateDEK

Generates a new 32-byte Data Encryption Key.

```go
func (c *Crypto) GenerateDEK() ([]byte, error)
```

**Returns**:
- `[]byte`: 32-byte DEK
- `error`: Generation error, if any

#### EncryptData

Encrypts data using AES-GCM with the provided DEK.

```go
func (c *Crypto) EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `plaintext`: Data to encrypt
- `dek`: 32-byte Data Encryption Key

**Returns**:
- `[]byte`: Encrypted data (includes nonce)
- `error`: Encryption error, if any

#### DecryptData

Decrypts data using AES-GCM with the provided DEK.

```go
func (c *Crypto) DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `ciphertext`: Data to decrypt
- `dek`: 32-byte Data Encryption Key

**Returns**:
- `[]byte`: Decrypted data
- `error`: Decryption error, if any

### DEK Operations

#### EncryptDEK

Encrypts a DEK using the current KEK.

```go
func (c *Crypto) EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `plaintextDEK`: DEK to encrypt

**Returns**:
- `[]byte`: Encrypted DEK
- `error`: Encryption error, if any

#### DecryptDEKWithVersion

Decrypts a DEK using a specific KEK version.

```go
func (c *Crypto) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `ciphertextDEK`: Encrypted DEK
- `kekVersion`: KEK version to use for decryption

**Returns**:
- `[]byte`: Decrypted DEK
- `error`: Decryption error, if any

### Hashing Operations

#### HashBasic

Creates a SHA-256 hash of the input.

```go
func (c *Crypto) HashBasic(ctx context.Context, value []byte) string
```

**Parameters**:
- `ctx`: Context for the operation
- `value`: Data to hash

**Returns**:
- `string`: Hex-encoded hash

**Note**: This is a fast, deterministic hash suitable for lookups but not cryptographically secure for passwords.

#### HashSecure

Creates an Argon2id hash of the input with pepper.

```go
func (c *Crypto) HashSecure(ctx context.Context, value []byte) (string, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `value`: Data to hash (typically passwords)

**Returns**:
- `string`: Encoded hash with parameters
- `error`: Hashing error, if any

**Note**: This is suitable for password storage and other security-critical hashing.

#### CompareSecureHashAndValue

Compares a value against an Argon2id hash.

```go
func (c *Crypto) CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `value`: Value to check (will be serialized)
- `hashValue`: Encoded hash to compare against

**Returns**:
- `bool`: True if value matches hash
- `error`: Comparison error, if any

#### CompareBasicHashAndValue

Compares a value against a SHA-256 hash.

```go
func (c *Crypto) CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
```

**Parameters**:
- `ctx`: Context for the operation
- `value`: Value to check (will be serialized)
- `hashValue`: Hash to compare against

**Returns**:
- `bool`: True if value matches hash
- `error`: Comparison error, if any

### Key Management

#### RotateKEK

Rotates the Key Encryption Key, generating a new version.

```go
func (c *Crypto) RotateKEK(ctx context.Context) error
```

**Parameters**:
- `ctx`: Context for the operation

**Returns**:
- `error`: Rotation error, if any

**Behavior**:
- Creates a new KEK version in KMS
- Updates metadata database
- Marks previous version as deprecated
- New encryptions will use the new key version
- Old data can still be decrypted with previous versions

### Stream Operations

#### EncryptStream

Encrypts data from a reader to a writer using streaming AES-GCM.

```go
func (c *Crypto) EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
```

**Parameters**:
- `ctx`: Context for the operation
- `reader`: Source of plaintext data
- `writer`: Destination for encrypted data
- `dek`: 32-byte Data Encryption Key

**Returns**:
- `error`: Streaming error, if any

#### DecryptStream

Decrypts data from a reader to a writer using streaming AES-GCM.

```go
func (c *Crypto) DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
```

**Parameters**:
- `ctx`: Context for the operation
- `reader`: Source of encrypted data
- `writer`: Destination for decrypted data
- `dek`: 32-byte Data Encryption Key

**Returns**:
- `error`: Streaming error, if any

## Validation Functions

### ValidateStruct

Validates a struct for proper encx usage at runtime.

```go
func ValidateStruct(object any) error
```

**Parameters**:
- `object`: Pointer to struct to validate

**Returns**:
- `error`: Validation errors, if any

**Checks**:
- Required fields (DEK, DEKEncrypted, KeyVersion) exist
- Tagged fields have proper companion fields
- Companion fields have correct types
- Tag syntax is valid

**Example**:
```go
if err := encx.ValidateStruct(&user); err != nil {
    log.Fatalf("Invalid struct: %v", err)
}
```

### NewStructTagValidator

Creates a compile-time struct tag validator.

```go
func NewStructTagValidator() *StructTagValidator
```

**Returns**:
- `*StructTagValidator`: Validator instance

**Usage**:
```go
validator := encx.NewStructTagValidator()
err := validator.ValidateSourceFile("user.go")
```

## Testing Utilities

### NewCryptoServiceMock

Creates a mock implementation of CryptoService for testing.

```go
func NewCryptoServiceMock() *CryptoServiceMock
```

**Returns**:
- `*CryptoServiceMock`: Mock instance using testify/mock

**Example**:
```go
mock := encx.NewCryptoServiceMock()
mock.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)

// Use mock in tests
service := NewUserService(mock)
err := service.ProcessUser(user)

mock.AssertExpectations(t)
```

### InMemorySecretStore

In-memory implementation of SecretManagementService for testing.

```go
func NewInMemorySecretStore() *InMemorySecretStore
```

**Returns**:
- `*InMemorySecretStore`: Thread-safe in-memory secret store

**Example**:
```go
// Create in-memory store
secretStore := encx.NewInMemorySecretStore()

// Use with NewCrypto
kms := encx.NewSimpleTestKMS()
cfg := encx.Config{
    KEKAlias:    "test-kek",
    PepperAlias: "test-service",
}

crypto, err := encx.NewCrypto(ctx, kms, secretStore, cfg)
```

**Features**:
- Thread-safe for concurrent testing
- Automatic pepper generation
- Isolated storage per PepperAlias
- Data lost on restart (in-memory only)

**Warning**: Only use for testing. Not suitable for production use.

### SimpleTestKMS

Mock KMS implementation for testing.

```go
func NewSimpleTestKMS() KeyManagementService
```

**Returns**:
- `KeyManagementService`: Mock KMS that simulates cloud KMS behavior

**Example**:
```go
kms := encx.NewSimpleTestKMS()

// Use with NewCrypto
secretStore := encx.NewInMemorySecretStore()
cfg := encx.Config{
    KEKAlias:    "test-kek",
    PepperAlias: "test-service",
}

crypto, err := encx.NewCrypto(ctx, kms, secretStore, cfg)
```

## Error Types

### Base Errors

```go
var (
    ErrUninitializedPepper = errors.New("pepper value appears to be uninitialized (all zeros)")
    ErrMissingField       = errors.New("missing required field")
    ErrMissingTargetField = errors.New("missing required target field")
    ErrInvalidFieldType   = errors.New("invalid field type")
    ErrUnsupportedType    = errors.New("unsupported type")
    ErrTypeConversion     = errors.New("type conversion failed")
    ErrNilPointer         = errors.New("nil pointer encountered")
    ErrOperationFailed    = errors.New("operation failed")
    ErrInvalidFormat      = errors.New("invalid format")
)
```

### Error Helper Functions

```go
func NewMissingFieldError(fieldName string, action Action) error
func NewMissingTargetFieldError(fieldName string, targetFieldName string, action Action) error
func NewInvalidFieldTypeError(fieldName string, expectedType, actualType string, action Action) error
func NewUnsupportedTypeError(fieldName string, typeName string, action Action) error
func NewTypeConversionError(fieldName string, typeName string, action Action) error
func NewNilPointerError(fieldName string, action Action) error
func NewOperationFailedError(fieldName string, action Action, details string) error
func NewInvalidFormatError(fieldName string, formatName string, action Action) error
```

## Configuration Options

### Configuration Approaches

ENCX v0.6.0+ provides two configuration approaches:

#### 1. Explicit Configuration (Recommended for Libraries)

Use explicit dependency injection with the `Config` struct:

```go
kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})

cfg := encx.Config{
    KEKAlias:    "alias/my-encryption-key",
    PepperAlias: "my-app-service",
}

crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
```

**Benefits**:
- Full control over dependencies
- No hidden environment variable dependencies
- Better for library code
- Easier to test with dependency injection

#### 2. Environment-Based Configuration (Recommended for Applications)

Use environment variables with `NewCryptoFromEnv`:

```go
// Set environment:
// export ENCX_KEK_ALIAS="alias/my-encryption-key"
// export ENCX_PEPPER_ALIAS="my-app-service"

kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})

crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
```

**Benefits**:
- 12-factor app compliant
- Environment-specific configuration
- Easier deployment across environments
- No hardcoded configuration values

### Advanced Option Functions

For advanced scenarios, you can pass additional options to `NewCrypto` or `NewCryptoFromEnv`:

#### WithArgon2Params

Sets custom Argon2id parameters for secure hashing.

```go
func WithArgon2Params(params *Argon2Params) Option
```

**Example**:
```go
params := &encx.Argon2Params{
    Memory:      131072, // 128 MB
    Iterations:  4,
    Parallelism: 8,
    SaltLength:  16,
    KeyLength:   32,
}

crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg, encx.WithArgon2Params(params))
```

**Use Cases**:
- Customizing password hashing strength
- Balancing security vs performance
- Meeting specific compliance requirements

#### WithSerializer

Sets a custom serializer for field values.

```go
func WithSerializer(serializer Serializer) Option
```

**Use Cases**:
- Custom encoding formats
- Legacy data format compatibility
- Performance optimization for specific data types

### Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENCX_KEK_ALIAS` | Yes | - | KMS key identifier (e.g., `alias/my-key`) |
| `ENCX_PEPPER_ALIAS` | Yes | - | Service identifier for pepper storage |
| `ENCX_DB_PATH` | No | `.encx` | Database directory path |
| `ENCX_DB_FILENAME` | No | `keys.db` | Database filename |

### Deprecated Options

The following options are **deprecated** in v0.6.0+ and replaced by explicit parameters:

- ~~`WithKMSService(kms)`~~ → Pass `kms` directly to `NewCrypto`
- ~~`WithPepper(pepper)`~~ → Pepper auto-managed via `SecretManagementService`
- ~~`WithKEKAlias(alias)`~~ → Use `Config.KEKAlias` field
- ~~`WithKeyMetadataDB(db)`~~ → Database auto-initialized from `Config.DBPath`

**Migration**: See the [Migration Guide](./MIGRATION_GUIDE.md) for upgrading from v0.5.x to v0.6.0+.

## Constants

### Struct Tags

```go
const (
    StructTag       = "encx"         // The struct tag name
    TagEncrypt      = "encrypt"      // Tag for encryption
    TagHashSecure   = "hash_secure"  // Tag for Argon2id hashing
    TagHashBasic    = "hash_basic"   // Tag for SHA-256 hashing
)
```

### Field Names

```go
const (
    FieldDEK          = "DEK"          // DEK field name
    FieldDEKEncrypted = "DEKEncrypted" // Encrypted DEK field name
    FieldKeyVersion   = "KeyVersion"   // Key version field name
)
```

### Field Suffixes

```go
const (
    SuffixEncrypted = "Encrypted"  // Suffix for encrypted companion fields
    SuffixHashed    = "Hash"       // Suffix for hash companion fields
)
```

## Thread Safety

- The `Crypto` struct is safe for concurrent use across multiple goroutines
- KMS operations are thread-safe (depends on provider implementation)
- Database operations use proper locking and transactions
- Hash operations are stateless and thread-safe

## Performance Considerations

- **DEK Generation**: Fast cryptographically secure random generation
- **AES-GCM Encryption**: Hardware-accelerated on modern CPUs
- **Argon2id Hashing**: CPU and memory intensive, tune parameters for your needs
- **KMS Operations**: Network latency dependent, consider connection pooling
- **Database Operations**: Use connection pooling for better performance
- **Serialization**: JSON serialization overhead for complex types

## Memory Management

- Sensitive data is cleared from memory when possible
- DEKs are not cached by default
- Use `defer` to clear sensitive variables when appropriate
- The library does not prevent memory dumps or swap to disk
