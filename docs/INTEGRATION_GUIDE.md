# ENCX Integration Guide

> **For teams integrating ENCX into their codebase**
>
> This guide provides step-by-step instructions for adding field-level encryption to your Go application.

## Table of Contents

1. [Quick Integration Checklist](#quick-integration-checklist)
2. [Installation & Setup](#installation--setup)
3. [API Reference Quick Links](#api-reference-quick-links)
4. [Integration Patterns](#integration-patterns)
5. [Database Schema Design](#database-schema-design)
6. [Testing Your Integration](#testing-your-integration)
7. [Production Deployment](#production-deployment)

## Quick Integration Checklist

- [ ] Install library: `go get github.com/hengadev/encx`
- [ ] Define structs with encx tags
- [ ] Set up database schema with encrypted columns (automatic in v0.5.2+)
- [ ] Configure crypto instance (KMS + pepper) - automatic KEK initialization in v0.5.3+
- [ ] Generate code (optional, for performance)
- [ ] Implement encryption in your data layer
- [ ] Add tests for encrypt/decrypt cycles
- [ ] Configure production KMS

> **🆕 v0.5.2+**: Database schema creation and KEK initialization are now automatic! No manual setup required.

## Installation & Setup

### 1. Install the Library

```bash
go get github.com/hengadev/encx
```

### 2. Install Code Generator (Optional but Recommended)

```bash
# Clone repo and build CLI
git clone https://github.com/hengadev/encx
cd encx
make build-cli && make install-cli
```

### 3. Initialize Configuration

```bash
# Create encx.yaml in your project root
encx-gen init
```

### 4. Quick Start with Automatic Initialization (v0.5.2+)

**ENCX now automatically handles database and KEK setup!** Just create your crypto instance and it's ready to use:

```go
package main

import (
    "context"
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/aws"
)

func main() {
    ctx := context.Background()

    // Create KMS service (your choice of provider)
    kms, err := aws.NewKMSService(ctx, aws.Config{
        Region: "us-east-1",
    })

    // Create crypto instance - everything else is automatic!
    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(kms),
        encx.WithKEKAlias("my-app-key"),
        encx.WithPepper([]byte("your-32-byte-secret-pepper-key!")),
    )
    if err != nil {
        panic(err)
    }

    // That's it! Ready to encrypt/decrypt
    // No manual database setup required
    // No KEK initialization required
}
```

**What happens automatically:**
1. ✅ Database schema created (kek_versions table + index)
2. ✅ Initial KEK created and stored in database
3. ✅ Configuration validated before any operations
4. ✅ Ready for immediate encryption operations

**Migration from older versions:** See [Migration Guide](./MIGRATION_GUIDE.md) for details.

## API Reference Quick Links

### Core Functions

**Generated code (recommended approach):**
```go
// Pattern: Process<YourStructName>Encx
ProcessStructNameEncx(ctx context.Context, crypto encx.CryptoService, source *StructName) (*StructNameEncx, error)

// Pattern: Decrypt<YourStructName>Encx
DecryptStructNameEncx(ctx context.Context, crypto encx.CryptoService, source *StructNameEncx) (*StructName, error)
```

**Note:** ENCX uses code generation to create type-safe encryption functions. Replace `StructName` with your actual struct name (e.g., `ProcessUserEncx`, `ProcessOrderEncx`).

### Configuration Functions

```go
// Production setup
encx.NewCrypto(ctx context.Context, opts ...CryptoOption) (*Crypto, error)

// Testing setup
encx.NewTestCrypto(t *testing.T, opts ...*TestCryptoOptions) (*Crypto, error)

// Options
encx.WithKMSService(kms KMSService)
encx.WithPepper(pepper []byte)
encx.WithKEKAlias(alias string)
```

### Validation Functions

```bash
# CLI validation
encx-gen validate -v ./path/to/package

# Runtime validation
encx.ValidateStruct(structPtr interface{}) error
```

See [API_REFERENCE.md](./API_REFERENCE.md) for complete API documentation.

## Integration Patterns

### Pattern 1: Basic User Model

**Step 1: Define Your Struct**

```go
package models

type User struct {
    ID        int    `json:"id" db:"id"`
    Email     string `json:"email" encx:"encrypt,hash_basic"`
    Name      string `json:"name" encx:"encrypt"`
    CreatedAt int64  `json:"created_at" db:"created_at"`
}
```

**Step 2: Generate Code**

**Method A: Direct command (recommended)**
```bash
encx-gen generate ./models
```

**Method B: Using go generate (optional)**
```bash
# First add the generate directive to your file:
//go:generate encx-gen generate .

# Then run:
go generate ./models
```

This creates `models/user_encx.go` with:
```go
type UserEncx struct {
    ID             int    `json:"id" db:"id"`
    EmailEncrypted []byte `json:"email_encrypted" db:"email_encrypted"`
    EmailHash      string `json:"email_hash" db:"email_hash"`
    NameEncrypted  []byte `json:"name_encrypted" db:"name_encrypted"`
    DEKEncrypted   []byte `json:"dek_encrypted" db:"dek_encrypted"`
    KeyVersion     int    `json:"key_version" db:"key_version"`
    Metadata       string `json:"metadata" db:"metadata"`
    CreatedAt      int64  `json:"created_at" db:"created_at"`
}
```

**Step 3: Use in Your Code**

```go
// Create user
func CreateUser(ctx context.Context, email, name string) error {
    user := &User{
        Email: email,
        Name:  name,
        CreatedAt: time.Now().Unix(),
    }

    // Encrypt
    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    if err != nil {
        return fmt.Errorf("encryption failed: %w", err)
    }

    // Save to database
    return db.Create(userEncx).Error
}

// Find user by email
func FindUserByEmail(ctx context.Context, email string) (*User, error) {
    // Create hash for search
    tempUser := &User{Email: email}
    tempEncx, _ := ProcessUserEncx(ctx, crypto, tempUser)

    // Search by hash
    var userEncx UserEncx
    err := db.Where("email_hash = ?", tempEncx.EmailHash).First(&userEncx).Error
    if err != nil {
        return nil, err
    }

    // Decrypt
    return DecryptUserEncx(ctx, crypto, &userEncx)
}
```

### Pattern 2: Password Authentication

```go
package models

type Account struct {
    ID       int    `json:"id" db:"id"`
    Username string `json:"username" db:"username"`
    Password string `json:"-" encx:"hash_secure,encrypt"`
}

// Generate code: encx-gen generate .
```

**Usage:**

```go
// Registration
func Register(ctx context.Context, username, password string) error {
    account := &Account{
        Username: username,
        Password: password,
    }

    accountEncx, err := ProcessAccountEncx(ctx, crypto, account)
    if err != nil {
        return err
    }

    return db.Create(accountEncx).Error
}

// Login
func Login(ctx context.Context, username, password string) (*Account, error) {
    var accountEncx AccountEncx
    err := db.Where("username = ?", username).First(&accountEncx).Error
    if err != nil {
        return nil, err
    }

    // Verify password using secure hash
    isValid := crypto.CompareSecureHashAndValue(ctx, password, accountEncx.PasswordHash)
    if !isValid {
        return nil, errors.New("invalid credentials")
    }

    return DecryptAccountEncx(ctx, crypto, &accountEncx)
}
```

### Pattern 3: Embedded Structs

```go
type Address struct {
    Street string `encx:"encrypt"`
    City   string `encx:"encrypt"`
    Zip    string `encx:"hash_basic"`
}

type Customer struct {
    ID      int     `json:"id" db:"id"`
    Name    string  `json:"name" encx:"encrypt"`
    Address Address `json:"address"` // Automatically processed
}

// Generate code: encx-gen generate .
```

## Database Schema Design

### PostgreSQL Schema

```sql
-- User table with encrypted fields
CREATE TABLE users (
    id SERIAL PRIMARY KEY,

    -- Encrypted fields (store as BYTEA)
    email_encrypted BYTEA,
    name_encrypted BYTEA,

    -- Hash fields for searching
    email_hash VARCHAR(64) UNIQUE NOT NULL,

    -- Required encryption metadata
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    metadata JSONB DEFAULT '{}',

    -- Regular fields
    created_at BIGINT NOT NULL,
    updated_at BIGINT
);

-- Indexes for performance
CREATE INDEX idx_users_email_hash ON users (email_hash);
CREATE INDEX idx_users_key_version ON users (key_version);
```

### MySQL Schema

```sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,

    -- Encrypted fields (store as BLOB)
    email_encrypted BLOB,
    name_encrypted BLOB,

    -- Hash fields
    email_hash VARCHAR(64) UNIQUE NOT NULL,

    -- Encryption metadata
    dek_encrypted BLOB NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    metadata JSON,

    created_at BIGINT NOT NULL,

    INDEX idx_email_hash (email_hash),
    INDEX idx_key_version (key_version)
);
```

### SQLite Schema

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_encrypted BLOB,
    name_encrypted BLOB,
    email_hash TEXT UNIQUE NOT NULL,
    dek_encrypted BLOB NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    metadata TEXT,
    created_at INTEGER NOT NULL
);

CREATE INDEX idx_users_email_hash ON users (email_hash);
```

## Testing Your Integration

### Unit Tests

```go
func TestUserEncryption(t *testing.T) {
    // Setup test crypto
    crypto, err := encx.NewTestCrypto(t)
    require.NoError(t, err)

    // Test encryption
    user := &User{
        Email: "test@example.com",
        Name:  "Test User",
    }

    userEncx, err := ProcessUserEncx(context.Background(), crypto, user)
    require.NoError(t, err)
    assert.NotEmpty(t, userEncx.EmailEncrypted)
    assert.NotEmpty(t, userEncx.EmailHash)
    assert.NotEmpty(t, userEncx.NameEncrypted)

    // Test decryption
    decrypted, err := DecryptUserEncx(context.Background(), crypto, userEncx)
    require.NoError(t, err)
    assert.Equal(t, user.Email, decrypted.Email)
    assert.Equal(t, user.Name, decrypted.Name)
}
```

### Integration Tests

```go
func TestUserRepository(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    crypto, _ := encx.NewTestCrypto(t)
    repo := NewUserRepository(db, crypto)

    // Test create
    user, err := repo.Create(context.Background(), "test@example.com", "John Doe")
    require.NoError(t, err)

    // Test find by email
    found, err := repo.FindByEmail(context.Background(), "test@example.com")
    require.NoError(t, err)
    assert.Equal(t, user.ID, found.ID)
    assert.Equal(t, "John Doe", found.Name)
}
```

## Production Deployment

### 1. Configure KMS

**AWS KMS:**
```go
import "github.com/hengadev/encx/providers/aws"

kmsService, err := aws.NewKMSService(ctx, aws.Config{
    Region: "us-east-1",
})

crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsService),
    encx.WithKEKAlias("alias/myapp-master-key"),
    encx.WithPepper([]byte(os.Getenv("ENCRYPTION_PEPPER"))),
)
```

**HashiCorp Vault:**
```go
import "github.com/hengadev/encx/providers/hashicorp"

vaultClient, _ := vault.NewClient(&vault.Config{
    Address: os.Getenv("VAULT_ADDR"),
})

kmsService, _ := hashicorp.NewTransitService(vaultClient)

crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsService),
    encx.WithKEKAlias("transit/keys/app-encryption-key"),
    encx.WithPepper([]byte(os.Getenv("ENCRYPTION_PEPPER"))),
)
```

### 2. Environment Variables

```bash
# .env
ENCRYPTION_PEPPER="your-32-byte-pepper-secret-here"
KEK_ALIAS="alias/production-master-key"
AWS_REGION="us-east-1"

# For Vault
VAULT_ADDR="https://vault.production.com"
VAULT_TOKEN="your-vault-token"
```

### 3. Monitoring & Logging

```go
// Add metrics
import "github.com/prometheus/client_golang/prometheus"

encryptionDuration := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "encx_encryption_duration_seconds",
        Help: "Time taken to encrypt data",
    },
    []string{"struct_type"},
)

