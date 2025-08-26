# encx Package Improvement Roadmap

## Overview
This document outlines a comprehensive improvement plan for the `encx` Go package based on code review and best practices analysis. The plan is organized into 5 phases with clear priorities and implementation details.

## Current State Assessment

### Strengths ✅
- **Security**: Proper cryptographic practices (AES-GCM, Argon2id, secure random generation)
- **Architecture**: Clean separation with good interface design (`CryptoService`, `KeyManagementService`)
- **Key Management**: DEK/KEK architecture with rotation support follows industry standards
- **Error Handling**: Comprehensive custom error types with structured messaging

### Issues Identified 🔧
- **File Size**: `crypto.go` (259 lines) and `process_struct.go` (282 lines) are too large
- **Naming**: Inconsistent casing between constants (`ENCRYPT` vs `camelCase`)
- **Testing**: Missing comprehensive test suite (noted in TODO)
- **Validation**: Struct tags only validated at runtime, not compile-time

---

## Phase 1: Code Organization & Standards 🗂️

### Priority: **IMMEDIATE**
### Estimated Effort: **2-3 days**

#### 1.1 File Restructuring

**Split `crypto.go` into:**
```
crypto/
├── dek.go          # DEK generation/encryption/decryption (GenerateDEK, EncryptDEK, DecryptDEKWithVersion)
├── encryption.go   # Data encryption/decryption + streaming (EncryptData, DecryptData, EncryptStream, DecryptStream)
├── hashing.go      # Basic & secure hashing (HashBasic, HashSecure, Compare* functions)
└── key_rotation.go # KEK rotation logic (RotateKEK, ensureInitialKEK)
```

**Split `process_struct.go` into:**
```
processor/
├── struct.go       # Main struct processing (ProcessStruct, processEmbeddedStruct)
├── field.go        # Individual field processing (processField)
├── validation.go   # Input validation (validateObjectForProcessing, validateDEKField)
└── embedded.go     # Embedded struct handling (processEmbeddedStruct improvements)
```

#### 1.2 Naming Standardization

**Replace inconsistent naming:**
```go
// Current inconsistencies → Proposed standard
const (
    // Tags (prefer lowercase with underscores)
    TagEncrypt    = "encrypt"     // was: ENCRYPT
    TagHashSecure = "hash_secure" // was: SECURE  
    TagHashBasic  = "hash_basic"  // was: BASIC
    
    // Fields (prefer PascalCase for exported, camelCase for internal)
    FieldKeyVersion   = "KeyVersion"    // was: VERSION_FIELD
    FieldDEK         = "DEK"            // was: DEK_FIELD
    SuffixEncrypted  = "Encrypted"      // was: ENCRYPTED_FIELD_SUFFIX
    SuffixHashed     = "Hash"           // was: HASHED_FIELD_SUFFIX
    
    // Internal constants (camelCase)
    defaultDBDirName  = ".encx"         // Keep consistent
    defaultDBFileName = "metadata.db"   // Keep consistent
)

// Rename functions for clarity
func NewCrypto(...) (*Crypto, error) // was: New()
```

#### 1.3 Package Structure Improvement
```
encx/
├── crypto/          # Core crypto operations
├── processor/       # Struct processing logic  
├── providers/       # KMS providers (existing)
├── internal/        # Internal utilities
│   ├── validation/  # Common validation functions
│   └── constants/   # Shared constants
├── cmd/            # CLI tools (new)
└── examples/       # Usage examples (existing)
```

---

## Phase 2: Compile-Time Validation ⚡

### Priority: **MEDIUM**
### Estimated Effort: **3-4 days**

#### 2.1 Static Analysis Tool
```go
// cmd/encx-lint/main.go
package main

import (
    "go/ast"
    "go/parser"
    "go/token"
)

// Tool that validates:
// - Struct tag syntax correctness
// - Required companion fields (Email requires EmailEncrypted)  
// - Field type compatibility (string can't be encrypted to int)
// - Missing DEK/KeyVersion fields in structs with encx tags
// - Invalid tag combinations

func main() {
    // Parse Go files in project
    // Extract structs with `encx` tags
    // Validate tag syntax and field relationships
    // Report errors with file:line references
}
```

