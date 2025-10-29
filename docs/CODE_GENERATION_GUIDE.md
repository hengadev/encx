# Encx Code Generation Guide

This guide explains how to use the encx code generation system to transition from reflection-based struct processing to high-performance generated code.

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Struct Tags](#struct-tags)
5. [Generated Code](#generated-code)
6. [CLI Commands](#cli-commands)
7. [Build Integration](#build-integration)
8. [Database Schema](#database-schema)
9. [Best Practices](#best-practices)
10. [Troubleshooting](#troubleshooting)

## Overview

The encx code generation system replaces runtime reflection with compile-time code generation, providing:

- **Performance**: Eliminates reflection overhead at runtime
- **Type Safety**: Generates strongly-typed functions for each struct
- **Validation**: Comprehensive validation of tags and companion fields
- **Incremental**: Only regenerates when source files change
- **Cross-Database**: Supports PostgreSQL, SQLite, and MySQL

## Quick Start

### 1. Install the CLI Tool

```bash
# Build and install the encx-gen CLI
make build-cli
make install-cli

# Or run directly with go
go run cmd/encx-gen/main.go version
```

### 2. Initialize Configuration

```bash
# Create default configuration
encx-gen init

# This creates encx.yaml with default settings
```

### 3. Define Your Struct

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    // Required fields - always encrypted even if zero value
    ID       int       `json:"id"`
    Email    string    `json:"email" encx:"encrypt,hash_basic"`
    Phone    string    `json:"phone" encx:"encrypt"`
    SSN      string    `json:"ssn" encx:"hash_secure"`
    IsActive bool      `json:"is_active" encx:"encrypt"`

    // Struct types with semantic zero values - checked automatically
    UserID    uuid.UUID `json:"user_id" encx:"encrypt"`    // Checks != uuid.Nil
    CreatedAt time.Time `json:"created_at" encx:"encrypt"` // Checks !.IsZero()

    // Optional fields - use pointers, checked for nil
    NickName  *string    `json:"nickname" encx:"encrypt"`
    UpdatedAt *time.Time `json:"updated_at" encx:"encrypt"`

    // No companion fields needed! Code generation creates separate output struct
}
```

**Note on Imports**: The code generator automatically tracks and includes required imports (e.g., `github.com/google/uuid`, `time`) in generated files.

### 4. Generate Code

```bash
# Validate your struct tags first
encx-gen validate -v .

# Generate the code
encx-gen generate -v .

# This creates user_encx.go with ProcessUserEncx and DecryptUserEncx functions
```

### 5. Use Generated Code

```go
ctx := context.Background()
crypto := encx.NewCrypto(config)

// Original user data
user := &User{
    Email: "user@example.com",
    Phone: "+1234567890",
    SSN:   "123-45-6789",
}

// Process (encrypt/hash) the user data
// Note: Function name follows pattern Process<StructName>Encx
// For a User struct, the generator creates ProcessUserEncx
userEncx, err := ProcessUserEncx(ctx, crypto, user)
if err != nil {
    log.Fatal(err)
}

// Store userEncx in database...

// Later, decrypt the data
decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### Basic Configuration (encx.yaml)

```yaml
version: "1"

generation:
  output_suffix: "_encx"
  package_name: "encx"

packages: {}
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `output_suffix` | Suffix for generated files | `_encx` |
| `package_name` | Package name for generated code | `encx` |

### Package-Specific Configuration

```yaml
packages:
  "./models":
    skip: false              # Generate code for this package
    output_dir: "./generated" # Custom output directory (optional)

  "./test":
    skip: true               # Skip code generation
```

**Note:** ENCX uses a custom compact binary serializer automatically. No serializer configuration is needed.

## Struct Tags

### Supported Tags

| Tag | Description | Companion Field Type | Example |
|-----|-------------|---------------------|---------|
| `encrypt` | Encrypt field data | `[]byte` | `EmailEncrypted []byte` |
| `hash_basic` | Basic hash for search | `string` | `EmailHash string` |
| `hash_secure` | Secure hash (no search) | `string` | `SSNHashSecure string` |

### Tag Combinations

```go
type User struct {
    // Encrypt only
    Phone string `encx:"encrypt"`
    PhoneEncrypted []byte

    // Hash only (basic)
    Username string `encx:"hash_basic"`
    UsernameHash string

    // Hash only (secure)
    SSN string `encx:"hash_secure"`
    SSNHashSecure string

    // Encrypt + Hash (searchable)
    Email string `encx:"encrypt,hash_basic"`
    EmailEncrypted []byte
    EmailHash string

    // Encrypt + Hash (non-searchable)
    CreditCard string `encx:"encrypt,hash_secure"`
    CreditCardEncrypted []byte
    CreditCardHashSecure string
}
```

### Invalid Combinations

```go
// ❌ INVALID - Cannot use both hash types
Email string `encx:"hash_basic,hash_secure"`

// ❌ INVALID - Missing companion field
Phone string `encx:"encrypt"`
// Missing: PhoneEncrypted []byte

// ❌ INVALID - Wrong companion field type
Email string `encx:"encrypt"`
EmailEncrypted string  // Should be []byte
```

### Field Types and Zero-Value Handling

The code generator intelligently handles different field types with type-aware zero-value checking:

#### Required Fields (Always Encrypted)

Basic types are always encrypted, even when they have zero values:

```go
type User struct {
    Email    string `encx:"encrypt"`  // Empty string "" is encrypted
    Age      int    `encx:"encrypt"`  // Zero value 0 is encrypted
    IsActive bool   `encx:"encrypt"`  // false is encrypted
    Score    float64 `encx:"encrypt"` // 0.0 is encrypted
}
```

**Rationale**: Zero values are semantically valid data that should be encrypted. Empty strings, zero integers, and false booleans all represent meaningful state.

#### Optional Fields (Pointer Types)

Use pointer types to indicate truly optional fields:

```go
import "github.com/google/uuid"

type User struct {
    // Optional fields checked for nil before encryption
    NickName    *string     `encx:"encrypt"` // Encrypts only if != nil
    MiddleName  *string     `encx:"encrypt"` // Skips if nil
    Age         *int        `encx:"encrypt"` // Encrypts only if != nil
    TenantID    *uuid.UUID  `encx:"encrypt"` // Skips if nil
}
```

**Generated code includes nil checks**:
```go
if source.NickName != nil {
    // Serialize and encrypt
}
```

#### Struct Types with Semantic Zero Values

Some struct types have special "not set" values that are automatically detected:

```go
import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    // uuid.UUID - checks against uuid.Nil
    UserID uuid.UUID `encx:"encrypt"` // Skips if == uuid.Nil

    // time.Time - checks .IsZero()
    CreatedAt time.Time `encx:"encrypt"` // Skips if .IsZero() == true
}
```

**Generated code**:
```go
// For uuid.UUID
if source.UserID != uuid.Nil {
    // Serialize and encrypt
}

// For time.Time
if !source.CreatedAt.IsZero() {
    // Serialize and encrypt
}
```

#### Type Decision Matrix

| Type | Example | Zero-Value Behavior | Check Performed |
|------|---------|---------------------|-----------------|
| `string` | `Email string` | Always encrypted | None (always process) |
| `int`, `uint`, `float` | `Age int` | Always encrypted | None (always process) |
| `bool` | `IsActive bool` | Always encrypted | None (always process) |
| `*string`, `*int`, etc. | `NickName *string` | Skip if nil | `!= nil` |
| `uuid.UUID` | `UserID uuid.UUID` | Skip if Nil | `!= uuid.Nil` |
| `time.Time` | `CreatedAt time.Time` | Skip if zero time | `!.IsZero()` |
| `*uuid.UUID` | `TenantID *uuid.UUID` | Skip if nil | `!= nil` |
| `*time.Time` | `UpdatedAt *time.Time` | Skip if nil | `!= nil` |

#### Best Practice: Use Types to Express Intent

```go
// ✅ GOOD - Type system expresses optionality
type User struct {
    Email    string     // Required, empty string is valid
    NickName *string    // Optional, use nil for "not set"
    UserID   uuid.UUID  // Required, uuid.Nil means "not set"
    TenantID *uuid.UUID // Optional, nil means "not set"
}

// ❌ AVOID - Ambiguous semantics
type User struct {
    Email    string     // Is "" a valid value or "not set"?
    NickName string     // Can't distinguish "not set" from empty
}
```

### Type Aliases

The code generator handles type aliases based on their usage patterns:

#### Scalar Type Aliases (Fully Supported ✅)

Simple type aliases to basic types work correctly:

```go
type State string
type UserID int64
type Priority int8
type Active bool

type Order struct {
    Status   State    `encx:"encrypt"` // Always encrypted - correct
    User     UserID   `encx:"encrypt"` // Always encrypted - correct
    Level    Priority `encx:"encrypt"` // Always encrypted - correct
    IsActive Active   `encx:"encrypt"` // Always encrypted - correct
}
```

**Behavior**: These are treated as required fields and always encrypted (even zero values). This is correct because:
- Empty string (`""`) is a valid state value
- Zero integer (`0`) is a valid ID
- False (`false`) is a valid active status

#### Special Type Aliases (Limited Support ⚠️)

Type aliases to `uuid.UUID` and `time.Time` have limitations:

```go
import (
    "time"
    "github.com/google/uuid"
)

type CustomUUID uuid.UUID
type Timestamp time.Time

type Resource struct {
    // ⚠️ LIMITATION: No automatic uuid.Nil or .IsZero() checks
    ID        CustomUUID `encx:"encrypt"` // Always encrypted (no uuid.Nil check)
    CreatedAt Timestamp  `encx:"encrypt"` // Always encrypted (no .IsZero() check)
}
```

**Why this happens**: The code generator sees the alias name (`CustomUUID`, `Timestamp`), not the underlying type (`uuid.UUID`, `time.Time`), so special zero-value checks aren't applied.

#### Recommended Pattern for Special Type Aliases

**For optional fields with special type aliases, use pointers**:

```go
type CustomUUID uuid.UUID
type Timestamp time.Time

type Resource struct {
    // ✅ RECOMMENDED - Required fields
    ID        CustomUUID  `encx:"encrypt"` // Always encrypted
    CreatedAt Timestamp   `encx:"encrypt"` // Always encrypted

    // ✅ RECOMMENDED - Optional fields use pointers
    TenantID  *CustomUUID `encx:"encrypt"` // nil checked
    UpdatedAt *Timestamp  `encx:"encrypt"` // nil checked
    DeletedAt *Timestamp  `encx:"encrypt"` // nil checked
}
```

**This works because**:
- Pointer nil checks happen regardless of the underlying type
- `*CustomUUID` gets `!= nil` check (correct)
- `*Timestamp` gets `!= nil` check (correct)

#### Alternative: Use Direct Types

If you need automatic zero-value checking, use the types directly:

```go
import (
    "time"
    "github.com/google/uuid"
)

type Resource struct {
    // ✅ Direct types get special handling
    ID        uuid.UUID `encx:"encrypt"` // Checks != uuid.Nil
    TenantID  uuid.UUID `encx:"encrypt"` // Checks != uuid.Nil
    CreatedAt time.Time `encx:"encrypt"` // Checks !.IsZero()
    UpdatedAt time.Time `encx:"encrypt"` // Checks !.IsZero()

    // Or use pointers for optional
    DeletedAt *time.Time `encx:"encrypt"` // Checks != nil
}
```

#### Summary: Type Alias Best Practices

| Alias Type | Pattern | Zero-Value Check | Recommendation |
|------------|---------|------------------|----------------|
| `type State string` | Value | None (always encrypts) | ✅ Use directly |
| `type UserID int` | Value | None (always encrypts) | ✅ Use directly |
| `type CustomUUID uuid.UUID` | Value | ⚠️ No special check | Use `uuid.UUID` directly |
| `type Timestamp time.Time` | Value | ⚠️ No special check | Use `time.Time` directly |
| `*State` | Pointer | `!= nil` | ✅ Use for optional |
| `*CustomUUID` | Pointer | `!= nil` | ✅ Use for optional |
| `*Timestamp` | Pointer | `!= nil` | ✅ Use for optional |

**Key Takeaway**: For aliases to `uuid.UUID` and `time.Time`, either:
1. Use the direct type for automatic zero-value checking, or
2. Use pointers to type aliases for optional fields with nil checking

### Automatic Import Tracking

The code generator automatically tracks and includes necessary imports:

```go
// Your source file
import "github.com/google/uuid"

type User struct {
    UserID uuid.UUID `encx:"encrypt"`
}
```

**Generated file automatically includes**:
```go
// Generated code
import (
    "context"
    "time"
    "github.com/hengadev/errsx"
    "github.com/hengadev/encx"
    "github.com/google/uuid"  // ✅ Automatically added
)
```

The generator:
- Parses all imports from your source file
- Detects which packages are used by field types
- Includes only necessary imports in generated code
- Avoids duplicate imports with hardcoded dependencies

## Generated Code

### Example Generated Functions

For a `User` struct, the generator creates:

```go
// ProcessUserEncx encrypts and hashes user data
// Note: Function name follows pattern Process<StructName>Encx
func ProcessUserEncx(ctx context.Context, crypto encx.CryptoService, source *User) (*UserEncx, error) {
    // Generated implementation with proper error handling
}

// DecryptUserEncx decrypts user data back to original form
// Note: Function name follows pattern Decrypt<StructName>Encx
func DecryptUserEncx(ctx context.Context, crypto encx.CryptoService, source *UserEncx) (*User, error) {
    // Generated implementation with proper error handling
}

// UserEncx contains only encrypted/hashed fields
// Note: Struct name follows pattern <StructName>Encx
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

### Error Handling

Generated functions use structured error handling with type-aware zero-value checks:

```go
// Generated code includes comprehensive error handling
func ProcessUserEncx(ctx context.Context, crypto encx.CryptoService, source *User) (*UserEncx, error) {
    errs := errsx.Map{}

    // DEK generation
    dek, err := crypto.GenerateDEK()
    if err != nil {
        errs.Set("DEK generation", err)
        return nil, errs.AsError()
    }

    // Required field - always encrypted (no condition)
    emailBytes, err := encx.SerializeValue(source.Email)
    if err != nil {
        errs.Set("Email serialization", err)
    } else {
        result.EmailEncrypted, err = crypto.EncryptData(ctx, emailBytes, dek)
        if err != nil {
            errs.Set("Email encryption", err)
        }
    }

    // Optional field - checked for nil
    if source.NickName != nil {
        nickNameBytes, err := encx.SerializeValue(source.NickName)
        if err != nil {
            errs.Set("NickName serialization", err)
        } else {
            result.NickNameEncrypted, err = crypto.EncryptData(ctx, nickNameBytes, dek)
            if err != nil {
                errs.Set("NickName encryption", err)
            }
        }
    }

    // Struct with semantic zero - checked automatically
    if source.UserID != uuid.Nil {
        userIDBytes, err := encx.SerializeValue(source.UserID)
        if err != nil {
            errs.Set("UserID serialization", err)
        } else {
            result.UserIDEncrypted, err = crypto.EncryptData(ctx, userIDBytes, dek)
            if err != nil {
                errs.Set("UserID encryption", err)
            }
        }
    }

    return result, errs.AsError()
}
```

**Note**: The generator automatically uses the internal compact binary serializer via `encx.SerializeValue()`.

## CLI Commands

### Running the Tool

You have 3 options to run encx-gen:

#### Option 1: Direct Commands (RECOMMENDED)
```bash
# Build the tool once
go build -o bin/encx-gen ./cmd/encx-gen

# Use the built tool
./bin/encx-gen generate .
./bin/encx-gen validate -v .
```

#### Option 2: Go Run (RELIABLE)
```bash
# Run directly from source (no building needed)
go run ./cmd/encx-gen generate .
go run ./cmd/encx-gen validate -v .
```

#### Option 3: Go Generate with Go Run (ADVANCED)
Add to your source file (path must be relative to your file):
```go
//go:generate go run ../../cmd/encx-gen generate .
//go:generate go run ../../cmd/encx-gen validate -v .
```

Then run:
```bash
go generate ./...
```

### generate

Generate encx code for structs:

```bash
# Using any of the 3 options above:

# Generate for current directory and all subdirectories recursively
# This discovers ALL Go packages in subdirectories automatically
go run ./cmd/encx-gen generate .

# Generate for specific packages only (no recursion)
go run ./cmd/encx-gen generate ./models ./api

# Generate with custom config
go run ./cmd/encx-gen generate -config=my-config.yaml .

# Verbose output (shows discovered packages)
go run ./cmd/encx-gen generate -v .

# Dry run (show what would be generated)
go run ./cmd/encx-gen generate -dry-run .
```

**Recursive Package Discovery**:
When you use `encx-gen generate .`, the tool automatically:
- Scans the current directory for Go packages
- Recursively discovers all Go packages in subdirectories
- Skips hidden directories, `vendor/`, and `node_modules/`
- Processes all discovered packages that contain Go source files
- Reports the number of packages found in verbose mode

This makes it ideal for processing entire projects from the root directory with a single command.

### validate

Validate configuration and struct tags:

```bash
# Validate current directory
go run ./cmd/encx-gen validate .

# Validate with verbose output
go run ./cmd/encx-gen validate -v .

# Validate specific packages
go run ./cmd/encx-gen validate ./models ./api
```

### init

Initialize configuration file:

```bash
# Create default configuration
go run ./cmd/encx-gen init

# Force overwrite existing file
go run ./cmd/encx-gen init -force
```

### version

Show version information:

```bash
go run ./cmd/encx-gen version
```

## Build Integration

### Three Working Approaches for Code Generation

#### 1. Direct Commands (RECOMMENDED)

Build once and use the binary:

```bash
# Build the tool once
go build -o bin/encx-gen ./cmd/encx-gen

# Validate structs
./bin/encx-gen validate -v ./models

# Generate code for entire project recursively
./bin/encx-gen generate -v .

# Generate for specific packages only
./bin/encx-gen generate -v ./models ./api ./internal
```

#### 2. Go Run (RELIABLE)

Run directly from source without building:

```bash
# Validate structs
go run ./cmd/encx-gen validate -v ./models

# Generate code for entire project recursively
go run ./cmd/encx-gen generate -v .

# Generate for specific packages only
go run ./cmd/encx-gen generate -v ./models ./api ./internal
```

#### 3. Go Generate with Go Run (ADVANCED)

If you prefer the Go generate workflow, add directives to your Go files:

```go
package models

//go:generate go run ../../cmd/encx-gen validate -v .
//go:generate go run ../../cmd/encx-gen generate -v .

type User struct {
    // Your struct definition
}
```

Then run:
```bash
# Generate for all packages
go generate ./...

# Generate for specific package
go generate ./models
```

**⚠️ Important Note:**
- Option 3 requires the correct relative path to `cmd/encx-gen` (e.g., `../../cmd/encx-gen`)
- Options 1 & 2 work consistently in all environments
- `//go:generate` is completely optional but can be useful for CI/CD automation
- When using `generate .`, the tool automatically discovers all Go packages in subdirectories recursively

### Makefile Integration

Use the provided Makefile targets:

```bash
# Generate all code
make generate

# Validate before generating
make validate

# Full development workflow
make dev

# CI pipeline
make ci
```

### Custom Integration

```bash
# Check if generation is needed
encx-gen validate . && echo "Valid" || exit 1

# Generate with error handling
encx-gen generate . || exit 1

# Verify nothing changed (for CI)
encx-gen generate -dry-run . | grep "Would generate" && exit 1 || echo "Up to date"
```

## Database Schema

### Hybrid Schema Approach

The generated code works with a hybrid database schema:

```sql
-- PostgreSQL example
CREATE TABLE users_encx (
    id SERIAL PRIMARY KEY,

    -- Individual encrypted/hashed columns
    email_encrypted BYTEA,
    email_hash VARCHAR(64),
    phone_encrypted BYTEA,
    ssn_hash_secure TEXT,

    -- Essential encryption fields
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL,

    -- Flexible metadata (JSONB for PostgreSQL)
    metadata JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_encx_email_hash ON users_encx (email_hash);
CREATE INDEX idx_users_encx_key_version ON users_encx (key_version);
```

### Cross-Database Support

The schema helpers support multiple databases:

```go
import "github.com/hengadev/encx/internal/schema"

// Get database-specific DDL
ddl := schema.GetDDLForDatabase("postgresql", "users_encx")
ddl := schema.GetDDLForDatabase("sqlite", "users_encx")
ddl := schema.GetDDLForDatabase("mysql", "users_encx")

// Use MetadataColumn for cross-database JSON support
type UserRecord struct {
    ID           int                     `db:"id"`
    EmailHash    string                  `db:"email_hash"`
    Metadata     schema.MetadataColumn   `db:"metadata"`
}
```

## Best Practices

### 1. Struct Design

```go
import (
    "time"
    "github.com/google/uuid"
)

// ✅ GOOD - Clear separation and type-based optionality
type User struct {
    // Business fields
    ID   int    `json:"id"`
    Name string `json:"name"`

    // Required sensitive fields - always encrypted
    Email    string `json:"email" encx:"encrypt,hash_basic"`
    Phone    string `json:"phone" encx:"encrypt"`
    SSN      string `json:"ssn" encx:"hash_secure"`
    IsActive bool   `json:"is_active" encx:"encrypt"` // false is valid and encrypted

    // Struct types with semantic zero values
    UserID    uuid.UUID `json:"user_id" encx:"encrypt"`    // Skips if uuid.Nil
    CreatedAt time.Time `json:"created_at" encx:"encrypt"` // Skips if .IsZero()

    // Optional fields - use pointers to express optionality
    NickName  *string    `json:"nickname" encx:"encrypt"`   // nil means "not set"
    UpdatedAt *time.Time `json:"updated_at" encx:"encrypt"` // nil means "not set"

    // No companion fields needed - code generation creates separate UserEncx struct
}

// ❌ AVOID - Ambiguous optionality
type User struct {
    Email    string `encx:"encrypt"` // Is "" a valid value or "not set"?
    NickName string `encx:"encrypt"` // Can't distinguish empty from unset
}

// ✅ BETTER - Use types to express intent
type User struct {
    Email    string  `encx:"encrypt"` // Required, "" is valid and encrypted
    NickName *string `encx:"encrypt"` // Optional, nil means "not set"
}
```

**Key Principles**:
- Use value types for required fields (even zero values are encrypted)
- Use pointer types for truly optional fields (nil = "not set")
- Use `uuid.UUID` and `time.Time` for fields with semantic zero values
- Let the type system express your intent

### 2. Package Organization

```
project/
├── models/           # Domain models with encx tags
│   ├── user.go
│   ├── user_encx.go  # Generated
│   └── encx.yaml     # Package-specific config
├── api/              # API models
│   ├── request.go
│   └── request_encx.go # Generated
└── encx.yaml         # Global config
```

### 3. Development Workflow

```bash
# 1. Design your structs
# 2. Validate tags
make validate

# 3. Generate code
make generate

# 4. Test integration
make test

# 5. Commit both source and generated code
git add . && git commit -m "Add encrypted user model"
```

### 4. CI/CD Integration

```yaml
# GitHub Actions example
- name: Validate encx code generation
  run: |
    make validate
    make generate
    git diff --exit-code  # Fail if generated code changed
```

## Troubleshooting

### Common Issues

#### 1. Missing Companion Fields

```
Error: missing companion field EmailEncrypted []byte for encrypt tag
```

**Solution**: Add the required companion field:
```go
Email string `encx:"encrypt"`
EmailEncrypted []byte `json:"email_encrypted"`  // Add this
```

#### 2. Invalid Tag Combinations

```
Error: cannot use both hash_basic and hash_secure tags
```

**Solution**: Use only one hash type:
```go
// ❌ Invalid
Email string `encx:"hash_basic,hash_secure"`

// ✅ Valid
Email string `encx:"encrypt,hash_basic"`
```

#### 3. Wrong Companion Field Type

```
Error: companion field EmailEncrypted has wrong type: expected []byte, got string
```

**Solution**: Fix the companion field type:
```go
Email string `encx:"encrypt"`
EmailEncrypted []byte `json:"email_encrypted"`  // Must be []byte for encrypt
```

#### 4. Generation Not Running

**Check**: Ensure encx-gen is installed and accessible:
```bash
which encx-gen
encx-gen version
```

**Check**: Verify configuration file:
```bash
encx-gen validate -v .
```

### Debugging Tips

1. **Use verbose mode**: `encx-gen generate -v .`
2. **Check dry-run output**: `encx-gen generate -dry-run .`
3. **Validate first**: `encx-gen validate .`
4. **Check cache**: Delete `.encx-gen-cache.json` if needed

### Performance Optimization

1. **Use incremental generation**: Cache automatically optimizes rebuilds
2. **Minimize structs**: Only add encx tags to fields that need encryption
3. **Proper indexing**: Use hash fields for database queries
4. **Monitor generation time**: Use benchmarks in large projects

## Migration from Reflection

See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) for detailed migration instructions from reflection-based encx to code generation.

## Advanced Topics

- [Custom Serializers](CUSTOM_SERIALIZERS.md)
- [Database Integration Patterns](DATABASE_PATTERNS.md)
- [Performance Benchmarking](PERFORMANCE.md)
- [Contributing to encx-gen](CONTRIBUTING.md)