// Use with middleware
func (r *Repository) Create(ctx context.Context, user *User) error {
    start := time.Now()
    defer func() {
        encryptionDuration.WithLabelValues("user").Observe(time.Since(start).Seconds())
    }()

    userEncx, err := ProcessUserEncx(ctx, r.crypto, user)
    if err != nil {
        log.Printf("Encryption failed for user: %v", err)
        return err
    }

    return r.db.Create(userEncx).Error
}
```

### 4. Key Rotation Strategy

```go
// Scheduled key rotation
func StartKeyRotation(crypto encx.CryptoService) {
    ticker := time.NewTicker(30 * 24 * time.Hour) // Rotate every 30 days
    defer ticker.Stop()

    for range ticker.C {
        if err := crypto.RotateKEK(context.Background()); err != nil {
            log.Printf("Key rotation failed: %v", err)
            // Alert operations team
        } else {
            log.Println("Key rotation completed successfully")
        }
    }
}
```

## Common Integration Issues

### ✅ RESOLVED: Database Schema Issues (v0.5.2+)
**Previous Error:** `no such table: kek_versions`

**Status:** **RESOLVED** - Database schema is now created automatically in v0.5.2+

**Previous Solution:** (No longer needed)
```sql
-- This manual table creation is no longer required
CREATE TABLE kek_versions (...);
```

### ✅ RESOLVED: KEK Initialization Issues (v0.5.3+)
**Previous Error:** `failed to get KMS Key ID for alias 'xxx' version 0`

**Status:** **RESOLVED** - KEK initialization is now automatic in v0.5.3+

**Previous Solution:** (No longer needed)
```go
// This manual KEK initialization is no longer needed
crypto.keyRotationOps.EnsureInitialKEK(ctx, crypto)
```

### Issue 1: Missing Companion Fields
**Error:** `missing companion field EmailEncrypted for encrypt tag`

**Solution:** Use code generation instead of manual struct definition:
```bash
encx-gen generate .
```

### Issue 2: Database Column Mismatch
**Error:** `sql: no rows in result set`

**Solution:** Ensure database columns match generated struct tags:
```go
type UserEncx struct {
    EmailHash string `db:"email_hash"` // Must match DB column name
}
```

### Issue 3: KMS Permission Errors
**Error:** `AccessDeniedException: User is not authorized`

**Solution:** Grant necessary KMS permissions:
```json
{
  "Effect": "Allow",
  "Action": [
    "kms:Encrypt",
    "kms:Decrypt",
    "kms:GenerateDataKey"
  ],
  "Resource": "arn:aws:kms:region:account:key/*"
}
```

### Issue 4: Pepper Validation Errors (New)
**Error:** `pepper is uninitialized`

**Solution:** Ensure pepper is properly provided as 32-byte array:
```go
// ✅ Correct - 32 bytes
encx.WithPepper([]byte("your-32-byte-secret-pepper-key!"))