**Usage:**
```bash
go install github.com/hengadev/encx/cmd/encx-lint
encx-lint ./...
```

#### 2.2 Code Generation Approach
```go
// cmd/encx-gen/main.go
//go:generate go run github.com/hengadev/encx/cmd/encx-gen

// Generates validation functions at build time:
// - func validateUserStructTags() error
// - Compile-time struct analysis
// - Generate type-safe tag constants
```

#### 2.3 Build-Time Reflection
```go
// validation_build.go
//go:build validate

package encx

import "reflect"

func init() {
    // Only runs when built with: go build -tags validate
    if err := validateAllRegisteredStructs(); err != nil {
        panic("Invalid struct tags detected: " + err.Error())
    }
}

func validateAllRegisteredStructs() error {
    // Use reflection to find all structs with encx tags
    // Validate at build time
    return nil
}
```

#### 2.4 IDE Integration
```json
// .vscode/settings.json
{
    "go.buildTags": "validate",
    "go.lintTool": "custom",
    "go.lintFlags": ["encx-lint"]
}
```

---

## Phase 3: Testing & Quality 🧪

### Priority: **CRITICAL**
### Estimated Effort: **5-7 days**

#### 3.1 Test Structure
```
test/
├── unit/
│   ├── crypto/
│   │   ├── dek_test.go           # DEK operations
│   │   ├── encryption_test.go    # Data encryption/decryption
│   │   ├── hashing_test.go       # Hash functions
│   │   └── key_rotation_test.go  # Key rotation
│   ├── processor/
│   │   ├── struct_test.go        # Struct processing
│   │   ├── field_test.go         # Field processing
│   │   ├── validation_test.go    # Input validation
│   │   └── embedded_test.go      # Embedded structs
│   └── errors_test.go            # Error scenarios
├── integration/
│   ├── end_to_end_test.go        # Full encrypt/decrypt flows
│   ├── provider_test.go          # KMS provider integration
│   ├── concurrency_test.go       # Race conditions & thread safety
│   └── streaming_test.go         # Large file streaming
├── benchmarks/
│   ├── crypto_bench_test.go      # Crypto operation performance
│   ├── struct_bench_test.go      # Struct processing performance
│   └── memory_bench_test.go      # Memory usage analysis
└── testdata/
    ├── fixtures/                 # Test struct definitions
    │   ├── simple_struct.go
    │   ├── complex_struct.go
    │   └── embedded_struct.go
    ├── golden/                   # Expected outputs
    └── certificates/             # Test certificates for KMS
```

#### 3.2 Test Coverage Targets
```go
// Minimum coverage requirements:
// - Unit tests: 90%+ coverage
// - Integration tests: All happy paths + major error scenarios
// - Benchmark tests: All crypto operations

// Property-based testing example:
func TestEncryptionRoundTrip(t *testing.T) {
    quick.Check(func(data []byte) bool {
        if len(data) == 0 { return true }
        
        crypto := setupTestCrypto(t)
        dek, _ := crypto.GenerateDEK()
        
        encrypted, err := crypto.EncryptData(context.Background(), data, dek)
        if err != nil { return false }
        
        decrypted, err := crypto.DecryptData(context.Background(), encrypted, dek)
        if err != nil { return false }
        
        return bytes.Equal(data, decrypted)
    }, nil)
}
```

#### 3.3 Test Utilities
```go
// testutil/
├── crypto.go      # Test crypto setup helpers
├── fixtures.go    # Common test data
├── assertions.go  # Custom test assertions
└── mocks/         # Mock implementations
    ├── kms.go     # Mock KMS service
    └── db.go      # Mock database
```

#### 3.4 CI/CD Integration
```yaml
# .github/workflows/test.yml
name: Test Suite
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go run ./cmd/encx-lint ./...
      - run: go test -bench=. -benchmem ./test/benchmarks/
```

---

## Phase 4: Enhanced Documentation 📚

