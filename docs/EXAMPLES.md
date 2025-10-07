# ENCX Examples

This document provides comprehensive examples for using the ENCX library in various scenarios.

## Table of Contents

- [Basic Examples](#basic-examples)
- [Combined Tags](#combined-tags)
- [Real-world Use Cases](#real-world-use-cases)
- [Testing Examples](#testing-examples)
- [Advanced Patterns](#advanced-patterns)

## Basic Examples

### Simple User Struct

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/hengadev/encx"
)

type User struct {
    // Basic encryption - for sensitive data that needs to be retrievable
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte
    
    // Basic hashing - for fast lookups, non-sensitive data
    Email            string `encx:"hash_basic"`
    EmailHash        string
    
    // Secure hashing - for passwords and sensitive identifiers
    Password         string `encx:"hash_secure"`
    PasswordHash     string
    
    // Required ENCX fields
    DEK              []byte // Data Encryption Key
    DEKEncrypted     []byte // Encrypted DEK
    KeyVersion       int    // Key version for rotation
}

func main() {
    // Create test crypto instance
    crypto, _ := encx.NewTestCrypto(nil)
    
    // Create user with plaintext data
    user := &User{
        Name:     "Alice Smith",
        Email:    "alice@example.com",
        Password: "securePassword123",
    }
    
    fmt.Printf("Before processing:\n")
    fmt.Printf("  Name: %s\n", user.Name)
    fmt.Printf("  Email: %s\n", user.Email)
    fmt.Printf("  Password: %s\n", user.Password)
    
    // Process the struct
    ctx := context.Background()
    if err := crypto.ProcessStruct(ctx, user); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("\nAfter processing:\n")
    fmt.Printf("  Name: '%s' (cleared)\n", user.Name)
    fmt.Printf("  NameEncrypted: %d bytes\n", len(user.NameEncrypted))
    fmt.Printf("  EmailHash: %s\n", user.EmailHash)
    fmt.Printf("  PasswordHash: %s\n", user.PasswordHash[:50]+"...")
    
    // Decrypt when needed
    if err := crypto.DecryptStruct(ctx, user); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("\nAfter decryption:\n")
    fmt.Printf("  Name: %s (restored)\n", user.Name)
}
```

## Combined Tags

### Email with Lookup and Privacy

Perfect for user management systems where you need both fast lookups and privacy protection:

```go
type User struct {
    // Email: encrypt for privacy + hash for lookups
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte // Stored in database
    EmailHash         string // Used for user lookups
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

func ExampleEmailLookup() {
    crypto, _ := encx.NewTestCrypto(nil)
    ctx := context.Background()
    
    // Process user
    user := &User{Email: "user@example.com"}
    crypto.ProcessStruct(ctx, user)
    
    // Later: find user by email
    searchEmail := "user@example.com"
    searchHash := crypto.HashBasic(ctx, []byte(searchEmail))
    
    if searchHash == user.EmailHash {
        fmt.Println("User found!")
        
        // Decrypt email for display
        crypto.DecryptStruct(ctx, user)
        fmt.Printf("User email: %s\n", user.Email)
    }
}
```

### Password with Authentication and Recovery

Secure password handling with both authentication and recovery capabilities:

```go
type User struct {
    // Password: secure hash for auth + encryption for recovery
    Password          string `encx:"hash_secure,encrypt"`
    PasswordHash      string // For login verification (Argon2id)
    PasswordEncrypted []byte // For password recovery scenarios
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}

func (u *User) CheckPassword(crypto *encx.Crypto, plaintext string) (bool, error) {
    ctx := context.Background()
    return crypto.CompareSecureHashAndValue(ctx, plaintext, u.PasswordHash)
}

func (u *User) RecoverPassword(crypto *encx.Crypto) (string, error) {
    ctx := context.Background()
    
    // Temporarily decrypt to get original password
    if err := crypto.DecryptStruct(ctx, u); err != nil {
        return "", err
    }
    
    password := u.Password
    
    // Clear the plaintext immediately
    u.Password = ""
    
    return password, nil
}
```

## Real-world Use Cases

### E-commerce Customer Data

```go
type Customer struct {
    // PII data - encrypt for privacy
    FirstName            string `encx:"encrypt"`
    FirstNameEncrypted   []byte
    LastName             string `encx:"encrypt"`
    LastNameEncrypted    []byte
    
    // Contact info - both searchable and private
    Email                string `encx:"encrypt,hash_basic"`
    EmailEncrypted       []byte
    EmailHash            string
    
    Phone                string `encx:"encrypt,hash_basic"`
    PhoneEncrypted       []byte
    PhoneHash            string
    
    // Address - encrypt for privacy
    Address              Address `encx:"encrypt"`
    AddressEncrypted     []byte
    
    // Account info - hash only (for deduplication)
    TaxID                string `encx:"hash_basic"`
    TaxIDHash            string
    
    // Required fields
    DEK                  []byte
    DEKEncrypted         []byte
    KeyVersion           int
}

type Address struct {
    Street    string
    City      string
    State     string
    ZipCode   string
    Country   string
}
```

### Healthcare Patient Records

```go
type Patient struct {
    // Patient identifiers - hash for fast lookup
    MedicalRecordNumber  string `encx:"hash_basic"`
    MedicalRecordHash    string
    
    // PII - encrypt for HIPAA compliance
    FirstName            string `encx:"encrypt"`
    FirstNameEncrypted   []byte
    LastName             string `encx:"encrypt"`
    LastNameEncrypted    []byte
    
    // Contact - searchable and private
    Email                string `encx:"encrypt,hash_basic"`
    EmailEncrypted       []byte
    EmailHash            string
    
    // SSN - highly sensitive, encrypt only
    SSN                  string `encx:"encrypt"`
    SSNEncrypted         []byte
    
    // Medical data - encrypt for privacy
    Diagnosis            []string `encx:"encrypt"`
    DiagnosisEncrypted   []byte
    
    // Required fields
    DEK                  []byte
    DEKEncrypted         []byte
    KeyVersion           int
}
```

### Financial Services

```go
type Account struct {
    // Account number - hash for lookups
    AccountNumber        string `encx:"hash_basic"`
    AccountNumberHash    string
    
    // Routing info - encrypt for security
    RoutingNumber        string `encx:"encrypt"`
    RoutingNumberEncrypted []byte
    
    // Customer info - both searchable and private
    CustomerID           string `encx:"encrypt,hash_basic"`
    CustomerIDEncrypted  []byte
    CustomerIDHash       string
    
    // Balance info - encrypt for confidentiality
    Balance              decimal.Decimal `encx:"encrypt"`
    BalanceEncrypted     []byte
    
    // Required fields
    DEK                  []byte
    DEKEncrypted         []byte
    KeyVersion           int
}
```

## Testing Examples

### Unit Testing with Mocks

```go
package service

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/hengadev/encx"
)

type UserService struct {
    crypto encx.CryptoService
}

func NewUserService(crypto encx.CryptoService) *UserService {
    return &UserService{crypto: crypto}
}

func (s *UserService) CreateUser(email, password string) error {
    user := &User{
        Email:    email,
        Password: password,
    }
    
    return s.crypto.ProcessStruct(context.Background(), user)
}

func TestUserService_CreateUser(t *testing.T) {
    // Create mock
    mockCrypto := encx.NewCryptoServiceMock()
    mockCrypto.On("ProcessStruct", mock.Anything, mock.MatchedBy(func(user *User) bool {
        return user.Email == "test@example.com" && user.Password == "secret"
    })).Return(nil)
    
    // Test service
    service := NewUserService(mockCrypto)
    err := service.CreateUser("test@example.com", "secret")
    
    assert.NoError(t, err)
    mockCrypto.AssertExpectations(t)
}
```

### Integration Testing

```go
func TestUserService_Integration(t *testing.T) {
    // Create test crypto with real operations
    crypto, _ := encx.NewTestCrypto(t)
    service := NewUserService(crypto)
    
    // Test actual encryption/hashing
    err := service.CreateUser("integration@example.com", "testPassword")
    assert.NoError(t, err)
}
```

### Table-driven Tests

```go
func TestPasswordValidation(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    ctx := context.Background()
    
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid password", "validPassword123", false},
        {"empty password", "", false}, // Empty passwords are processed
        {"unicode password", "пароль123", false},
        {"very long password", strings.Repeat("a", 1000), false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            user := &User{Password: tt.password}
            err := crypto.ProcessStruct(ctx, user)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, user.PasswordHash)
                assert.NotEmpty(t, user.PasswordEncrypted)
            }
        })
    }
}
```

## Advanced Patterns

### Batch Processing

```go
func ProcessUsersBatch(crypto *encx.Crypto, users []*User) error {
    ctx := context.Background()
    
    for i, user := range users {
        if err := crypto.ProcessStruct(ctx, user); err != nil {
            return fmt.Errorf("failed to process user %d: %w", i, err)
        }
    }
    
    return nil
}
```

### Conditional Processing

```go
type Document struct {
    Title                string
    TitleEncrypted       []byte
    
    Content              string `encx:"encrypt"`
    ContentEncrypted     []byte
    
    // Only encrypt sensitive documents
    IsConfidential       bool
    
    DEK                  []byte
    DEKEncrypted         []byte
    KeyVersion           int
}

