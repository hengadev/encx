# ENCX Migration Guide

> **For users upgrading ENCX to v0.5.2+**
>
> This guide helps you migrate from older versions of ENCX to the latest version with automatic initialization.

## Table of Contents

1. [Overview of Changes](#overview-of-changes)
2. [Breaking Changes](#breaking-changes)
3. [Migration Steps](#migration-steps)
4. [Code Changes Required](#code-changes-required)
5. [Troubleshooting Migration Issues](#troubleshooting-migration-issues)
6. [Rollback Plan](#rollback-plan)

## Overview of Changes

### v0.5.2: Automatic Database Schema Initialization
- **What changed**: `NewCrypto()` now automatically creates the required `kek_versions` table and index
- **Benefit**: No more manual database setup or migration scripts
- **Status**: ✅ **Fully backward compatible**

### v0.5.3: Automatic KEK Initialization
- **What changed**: `NewCrypto()` now automatically calls `EnsureInitialKEK()` to create the first KEK
- **Benefit**: No more manual KEK setup required
- **Status**: ✅ **Fully backward compatible**

### Enhanced Configuration Validation
- **What changed**: Configuration validation now happens before database operations
- **Benefit**: Better error messages, faster failure detection
- **Status**: ✅ **Fully backward compatible**

## Breaking Changes

### ✅ **No Breaking Changes**

All changes in v0.5.2+ are **fully backward compatible**. Your existing code will continue to work without modifications.

**Optional improvements** you can make:
- Remove manual database setup code (recommended)
- Remove manual KEK initialization code (recommended)
- Update error handling for deprecated errors (optional)

## Migration Steps

### Step 1: Update ENCX Version

```bash
# Update to the latest version
go get github.com/hengadev/encx@v0.5.3
```

### Step 2: Run Your Tests

```bash
# Run your existing tests to ensure compatibility
go test ./...
```

### Step 3: Remove Manual Setup Code (Optional but Recommended)

#### Before (v0.5.1 and earlier):

```go
// ❌ Manual database setup (no longer needed)
func setupDatabase(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS kek_versions (
            alias TEXT NOT NULL,
            version INTEGER NOT NULL,
            kms_key_id TEXT NOT NULL,
            is_deprecated BOOLEAN DEFAULT FALSE,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (alias, version)
        )
    `)
    return err
}

// ❌ Manual KEK initialization (no longer needed)
func initializeKEK(ctx context.Context, crypto *encx.Crypto) error {
    // This was required before v0.5.3
    return crypto.keyRotationOps.EnsureInitialKEK(ctx, crypto)
}
```

#### After (v0.5.3+):

```go
// ✅ Automatic setup - just create the crypto instance
func main() {
    ctx := context.Background()

    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(kms),
        encx.WithKEKAlias("my-app-key"),
        encx.WithPepper([]byte("your-32-byte-secret-pepper-key!")),
    )
    if err != nil {
        panic(err)
    }

    // Ready to use! Database schema and KEK are automatically set up
}
```

### Step 4: Update Error Handling (Optional)

#### Old Errors (No Longer Occur):
- `no such table: kek_versions`
- `failed to get KMS Key ID for alias 'xxx' version 0`

#### New Errors (Better Messages):
- `pepper is uninitialized`
- `database cannot be configured both via connection and path`

#### Update Error Handling:

```go
// Before - handling deprecated errors
switch {
case strings.Contains(err.Error(), "no such table: kek_versions"):
    log.Fatal("Database not initialized - run migrations first")
case strings.Contains(err.Error(), "KEK version 0 not found"):
    log.Fatal("KEK not initialized - call EnsureInitialKEK first")
}

// After - new errors are clearer and occur earlier
switch {
case strings.Contains(err.Error(), "pepper is uninitialized"):
    log.Fatal("Invalid pepper configuration - provide a 32-byte non-zero pepper")
case strings.Contains(err.Error(), "database cannot be configured both via connection and path"):
    log.Fatal("Invalid database configuration - use either connection OR path, not both")
}
```

## Code Changes Required

### 1. Update Imports (If Using Old API)

```go
// Old import pattern
import "github.com/hengadev/encx"
import "github.com/hengadev/encx/crypto" // Changed

// New import pattern
import "github.com/hengadev/encx"
// All crypto operations are now through encx package
```

### 2. Update Database Initialization

```go
// Remove this code (if you have it)
func initDatabase(dbPath string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return err
    }

    // Manual table creation is no longer needed
    _, err = db.Exec(`CREATE TABLE kek_versions...`)
    return err
}
```

### 3. Update Application Startup

```go
// Before
func main() {
    ctx := context.Background()

    // Manual database setup
    db := setupDatabase()

    // Manual KEK setup
    crypto := setupCrypto(ctx)
    err := crypto.keyRotationOps.EnsureInitialKEK(ctx, crypto)
    if err != nil {
        panic(err)
    }

    // Application logic
}

// After
func main() {
    ctx := context.Background()

    // Everything is automatic now
    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(kms),
        encx.WithKEKAlias("my-app-key"),
        encx.WithPepper([]byte("your-32-byte-secret-pepper-key!")),
    )
    if err != nil {
        panic(err)
    }

    // Application logic - ready to go!
}
```

## Troubleshooting Migration Issues

### Issue: "pepper is uninitialized" after migration

**Cause**: You're providing a pepper as a string instead of bytes, or providing all-zero bytes.

**Solution**:
```go
// ❌ Wrong
encx.WithPepper("your-pepper-string")
encx.WithPepper(make([]byte, 32))

// ✅ Correct
encx.WithPepper([]byte("your-32-byte-secret-pepper-key!"))
```

### Issue: "database cannot be configured both via connection and path"

**Cause**: You're providing both database connection and path options.

**Solution**:
```go
// ❌ Wrong
encx.WithKeyMetadataDB(db)
encx.WithKeyMetadataDBPath("/path/to/db")

// ✅ Correct - choose one
encx.WithKeyMetadataDB(db)
// OR
encx.WithKeyMetadataDBPath("/path/to/db")
```

### Issue: Tests failing with validation errors

**Cause**: Tests might be using conflicting configuration options.

**Solution**:
```go
// Use dedicated test setup
crypto, err := encx.NewTestCrypto(t)
// OR
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(testKMS),
    encx.WithKEKAlias("test-key"),
    encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
)
```

## Rollback Plan

If you need to rollback to v0.5.1:

### Step 1: Downgrade ENCX

```bash
go get github.com/hengadev/encx@v0.5.1
```

### Step 2: Restore Manual Setup Code

Add back the manual setup code you removed:

```go
// Restore database initialization
func setupDatabase(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS kek_versions (
            alias TEXT NOT NULL,
            version INTEGER NOT NULL,
            kms_key_id TEXT NOT NULL,
            is_deprecated BOOLEAN DEFAULT FALSE,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (alias, version)
        )
    `)
    return err
}

// Restore KEK initialization
func initializeKEK(ctx context.Context, crypto *encx.Crypto) error {
    return crypto.keyRotationOps.EnsureInitialKEK(ctx, crypto)
}
```

### Step 3: Update Application Startup

```go
func main() {
    ctx := context.Background()

    // Restore manual setup
    db := setupDatabase()
    crypto := setupCrypto(ctx)

    // Restore manual KEK init
    err := crypto.keyRotationOps.EnsureInitialKEK(ctx, crypto)
    if err != nil {
        panic(err)
    }

    // Application logic
}
```

## Migration Checklist

- [ ] Update ENCX to v0.5.3+
- [ ] Run existing tests to verify compatibility
- [ ] Remove manual database setup code (recommended)
- [ ] Remove manual KEK initialization code (recommended)
- [ ] Update error handling for new validation messages (optional)
- [ ] Test with real data to ensure encryption/decryption works
- [ ] Update deployment scripts (remove manual migration steps)
- [ ] Update documentation and README files
- [ ] Monitor production for any issues after deployment

## Benefits of Upgrading

1. **Simplified Setup**: No more manual database or KEK initialization
2. **Better Error Messages**: Clearer validation errors that occur earlier
3. **Reduced Maintenance**: Less code to maintain and fewer moving parts
4. **Faster Development**: Get started immediately without setup overhead
5. **Fewer Migration Issues**: Automatic initialization prevents common setup errors

## Support

If you encounter issues during migration:

1. **Check the [Integration Guide](./INTEGRATION_GUIDE.md)** for updated examples
2. **Review the [Troubleshooting Guide](./TROUBLESHOOTING.md)** for common issues
3. **Open a GitHub issue** with your specific error and code
4. **Check existing issues** for similar migration problems

---

**Next Steps:**
1. Update to v0.5.3+
2. Test thoroughly
3. Remove deprecated manual setup code
4. Enjoy the simplified automatic initialization!