# ENCX - Enterprise Cryptography for Go

> **Context7 Users**: See [Quick Integration Guide](./docs/CONTEXT7_GUIDE.md) for structured examples and patterns

A production-ready Go library for field-level encryption, hashing, and key management. ENCX provides struct-based cryptographic operations with support for key rotation, multiple KMS backends, and comprehensive testing utilities.

## 🚀 Context7 Quick Start

```go
// Install: go get github.com/hengadev/encx

// Define struct with encryption tags (no companion fields needed)
type User struct {
    Email    string `encx:"encrypt,hash_basic"` // Encrypt + searchable
    Password string `encx:"hash_secure"`        // Secure password hash
}

// Generate code using one of 3 options:
// 1. go run ./cmd/encx-gen generate .
// 2. Build first: go build -o bin/encx-gen ./cmd/encx-gen && ./bin/encx-gen generate .
// 3. Add: //go:generate go run ../../cmd/encx-gen generate .  (path must be relative)

// Use generated functions for type-safe encryption
crypto, _ := encx.NewTestCrypto(nil)
user := &User{Email: "user@example.com", Password: "secret123"}

// Process returns separate struct with encrypted/hashed fields
// Note: Function name follows pattern Process<YourStructName>Encx
// For a User struct, it generates ProcessUserEncx
userEncx, err := ProcessUserEncx(ctx, crypto, user)

// userEncx.EmailEncrypted contains encrypted email
// userEncx.EmailHash contains searchable hash
// userEncx.PasswordHashSecure contains secure hash

// Decrypt when needed
// Note: Function name follows pattern Decrypt<YourStructName>Encx
decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
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

// Define your struct with encx tags (no companion fields needed)
type User struct {
    Name     string `encx:"encrypt"`
    Email    string `encx:"hash_basic"`
    Password string `encx:"hash_secure"`
}

// Run code generation using one of 3 options:
// 1. go run ./cmd/encx-gen generate .
// 2. Build first: go build -o bin/encx-gen ./cmd/encx-gen && ./bin/encx-gen generate .
// 3. Add: //go:generate go run ../../cmd/encx-gen generate .  (path must be relative)

func main() {
    ctx := context.Background()
    crypto, _ := encx.NewTestCrypto(nil)

    // Create user with sensitive data
    user := &User{
        Name:     "John Doe",
        Email:    "john@example.com",
        Password: "secret123",
    }

    // Process returns encrypted struct (generated function)
    // Note: For your struct, replace "User" with your actual struct name
    // Example: ProcessCustomerEncx, ProcessOrderEncx, etc.
    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    if err != nil {
        log.Fatal(err)
    }

    // Store encrypted data in database
    fmt.Printf("NameEncrypted: %d bytes\n", len(userEncx.NameEncrypted))
    fmt.Printf("EmailHash: %s\n", userEncx.EmailHash[:16]+"...")
    fmt.Printf("PasswordHashSecure: %s...\n", userEncx.PasswordHashSecure[:20]+"...")

    // Decrypt when needed (generated function)
    decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Decrypted Name: %s\n", decryptedUser.Name)
}
```

## Struct Tags Reference

### Single Operation Tags

- `encx:"encrypt"` - Encrypts field value
- `encx:"hash_basic"` - Creates SHA-256 hash for searchable indexing
- `encx:"hash_secure"` - Creates Argon2id hash with pepper (for passwords)

### Combined Operation Tags

- `encx:"encrypt,hash_basic"` - Both encrypts AND hashes the field (searchable encryption)
- `encx:"hash_secure,encrypt"` - Secure hash for auth + encryption for recovery

### How It Works

encx-gen discovers structs automatically by parsing Go source files. No special directives required.

**Code Generation Methods:**

## Code Generation: 3 Working Options

### Option 1: Direct Commands (RECOMMENDED)
```bash
# Build the tool once
go build -o bin/encx-gen ./cmd/encx-gen

# Generate code for current directory and all subdirectories recursively
./bin/encx-gen generate .

# Generate code for specific packages
./bin/encx-gen generate ./path/to/package
```

