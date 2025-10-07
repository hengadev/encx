# ENCX Troubleshooting Guide

Common issues, solutions, and debugging techniques for the ENCX library.

## Table of Contents

- [Common Errors](#common-errors)
- [Struct Configuration Issues](#struct-configuration-issues)
- [Tag Validation Problems](#tag-validation-problems)
- [Runtime Issues](#runtime-issues)
- [Performance Problems](#performance-problems)
- [Testing Issues](#testing-issues)
- [Debugging Techniques](#debugging-techniques)

## Common Errors

### Error: "ProcessStruct requires a pointer to a struct"

**Symptoms**:
```
Error: ProcessStruct requires a pointer to a struct, got User. 
Usage: crypto.ProcessStruct(ctx, &myStruct) not crypto.ProcessStruct(ctx, myStruct)
```

**Cause**: Passing a struct value instead of a pointer.

**Solution**:
```go
// ❌ Wrong - passing struct value
user := User{Name: "John"}
err := crypto.ProcessStruct(ctx, user)

// ✅ Correct - passing struct pointer
user := &User{Name: "John"}
err := crypto.ProcessStruct(ctx, user)

// ✅ Also correct
user := User{Name: "John"}
err := crypto.ProcessStruct(ctx, &user)
```

### Error: "missing required field: DEK"

**Symptoms**:
```
Error: struct validation errors:
missing required field: DEK
missing required field: DEKEncrypted
missing required field: KeyVersion
```

**Cause**: Struct is missing required ENCX fields.

**Solution**:
```go
// ❌ Wrong - missing required fields
type User struct {
    Name string `encx:"encrypt"`
    NameEncrypted []byte
}

// ✅ Correct - includes all required fields
type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte
    
    // Required ENCX fields
    DEK              []byte  // Data Encryption Key
    DEKEncrypted     []byte  // Encrypted DEK
    KeyVersion       int     // Key version for rotation
}
```

### Error: "encryption requires companion field"

**Symptoms**:
```
Error: encryption requires companion field: 'NameEncrypted' field must exist to store encrypted data for field 'Name'. Add 'NameEncrypted []byte' to your struct
```

**Cause**: Tagged field is missing its companion field.

**Solution**:
```go
// ❌ Wrong - missing companion field
type User struct {
    Name string `encx:"encrypt"`
    // Missing: NameEncrypted []byte
}

// ✅ Correct - includes companion field
type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte  // Companion field for encrypted data
}
```

### Error: "field type must be []byte"

**Symptoms**:
```
Error: companion field 'NameEncrypted' must be of type []byte, got string
```

**Cause**: Companion field has wrong type.

**Solution**:
```go
// ❌ Wrong - companion field is string
type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    string // Wrong type
}

// ✅ Correct - companion field is []byte
type User struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte // Correct type
}
```

### Error: "invalid encx tag"

**Symptoms**:
```
Error: field 'Name' has invalid encx tag 'encript', supported values: encrypt, hash_secure, hash_basic
```

**Cause**: Typo in tag name or using unsupported tag.

**Solution**:
```go
// ❌ Wrong - typo in tag
type User struct {
    Name string `encx:"encript"` // Should be "encrypt"
}

// ✅ Correct - proper tag
type User struct {
    Name string `encx:"encrypt"`
}

// ✅ Supported tags
// encx:"encrypt"      - AES-GCM encryption
// encx:"hash_basic"   - SHA-256 hashing
// encx:"hash_secure"  - Argon2id hashing
// encx:"encrypt,hash_basic" - Combined operations
```

## Struct Configuration Issues

### Issue: Fields Not Being Processed

**Symptoms**: Tagged fields remain unchanged after ProcessStruct.

**Debug Steps**:

1. **Check if field is exported**:
```go
// ❌ Wrong - unexported field (lowercase)
type User struct {
    name string `encx:"encrypt"` // Won't be processed
}

// ✅ Correct - exported field (uppercase)
type User struct {
    Name string `encx:"encrypt"` // Will be processed
}
```

2. **Verify tag syntax**:
```go
// ❌ Wrong syntax variations
Name string `encx: "encrypt"`  // Extra space
Name string `enc:"encrypt"`    // Wrong tag name
Name string `encx:encrypt`     // Missing quotes

// ✅ Correct syntax
Name string `encx:"encrypt"`
```

3. **Check struct validation**:
```go
if err := encx.ValidateStruct(&user); err != nil {
    log.Printf("Struct validation failed: %v", err)
}
```

### Issue: Embedded Structs Not Processing

**Symptoms**: Fields in embedded structs are ignored.

**Solution**:
```go
// ❌ Problematic - unexported embedded struct
type user struct { // lowercase - unexported
    Name string `encx:"encrypt"`
}

type Profile struct {
    user // Won't be processed
}

// ✅ Correct - exported embedded struct
type User struct { // uppercase - exported
    Name string `encx:"encrypt"`
}

type Profile struct {
    User // Will be processed
    // Or with field name:
    UserData User `encx:"-"` // Will also be processed
}
```

## Tag Validation Problems

### Issue: Combined Tags Not Recognized

**Symptoms**:
```
Error: invalid encx tag 'encrypt,hash_basic'
```

**Cause**: Using v1.1.x library with v1.2.x tag syntax, or validation tool issues.

**Solution**:
1. **Update to latest version**:
```bash
go get github.com/hengadev/encx@latest
go mod tidy
```

2. **Verify version**:
```go
import "github.com/hengadev/encx"

// This should work in v1.2.x+
type User struct {
    Email string `encx:"encrypt,hash_basic"`
    EmailEncrypted []byte
    EmailHash string
}
```

3. **Check validation tool version**:
```bash
go run github.com/hengadev/encx/cmd/validate-tags@latest -v
```

### Issue: Validation Tool False Positives

**Symptoms**: Validation tool reports errors for valid structs.

**Debug Steps**:

1. **Check file parsing**:
```bash
# Test with specific file
go run github.com/hengadev/encx/cmd/validate-tags -pattern="myfile.go" -v
```

2. **Verify syntax**:
```go
// Make sure struct follows ENCX requirements
type ValidUser struct {
    Email             string `encx:"encrypt,hash_basic"`
    EmailEncrypted    []byte
    EmailHash         string
    DEK               []byte
    DEKEncrypted      []byte
    KeyVersion        int
}
```

## Runtime Issues

### Issue: "DEK not found in context"

**Symptoms**:
```
Error: DEK not found in processing context. This indicates an internal error in DEK management
```

**Cause**: Internal processing error, usually during complex struct processing.

**Debug Steps**:

1. **Validate struct first**:
```go
if err := encx.ValidateStruct(&user); err != nil {
    log.Fatalf("Fix struct first: %v", err)
}
```

2. **Check for nil fields**:
```go
// Ensure no required fields are nil pointers
user := &User{
    Name: "John", // Don't leave required data nil
}
```

3. **Process in isolation**:
```go
// Test with minimal struct
type TestUser struct {
    Name             string `encx:"encrypt"`
    NameEncrypted    []byte
    DEK              []byte
    DEKEncrypted     []byte
    KeyVersion       int
}

testUser := &TestUser{Name: "Test"}
err := crypto.ProcessStruct(ctx, testUser)
```

### Issue: "failed to encrypt DEK"

**Symptoms**:
```
Error: DEK encryption failed using KEK: KMS operation failed
```

**Cause**: KMS connectivity or configuration issues.

**Solutions**:

1. **Check KMS connectivity**:
```go
// Test KMS directly
keyID, err := kmsService.CreateKey(ctx, "test-key")
if err != nil {
    log.Printf("KMS not accessible: %v", err)
}
```

2. **Verify credentials**:
```go
// For AWS KMS - check credentials
// For Vault - check token/authentication
// For test environments - ensure test KMS is running
```

3. **Use test KMS for debugging**:
```go
// Switch to test KMS to isolate issue
crypto, _ := encx.NewTestCrypto(t)
err := crypto.ProcessStruct(ctx, user)
```

### Issue: "serialization failed"

**Symptoms**:
```
Error: serialization failed for field 'Data' of type map[string]interface{} during encryption
```

**Cause**: Field contains non-serializable data.

**Solutions**:

1. **Check supported types**:
```go
// ✅ Supported types for encryption/hashing
string, int, int8, int16, int32, int64
uint, uint8, uint16, uint32, uint64
float32, float64, bool
[]byte, []string, []int (and other slices)
map[string]string, map[string]interface{}
struct types

// ❌ Problematic types
chan int        // Channels
func()         // Functions
unsafe.Pointer // Unsafe pointers
```

2. **Convert unsupported types**:
```go
// ❌ Problematic
type User struct {
    Data chan string `encx:"encrypt"` // Can't serialize channels
}

// ✅ Solution - use serializable representation
type User struct {
    Data string `encx:"encrypt"` // Serialize channel data to string
}
```

## Performance Problems

### Issue: Slow Encryption/Decryption

**Symptoms**: ProcessStruct/DecryptStruct takes too long.

**Debug Steps**:

1. **Profile operations**:
```go
start := time.Now()
err := crypto.ProcessStruct(ctx, user)
duration := time.Since(start)
log.Printf("ProcessStruct took: %v", duration)
```

2. **Check Argon2 parameters**:
```go
// Heavy parameters slow down secure hashing
params := &encx.Argon2Params{
    Memory:      65536, // 64MB - reduce for faster hashing
    Iterations:  3,     // Reduce iterations if too slow
    Parallelism: 4,     // Adjust based on CPU cores
}

crypto, err := encx.New(ctx, encx.WithArgon2Params(params))
```

3. **Optimize struct design**:
```go
// ❌ Inefficient - many small encrypted fields
type User struct {
    Field1    string `encx:"encrypt"`
    Field1Enc []byte
    Field2    string `encx:"encrypt"`
    Field2Enc []byte
    // ... 20 more fields
}

// ✅ Better - group related data
type User struct {
    PersonalData    PersonalInfo `encx:"encrypt"`
    PersonalDataEnc []byte
}

type PersonalInfo struct {
    Name    string
    Email   string
    Phone   string
    // All encrypted as one unit
}
```

### Issue: High Memory Usage

**Symptoms**: Memory usage grows significantly during processing.

**Solutions**:

1. **Process in batches**:
```go
// ❌ Process all at once
for _, user := range users { // 10,000 users
    crypto.ProcessStruct(ctx, user)
}

// ✅ Process in batches
batchSize := 100
for i := 0; i < len(users); i += batchSize {
    end := i + batchSize
    if end > len(users) {
        end = len(users)
    }
    
    for _, user := range users[i:end] {
        crypto.ProcessStruct(ctx, user)
    }
    
    // Optional: force GC between batches
    runtime.GC()
}
```

2. **Clear sensitive data**:
```go
defer func() {
    // Clear sensitive data from memory
    user.Password = ""
    // Note: This doesn't guarantee memory clearing,
    // but helps reduce exposure time
}()
```

## Testing Issues

### Issue: Tests Fail with "pepper value appears to be uninitialized"

**Symptoms**:
```
Error: pepper value appears to be uninitialized (all zeros)
```

**Cause**: Test crypto not configured properly.

**Solution**:
```go
// ❌ Wrong - may have zero pepper
crypto, _ := encx.NewTestCrypto(t)

// ✅ Correct - explicitly set test pepper
crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
    Pepper: []byte("test-pepper-exactly-32-bytes!!"),
})
```

### Issue: Mocks Not Working

**Symptoms**: Mock expectations not being met.

**Debug Steps**:

1. **Check mock setup**:
```go
mock := encx.NewCryptoServiceMock()

// ✅ Use mock.Anything for complex arguments
mock.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)

// ✅ Or be specific with MatchedBy
mock.On("ProcessStruct", mock.Anything, mock.MatchedBy(func(user *User) bool {
    return user.Name == "John"
})).Return(nil)

service := NewUserService(mock)
err := service.CreateUser("John")

// Must call this to verify expectations
mock.AssertExpectations(t)
```

2. **Debug mock calls**:
```go
// Add debug output
mock.On("ProcessStruct", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
    user := args.Get(1).(*User)
    log.Printf("Mock called with user: %+v", user)
}).Return(nil)
```

### Issue: Integration Tests Fail

**Symptoms**: Tests work in isolation but fail when run together.

**Cause**: Shared state or database conflicts.

**Solutions**:

1. **Use unique test data**:
```go
func TestUserProcessing(t *testing.T) {
    // Create unique identifiers
    testID := uuid.New().String()
    
    user := &User{
        Name:  "TestUser-" + testID,
        Email: fmt.Sprintf("test-%s@example.com", testID),
    }
}
```

2. **Isolate test databases**:
```go
func TestUserProcessing(t *testing.T) {
    // Create temporary test database
    tempDB := createTestDB(t)
    defer tempDB.Close()
    
    crypto, _ := encx.NewTestCryptoWithDB(t, tempDB)
}
```

## Debugging Techniques

### Enable Debug Logging

```go
import "log"

// Add before processing
log.Printf("Processing user: %+v", user)

err := crypto.ProcessStruct(ctx, user)
if err != nil {
    log.Printf("ProcessStruct error: %v", err)
} else {
    log.Printf("ProcessStruct success, DEK len: %d", len(user.DEK))
}
```

### Inspect Struct State

```go
// Before processing
log.Printf("Before - Name: '%s', DEK: %v", user.Name, len(user.DEK))

err := crypto.ProcessStruct(ctx, user)

// After processing
log.Printf("After - Name: '%s', NameEncrypted: %v, DEK: %v", 
    user.Name, len(user.NameEncrypted), len(user.DEK))
```

### Validate at Multiple Points

```go
// Validate before processing
if err := encx.ValidateStruct(user); err != nil {
    t.Fatalf("Pre-validation failed: %v", err)
}

// Process
err := crypto.ProcessStruct(ctx, user)
require.NoError(t, err)

// Validate expected state
assert.Empty(t, user.Name, "Original field should be cleared")
assert.NotEmpty(t, user.NameEncrypted, "Encrypted field should be populated")
assert.NotEmpty(t, user.DEK, "DEK should be generated")
```

### Test Individual Components

```go
// Test DEK generation separately
dek, err := crypto.GenerateDEK()
require.NoError(t, err)
require.Len(t, dek, 32)

// Test data encryption separately
plaintext := []byte("test data")
encrypted, err := crypto.EncryptData(ctx, plaintext, dek)
require.NoError(t, err)

decrypted, err := crypto.DecryptData(ctx, encrypted, dek)
require.NoError(t, err)
require.Equal(t, plaintext, decrypted)
```

## Getting Additional Help

### Collecting Debug Information

When reporting issues, include:

1. **ENCX version**:
```bash
go list -m github.com/hengadev/encx
```

2. **Go version**:
```bash
go version
```

3. **Struct definition**:
```go
// Include your struct definition with tags
```

4. **Error messages** (full text, not paraphrased)

5. **Minimal reproduction case**:
```go
func TestReproduceBug(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    user := &User{Name: "Test"}
    err := crypto.ProcessStruct(context.Background(), user)
    // Show the unexpected behavior
}
```

### Using the Validation Utility

```bash
# Check for common struct issues
go run github.com/hengadev/encx/cmd/validate-tags -v

# Check specific files
go run github.com/hengadev/encx/cmd/validate-tags -pattern="user*.go" -v
```

### Community Resources

- **GitHub Issues**: Search existing issues or create new ones
- **Examples**: Check [docs/EXAMPLES.md](./EXAMPLES.md) for working code
- **API Docs**: See [docs/API.md](./API.md) for detailed API reference
