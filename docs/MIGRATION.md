# ENCX Migration Guide

This guide helps you migrate between different versions of ENCX and provides information about breaking changes.

## Table of Contents

- [Version 1.1.x to 1.2.x](#version-11x-to-12x)
- [Breaking Changes Summary](#breaking-changes-summary)
- [Struct Tag Changes](#struct-tag-changes)
- [API Changes](#api-changes)
- [Testing Changes](#testing-changes)
- [Migration Steps](#migration-steps)

## Version 1.1.x to 1.2.x

### New Features in 1.2.x

- **Combined Tags**: Support for comma-separated tags like `encx:"encrypt,hash_basic"`
- **Compile-time Validation**: New validation utility for struct tags
- **Enhanced Error Messages**: Better error context and actionable guidance
- **Improved Testing**: Comprehensive testing utilities and examples
- **Better Documentation**: Extensive examples and API documentation

### Backward Compatibility

✅ **Good News**: Version 1.2.x is **fully backward compatible** with 1.1.x

- All existing struct tags continue to work without changes
- All existing API calls remain the same
- No migration required for basic usage

### Optional Enhancements

While not required, you may want to take advantage of new features:

#### 1. Use Combined Tags (Optional)

**Before (v1.1.x)**:
```go
type User struct {
    Email             string `encx:"hash_basic"`
    EmailHash         string
    Password          string `encx:"encrypt"`
    PasswordEncrypted []byte
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}
```

**After (v1.2.x with combined tags)**:
```go
type User struct {
    // Now you can encrypt AND hash the same field
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte // For secure storage
    EmailHash         string // For fast lookups
    
    Password          string `encx:"hash_secure,encrypt"`
    PasswordHash      string // For authentication
    PasswordEncrypted []byte // For recovery
    
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}
```

#### 2. Add Validation (Recommended)

**New in v1.2.x**: Compile-time validation

Add to your CI/CD pipeline:
```bash
# Validate struct tags in your codebase
go run github.com/hengadev/encx/cmd/validate-tags -v
```

Add to your code for runtime validation:
```go
// Validate during development
func init() {
    if err := encx.ValidateStruct(&User{}); err != nil {
        panic(fmt.Sprintf("Invalid User struct: %v", err))
    }
}
```

#### 3. Enhanced Testing (Recommended)

**Before (v1.1.x)**:
```go
func TestUserEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    // Basic testing
}
```

**After (v1.2.x with enhanced testing)**:
```go
func TestUserEncryption(t *testing.T) {
    // Option 1: Use improved test utilities
    crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
        Pepper: []byte("test-pepper-exactly-32-bytes!!"),
    })
    
    // Option 2: Use mocks for unit tests
    mockCrypto := encx.NewCryptoServiceMock()
    mockCrypto.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
    
    // Your existing test code remains the same
}
```

## Breaking Changes Summary

### None for v1.1.x → v1.2.x

There are **no breaking changes** in version 1.2.x. All v1.1.x code continues to work without modification.

### Future Breaking Changes (v2.0.x - Planned)

Future major versions may include breaking changes. We'll provide migration guides when they're released.

## Struct Tag Changes

### Tag Syntax Evolution

| Version | Syntax | Example | Status |
|---------|--------|---------|--------|
| v1.0.x | Single tags only | `encx:"encrypt"` | ✅ Supported |
| v1.1.x | Single tags only | `encx:"hash_secure"` | ✅ Supported |
| v1.2.x | Single + Combined | `encx:"encrypt,hash_basic"` | ✅ New Feature |

### Tag Validation

| Version | Validation | Method |
|---------|------------|--------|
| v1.0.x | Runtime only | Manual checks |
| v1.1.x | Runtime only | Manual checks |
| v1.2.x | Runtime + Compile-time | `ValidateStruct()` + validation utility |

## API Changes

### Constants Naming (Internal Change)

**Background**: Internal constant naming was standardized in v1.2.x, but this doesn't affect public API.

**Before (internal)**:
```go
const (
    ENCRYPT_TAG = "encrypt"  // Old internal naming
    DEK_FIELD   = "DEK"      // Old internal naming
)
```

**After (internal)**:
```go
const (
    TagEncrypt = "encrypt"   // New internal naming
    FieldDEK   = "DEK"       // New internal naming
)
```

**Impact**: None - these were internal constants. Your code is unaffected.

### Enhanced Error Messages

**Before (v1.1.x)**:
```
Error: missing field EmailEncrypted
```

**After (v1.2.x)**:
```
Error: encryption requires companion field: 'EmailEncrypted' field must exist to store encrypted data for field 'Email'. Add 'EmailEncrypted []byte' to your struct
```

**Impact**: Better debugging experience, but error handling code remains the same.

## Testing Changes

### Test Utilities Evolution

| Version | Test Creation | Mocking |
|---------|---------------|---------|
| v1.0.x | Basic test crypto | Manual mocks |
| v1.1.x | Basic test crypto | Manual mocks |
| v1.2.x | Enhanced test utilities | Built-in mock support |

### Migration to Enhanced Testing

**Old approach (still works)**:
```go
func TestUserService(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    
    user := &User{Name: "Test"}
    err := crypto.ProcessStruct(context.Background(), user)
    assert.NoError(t, err)
}
```

**New enhanced approach (optional)**:
```go
func TestUserService(t *testing.T) {
    // Option 1: Enhanced configuration
    crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
        Pepper: []byte("custom-test-pepper-32-bytes!"),
    })
    
    // Option 2: Use mocks for faster unit tests
    mock := encx.NewCryptoServiceMock()
    mock.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
    
    // Your existing test logic remains unchanged
}
```

## Migration Steps

### Step 1: Update Dependency

```bash
go get github.com/hengadev/encx@v1.2.x
go mod tidy
```

### Step 2: Verify Compatibility

Run your existing tests to ensure everything still works:

```bash
go test ./...
```

### Step 3: Add Validation (Recommended)

Add struct validation to catch issues early:

```go
// Add to your main package or init functions
if err := encx.ValidateStruct(&User{}); err != nil {
    log.Fatalf("Struct validation failed: %v", err)
}
```

### Step 4: Update CI/CD (Recommended)

Add tag validation to your build process:

```yaml
# .github/workflows/ci.yml
- name: Validate ENCX struct tags
  run: go run github.com/hengadev/encx/cmd/validate-tags -v
```

### Step 5: Consider Combined Tags (Optional)

Evaluate if combined tags would benefit your use case:

```go
// Before: Two separate operations
Email    string `encx:"hash_basic"`
EmailHash string
Password string `encx:"encrypt"`
PasswordEncrypted []byte

// After: Combined operations where it makes sense
Email    string `encx:"encrypt,hash_basic"`
EmailEncrypted []byte
EmailHash string
```

### Step 6: Enhance Tests (Optional)

Update your test suite to use new testing utilities:

```go
// Add mock tests for faster unit testing
func TestUserService_Unit(t *testing.T) {
    mock := encx.NewCryptoServiceMock()
    mock.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
    
    service := NewUserService(mock)
    err := service.CreateUser("test@example.com")
    
    assert.NoError(t, err)
    mock.AssertExpectations(t)
}

// Keep integration tests with real crypto
func TestUserService_Integration(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    service := NewUserService(crypto)
    
    err := service.CreateUser("test@example.com")
    assert.NoError(t, err)
}
```

## Rollback Plan

If you need to rollback to v1.1.x:

```bash
go get github.com/hengadev/encx@v1.1.x
go mod tidy
```

**Note**: Rollback is safe since v1.2.x doesn't change data formats or database schemas.

## Common Issues and Solutions

### Issue: Validation Utility Not Found

**Problem**:
```
go run github.com/hengadev/encx/cmd/validate-tags -v
# Error: cannot find package
```

**Solution**:
Make sure you're using v1.2.x and the utility is installed:
```bash
go install github.com/hengadev/encx/cmd/validate-tags@latest
validate-tags -v
```

### Issue: Combined Tag Validation Errors

**Problem**:
```
Error: invalid encx tag 'encrypt,hash_basic'
```

**Solution**:
This occurs if you're mixing v1.2.x struct tags with v1.1.x library. Ensure you're using v1.2.x:
```bash
go get github.com/hengadev/encx@v1.2.x
```

### Issue: New Error Message Format

**Problem**: Your error parsing breaks due to enhanced error messages.

**Solution**: Update error handling to be more flexible:

**Before**:
```go
if err != nil && strings.Contains(err.Error(), "missing field") {
    // Handle missing field
}
```

**After**:
```go
if err != nil && (strings.Contains(err.Error(), "missing field") || 
                  strings.Contains(err.Error(), "requires companion field")) {
    // Handle missing field (more flexible)
}
```

## Getting Help

### Resources

- **Documentation**: Check the updated [README.md](../README.md) and [docs/](../) folder
- **Examples**: See [docs/EXAMPLES.md](./EXAMPLES.md) for comprehensive usage examples
- **API Reference**: See [docs/API.md](./API.md) for complete API documentation

### Support Channels

- **Issues**: Report bugs or ask questions on GitHub Issues
- **Discussions**: Join community discussions on GitHub Discussions
- **Security**: Report security issues privately to the maintainers

### Version Support

| Version | Support Status | End of Life |
|---------|----------------|-------------|
| v1.2.x | ✅ Active development | TBD |
| v1.1.x | ✅ Critical fixes only | 6 months after v1.2.x |
| v1.0.x | ❌ No longer supported | N/A |

## Best Practices for Future Migrations

1. **Pin Dependencies**: Use exact versions in production
2. **Test Early**: Test pre-release versions in staging environments
3. **Read Changelogs**: Always review CHANGELOG.md before upgrading
4. **Gradual Rollout**: Migrate incrementally in large applications
5. **Backup Data**: Always backup before major version upgrades
6. **Validation**: Use built-in validation tools before and after migration
