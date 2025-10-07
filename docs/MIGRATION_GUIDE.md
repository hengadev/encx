# Migration Guide: From Reflection to Code Generation

This guide helps you migrate from reflection-based encx struct processing to the new high-performance code generation system.

## Table of Contents

1. [Overview](#overview)
2. [Before You Start](#before-you-start)
3. [Migration Steps](#migration-steps)
4. [Code Changes](#code-changes)
5. [Database Schema](#database-schema)
6. [Testing Migration](#testing-migration)
7. [Performance Comparison](#performance-comparison)
8. [Troubleshooting](#troubleshooting)

## Overview

### What's Changing

**Old Approach (Reflection-based):**
```go
// Runtime reflection processing
processor := encx.NewStructProcessor(crypto, serializer)
result, err := processor.Process(ctx, userStruct)
```

**New Approach (Code Generation):**
```go
// Compile-time generated functions
userEncx, err := ProcessUserEncx(ctx, crypto, user)
```

### Benefits of Migration

- **Performance**: 10-100x faster processing (no reflection overhead)
- **Type Safety**: Compile-time validation and strongly-typed functions
- **Maintainability**: Generated code is readable and debuggable
- **Validation**: Comprehensive tag validation at build time
- **Incremental**: Only regenerate when source files change

## Before You Start

### Prerequisites

1. **encx version**: Ensure you're using encx v2.0+ with code generation support
2. **Go version**: Go 1.21+ recommended for best performance
3. **Backup**: Create a backup of your current codebase
4. **Testing**: Ensure you have comprehensive tests for your encryption logic

### Assessment

Run this assessment to understand your migration scope:

```bash
# Count structs using reflection-based processing
grep -r "NewStructProcessor\|processor.Process" . | wc -l

# Find structs with encx tags
grep -r "encx:" . --include="*.go" | wc -l

# Check for custom serializers
grep -r "Serializer" . --include="*.go" | grep -v "json"
```

## Migration Steps

### Step 1: Install Code Generation Tools

```bash
# Build the encx-gen CLI tool
cd path/to/encx
make build-cli
make install-cli

# Verify installation
encx-gen version
```

### Step 2: Initialize Configuration

```bash
# In your project root
encx-gen init

# This creates encx.yaml - review and customize as needed
```

### Step 3: Update Struct Definitions

**Before (Reflection-based):**
```go
type User struct {
    ID    int    `json:"id"`
    Email string `json:"email" encx:"encrypt,hash_basic"`
    Phone string `json:"phone" encx:"encrypt"`
    SSN   string `json:"ssn" encx:"hash_secure"`
}
```

**After (Code Generation):**
```go
//go:generate encx-gen generate .

// Source struct - keep it clean with just encx tags
type User struct {
    ID    int    `json:"id"`
    Email string `json:"email" encx:"encrypt,hash_basic"`
    Phone string `json:"phone" encx:"encrypt"`
    SSN   string `json:"ssn" encx:"hash_secure"`
}

// Generated UserEncx struct (created automatically by encx-gen)
// You don't write this - it's generated in user_encx.go
type UserEncx struct {
    EmailEncrypted []byte `db:"email_encrypted" json:"email_encrypted"`
    EmailHash      string `db:"email_hash" json:"email_hash"`
    PhoneEncrypted []byte `db:"phone_encrypted" json:"phone_encrypted"`
    SSNHashSecure  string `db:"ssn_hash_secure" json:"ssn_hash_secure"`

    // Essential encryption fields
    DEKEncrypted []byte                      `db:"dek_encrypted" json:"dek_encrypted"`
    KeyVersion   int                         `db:"key_version" json:"key_version"`
    Metadata     metadata.EncryptionMetadata `db:"metadata" json:"metadata"`
}
```

**Key Change:** Code generation creates a **separate** `UserEncx` struct. You don't add companion fields to your source struct anymore!

#### 3.1 Validate Struct Definitions

```bash
# Validate your struct tags
encx-gen validate -v .

# Fix any validation errors before proceeding
```

### Step 4: Generate Code

```bash
# Generate encx code for all packages
encx-gen generate -v .

# This creates *_encx.go files with ProcessXxxEncx functions
```

### Step 5: Update Application Code

#### 5.1 Replace Reflection Calls

**Before:**
```go
// Old reflection-based approach
processor := encx.NewStructProcessor(crypto, serializer)
processed, err := processor.Process(ctx, user)
if err != nil {
    return err
}

// Cast result back to expected type
userProcessed := processed.(*UserProcessed)
```

**After:**
```go
// New code generation approach
// Note: Returns separate UserEncx struct with encrypted fields
userEncx, err := ProcessUserEncx(ctx, crypto, user)
if err != nil {
    return err
}

// Store userEncx (not user) in database
// userEncx contains all encrypted/hashed fields
```

#### 5.2 Update Decryption Code

**Before:**
```go
// Old reflection-based decryption
processor := encx.NewStructProcessor(crypto, serializer)
decrypted, err := processor.Decrypt(ctx, encryptedUser)
if err != nil {
    return err
}

user := decrypted.(*User)
```

**After:**
```go
// New code generation approach
// Note: Takes UserEncx and returns original User with decrypted data
user, err := DecryptUserEncx(ctx, crypto, userEncx)
if err != nil {
    return err
}

// user now has decrypted Email, Phone, SSN fields
```

#### 5.3 Remove Old Dependencies

```go
// Remove these imports if no longer needed
import (
    "github.com/hengadev/encx/internal/processor"  // Remove
    "github.com/hengadev/encx/internal/reflection" // Remove
)
```

### Step 6: Update Database Schema

**Important:** With code generation, you store the `UserEncx` struct (not `User`) in the database. Consider creating a separate table or renaming your existing table.

#### 6.1 Option A: Create New Table

```sql
-- PostgreSQL example
CREATE TABLE users_encx (
    id SERIAL PRIMARY KEY,

    -- Encrypted/hashed fields
    email_encrypted BYTEA,
    email_hash VARCHAR(64),
    phone_encrypted BYTEA,
    ssn_hash_secure TEXT,

    -- Essential encryption fields
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Add indexes for performance
CREATE INDEX idx_users_encx_email_hash ON users_encx (email_hash);
CREATE INDEX idx_users_encx_key_version ON users_encx (key_version);
```

#### 6.2 Option B: Migrate Existing Table

```sql
-- PostgreSQL example
ALTER TABLE users ADD COLUMN email_encrypted BYTEA;
ALTER TABLE users ADD COLUMN email_hash VARCHAR(64);
ALTER TABLE users ADD COLUMN phone_encrypted BYTEA;
ALTER TABLE users ADD COLUMN ssn_hash_secure TEXT;
ALTER TABLE users ADD COLUMN dek_encrypted BYTEA;
ALTER TABLE users ADD COLUMN key_version INTEGER DEFAULT 1;
ALTER TABLE users ADD COLUMN metadata JSONB DEFAULT '{}';

-- Add indexes
CREATE INDEX idx_users_email_hash ON users (email_hash);
CREATE INDEX idx_users_key_version ON users (key_version);
```

#### 6.3 Data Migration Script

```go
func migrateUserData(db *sql.DB, crypto *encx.Crypto) error {
    // Read existing users
    rows, err := db.Query("SELECT id, email, phone, ssn FROM users WHERE email_encrypted IS NULL")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var user User
        err := rows.Scan(&user.ID, &user.Email, &user.Phone, &user.SSN)
        if err != nil {
            continue
        }

        // Process with new code generation
        userEncx, err := ProcessUserEncx(ctx, crypto, &user)
        if err != nil {
            log.Printf("Failed to process user %d: %v", user.ID, err)
            continue
        }

        // Update database with encrypted data
        _, err = db.Exec(`
            UPDATE users SET
                email_encrypted = $1,
                email_hash = $2,
                phone_encrypted = $3,
                ssn_hash_secure = $4,
                dek_encrypted = $5,
                key_version = $6,
                metadata = $7
            WHERE id = $8`,
            userEncx.EmailEncrypted,
            userEncx.EmailHash,
            userEncx.PhoneEncrypted,
            userEncx.SSNHashSecure,
            userEncx.DEKEncrypted,
            userEncx.KeyVersion,
            userEncx.Metadata,
            user.ID,
        )
        if err != nil {
            log.Printf("Failed to update user %d: %v", user.ID, err)
        }
    }

    return nil
}
```

### Step 7: Update Build Process

#### 7.1 Add Code Generation to Build

Add to your Makefile:
```makefile
build: generate
	go build ./...

generate:
	encx-gen generate -v ./...

validate:
	encx-gen validate -v ./...
```

#### 7.2 Add Go Generate Directives

```go
package models

//go:generate encx-gen validate -v .
//go:generate encx-gen generate -v .

type User struct {
    // Your struct definition
}
```

#### 7.3 Update CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Generate encx code
  run: |
    make validate
    make generate
    git diff --exit-code  # Ensure generated code is up to date

- name: Build and test
  run: |
    make build
    make test
```

## Code Changes

### Pattern 1: Simple Struct Processing

**Before:**
```go
func processUser(ctx context.Context, user *User) (*UserProcessed, error) {
    processor := encx.NewStructProcessor(crypto, &serialization.JSONSerializer{})
    result, err := processor.Process(ctx, user)
    if err != nil {
        return nil, err
    }
    return result.(*UserProcessed), nil
}
```

**After:**
```go
func processUser(ctx context.Context, user *User) (*UserEncx, error) {
    return ProcessUserEncx(ctx, crypto, user)
}
```

### Pattern 2: Batch Processing

**Before:**
```go
func processUsers(ctx context.Context, users []*User) ([]*UserProcessed, error) {
    processor := encx.NewStructProcessor(crypto, &serialization.JSONSerializer{})
    var results []*UserProcessed

    for _, user := range users {
        result, err := processor.Process(ctx, user)
        if err != nil {
            return nil, err
        }
        results = append(results, result.(*UserProcessed))
    }

    return results, nil
}
```

**After:**
```go
func processUsers(ctx context.Context, users []*User) ([]*UserEncx, error) {
    var results []*UserEncx

    for _, user := range users {
        userEncx, err := ProcessUserEncx(ctx, crypto, user)
        if err != nil {
            return nil, err
        }
        results = append(results, userEncx)
    }

    return results, nil
}
```

### Pattern 3: Generic Processing (Advanced)

**Before:**
```go
func processAnyStruct(ctx context.Context, data interface{}) (interface{}, error) {
    processor := encx.NewStructProcessor(crypto, &serialization.JSONSerializer{})
    return processor.Process(ctx, data)
}
```

**After:**
```go
// Use type-specific generated functions instead
func processUser(ctx context.Context, user *User) (*UserEncx, error) {
    return ProcessUserEncx(ctx, crypto, user)
}

func processOrder(ctx context.Context, order *Order) (*OrderEncx, error) {
    return ProcessOrderEncx(ctx, crypto, order)
}

// Or use type switches if you need generic processing
func processAnyStruct(ctx context.Context, data interface{}) (interface{}, error) {
    switch v := data.(type) {
    case *User:
        return ProcessUserEncx(ctx, crypto, v)
    case *Order:
        return ProcessOrderEncx(ctx, crypto, v)
    default:
        return nil, fmt.Errorf("unsupported type: %T", data)
    }
}
```

## Database Schema

### Schema Migration

#### PostgreSQL Migration

```sql
-- Add new columns
ALTER TABLE users
    ADD COLUMN email_encrypted BYTEA,
    ADD COLUMN email_hash VARCHAR(64),
    ADD COLUMN phone_encrypted BYTEA,
    ADD COLUMN ssn_hash_secure TEXT,
    ADD COLUMN dek_encrypted BYTEA,
    ADD COLUMN key_version INTEGER DEFAULT 1,
    ADD COLUMN metadata JSONB DEFAULT '{}';

-- Add indexes
CREATE INDEX CONCURRENTLY idx_users_email_hash ON users (email_hash);
CREATE INDEX CONCURRENTLY idx_users_key_version ON users (key_version);
CREATE INDEX CONCURRENTLY idx_users_metadata_gin ON users USING GIN (metadata);

-- After data migration, make required fields NOT NULL
-- ALTER TABLE users ALTER COLUMN dek_encrypted SET NOT NULL;
```

#### SQLite Migration

```sql
-- SQLite requires recreation for complex schema changes
CREATE TABLE users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Existing columns
    name TEXT,
    created_at DATETIME,

    -- New encrypted columns
    email_encrypted BLOB,
    email_hash TEXT,
    phone_encrypted BLOB,
    ssn_hash_secure TEXT,
    dek_encrypted BLOB,
    key_version INTEGER DEFAULT 1,
    metadata TEXT DEFAULT '{}' CHECK (json_valid(metadata))
);

-- Copy existing data
INSERT INTO users_new (id, name, created_at)
SELECT id, name, created_at FROM users;

-- Drop old table and rename
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

-- Add indexes
CREATE INDEX idx_users_email_hash ON users (email_hash);
CREATE INDEX idx_users_key_version ON users (key_version);
```

### Cross-Database Compatibility

Use the schema helpers for cross-database compatibility:

```go
import "github.com/hengadev/encx/internal/schema"

// Get database-specific DDL
dbType := schema.ParseDatabaseType("postgresql")
ddl := schema.GetDDLForDatabase(dbType.String(), "users_encx")

// Use MetadataColumn for JSON storage
type UserRecord struct {
    ID           int                   `db:"id"`
    EmailHash    string                `db:"email_hash"`
    Metadata     schema.MetadataColumn `db:"metadata"`
}
```

## Testing Migration

### 1. Create Migration Tests

```go
func TestMigration(t *testing.T) {
    // Test data
    originalUser := &User{
        Email: "test@example.com",
        Phone: "+1234567890",
        SSN:   "123-45-6789",
    }

    // Process with new code generation
    userEncx, err := ProcessUserEncx(ctx, crypto, originalUser)
    require.NoError(t, err)

    // Decrypt and verify
    decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
    require.NoError(t, err)

    // Verify data integrity
    assert.Equal(t, originalUser.Email, decryptedUser.Email)
    assert.Equal(t, originalUser.Phone, decryptedUser.Phone)
    assert.Equal(t, originalUser.SSN, decryptedUser.SSN)

    // Verify encrypted fields are populated
    assert.NotEmpty(t, userEncx.EmailEncrypted)
    assert.NotEmpty(t, userEncx.EmailHash)
    assert.NotEmpty(t, userEncx.PhoneEncrypted)
    assert.NotEmpty(t, userEncx.SSNHashSecure)
}
```

### 2. Performance Comparison Tests

```go
func BenchmarkOldVsNew(b *testing.B) {
    user := &User{Email: "test@example.com", Phone: "+1234567890"}

    b.Run("Old_Reflection", func(b *testing.B) {
        processor := encx.NewStructProcessor(crypto, &serialization.JSONSerializer{})
        for i := 0; i < b.N; i++ {
            _, err := processor.Process(ctx, user)
            if err != nil {
                b.Fatal(err)
            }
        }
    })

    b.Run("New_CodeGen", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _, err := ProcessUserEncx(ctx, crypto, user)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

### 3. Integration Tests

```go
func TestEndToEndMigration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)

    // Create test data
    user := &User{Email: "test@example.com"}

    // Process and store
    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    require.NoError(t, err)

    err = storeUserEncx(db, userEncx)
    require.NoError(t, err)

    // Retrieve and decrypt
    retrievedUserEncx, err := loadUserEncx(db, user.ID)
    require.NoError(t, err)

    decryptedUser, err := DecryptUserEncx(ctx, crypto, retrievedUserEncx)
    require.NoError(t, err)

    assert.Equal(t, user.Email, decryptedUser.Email)
}
```

## Performance Comparison

### Expected Performance Improvements

| Operation | Reflection-based | Code Generation | Improvement |
|-----------|-----------------|-----------------|-------------|
| Small struct (3 fields) | 1000 ns/op | 50 ns/op | 20x faster |
| Medium struct (10 fields) | 3500 ns/op | 150 ns/op | 23x faster |
| Large struct (25 fields) | 8500 ns/op | 350 ns/op | 24x faster |
| Memory allocations | High | Low | 70% reduction |

### Benchmark Your Migration

```bash
# Run benchmarks before migration
go test -bench=BenchmarkOldReflection -benchmem ./...

# Run benchmarks after migration
go test -bench=BenchmarkNewCodeGen -benchmem ./...

# Compare results
go test -bench=BenchmarkOldVsNew -benchmem ./...
```

## Troubleshooting

### Common Migration Issues

#### 1. Missing Companion Fields

**Error**: `missing companion field EmailEncrypted []byte for encrypt tag`

**Solution**: Add all required companion fields to your struct:
```go
Email string `encx:"encrypt"`
EmailEncrypted []byte `json:"email_encrypted"` // Add this
```

#### 2. Database Schema Mismatch

**Error**: `column "email_encrypted" does not exist`

**Solution**: Run database migration before using generated code:
```sql
ALTER TABLE users ADD COLUMN email_encrypted BYTEA;
```

#### 3. Type Conversion Issues

**Error**: `cannot convert *UserProcessed to *UserEncx`

**Solution**: Update your code to use the new generated types:
```go
// Old
userProcessed := result.(*UserProcessed)

// New
userEncx, err := ProcessUserEncx(ctx, crypto, user)
```

#### 4. Serializer Issues

**Error**: `serializer not found`

**Solution**: The new system uses per-struct serializers. Ensure your configuration specifies the serializer:
```yaml
generation:
  default_serializer: "json"
```

### Migration Checklist

- [ ] ✅ Install encx-gen CLI tool
- [ ] ✅ Initialize configuration (encx.yaml)
- [ ] ✅ Add companion fields to all structs
- [ ] ✅ Validate struct definitions
- [ ] ✅ Generate code for all packages
- [ ] ✅ Update application code to use generated functions
- [ ] ✅ Migrate database schema
- [ ] ✅ Migrate existing encrypted data
- [ ] ✅ Update build process and CI/CD
- [ ] ✅ Run comprehensive tests
- [ ] ✅ Benchmark performance improvements
- [ ] ✅ Remove old reflection-based code

### Getting Help

1. **Validation errors**: Run `encx-gen validate -v .` for detailed error messages
2. **Generation issues**: Use `encx-gen generate -dry-run -v .` to debug
3. **Performance issues**: Run benchmarks to verify improvements
4. **Schema questions**: Check the database schema examples in the docs

## Next Steps

After completing the migration:

1. **Monitor Performance**: Verify the expected performance improvements
2. **Optimize Queries**: Use hash fields for efficient database searches
3. **Update Documentation**: Document your new encrypted data patterns
4. **Team Training**: Train your team on the new code generation workflow

The migration to code generation will significantly improve your application's performance while providing better type safety and maintainability!