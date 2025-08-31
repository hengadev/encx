// Package encx provides production-ready field-level encryption, hashing, and key management for Go applications.
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
//	    Name             string `encx:"encrypt"`
//	    NameEncrypted    []byte
//	    Email            string `encx:"hash_basic"`
//	    EmailHash        string
//	    Password         string `encx:"hash_secure"`
//	    PasswordHash     string
//
//	    // Required fields
//	    DEK              []byte
//	    DEKEncrypted     []byte
//	    KeyVersion       int
//	}
//
// Create crypto instance and process:
//
//	crypto, _ := encx.NewTestCrypto(nil)
//	user := &User{
//	    Name:     "John Doe",
//	    Email:    "john@example.com",
//	    Password: "secret123",
//	}
//
//	err := crypto.ProcessStruct(ctx, user)
//	// Original fields are now cleared/hashed
//	// Encrypted data stored in companion fields
//
// # Struct Tags
//
// Single operation tags:
//   - encx:"encrypt" - Encrypts field, stores in companion *Encrypted []byte field
//   - encx:"hash_basic" - SHA-256 hash, stores in companion *Hash string field
//   - encx:"hash_secure" - Argon2id hash with pepper, stores in companion *Hash string field
//
// Combined operation tags:
//   - encx:"encrypt,hash_basic" - Both encrypts AND hashes the field
//   - encx:"hash_secure,encrypt" - Secure hash for auth + encryption for recovery
//
// # Required Struct Fields
//
// Every struct must include these fields:
//
//	DEK          []byte  // Data Encryption Key (auto-generated)
//	DEKEncrypted []byte  // Encrypted DEK (set automatically)
//	KeyVersion   int     // Key version for rotation (set automatically)
//
// # Advanced Example: Combined Tags
//
// Perfect for user lookup with privacy protection:
//
//	type User struct {
//	    Email             string `encx:"encrypt,hash_basic"`
//	    EmailEncrypted    []byte // For secure storage
//	    EmailHash         string // For fast user lookups
//
//	    DEK               []byte
//	    DEKEncrypted      []byte
//	    KeyVersion        int
//	}
//
//	// Usage
//	user := &User{Email: "user@example.com"}
//	crypto.ProcessStruct(ctx, user)
//
//	// Now you can:
//	// 1. Store user.EmailEncrypted securely in database
//	// 2. Use user.EmailHash for fast user lookups
//	// 3. Decrypt user.Email when needed for display
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
// Compile-time validation:
//
//	go run github.com/hengadev/encx/cmd/validate-tags -v
//
// Runtime validation:
//
//	if err := encx.ValidateStruct(&user); err != nil {
//	    log.Fatalf("Invalid struct: %v", err)
//	}
//
// # Testing
//
// Unit testing with mocks:
//
//	func TestUserService(t *testing.T) {
//	    mockCrypto := encx.NewCryptoServiceMock()
//	    mockCrypto.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
//
//	    service := NewUserService(mockCrypto)
//	    err := service.CreateUser("test@example.com")
//	    assert.NoError(t, err)
//
//	    mockCrypto.AssertExpectations(t)
//	}
//
// Integration testing:
//
//	func TestUserServiceIntegration(t *testing.T) {
//	    crypto, _ := encx.NewTestCrypto(t)
//	    service := NewUserService(crypto)
//
//	    err := service.CreateUser("test@example.com")
//	    assert.NoError(t, err)
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

