# Testing Implementation Issues & TODO

## Current Status
The testing utilities have been implemented and provide comprehensive mocking and testing capabilities. However, there are some technical issues that need to be resolved for full functionality.

## ✅ What's Working
- `CryptoServiceMock`: Full mock implementation works correctly
- `NewTestCrypto`: Basic factory creation works
- Basic encryption/decryption operations work
- Documentation is complete
- Test structure and patterns are established

## 🐛 Current Issues

### 1. Struct-Level Encryption/Decryption Inconsistency
**Problem**: Integration tests that use struct encryption fail during decryption phase.

**Symptoms**:
- `TestPhoneEncryption_IntegrationTest` fails
- `TestStructEncryption` fails  
- `TestDebugCrypto` fails at decryption step
- Error: "cipher: message authentication failed"

**Root Cause**: The mock KMS implementation for `EncryptDEK`/`DecryptDEK` doesn't maintain consistency between encryption and decryption operations. When a DEK is encrypted by the real crypto code and then needs to be decrypted, the mock doesn't return the original DEK.

**Location**: `test_factory.go` lines 134-148 (setupDefaultKMSMockExpectations function)

### 2. Mock DEK Cycle Broken
**Problem**: The testify mock approach for DEK operations is too complex and fragile.

**Current Implementation Issues**:
```go
// This doesn't work reliably:
kmsMock.On("EncryptDEK", mock.Anything, mock.Anything, mock.Anything).
    Return([]byte("mock-encrypted-dek-data"), nil)

kmsMock.On("DecryptDEK", mock.Anything, mock.Anything, mock.Anything).
    Return(testDEK, nil) // This breaks the encrypt->decrypt cycle
```

**Why It Fails**: Real encryption creates a DEK, encrypts data with it, then encrypts the DEK with KMS. During decryption, the KMS must return the exact same DEK that was originally encrypted. The mock returns a fixed DEK instead of the original one.

### 3. Alternative Approach Started But Not Completed
**Attempted Solution**: `SimpleTestKMS` in `simple_test_kms.go` was created to avoid mocking complexity by providing a real in-memory KMS implementation.

**Current Issue**: Database path handling error in `NewTestCryptoWithSimpleKMS`:
```
Error: failed to get database path from connection: sql: Scan error on column index 0, name "seq": destination not a pointer
```

**Location**: `simple_test_kms.go` line 131-140, related to `getDatabasePathFromDB()` function in `crypto.go`

## 🔧 Solutions to Implement

### Option 1: Fix Mock KMS Consistency (Recommended)
**Approach**: Store actual DEKs in the mock and return them consistently.

**Implementation**:
```go
// In setupDefaultKMSMockExpectations:
dekStore := make(map[string][]byte) // Shared between encrypt/decrypt

kmsMock.On("EncryptDEK", mock.Anything, mock.Anything, mock.Anything).
    Return(func(ctx context.Context, keyID string, plaintextDEK []byte) []byte {
        key := fmt.Sprintf("dek-%x", plaintextDEK[:8])
        dekStore[key] = plaintextDEK  // Store the actual DEK
        return []byte(key)           // Return mock ciphertext
    }, nil)

kmsMock.On("DecryptDEK", mock.Anything, mock.Anything, mock.Anything).
    Return(func(ctx context.Context, keyID string, ciphertextDEK []byte) []byte {
        key := string(ciphertextDEK)
        if dek, exists := dekStore[key]; exists {
            return dek              // Return the original DEK
        }
        panic("DEK not found in test store")
    }, nil)
```

### Option 2: Complete SimpleTestKMS Implementation
**Approach**: Fix the database path issue and use a real KMS implementation for testing.

**Steps**:
1. Fix `getDatabasePathFromDB()` compatibility with file-based databases
2. Ensure `SimpleTestKMS.EncryptDEK`/`DecryptDEK` cycle is consistent
3. Update `NewTestCryptoWithSimpleKMS` to handle database initialization properly

### Option 3: Hybrid Approach
**Approach**: Use mocks for unit tests, SimpleTestKMS for integration tests.

**Implementation**:
- Keep `CryptoServiceMock` for pure unit testing
- Fix `SimpleTestKMS` for integration testing
- Update `NewTestCrypto` to choose implementation based on options

## 📋 Priority TODO List

### High Priority
1. **Fix Mock DEK Consistency**: Implement Option 1 above to make struct encryption tests work
2. **Test All Examples**: Ensure `TestPhoneEncryption_*` tests pass
3. **Validate Documentation**: Ensure all examples in `TESTING.md` work

### Medium Priority
1. **Complete SimpleTestKMS**: Fix database path issues
2. **Add More Test Scenarios**: Edge cases, error conditions
3. **Performance Testing**: Benchmark test setup overhead

### Low Priority
1. **Cleanup Debug Files**: Remove or consolidate debug test files
2. **Add Integration with Existing Tests**: Ensure compatibility with current test suite
3. **Advanced Mock Scenarios**: Complex KMS behaviors (key rotation, failures)

## 🔍 How to Debug

### Test Individual Components
```bash
# Test basic mock functionality (should pass)
go test -v -run "TestCryptoServiceMock$"

# Test basic crypto creation (should pass)  
go test -v -run "TestNewTestCrypto$"

# Test problematic struct encryption (currently fails)
go test -v -run "TestDebugCrypto$"

# Test phone integration (currently fails)
go test -v -run "TestPhoneEncryption_IntegrationTest$"
```

### Key Files to Examine
- `test_factory.go`: Mock setup and test crypto factory
- `crypto.go`: DEK encryption/decryption logic  
- `process_struct.go`: Struct processing logic
- `key_metadata.go`: Database path handling

## 💡 Notes for Future Development

### Design Principles
1. **Keep It Simple**: Avoid over-complex mocking
2. **Real Encryption**: Use actual encryption algorithms when possible
3. **Isolated Tests**: Each test should be independent
4. **Fast Execution**: In-memory operations, no external dependencies

### Testing Strategy
1. **Unit Tests**: Use `CryptoServiceMock` for business logic
2. **Integration Tests**: Use real crypto with test KMS  
3. **API Tests**: Full request/response cycle testing
4. **Performance Tests**: Ensure test setup is fast

The core testing infrastructure is in place and the design is sound. The remaining work is primarily about fixing the DEK encryption/decryption cycle consistency in the mock implementation.
