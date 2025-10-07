# Encx for Context7: Quick Integration Guide

> **Library**: `github.com/hengadev/encx`
> **Type**: Go field-level encryption and hashing library
> **Complexity**: Intermediate
> **Use Cases**: Data encryption, PII protection, secure hashing, key management

## What is Encx?

Encx is a production-ready Go library that provides **field-level encryption** and **secure hashing** using simple struct tags. It automatically handles key management, supports multiple KMS backends, and offers both reflection-based and high-performance code generation approaches.

## Quick Start (30 seconds)

```go
// 1. Install
go get github.com/hengadev/encx

// 2. Define struct with encx tags
type User struct {
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte // Generated automatically
    EmailHash         string // For fast lookups

    // Required fields
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

// 3. Process data
crypto, _ := encx.NewTestCrypto(nil)
user := &User{Email: "user@example.com"}
err := crypto.ProcessStruct(ctx, user)
// user.Email is now encrypted and hashed
```

## Use Case Categories

### 🔐 Data Protection & Privacy
- **User PII encryption**: Names, addresses, phone numbers
- **Financial data**: Account numbers, SSNs, payment info
- **Healthcare data**: Patient records, medical IDs
- **Legal compliance**: GDPR, HIPAA, SOX requirements

### 🔍 Searchable Encryption
- **User lookup by email**: Encrypt for privacy + hash for search
- **Customer search**: Find users while protecting sensitive data
- **Audit trails**: Searchable logs with encrypted details

### 🔑 Authentication & Passwords
- **Password hashing**: Argon2id with automatic salt/pepper
- **Password recovery**: Secure hash + encrypted backup
- **Multi-factor auth**: Encrypted backup codes

### 📊 Analytics & Reporting
- **Pseudonymization**: Hash IDs for analytics
- **Data masking**: Show partial data in reports
- **Compliance reporting**: Decrypt only when needed

## Implementation Patterns

### Pattern 1: Basic Field Encryption
**When to use**: Simple data protection, no search needed

```go
type Document struct {
    Content           string `encx:"encrypt"`
    ContentEncrypted  []byte

    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}
```

### Pattern 2: Searchable Fields
**When to use**: Need to find records by encrypted field

```go
type User struct {
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte // Store securely
    EmailHash         string // Search index

    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

// Search by hash
users := db.Where("email_hash = ?", user.EmailHash).Find(&users)
```

### Pattern 3: Password Management
**When to use**: Authentication with recovery capability

```go
type Account struct {
    Password          string `encx:"hash_secure,encrypt"`
    PasswordHash      string // For login verification
    PasswordEncrypted []byte // For recovery scenarios

    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

// Verify password
isValid := crypto.CompareSecureHashAndValue(ctx, inputPassword, account.PasswordHash)

// Recover password (admin function)
crypto.DecryptStruct(ctx, account)
recoveredPassword := account.Password
```

## Code Generation (Recommended)

For **high performance** and **type safety**, use code generation:

```bash
# 1. Install CLI
make build-cli && make install-cli

# 2. Generate code
encx-gen generate .

# 3. Use generated functions
userEncx, err := ProcessUserEncx(ctx, crypto, user)     // Encrypt
user, err := DecryptUserEncx(ctx, crypto, userEncx)     // Decrypt
```

**Benefits**:
- 10x faster than reflection
- Compile-time type safety
- Better IDE support

## Configuration Examples

### Development/Testing
```go
crypto, _ := encx.NewTestCrypto(nil)
```

### Production with AWS KMS
```go
crypto, err := encx.New(ctx,
    encx.WithKMSService(awsKMS),
    encx.WithDatabase(db),
    encx.WithPepper(pepper),
    encx.WithKEKAlias("myapp-kek"),
)
```

### Production with HashiCorp Vault
```go
import "github.com/hengadev/encx/providers/vault"

kms, err := vault.NewKMSService(client)
crypto, err := encx.New(ctx, encx.WithKMSService(kms))
```

## Database Integration

### Schema Design
```sql
-- PostgreSQL example
CREATE TABLE users (
    id SERIAL PRIMARY KEY,

    -- Encrypted columns
    email_encrypted BYTEA,
    name_encrypted BYTEA,

    -- Hash columns for search
    email_hash VARCHAR(64) UNIQUE,

    -- Essential encryption fields
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- Search index
CREATE INDEX idx_users_email_hash ON users (email_hash);
```

### Database Operations
```go
// Insert encrypted user
userEncx, _ := ProcessUserEncx(ctx, crypto, user)
db.Create(userEncx)

// Search by email hash
var users []UserEncx
db.Where("email_hash = ?", userEncx.EmailHash).Find(&users)

// Decrypt for display
user, _ := DecryptUserEncx(ctx, crypto, userEncx)
```

## Common Recipes

### Recipe 1: User Registration & Login
```go
// Registration
func RegisterUser(email, password string) error {
    user := &User{
        Email:    email,
        Password: password,
    }

    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    if err != nil {
        return err
    }

    return db.Create(userEncx).Error
}

// Login
func LoginUser(email, password string) (*User, error) {
    // Find by email hash
    tempUser := &User{Email: email}
    tempEncx, _ := ProcessUserEncx(ctx, crypto, tempUser)

    var userEncx UserEncx
    err := db.Where("email_hash = ?", tempEncx.EmailHash).First(&userEncx).Error
    if err != nil {
        return nil, err
    }

    // Verify password
    user, _ := DecryptUserEncx(ctx, crypto, &userEncx)
    if !crypto.CompareSecureHashAndValue(ctx, password, userEncx.PasswordHash) {
        return nil, errors.New("invalid password")
    }

    return user, nil
}
```