### Priority: **MEDIUM**
### Estimated Effort: **2-3 days**

#### 4.1 Comprehensive Examples
```go
// examples/
├── 01_basic_usage/
│   ├── main.go              # Simple encrypt/decrypt example
│   ├── user_struct.go       # Example struct definition
│   └── README.md           # Step-by-step guide
├── 02_custom_providers/
│   ├── vault_example.go     # HashiCorp Vault integration
│   ├── aws_kms_example.go   # AWS KMS integration  
│   ├── azure_kv_example.go  # Azure Key Vault integration
│   └── mock_provider.go     # Testing with mock provider
├── 03_advanced/
│   ├── key_rotation.go      # Key rotation workflows
│   ├── streaming.go         # Large file encryption
│   ├── embedded_structs.go  # Complex nested structures
│   ├── concurrent_usage.go  # Thread-safe operations
│   └── custom_serializer.go # Custom serialization
├── 04_patterns/
│   ├── repository_pattern.go # Using with repositories
│   ├── middleware.go        # HTTP middleware example
│   └── batch_processing.go  # Bulk operations
└── 05_migration/
    ├── key_migration.go     # Migrating between key versions
    └── data_migration.go    # Migrating encrypted data
```

#### 4.2 API Documentation Enhancement
```go
// Comprehensive godoc comments for all public APIs

// ProcessStruct encrypts, hashes, and processes fields in a struct based on `encx` tags.
//
// Supported tags:
//   - encrypt: AES-GCM encryption, requires companion *Encrypted field
//   - hash_secure: Argon2id hashing with pepper, requires companion *Hash field  
//   - hash_basic: SHA-256 hashing, requires companion *Hash field
//
// Required struct fields:
//   - DEK []byte: Data Encryption Key (auto-generated if nil)
//   - DEKEncrypted []byte: Encrypted DEK (set automatically)
//   - KeyVersion int: KEK version used (set automatically)
//
// Example:
//   type User struct {
//       Email        string `encx:"hash_basic"`
//       EmailHash    string
//       Password     string `encx:"hash_secure"`  
//       PasswordHash string
//       Address      string `encx:"encrypt"`
//       AddressEncrypted []byte
//       DEK          []byte
//       DEKEncrypted []byte
//       KeyVersion   int
//   }
//
//   user := &User{Email: "test@example.com", Password: "secret", Address: "123 Main St"}
//   if err := crypto.ProcessStruct(ctx, user); err != nil {
//       return fmt.Errorf("encryption failed: %w", err)
//   }
//   // user.EmailHash, user.PasswordHash, user.AddressEncrypted are now populated
func (c *Crypto) ProcessStruct(ctx context.Context, object any) error
```

#### 4.3 Documentation Website
```markdown
docs/
├── README.md                 # Getting started guide
├── SECURITY.md              # Security considerations
├── PERFORMANCE.md           # Performance characteristics  
├── MIGRATION.md             # Migration guides
├── TROUBLESHOOTING.md       # Common issues
├── API_REFERENCE.md         # Complete API documentation
└── CONTRIBUTING.md          # Development guide
```

#### 4.4 Interactive Documentation
```go
// Use go-swagger or similar to generate interactive API docs
// Add runnable examples in documentation
// Create playground for testing struct tag combinations
```

---

## Phase 5: Advanced Features 🚀

### Priority: **FUTURE**
### Estimated Effort: **1-2 weeks**

#### 5.1 Enhanced Input Validation
```go
// Validate all parameters at construction time
func NewCrypto(ctx context.Context, opts ...Option) (*Crypto, error) {
    config := &Config{}
    for _, opt := range opts {
        if err := opt(config); err != nil {
            return nil, fmt.Errorf("invalid option: %w", err)
        }
    }
    
    // Comprehensive validation
    if err := validateConfig(config); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    return &Crypto{config: config}, nil
}

func validateConfig(c *Config) error {
    if err := validateArgon2Params(c.Argon2); err != nil {
        return fmt.Errorf("argon2 params: %w", err)
    }
    if err := validateKMSConfig(c.KMS); err != nil {
        return fmt.Errorf("kms config: %w", err)
    }
    // Additional validation...
    return nil
}
```