### Option 2: Go Run (RELIABLE)
```bash
# No building needed - runs directly from source
go run ./cmd/encx-gen generate ./path/to/package

# Generate code for current directory and all subdirectories recursively
go run ./cmd/encx-gen generate .

# Works from any directory with correct relative path
go run ../../cmd/encx-gen generate .
```

### Option 3: Go Generate with Go Run (ADVANCED)
Add this to your Go source file (path must be relative to your file):
```go
//go:generate go run ../../cmd/encx-gen generate .
```

Then run: `go generate ./...`

**Note**: When using `encx-gen generate .`, the tool automatically discovers all Go packages in subdirectories recursively, making it ideal for processing entire projects from the root directory.

**⚠️ Note:** Option 3 requires the correct relative path to `cmd/encx-gen`. Options 1 & 2 work consistently in all environments.

When you define a struct with encx tags:

```go
// Your source struct - clean and simple
type User struct {
    Email string `encx:"encrypt,hash_basic"`
}

// encx-gen automatically generates a UserEncx struct
type UserEncx struct {
    EmailEncrypted []byte   // Encrypted email data
    EmailHash      string   // Searchable hash
    DEKEncrypted   []byte   // Encrypted data encryption key
    KeyVersion     int      // Key version for rotation
    Metadata       string   // Serialization metadata
}

// And generates these functions:
// - ProcessUserEncx(ctx, crypto, user) (*UserEncx, error)
// - DecryptUserEncx(ctx, crypto, userEncx) (*User, error)
//
// Note: Function names follow the pattern Process<StructName>Encx
// Replace "User" with your actual struct name
```

## Advanced Examples

### Combined Tags for Email (Searchable Encryption)

Perfect for user lookup + privacy using code generation:

```go
// Source struct - clean definition
type User struct {
    Email string `encx:"encrypt,hash_basic"`
}

// Run: go run ./cmd/encx-gen generate .

// Usage
user := &User{Email: "user@example.com"}
userEncx, err := ProcessUserEncx(ctx, crypto, user)

// Generated UserEncx has:
// - EmailEncrypted []byte  // For secure storage
// - EmailHash      string  // For fast user lookups

// Database search example
db.Where("email_hash = ?", userEncx.EmailHash).First(&foundUser)

// Decrypt when needed
decrypted, _ := DecryptUserEncx(ctx, crypto, foundUser)
fmt.Println(decrypted.Email) // "user@example.com"
```

### Password with Recovery

Secure authentication + recovery capability:

```go
type User struct {
    Password string `encx:"hash_secure,encrypt"`
}

// Run: go run ./cmd/encx-gen generate .

// Example: Registration
// (Replace "User" with your actual struct name)
user := &User{Password: "secret123"}
userEncx, _ := ProcessUserEncx(ctx, crypto, user)

// Generated UserEncx has:
// - PasswordHashSecure string // For authentication (Argon2id)
// - PasswordEncrypted  []byte // For recovery scenarios

// Login verification
isValid := crypto.CompareSecureHashAndValue(ctx, inputPassword, userEncx.PasswordHashSecure)

// Password recovery (admin function)
recovered, _ := DecryptUserEncx(ctx, crypto, userEncx)
fmt.Println(recovered.Password) // Original password temporarily available
```

### Embedded Structs

Code generation handles embedded structs automatically:

```go
//go:generate go run ../../cmd/encx-gen generate .

type Address struct {
    Street string `encx:"encrypt"`
    City   string `encx:"hash_basic"`
}

type User struct {
    Name    string  `encx:"encrypt"`
    Address Address // Embedded struct, automatically processed
}

// Example: Usage
// (Replace "User" with your actual struct name)
user := &User{
    Name: "John Doe",
    Address: Address{
        Street: "123 Main St",
        City:   "Springfield",
    },
}

userEncx, _ := ProcessUserEncx(ctx, crypto, user)
// Generated UserEncx includes all encrypted/hashed fields from embedded struct
```

## Configuration

### Production Setup

```go
// With AWS KMS
pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(awsKMS),
    encx.WithKEKAlias("alias/myapp-kek"),
    encx.WithPepper(pepper),
)

// With HashiCorp Vault
pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(vaultKMS),
    encx.WithKEKAlias("transit/keys/myapp-kek"),
    encx.WithPepper(pepper),
)
```

