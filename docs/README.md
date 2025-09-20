# ENCX Documentation

Welcome to the comprehensive documentation for ENCX, a production-ready Go library for field-level encryption, hashing, and key management.

## Documentation Structure

### Getting Started
- **[Main README](../README.md)** - Quick start, installation, and basic usage
- **[Examples](./EXAMPLES.md)** - Comprehensive examples for all use cases
- **[API Reference](./API.md)** - Complete API documentation

### Advanced Topics
- **[Migration Guide](./MIGRATION.md)** - Upgrading between versions
- **[Troubleshooting](./TROUBLESHOOTING.md)** - Common issues and solutions

## Quick Navigation

### I'm New to ENCX
1. Start with the [Main README](../README.md) for installation and basic usage
2. Browse [Examples](./EXAMPLES.md) to see practical implementations
3. Try the validation utility: `go run github.com/hengadev/encx/cmd/validate-tags -v`

### I Need Examples
- **[Basic Usage](./EXAMPLES.md#basic-examples)** - Simple encryption and hashing
- **[Combined Tags](./EXAMPLES.md#combined-tags)** - Encrypt AND hash the same field
- **[Real-world Use Cases](./EXAMPLES.md#real-world-use-cases)** - E-commerce, healthcare, finance
- **[Testing Examples](./EXAMPLES.md#testing-examples)** - Unit and integration testing

### I'm Upgrading
- **[Migration Guide](./MIGRATION.md)** - Step-by-step upgrade instructions
- **[Breaking Changes](./MIGRATION.md#breaking-changes-summary)** - What changed between versions

### I Have Issues
- **[Troubleshooting Guide](./TROUBLESHOOTING.md)** - Common problems and solutions
- **[Error Reference](./TROUBLESHOOTING.md#common-errors)** - Specific error messages and fixes
- **[Debugging Techniques](./TROUBLESHOOTING.md#debugging-techniques)** - How to debug issues

### I Need API Details
- **[API Reference](./API.md)** - Complete function documentation
- **[Error Types](./API.md#error-types)** - All error types and meanings
- **[Configuration Options](./API.md#configuration-options)** - All available options

## Key Features Covered

### 🔒 Field-level Encryption
- AES-GCM encryption with automatic key management
- Support for all Go data types
- Streaming encryption for large data

### 🔑 Secure Hashing  
- Argon2id for password hashing
- SHA-256 for fast lookups
- Pepper support for enhanced security

### 🏷️ Combined Tags
- **New in v1.2.x**: `encx:"encrypt,hash_basic"`
- Process the same field with multiple operations
- Perfect for user lookups with privacy protection

### ✅ Validation & Testing
- Compile-time struct tag validation
- Runtime struct validation
- Comprehensive testing utilities with mocks

### 🔄 Key Management
- Automatic DEK/KEK architecture
- Key rotation with version support
- Multiple KMS provider support

## Code Examples by Use Case

### Basic User Management
```go
type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte
    Email            string `encx:"hash_basic"`
    EmailHash        string
    Password         string `encx:"hash_secure"`
    PasswordHash     string
    
    DEK              []byte
    DEKEncrypted     []byte
    KeyVersion       int
}
```

### Advanced: Email with Lookup & Privacy
```go
type User struct {
    // Encrypt for privacy + hash for lookups
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte // Store securely
    EmailHash         string // Fast user lookup
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}
```

### Testing Setup
```go
func TestUserService(t *testing.T) {
    // Option 1: Integration testing
    crypto, _ := encx.NewTestCrypto(t)
    
    // Option 2: Unit testing with mocks
    mock := encx.NewCryptoServiceMock()
    mock.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
}
```

## Validation Tools

### Compile-time Validation
```bash
# Check all files
go run github.com/hengadev/encx/cmd/validate-tags -v

# Check specific patterns
go run github.com/hengadev/encx/cmd/validate-tags -pattern="user*.go"
```

### Runtime Validation
```go
// Validate struct definition
if err := encx.ValidateStruct(&user); err != nil {
    log.Fatalf("Invalid struct: %v", err)
}
```

## Best Practices Summary

1. **Structure Design**
   - Always include required fields: `DEK`, `DEKEncrypted`, `KeyVersion`
   - Use combined tags strategically: `encx:"encrypt,hash_basic"`
   - Provide companion fields with correct types

2. **Security**
   - Store pepper separately from database
   - Use proper KMS services in production
   - Implement regular key rotation

3. **Testing**
   - Use validation tools during development
   - Mock for unit tests, real crypto for integration tests
   - Test with various data types and edge cases

4. **Performance**
   - Process structs in batches for large datasets
   - Tune Argon2 parameters based on your security/performance needs
   - Monitor KMS API calls and database performance

## Version Compatibility

| Version | Status | Combined Tags | Validation | Docs |
|---------|--------|---------------|------------|------|
| v1.2.x  | ✅ Current | ✅ Yes | ✅ Full | ✅ Complete |
| v1.1.x  | ✅ Maintained | ❌ No | ⚠️ Basic | ⚠️ Basic |
| v1.0.x  | ❌ EOL | ❌ No | ❌ No | ❌ Minimal |

## Getting Help

### Resources
- 📚 **Documentation**: You're here! Browse the guides above
- 💡 **Examples**: Check `examples/` directory for working code
- 🔧 **Tools**: Use the validation utility for struct checking

### Community
- 🐛 **Issues**: [GitHub Issues](https://github.com/hengadev/encx/issues) for bugs
- 💬 **Discussions**: [GitHub Discussions](https://github.com/hengadev/encx/discussions) for questions
- 🔒 **Security**: Report security issues privately

### Quick Reference Card

```go
// Required struct pattern
type MyStruct struct {
    Field          string `encx:"encrypt"`     // ← Your data
    FieldEncrypted []byte                     // ← Companion field
    
    HashField      string `encx:"hash_basic"` // ← Hash operation
    HashFieldHash  string                     // ← Hash companion
    
    ComboField     string `encx:"encrypt,hash_basic"` // ← Combined
    ComboFieldEncrypted []byte                        // ← Both companions
    ComboFieldHash      string                        // ← needed
    
    // Always required
    DEK            []byte // ← Generated automatically
    DEKEncrypted   []byte // ← Set automatically  
    KeyVersion     int    // ← Set automatically
}

// Usage pattern
user := &MyStruct{Field: "sensitive data"}
err := crypto.ProcessStruct(ctx, user)
// user.Field is now cleared
// user.FieldEncrypted contains encrypted data

// Later decrypt
err = crypto.DecryptStruct(ctx, user) 
// user.Field is restored
```

---

**Next Steps**: Start with the [Main README](../README.md) or jump to [Examples](./EXAMPLES.md) for hands-on learning!