// ❌ Wrong - string instead of bytes
encx.WithPepper("your-32-byte-secret-pepper-key!")

// ❌ Wrong - all zeros
encx.WithPepper(make([]byte, 32))
```

### Issue 5: Configuration Validation Errors (New)
**Error:** `database cannot be configured both via connection and path`

**Solution:** Use either database connection OR path, not both:
```go
// ✅ Option 1: Use database connection
encx.WithKeyMetadataDB(db)

// ✅ Option 2: Use database path
encx.WithKeyMetadataDBPath("/path/to/database.db")

// ❌ Wrong: Both at the same time
encx.WithKeyMetadataDB(db)
encx.WithKeyMetadataDBPath("/path/to/database.db")
```

## Additional Resources

- **[Main README](../README.md)** - Overview and quick start
- **[API Reference](./API_REFERENCE.md)** - Complete API documentation
- **[Code Generation Guide](./CODE_GENERATION_GUIDE.md)** - Performance optimization
- **[Context7 Guide](./CONTEXT7_GUIDE.md)** - Integration patterns
- **[Examples](../examples/)** - Working code samples
- **[KMS Provider Docs](../providers/)** - Provider-specific configuration

## Support & Contribution

- **Issues**: Report bugs or request features via GitHub issues
- **Contributions**: See CONTRIBUTING.md for guidelines
- **Security**: Report security issues to security@example.com

---

**Next Steps:**
1. Follow the [Quick Integration Checklist](#quick-integration-checklist)
2. Review the [Integration Pattern](#integration-patterns) that matches your use case
3. Set up your [Database Schema](#database-schema-design)
4. Deploy to production following the [Production Deployment](#production-deployment) guide