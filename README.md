# ENCX - Enterprise Cryptography for Go

> **Context7 Users**: See [Quick Integration Guide](./docs/CONTEXT7_GUIDE.md) for structured examples and patterns

A production-ready Go library for field-level encryption, hashing, and key management. ENCX provides struct-based cryptographic operations with support for key rotation, multiple KMS backends, and comprehensive testing utilities.

## 🚀 Context7 Quick Start

```go
// Install: go get github.com/hengadev/encx

// Define struct with encryption tags
type User struct {
    Email             string `encx:"encrypt,hash_basic"` // Encrypt + searchable
    EmailEncrypted    []byte // Auto-populated
    EmailHash         string // For fast lookups

    Password          string `encx:"hash_secure"`       // Secure password hash
    PasswordHash      string // For authentication

    // Required encryption fields
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

// Process (encrypt/hash) user data
crypto, _ := encx.NewTestCrypto(nil)
user := &User{Email: "user@example.com", Password: "secret123"}
err := crypto.ProcessStruct(ctx, user)

// Now: user.EmailEncrypted contains encrypted email
//      user.EmailHash contains searchable hash
//      user.PasswordHash contains secure hash
```

**→ [See all patterns and use cases](./docs/CONTEXT7_GUIDE.md)**

## Features

- **Field-level encryption** with AES-GCM
- **Secure hashing** with Argon2id and basic SHA-256
- **Combined operations** - encrypt AND hash the same field
- **Automatic key management** with DEK/KEK architecture
- **Key rotation** support with version tracking
- **Multiple KMS backends** (AWS KMS, HashiCorp Vault, etc.)
- **Comprehensive testing** utilities and mocks
- **Compile-time validation** for struct tags

## Quick Start

### Installation

```bash
go get github.com/hengadev/encx
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/hengadev/encx"
)

// Define your struct with encx tags
type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte
    Email            string `encx:"hash_basic"`
    EmailHash        string
    Password         string `encx:"hash_secure"`
    PasswordHash     string
    
    // Required fields
    DEK              []byte
    DEKEncrypted     []byte
    KeyVersion       int
}

func main() {
    // Create crypto instance (see Configuration section for production setup)
    crypto, _ := encx.NewTestCrypto(nil)
    
    // Create user with sensitive data
    user := &User{
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "secret123",
    }
    
    // Process the struct (encrypt/hash operations)
    ctx := context.Background()
    if err := crypto.ProcessStruct(ctx, user); err != nil {
        log.Fatal(err)
    }
    
    // Original sensitive fields are now cleared/processed
    fmt.Printf("Name: '%s' (cleared)\n", user.Name)
    fmt.Printf("NameEncrypted: %d bytes\n", len(user.NameEncrypted))
    fmt.Printf("EmailHash: %s\n", user.EmailHash)
    fmt.Printf("PasswordHash: %s\n", user.PasswordHash[:20]+"...")
    
    // Decrypt when needed
    if err := crypto.DecryptStruct(ctx, user); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Decrypted Name: %s\n", user.Name)
}
```

## Struct Tags Reference

### Single Operation Tags

- `encx:"encrypt"` - Encrypts field, stores in companion `*Encrypted []byte` field
- `encx:"hash_basic"` - SHA-256 hash, stores in companion `*Hash string` field  
- `encx:"hash_secure"` - Argon2id hash with pepper, stores in companion `*Hash string` field

### Combined Operation Tags

- `encx:"encrypt,hash_basic"` - Both encrypts AND hashes the field
- `encx:"hash_secure,encrypt"` - Secure hash for auth + encryption for recovery

### Required Struct Fields

Every struct must include these fields:

```go
type YourStruct struct {
    // Your tagged fields...
    
    DEK          []byte  // Data Encryption Key (auto-generated)
    DEKEncrypted []byte  // Encrypted DEK (set automatically)  
    KeyVersion   int     // Key version for rotation (set automatically)
}
```

## Advanced Examples

### Combined Tags for Email

Perfect for user lookup + privacy:

```go
type User struct {
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte // For secure storage
    EmailHash         string // For fast user lookups
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

// Usage
user := &User{Email: "user@example.com"}
crypto.ProcessStruct(ctx, user)

// Now you can:
// 1. Store user.EmailEncrypted securely in database
// 2. Use user.EmailHash for fast user lookups
// 3. Decrypt user.Email when needed for display
```

### Password with Recovery

Secure authentication + recovery capability:

```go
type User struct {
    Password          string `encx:"hash_secure,encrypt"`
    PasswordHash      string // For authentication (Argon2id)
    PasswordEncrypted []byte // For recovery scenarios
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

// Usage for login
func (u *User) CheckPassword(plaintext string) bool {
    return crypto.CompareSecureHashAndValue(ctx, plaintext, u.PasswordHash)
}

// Usage for password recovery
func (u *User) RecoverPassword() string {
    crypto.DecryptStruct(ctx, u)
    return u.Password // Temporarily available for recovery
}
```

### Embedded Structs

ENCX automatically processes embedded structs:

```go
type Address struct {
    Street           string `encx:"encrypt"`
    StreetEncrypted  []byte
    City             string `encx:"hash_basic"`
    CityHash         string
}

type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte
    
    Address          Address // Automatically processed
    
    DEK              []byte
    DEKEncrypted     []byte
    KeyVersion       int
}
```

## Configuration

### Production Setup

```go
// With AWS KMS
pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.New(ctx, awsKMS, "alias/myapp-kek", "",
    encx.WithPepper(pepper),
)

// With HashiCorp Vault
pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.New(ctx, vaultKMS, "transit/keys/myapp-kek", "",
    encx.WithPepper(pepper),
)
```

### Testing Setup

```go
// For unit tests with mocking
func TestUserEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    
    user := &User{Name: "Test User"}
    err := crypto.ProcessStruct(ctx, user)
    assert.NoError(t, err)
    assert.NotEmpty(t, user.NameEncrypted)
}

// For integration tests
func TestUserEncryptionIntegration(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
        Pepper: []byte("test-pepper-exactly-32-bytes!!"),
    })
    
    // Test with real crypto operations
}
```

### Serializer Configuration

ENCX supports multiple serialization methods with both global and per-struct configuration:

#### Global Configuration (encx.yaml)

```yaml
codegen:
  output_dir: "generated"
  default_serializer: json  # Options: json, gob, basic
```

#### Per-Struct Override

Use comment-based configuration to override the global serializer for specific structs:

```go
//encx:options serializer=gob
type HighPerformanceData struct {
    Data     []byte `encx:"encrypt"`
    Metadata string `encx:"hash_basic"`
}

//encx:options serializer=basic
type SimpleConfig struct {
    APIKey  string `encx:"hash_secure"`
    Timeout int    `encx:"encrypt"`
}

// Uses default serializer from encx.yaml
type RegularUser struct {
    Email string `encx:"encrypt,hash_basic"`
}
```

#### Serializer Types

- **JSON** (`json`) - Standard JSON serialization, good compatibility and human-readable
- **GOB** (`gob`) - Go-specific binary format, faster and more compact for Go-to-Go communication
- **Basic** (`basic`) - Direct conversion for primitives with JSON fallback, minimal overhead

Choose based on your performance needs and interoperability requirements.

## Validation

### Compile-time Validation

Use the validation utility to check your structs:

```bash
# Validate all Go files in current directory
go run github.com/hengadev/encx/cmd/validate-tags -v

# Validate specific files
go run github.com/hengadev/encx/cmd/validate-tags -pattern="user*.go"
```

### Runtime Validation

```go
// Validate struct definition before processing
if err := encx.ValidateStruct(&user); err != nil {
    log.Fatalf("Invalid struct: %v", err)
}
```

## Key Management

### Key Rotation

```go
// Rotate the Key Encryption Key (KEK)
if err := crypto.RotateKEK(ctx); err != nil {
    log.Fatalf("Key rotation failed: %v", err)
}

// Data encrypted with old keys can still be decrypted
// New encryptions will use the new key version
```

### Multiple Key Versions

ENCX automatically handles multiple key versions:

```go
// User encrypted with key version 1
oldUser := &User{Name: "Alice"}
crypto.ProcessStruct(ctx, oldUser) // Uses current key (v1)

// Rotate key
crypto.RotateKEK(ctx)

// New user encrypted with key version 2  
newUser := &User{Name: "Bob"}
crypto.ProcessStruct(ctx, newUser) // Uses current key (v2)

// Both can be decrypted
crypto.DecryptStruct(ctx, oldUser) // Uses key v1
crypto.DecryptStruct(ctx, newUser) // Uses key v2
```

## KMS Providers

### AWS KMS

```go
import "github.com/hengadev/encx/providers/awskms"

kmsService, err := awskms.New(ctx, awskms.Config{
    Region: "us-east-1",
})
if err != nil {
    log.Fatal(err)
}

pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.New(ctx, kmsService, "alias/my-encryption-key", "",
    encx.WithPepper(pepper),
)
```

**[→ Full AWS KMS Documentation](./providers/awskms/README.md)**

### HashiCorp Vault

```go
import "github.com/hengadev/encx/providers/hashicorpvault"

vaultClient, err := vault.NewClient(&vault.Config{
    Address: "https://vault.example.com",
})
if err != nil {
    log.Fatal(err)
}

kmsService, err := hashicorpvault.NewKMSService(vaultClient)
if err != nil {
    log.Fatal(err)
}

pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.New(ctx, kmsService, "transit/keys/my-key", "",
    encx.WithPepper(pepper),
)
```

**[→ Full HashiCorp Vault Documentation](./providers/hashicorpvault/README.md)**

## Examples

### S3 Streaming Upload with Encryption

Example showing how to encrypt files on-the-fly and stream them directly to AWS S3:

```go
import (
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/awskms"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Encrypt file and stream to S3
err := crypto.EncryptStream(ctx, fileReader, s3Writer, dek)
```

**[→ Full S3 Streaming Example](./examples/s3-streaming-upload/README.md)**

Features:
- Zero-copy streaming encryption
- Minimal memory usage (4KB buffer)
- Production-ready with error handling
- Includes HTTP upload server example

## Best Practices

### 1. Use Combined Tags Strategically

```go
// Good: Email needs both lookup and privacy
Email string `encx:"encrypt,hash_basic"`

// Good: Password needs auth and recovery
Password string `encx:"hash_secure,encrypt"`

// Avoid: Unnecessary combinations
InternalID string `encx:"encrypt,hash_basic,hash_secure"` // Too much
```

### 2. Proper Error Handling

```go
if err := crypto.ProcessStruct(ctx, user); err != nil {
    // Log the error for debugging
    log.Printf("Encryption failed: %v", err)
    
    // Return user-friendly error
    return fmt.Errorf("failed to process user data")
}
```

### 3. Validate Structs Early

```go
// Validate during development/testing
func init() {
    if err := encx.ValidateStruct(&User{}); err != nil {
        panic(fmt.Sprintf("Invalid User struct: %v", err))
    }
}
```

### 4. Handle Key Rotation Gracefully

```go
// Schedule regular key rotation
go func() {
    ticker := time.NewTicker(30 * 24 * time.Hour) // 30 days
    defer ticker.Stop()
    
    for range ticker.C {
        if err := crypto.RotateKEK(ctx); err != nil {
            log.Printf("Key rotation failed: %v", err)
        }
    }
}()
```

## Performance Considerations

- **Batch Operations**: Process multiple structs in batches when possible
- **Connection Pooling**: Use connection pooling for KMS and database
- **Caching**: Consider caching decrypted DEKs for frequently accessed data
- **Monitoring**: Monitor KMS API calls and database performance

## Security Considerations

- **Pepper Management**: Store pepper securely, separate from database
- **KMS Permissions**: Use least-privilege access for KMS operations  
- **Database Security**: Encrypt database at rest and in transit
- **Memory Management**: Clear sensitive data from memory when possible
- **Audit Logging**: Log all cryptographic operations for compliance

## Testing

The library includes comprehensive testing utilities:

```go
// Unit testing with mocks
func TestUserService(t *testing.T) {
    mockCrypto := encx.NewCryptoServiceMock()
    mockCrypto.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
    
    service := NewUserService(mockCrypto)
    err := service.CreateUser("test@example.com")
    assert.NoError(t, err)
    
    mockCrypto.AssertExpectations(t)
}

// Integration testing
func TestUserServiceIntegration(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    service := NewUserService(crypto)
    
    err := service.CreateUser("test@example.com")
    assert.NoError(t, err)
}
```

## Important: Version Control (.gitignore)

When using the `encx` package, add the following to your `.gitignore`:

```gitignore
.encx/
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Run the validation utility: `go run ./cmd/validate-tags -v`
5. Ensure all tests pass: `go test ./...`
6. Submit a pull request

## 🚧 TODOs

- [ ] implement example for different key management services:
    - [x] HashiCorp Vault
    - [x] AWS KMS
    - [ ] Azure Key Vault
    - [ ] Google Cloud KMS
    - [ ] Thales CipherTrust (formerly Vormetric)
    - [ ] AWS CloudHSM
- [x] explore concurrency for performance improvements
- [x] comprehensive tests
- [x] combined tags support
- [x] compile-time validation
- [x] enhanced error handling
- [x] improved documentation

## License

[Add your license here]

## Support

[Add support information here]

## 📚 Documentation

### Complete Guides
- **[Context7 Integration Guide](./docs/CONTEXT7_GUIDE.md)** - Quick reference for Context7 users
- **[Code Generation Guide](./docs/CODE_GENERATION_GUIDE.md)** - High-performance code generation
- **[Examples Documentation](./docs/EXAMPLES.md)** - Real-world use cases
- **[API Reference](./docs/API_REFERENCE.md)** - Complete API documentation
- **[Migration Guide](./docs/MIGRATION_GUIDE.md)** - Upgrade instructions

### Quick References
- **Use Cases**: Data encryption, PII protection, searchable encryption, password management
- **Performance**: Code generation provides 10x speed improvement over reflection
- **Security**: AES-GCM encryption, Argon2id hashing, automatic key management
- **Integration**: Works with PostgreSQL, SQLite, MySQL, AWS KMS, HashiCorp Vault

## Context7 Quick Queries

For Context7 users, here are optimized query patterns:

### Common Use Cases
| Query Pattern | Documentation Section |
|---------------|----------------------|
| "encrypt user email golang" | [Quick Start](#quick-start) + [Context7 Guide](./docs/CONTEXT7_GUIDE.md#pattern-2-searchable-fields) |
| "password hashing with encryption" | [Advanced Examples](#password-with-recovery) + [Context7 Guide](./docs/CONTEXT7_GUIDE.md#pattern-3-password-management) |
| "database schema for encrypted fields" | [Context7 Guide](./docs/CONTEXT7_GUIDE.md#database-integration) |
| "performance optimization encryption" | [Code Generation Guide](./docs/CODE_GENERATION_GUIDE.md) |
| "struct tag validation" | [Validation](#validation) + [API Reference](./docs/API_REFERENCE.md#validation-api) |

### Implementation Patterns
| Pattern | Use Case | Documentation |
|---------|----------|---------------|
| `encx:"encrypt"` | Simple data protection | [Struct Tags Reference](#struct-tags-reference) |
| `encx:"hash_basic"` | Fast search/lookup | [Quick Start](#quick-start) |
| `encx:"hash_secure"` | Password security | [Advanced Examples](#password-with-recovery) |
| `encx:"encrypt,hash_basic"` | Searchable encryption | [Combined Tags](#combined-operation-tags) |

### Technology Integration
| Technology | Integration Guide |
|------------|------------------|
| PostgreSQL | [Context7 Guide - Database](./docs/CONTEXT7_GUIDE.md#database-integration) |
| AWS KMS | [Configuration](#production-setup) |
| HashiCorp Vault | [KMS Providers](#hashicorp-vault) |
| Docker | [Complete Web App Example](./examples/complete-webapp/README.md#docker-configuration) |
| Per-Struct Serializers | [Serializer Example](./examples/per_struct_serializers.go) |