### Recipe 2: Data Export with Decryption
```go
func ExportUserData(userID int) (*UserProfile, error) {
    var userEncx UserEncx
    db.First(&userEncx, userID)

    // Decrypt for export
    user, err := DecryptUserEncx(ctx, crypto, &userEncx)
    if err != nil {
        return nil, err
    }

    return &UserProfile{
        Email:    user.Email,
        Name:     user.Name,
        Phone:    user.Phone,
    }, nil
}
```

### Recipe 3: Bulk Processing
```go
func ProcessUsersBatch(users []*User) ([]*UserEncx, error) {
    var results []*UserEncx

    for _, user := range users {
        userEncx, err := ProcessUserEncx(ctx, crypto, user)
        if err != nil {
            return nil, fmt.Errorf("failed to process user %s: %w", user.Email, err)
        }
        results = append(results, userEncx)
    }

    return results, nil
}
```

## Performance Guidelines

### Optimization Tips
1. **Use code generation** for production (10x faster)
2. **Batch operations** when processing multiple records
3. **Cache crypto instances** - don't recreate for each operation
4. **Use proper indexing** on hash fields for searches
5. **Monitor KMS calls** - they can be expensive

### Benchmarks
```go
// Code generation vs reflection (typical results)
BenchmarkCodeGeneration-8    	1000000	    1200 ns/op
BenchmarkReflection-8        	 100000	   12000 ns/op  // 10x slower
```

## Error Handling

### Structured Error Handling
```go
if err := crypto.ProcessStruct(ctx, user); err != nil {
    switch {
    case encx.IsRetryableError(err):
        // KMS temporarily unavailable - retry with backoff
        return handleRetry(err)

    case encx.IsAuthError(err):
        // Authentication failed - check credentials
        return handleAuthError(err)

    case encx.IsValidationError(err):
        // Invalid struct definition
        return handleValidationError(err)

    default:
        return fmt.Errorf("encryption failed: %w", err)
    }
}
```

### Common Errors & Solutions
| Error | Cause | Solution |
|-------|-------|----------|
| Missing companion field | `encx:"encrypt"` without `FieldEncrypted []byte` | Add companion field |
| Invalid tag combination | `encx:"hash_basic,hash_secure"` | Use only one hash type |
| KMS unavailable | Network/auth issues | Implement retry logic |
| Invalid struct | Missing required fields | Add DEK, DEKEncrypted, KeyVersion |

## Testing Strategies

### Unit Testing
```go
func TestUserEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)

    user := &User{Email: "test@example.com"}
    userEncx, err := ProcessUserEncx(ctx, crypto, user)

    assert.NoError(t, err)
    assert.NotEmpty(t, userEncx.EmailEncrypted)
    assert.NotEmpty(t, userEncx.EmailHash)
}
```

### Integration Testing
```go
func TestEndToEndEncryption(t *testing.T) {
    // Test with real crypto operations
    crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
        Pepper: []byte("test-pepper-exactly-32-bytes!!"),
    })

    // Test full encrypt/decrypt cycle
    original := &User{Email: "integration@test.com"}
    userEncx, _ := ProcessUserEncx(ctx, crypto, original)
    decrypted, _ := DecryptUserEncx(ctx, crypto, userEncx)

    assert.Equal(t, original.Email, decrypted.Email)
}
```

## Migration & Upgrades

### From Basic to Code Generation
```bash
# 1. Add go:generate directive
//go:generate encx-gen generate .

# 2. Generate code
go generate

# 3. Update function calls
// Old: crypto.ProcessStruct(ctx, user)
// New: ProcessUserEncx(ctx, crypto, user)
```

### Key Rotation
```go
// Rotate encryption keys
if err := crypto.RotateKEK(ctx); err != nil {
    log.Printf("Key rotation failed: %v", err)
}

// Old data can still be decrypted
// New operations use new key version
```

## Security Best Practices

### 1. Key Management
- Store KEK and pepper separately from database
- Use proper KMS services in production
- Implement regular key rotation (30-90 days)
- Monitor key usage and access

### 2. Data Handling
- Clear sensitive data from memory after use
- Use HTTPS for all data transmission
- Implement proper access controls
- Audit all encryption/decryption operations

### 3. Database Security
- Encrypt database at rest
- Use TLS for database connections
- Implement proper backup encryption
- Regular security audits

## Resources & References

### Documentation
- **[Main README](../README.md)** - Complete getting started guide
- **[Code Generation Guide](./CODE_GENERATION_GUIDE.md)** - Performance optimization
- **[Examples](./EXAMPLES.md)** - Real-world implementations
- **[API Reference](./API_REFERENCE.md)** - Complete API docs

### Examples
- **[Complete Web App](../examples/complete-webapp/)** - Full implementation
- **[Basic Examples](../examples/)** - Simple use cases

### Tools
- **CLI Validation**: `go run github.com/hengadev/encx/cmd/validate-tags -v`
- **Code Generation**: `encx-gen generate .`
- **Runtime Validation**: `encx.ValidateStruct(&yourStruct)`

## Context7 Quick Queries

Use these patterns for Context7 queries:

- **"How to encrypt user email with encx"** → Pattern 2: Searchable Fields
- **"Encx password hashing example"** → Pattern 3: Password Management
- **"Encx database schema setup"** → Database Integration section
- **"Encx performance optimization"** → Code generation + Performance Guidelines
- **"Encx error handling patterns"** → Error Handling section
- **"Encx testing strategies"** → Testing Strategies section

---

**Next Steps**: Start with the Quick Start example, then explore the specific use case pattern that matches your needs.