### Testing Setup

```go
// For unit tests with code generation
func TestUserEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)

    user := &User{Name: "Test User"}
    userEncx, err := ProcessUserEncx(ctx, crypto, user)

    assert.NoError(t, err)
    assert.NotEmpty(t, userEncx.NameEncrypted)
}

// For integration tests
func TestUserEncryptionIntegration(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
        Pepper: []byte("test-pepper-exactly-32-bytes!!"),
    })

    // Test full encrypt/decrypt cycle
    user := &User{Name: "Integration Test"}
    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    assert.NoError(t, err)

    decrypted, err := DecryptUserEncx(ctx, crypto, userEncx)
    assert.NoError(t, err)
    assert.Equal(t, "Integration Test", decrypted.Name)
}
```

### How It Works

ENCX uses a custom compact binary serializer that provides deterministic encryption with minimal overhead. The serialization is handled automatically by the generated code - you don't need to configure anything.

## Validation

Validate your struct tags before generating code:

```bash
# Validate all Go files in current directory
encx-gen validate -v .

# Validate specific packages
encx-gen validate -v ./models ./api

# Validation is automatically run before generation
go run ./cmd/encx-gen generate -v .
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
oldUserEncx, _ := ProcessUserEncx(ctx, crypto, oldUser) // Uses current key (v1)

// Rotate key
crypto.RotateKEK(ctx)

// New user encrypted with key version 2
newUser := &User{Name: "Bob"}
newUserEncx, _ := ProcessUserEncx(ctx, crypto, newUser) // Uses current key (v2)

// Both can be decrypted regardless of key version
DecryptUserEncx(ctx, crypto, oldUserEncx) // Automatically uses key v1
DecryptUserEncx(ctx, crypto, newUserEncx) // Automatically uses key v2
```

## KMS Providers

### AWS KMS

```go
import "github.com/hengadev/encx/providers/aws"

kmsService, err := aws.NewKMSService(ctx, aws.Config{
    Region: "us-east-1",
})
if err != nil {
    log.Fatal(err)
}

pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsService),
    encx.WithKEKAlias("alias/my-encryption-key"),
    encx.WithPepper(pepper),
)
```

**[→ Full AWS KMS Documentation](./providers/awskms/README.md)**

### HashiCorp Vault

```go
import "github.com/hengadev/encx/providers/hashicorp"

vaultClient, err := vault.NewClient(&vault.Config{
    Address: "https://vault.example.com",
})
if err != nil {
    log.Fatal(err)
}

kmsService, err := hashicorp.NewTransitServiceKMSService(vaultClient)
if err != nil {
    log.Fatal(err)
}

pepper := []byte("your-32-byte-pepper-here!!!!")
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsService),
    encx.WithKEKAlias("transit/keys/my-key"),
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
    "github.com/hengadev/encx/providers/aws"
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
userEncx, err := ProcessUserEncx(ctx, crypto, user)
if err != nil {
    // Log the error for debugging
    log.Printf("Encryption failed: %v", err)

    // Return user-friendly error
    return nil, fmt.Errorf("failed to process user data: %w", err)
}
```

### 3. Use Go Generate in Development

```go
//go:generate encx-gen validate -v .
//go:generate go run ./cmd/encx-gen generate -v .

// Run validation and generation during development
// Run: go generate ./...
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
// Unit testing with generated code
func TestUserEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)

    user := &User{
        Email:    "test@example.com",
        Password: "secret123",
    }

    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    assert.NoError(t, err)
    assert.NotEmpty(t, userEncx.EmailEncrypted)
    assert.NotEmpty(t, userEncx.EmailHash)
}

// Integration testing with full cycle
func TestUserEncryptDecryptCycle(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)

    original := &User{Email: "integration@test.com"}
    userEncx, err := ProcessUserEncx(ctx, crypto, original)
    assert.NoError(t, err)

    decrypted, err := DecryptUserEncx(ctx, crypto, userEncx)
    assert.NoError(t, err)
    assert.Equal(t, original.Email, decrypted.Email)
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
- **[Integration Guide](./docs/INTEGRATION_GUIDE.md)** - Step-by-step integration into your codebase
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