func ProcessDocument(crypto *encx.Crypto, doc *Document) error {
    ctx := context.Background()
    
    if doc.IsConfidential {
        // Encrypt title for confidential documents
        doc.Title = ""  // Will be processed as encrypt
        // Manually set the encx tag or use reflection to add it
    }
    
    return crypto.ProcessStruct(ctx, doc)
}
```

### Error Recovery

```go
func ProcessUserSafely(crypto *encx.Crypto, user *User) error {
    ctx := context.Background()
    
    // Save original values for rollback
    originalName := user.Name
    originalEmail := user.Email
    originalPassword := user.Password
    
    if err := crypto.ProcessStruct(ctx, user); err != nil {
        // Rollback on error
        user.Name = originalName
        user.Email = originalEmail
        user.Password = originalPassword
        
        return fmt.Errorf("encryption failed, rolled back: %w", err)
    }
    
    return nil
}
```

### Custom Validation

```go
func ValidateAndProcessUser(crypto *encx.Crypto, user *User) error {
    // Pre-validation
    if user.Email == "" {
        return fmt.Errorf("email is required")
    }
    
    if len(user.Password) < 8 {
        return fmt.Errorf("password must be at least 8 characters")
    }
    
    // Validate struct tags
    if err := encx.ValidateStruct(user); err != nil {
        return fmt.Errorf("struct validation failed: %w", err)
    }
    
    // Process
    ctx := context.Background()
    if err := crypto.ProcessStruct(ctx, user); err != nil {
        return fmt.Errorf("processing failed: %w", err)
    }
    
    // Post-validation
    if len(user.DEK) == 0 {
        return fmt.Errorf("DEK was not generated")
    }
    
    return nil
}
```

### Performance Monitoring

```go
func ProcessUserWithMetrics(crypto *encx.Crypto, user *User) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        log.Printf("User processing took: %v", duration)
    }()
    
    ctx := context.Background()
    return crypto.ProcessStruct(ctx, user)
}
```

## Running Examples

To run these examples:

1. Clone the repository
2. Navigate to the examples directory
3. Run individual examples:

```bash
go run basic_example.go
go run combined_tags_example.go
go run ecommerce_example.go
```

Or run all examples:

```bash
go run examples/*.go
```

## Best Practices Summary

1. **Use combined tags strategically** - only when you need both operations
2. **Validate early** - use `encx.ValidateStruct()` during development
3. **Handle errors gracefully** - provide meaningful error messages
4. **Test thoroughly** - use both unit tests with mocks and integration tests
5. **Monitor performance** - track encryption/decryption times in production
6. **Secure key management** - use proper KMS services in production
7. **Regular key rotation** - implement automated key rotation schedules
