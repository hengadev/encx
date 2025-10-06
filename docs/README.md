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
- Type-safe generated functions
- Comprehensive testing utilities

### 🔄 Key Management
- Automatic DEK/KEK architecture
- Key rotation with version support
- Multiple KMS provider support

## Code Examples by Use Case

### Basic User Management
```go
//go:generate encx-gen generate .

// Clean struct definition - no companion fields needed
type User struct {
    Name     string `encx:"encrypt"`
    Email    string `encx:"hash_basic"`
    Password string `encx:"hash_secure"`
}

// Usage
user := &User{Name: "John", Email: "john@example.com"}
userEncx, err := ProcessUserEncx(ctx, crypto, user)
// userEncx.NameEncrypted, EmailHash, PasswordHash are auto-generated
```

### Advanced: Email with Lookup & Privacy
```go
//go:generate encx-gen generate .

type User struct {
    // Encrypt for privacy + hash for lookups
    Email string `encx:"encrypt,hash_basic"`
}

// Usage
user := &User{Email: "user@example.com"}
userEncx, err := ProcessUserEncx(ctx, crypto, user)
// Generated: userEncx.EmailEncrypted (for storage)
//           userEncx.EmailHash (for lookups)
```

### Testing Setup
```go
func TestUserEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)

    user := &User{Email: "test@example.com"}
    userEncx, err := ProcessUserEncx(ctx, crypto, user)

    assert.NoError(t, err)
    assert.NotEmpty(t, userEncx.EmailEncrypted)
}
```

## Validation Tools

### Compile-time Validation
```bash
# Validate all files in current directory
encx-gen validate -v .

# Validate specific packages
encx-gen validate -v ./models ./api

# Validation runs automatically before generation
encx-gen generate -v .
```

## Best Practices Summary

1. **Structure Design**
   - No companion fields needed - automatically generated
   - Use combined tags strategically: `encx:"encrypt,hash_basic"`
   - Run code generation: `go generate ./...`

2. **Security**
   - Store pepper separately from database
   - Use proper KMS services in production
   - Implement regular key rotation

3. **Testing**
   - Use validation tools during development: `encx-gen validate`
   - Test with generated functions for type safety
   - Test with various data types and edge cases

4. **Performance**
   - Use code generation for best performance
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
//go:generate encx-gen generate .

// Source struct - clean and simple (no companion fields!)
type MyStruct struct {
    Field      string `encx:"encrypt"`                // Encrypted field
    HashField  string `encx:"hash_basic"`             // Hashed field
    ComboField string `encx:"encrypt,hash_basic"`     // Both operations
}

// Generated MyStructEncx struct (automatic):
// type MyStructEncx struct {
//     FieldEncrypted      []byte
//     HashFieldHash       string
//     ComboFieldEncrypted []byte
//     ComboFieldHash      string
//     DEKEncrypted        []byte
//     KeyVersion          int
//     Metadata            string
// }

// Usage pattern
data := &MyStruct{Field: "sensitive data"}
dataEncx, err := ProcessMyStructEncx(ctx, crypto, data)
// dataEncx contains all encrypted/hashed fields

// Later decrypt
decrypted, err := DecryptMyStructEncx(ctx, crypto, dataEncx)
// decrypted.Field is restored
```

---

**Next Steps**: Start with the [Main README](../README.md) or jump to [Examples](./EXAMPLES.md) for hands-on learning!
