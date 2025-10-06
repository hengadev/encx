// Package encx provides production-ready field-level encryption, hashing, and key management for Go applications.
//
// Context7 Metadata:
// - Library Type: Encryption & Security
// - Use Cases: Data protection, PII encryption, password hashing, searchable encryption
// - Complexity: Intermediate to Advanced
// - Performance: High (10x improvement with code generation)
// - Compliance: HIPAA, GDPR, SOX ready
// - Integration: PostgreSQL, MySQL, SQLite, AWS KMS, HashiCorp Vault
//
// ENCX enables you to encrypt and hash struct fields using simple struct tags, with automatic
// key management through a DEK/KEK architecture. It supports multiple KMS backends, key rotation,
// combined operations, and comprehensive testing utilities.
//
// # Key Features
//
//   - Field-level encryption with AES-GCM
//   - Secure hashing with Argon2id and basic SHA-256
//   - Combined operations - encrypt AND hash the same field
//   - Automatic key management with DEK/KEK architecture
//   - Key rotation support with version tracking
//   - Multiple KMS backends (AWS KMS, HashiCorp Vault, etc.)
//   - Comprehensive testing utilities and mocks
//   - Compile-time validation for struct tags
//
// # Quick Start
//
// Define your struct with encx tags:
//
//	type User struct {
//	    Name     string `encx:"encrypt"`
//	    Email    string `encx:"hash_basic"`
//	    Password string `encx:"hash_secure"`
//
//	    // No companion fields needed! Code generation creates separate output struct
//	}
//
// Generate type-safe functions (recommended approach):
//
//	//go:generate encx-gen generate .
//
// Create crypto instance and process with generated functions:
//
//	crypto, _ := encx.NewTestCrypto(nil)
//	user := &User{
//	    Name:     "John Doe",
//	    Email:    "john@example.com",
//	    Password: "secret123",
//	}
//
//	// Use generated type-safe functions
//	userEncx, err := ProcessUserEncx(ctx, crypto, user)
//	orderEncx, err := ProcessOrderEncx(ctx, crypto, order)
//	// Pattern: Process{YourStructName}Encx
//
// # Struct Tags
//
// Single operation tags:
//   - encx:"encrypt" - Encrypts field value
//   - encx:"hash_basic" - Creates SHA-256 hash for searchable indexing
//   - encx:"hash_secure" - Creates Argon2id hash with pepper (for passwords)
//
// Combined operation tags:
//   - encx:"encrypt,hash_basic" - Both encrypts AND hashes the field (searchable encryption)
//   - encx:"hash_secure,encrypt" - Secure hash for auth + encryption for recovery
//
// # Code Generation
//
// Code generation creates a separate {StructName}Encx struct with all encrypted/hashed fields:
//
//	// Your source struct
//	type User struct {
//	    Email string `encx:"encrypt,hash_basic"`
//	}
//
//	// Generated UserEncx struct (automatic)
//	type UserEncx struct {
//	    EmailEncrypted []byte
//	    EmailHash      string
//	    DEKEncrypted   []byte
//	    KeyVersion     int
//	    Metadata       string
//	}
//
// # Advanced Example: Combined Tags
//
// Perfect for user lookup with privacy protection:
//
//	type User struct {
//	    Email string `encx:"encrypt,hash_basic"`
//
//	    // No companion fields needed! Code generation creates:
//	    // - UserEncx.EmailEncrypted []byte (for secure storage)
//	    // - UserEncx.EmailHash string (for fast lookups)
//	}
//
//	// Usage
//	user := &User{Email: "user@example.com"}
//	userEncx, err := ProcessUserEncx(ctx, crypto, user)
//
//	// Now you can:
//	// 1. Store userEncx.EmailEncrypted securely in database
//	// 2. Use userEncx.EmailHash for fast user lookups
//	// 3. Decrypt with DecryptUserEncx when needed for display
//
// # Production Configuration
//
//	// With AWS KMS
//	crypto, err := encx.New(ctx,
//	    encx.WithKMSService(awsKMS),
//	    encx.WithDatabase(db),
//	    encx.WithPepper(pepper),
//	    encx.WithKEKAlias("myapp-kek"),
//	)
//
//	// With HashiCorp Vault
//	crypto, err := encx.New(ctx,
//	    encx.WithKMSService(vaultKMS),
//	    encx.WithDatabase(db),
//	    encx.WithPepper(pepper),
//	    encx.WithKEKAlias("myapp-kek"),
//	)
//
// # Validation
//
// Validate struct tags before generating code:
//
//	encx-gen validate -v .
//	encx-gen validate -v ./models ./api
//
// Validation runs automatically before generation:
//
//	encx-gen generate -v .
//
// # Error Handling
//
// ENCX provides structured error handling with sentinel errors for precise error classification:
//
//	user := &User{Name: "John", Email: "john@example.com"}
//	userEncx, err := ProcessUserEncx(ctx, crypto, user)
//	if err != nil {
//	    switch {
//	    case encx.IsRetryableError(err):
//	        // KMS or database temporarily unavailable - retry with backoff
//	        log.Warn("Retryable error: %v", err)
//	        return handleRetry(err)
//	    
//	    case encx.IsConfigurationError(err):
//	        // Invalid configuration - fix setup
//	        log.Error("Configuration error: %v", err)
//	        return handleConfigError(err)
//	    
//	    case encx.IsAuthError(err):
//	        // Authentication failed - check credentials
//	        log.Error("Authentication failed: %v", err)
//	        return handleAuthError(err)
//	    
//	    case encx.IsOperationError(err):
//	        // Encryption/decryption failed - check data/keys
//	        log.Error("Operation failed: %v", err)
//	        return handleOperationError(err)
//	    
//	    case encx.IsValidationError(err):
//	        // Data validation failed - check input
//	        log.Error("Validation error: %v", err)
//	        return handleValidationError(err)
//	    
//	    default:
//	        log.Error("Unknown error: %v", err)
//	        return err
//	    }
//	}
//
// Checking specific errors:
//
//	if errors.Is(err, encx.ErrKMSUnavailable) {
//	    // Implement retry logic
//	    return retryWithBackoff(operation)
//	}
//	
//	if errors.Is(err, encx.ErrAuthenticationFailed) {
//	    // Refresh credentials and retry
//	    return refreshAuthAndRetry(operation)
//	}
//
// # Testing
//
// Unit testing with generated functions:
//
//	func TestUserEncryption(t *testing.T) {
//	    crypto, _ := encx.NewTestCrypto(t)
//
//	    user := &User{Email: "test@example.com"}
//	    userEncx, err := ProcessUserEncx(ctx, crypto, user)
//
//	    assert.NoError(t, err)
//	    assert.NotEmpty(t, userEncx.EmailEncrypted)
//	}
//
// Integration testing with full cycle:
//
//	func TestUserEncryptDecryptCycle(t *testing.T) {
//	    crypto, _ := encx.NewTestCrypto(t)
//
//	    original := &User{Email: "test@example.com"}
//	    userEncx, err := ProcessUserEncx(ctx, crypto, original)
//	    assert.NoError(t, err)
//
//	    decrypted, err := DecryptUserEncx(ctx, crypto, userEncx)
//	    assert.NoError(t, err)
//	    assert.Equal(t, original.Email, decrypted.Email)
//	}
//
// # Documentation
//
// For comprehensive documentation, examples, and advanced usage:
//   - README.md - Complete getting started guide
//   - docs/EXAMPLES.md - Detailed examples for all use cases
//   - docs/API.md - Complete API reference
//   - docs/MIGRATION.md - Version upgrade guide
//   - docs/TROUBLESHOOTING.md - Common issues and solutions
//
// # Important: Version Control
//
// Add to your .gitignore:
//
//	.encx/
package encx