#### 5.2 Performance Optimizations
```go
// Connection pooling for KMS operations
type CryptoPool struct {
    pool    chan *Crypto
    factory func() (*Crypto, error)
    maxSize int
}

func (p *CryptoPool) Get() (*Crypto, error) {
    select {
    case crypto := <-p.pool:
        return crypto, nil
    default:
        return p.factory()
    }
}

// Async batch operations
func (c *Crypto) ProcessStructBatch(ctx context.Context, objects []any) error {
    // Process multiple structs concurrently
    // Use worker pools for CPU-bound operations
}

// Streaming encryption for large data
func (c *Crypto) EncryptReader(ctx context.Context, r io.Reader, dek []byte) io.Reader {
    // Return streaming cipher reader
}
```

#### 5.3 Monitoring & Observability
```go
// Metrics interface
type Metrics interface {
    IncEncryptions(algorithm string)
    IncDecryptions(algorithm string)  
    IncHashOperations(algorithm string)
    RecordDuration(operation string, duration time.Duration)
    RecordError(operation string, error error)
}

// OpenTelemetry integration
func (c *Crypto) ProcessStruct(ctx context.Context, object any) error {
    span := trace.SpanFromContext(ctx)
    defer span.End()
    
    // Add tracing and metrics
    start := time.Now()
    err := c.processStruct(ctx, object)
    c.metrics.RecordDuration("process_struct", time.Since(start))
    if err != nil {
        c.metrics.RecordError("process_struct", err)
        span.SetStatus(codes.Error, err.Error())
    }
    return err
}
```

#### 5.4 Advanced Key Management
```go
// Multi-region key support
type MultiRegionCrypto struct {
    regions map[string]*Crypto
    primary string
}

// Key escrow and recovery
type KeyEscrow interface {
    BackupKey(ctx context.Context, keyID string) error
    RecoverKey(ctx context.Context, keyID string) ([]byte, error)
}

// Compliance features
type ComplianceManager struct {
    auditLog    AuditLogger
    retention   RetentionPolicy
    permissions PermissionChecker
}
```

---

## Implementation Timeline

### Sprint 1 (Week 1): Foundation
- **Phase 1**: Complete file restructuring and naming standardization
- **Phase 3**: Set up basic test structure and achieve 70% unit test coverage

### Sprint 2 (Week 2): Quality & Validation  
- **Phase 2**: Implement static analysis tool and compile-time validation
- **Phase 3**: Complete comprehensive test suite (90% coverage)

### Sprint 3 (Week 3): Documentation & Polish
- **Phase 4**: Create comprehensive documentation and examples
- **Phase 3**: Add benchmarks and performance testing

### Sprint 4 (Week 4): Advanced Features (Optional)
- **Phase 5**: Implement selected advanced features based on priorities
- Final testing and optimization

---

## Success Metrics

### Code Quality
- [ ] All files under 200 lines
- [ ] Consistent naming throughout codebase
- [ ] 90%+ test coverage
- [ ] Zero linting warnings
- [ ] All examples working and documented

### Developer Experience
- [ ] Compile-time struct tag validation
- [ ] Comprehensive error messages with context
- [ ] Clear documentation with examples
- [ ] Easy setup and getting started guide

### Performance
- [ ] Benchmarks for all crypto operations
- [ ] Memory usage profiling
- [ ] Concurrent operation safety verified
- [ ] Large file streaming support tested

### Security
- [ ] Security audit checklist completed
- [ ] All cryptographic best practices followed
- [ ] Key rotation workflows tested
- [ ] Error handling doesn't leak sensitive data

---

## Notes

This roadmap represents a comprehensive improvement plan that would transform the `encx` package from a good utility into an excellent, production-ready library. The phases are prioritized based on impact and dependencies, with Phase 1 and Phase 3 being most critical for immediate improvement.

Each phase builds upon the previous ones, ensuring steady progress toward a more maintainable, testable, and user-friendly codebase.
