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

type User struct {
    ID    int    `json:"id"`
    Email string `json:"email" encx:"encrypt,hash_basic"`
    Phone string `json:"phone" encx:"encrypt"`
    SSN   string `json:"ssn" encx:"hash_secure"`

    // No companion fields needed! Code generation creates separate output struct
}
```

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

Generated functions use structured error handling:

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

    // Field processing with individual error tracking
    // Uses internal compact binary serializer automatically
    if source.Email != "" {
        emailBytes, err := serialization.Serialize(source.Email)
        if err != nil {
            errs.Set("Email serialization", err)
        } else {
            result.EmailEncrypted, err = crypto.EncryptData(ctx, emailBytes, dek)
            if err != nil {
                errs.Set("Email encryption", err)
            }
        }
    }

    return result, errs.AsError()
}
```

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

# Generate for current directory
go run ./cmd/encx-gen generate .

# Generate for specific packages
go run ./cmd/encx-gen generate ./models ./api

# Generate with custom config
go run ./cmd/encx-gen generate -config=my-config.yaml .

# Verbose output
go run ./cmd/encx-gen generate -v .

# Dry run (show what would be generated)
go run ./cmd/encx-gen generate -dry-run .
```

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

# Generate code
./bin/encx-gen generate -v ./models

# Generate for multiple packages
./bin/encx-gen generate -v ./models ./api ./internal
```

#### 2. Go Run (RELIABLE)

Run directly from source without building:

```bash
# Validate structs
go run ./cmd/encx-gen validate -v ./models

# Generate code
go run ./cmd/encx-gen generate -v ./models

# Generate for multiple packages
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
// ✅ GOOD - Clear separation of concerns
type User struct {
    // Business fields
    ID   int    `json:"id"`
    Name string `json:"name"`

    // Sensitive fields with encx tags
    Email string `json:"email" encx:"encrypt,hash_basic"`
    Phone string `json:"phone" encx:"encrypt"`
    SSN   string `json:"ssn" encx:"hash_secure"`

    // Companion fields grouped together
    EmailEncrypted []byte `json:"email_encrypted" db:"email_encrypted"`
    EmailHash      string `json:"email_hash" db:"email_hash"`
    PhoneEncrypted []byte `json:"phone_encrypted" db:"phone_encrypted"`
    SSNHashSecure  string `json:"ssn_hash_secure" db:"ssn_hash_secure"`

    // Essential encryption fields
    DEKEncrypted []byte `json:"dek_encrypted" db:"dek_encrypted"`
    KeyVersion   int    `json:"key_version" db:"key_version"`
    Metadata     string `json:"metadata" db:"metadata"`
}
```

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