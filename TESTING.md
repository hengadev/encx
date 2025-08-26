# Testing with encx

This guide explains how to effectively test applications that use the encx encryption library. The encx package provides comprehensive testing utilities that eliminate the need for external dependencies while ensuring your encryption logic works correctly.

## Quick Start

The most common testing scenarios are now simple:

```go
func TestMyEncryptionFeature(t *testing.T) {
    ctx := context.Background()
    
    // Create a test crypto instance with no external dependencies
    crypto, _ := encx.NewTestCrypto(t)
    
    // Your test code here
    encrypted, err := crypto.EncryptData(ctx, []byte("test data"), dek)
    require.NoError(t, err)
    // ... rest of your test
}
```

## Key Testing Utilities

### 1. CryptoService Mock (`CryptoServiceMock`)

For unit testing where you want to control all crypto behavior:

```go
func TestBusinessLogic(t *testing.T) {
    mockCrypto := encx.NewCryptoServiceMock()
    
    // Set up expectations
    mockCrypto.On("GenerateDEK").Return([]byte("fake-dek-32-chars"), nil)
    mockCrypto.On("EncryptData", mock.Anything, []byte("phone"), mock.Anything).
        Return([]byte("encrypted-phone"), nil)
    
    // Inject into your service
    service := NewUserService(mockCrypto)
    result, err := service.CreateUser("john@example.com", "+1-555-0123")
    
    require.NoError(t, err)
    mockCrypto.AssertExpectations(t)
}
```

### 2. Test Crypto Factory (`NewTestCrypto`)

For integration-style testing with real encryption but no external dependencies:

```go
func TestUserEncryption(t *testing.T) {
    ctx := context.Background()
    
    // Creates real crypto instance with in-memory database and mock KMS
    crypto, kmsMock := encx.NewTestCrypto(t)
    
    user := &User{
        Name:        "John Doe",
        PhoneNumber: "+1-555-0123",
    }
    
    // Real encryption happens here
    err := crypto.ProcessStruct(ctx, user)
    require.NoError(t, err)
    
    // Phone number is now encrypted and cleared
    assert.Empty(t, user.PhoneNumber)
    
    // Decrypt it back
    err = crypto.DecryptStruct(ctx, user)
    require.NoError(t, err)
    assert.Equal(t, "+1-555-0123", user.PhoneNumber)
}
```

### 3. Test Data Factory (`TestDataFactory`)

For creating predictable encrypted data:

```go
func TestEncryptedDataComparison(t *testing.T) {
    ctx := context.Background()
    crypto, _ := encx.NewTestCrypto(t)
    factory := encx.NewTestDataFactory(crypto)
    
    // Creates encrypted data with fixed DEK for predictable testing
    encrypted, dek, err := factory.CreatePredictableEncryptedData(ctx, "test-value")
    require.NoError(t, err)
    
    // Can decrypt and verify
    decrypted, err := crypto.DecryptData(ctx, encrypted, dek)
    require.NoError(t, err)
    assert.Equal(t, []byte("test-value"), decrypted)
}
```

## Testing Patterns

### Pattern 1: Unit Testing Business Logic

When testing business logic that uses encryption, use the mock:

```go
type UserService struct {
    crypto encx.CryptoService // Use interface, not concrete type
    db     UserRepository
}

func NewUserService(crypto encx.CryptoService, db UserRepository) *UserService {
    return &UserService{crypto: crypto, db: db}
}

func (s *UserService) CreateUser(email, phone string) (*User, error) {
    user := &User{Email: email, PhoneNumber: phone}
    
    if err := s.crypto.ProcessStruct(context.Background(), user); err != nil {
        return nil, err
    }
    
    return s.db.Save(user)
}

// Test the business logic without real encryption
func TestUserService_CreateUser(t *testing.T) {
    mockCrypto := encx.NewCryptoServiceMock()
    mockDB := &MockUserRepository{}
    
    // Mock the crypto operations
    mockCrypto.On("ProcessStruct", mock.Anything, mock.Anything).Return(nil)
    mockDB.On("Save", mock.Anything).Return(&User{ID: 123}, nil)
    
    service := NewUserService(mockCrypto, mockDB)
    user, err := service.CreateUser("test@example.com", "+1-555-0123")
    
    require.NoError(t, err)
    assert.Equal(t, 123, user.ID)
    
    mockCrypto.AssertExpectations(t)
    mockDB.AssertExpectations(t)
}
```

### Pattern 2: Integration Testing Encryption

When testing the actual encryption behavior:

```go
func TestUserEncryptionIntegration(t *testing.T) {
    ctx := context.Background()
    
    // Real crypto operations, no external dependencies
    crypto, _ := encx.NewTestCrypto(t)
    
    user := &User{
        Name:        "John Doe",
        PhoneNumber: "+1-555-0123",
        Email:       "john@example.com",
        
        // Required encx fields
        DEK:          nil, // Will be generated
        DEKEncrypted: nil, // Will be populated
        KeyVersion:   0,   // Will be set
    }
    
    // Test encryption
    err := crypto.ProcessStruct(ctx, user)
    require.NoError(t, err)
    
    // Verify sensitive data is encrypted
    assert.Equal(t, "John Doe", user.Name)       // Plain field unchanged
    assert.Equal(t, "john@example.com", user.Email) // Plain field unchanged
    assert.Empty(t, user.PhoneNumber)             // Encrypted field cleared
    assert.NotEmpty(t, user.DEKEncrypted)         // DEK was encrypted
    assert.Greater(t, user.KeyVersion, 0)         // Version was set
    
    // Test decryption
    err = crypto.DecryptStruct(ctx, user)
    require.NoError(t, err)
    
    // Verify sensitive data is decrypted
    assert.Equal(t, "+1-555-0123", user.PhoneNumber)
}
```

### Pattern 3: API Endpoint Testing

This addresses the original comment about phone GET endpoint testing:

```go
func TestUserAPI_GetUser(t *testing.T) {
    ctx := context.Background()
    
    // Set up test crypto and data
    crypto, _ := encx.NewTestCrypto(t)
    
    // Create test user with encrypted data
    user := &User{
        ID:          123,
        Name:        "John Doe", 
        PhoneNumber: "+1-555-0123",
        Email:       "john@example.com",
    }
    
    // Encrypt for storage
    err := crypto.ProcessStruct(ctx, user)
    require.NoError(t, err)
    
    // Store in test database (user.PhoneNumber is now empty)
    testDB := setupTestDatabase(t)
    err = testDB.Save(user)
    require.NoError(t, err)
    
    // Set up API handler with test crypto
    handler := NewUserHandler(crypto, testDB)
    
    // Test the GET endpoint
    req := httptest.NewRequest("GET", "/users/123", nil)
    w := httptest.NewRecorder()
    
    handler.GetUser(w, req)
    
    // Verify response
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response User
    err = json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err)
    
    // Phone number should be decrypted in response
    assert.Equal(t, "+1-555-0123", response.PhoneNumber)
    assert.Equal(t, "John Doe", response.Name)
}
```

## Advanced Configuration

### Custom Test Crypto Options

```go
func TestWithCustomConfig(t *testing.T) {
    // Create custom KMS mock
    kmsMock := encx.NewKeyManagementServiceMock()
    kmsMock.On("GetKeyID", mock.Anything, "my-custom-alias").
        Return("custom-key-id", nil)
    
    // Create crypto with custom settings
    crypto, _ := encx.NewTestCrypto(t, &encx.TestCryptoOptions{
        UseRealDatabase: true,                                      // Use file-based DB
        CustomPepper:    []byte("my-custom-pepper-32-chars-test"), // Custom pepper  
        CustomKMSMock:   kmsMock,                                   // Custom KMS behavior
        DBPath:          "/tmp/test.db",                           // Custom DB path
    })
    
    // Use crypto with custom configuration
}
```

### Testing Key Rotation

```go
func TestKeyRotation(t *testing.T) {
    ctx := context.Background()
    crypto, kmsMock := encx.NewTestCrypto(t)
    
    // Set up mock expectations for rotation
    kmsMock.On("CreateKey", ctx, "test-key-alias").
        Return("new-key-id", nil)
    
    // Test rotation
    err := crypto.RotateKEK(ctx)
    require.NoError(t, err)
    
    kmsMock.AssertExpectations(t)
}
```

## Migration from Complex Testing

### Before (Complex Setup)

```go
// DON'T DO THIS ANYMORE
func TestPhoneEndpoint_OLD_WAY(t *testing.T) {
    // Complex Vault container setup
    vaultContainer, err := testutils.SetupVault(context.Background(), t)
    require.NoError(t, err)
    defer testutils.TeardownVault(context.Background(), t, vaultContainer)
    
    // Create real vault KMS service  
    kms, err := hashicorpvault.New(/* complex config */)
    require.NoError(t, err)
    
    // Create real crypto with external dependencies
    crypto, err := encx.New(context.Background(), kms, "alias", "secret/pepper")
    require.NoError(t, err)
    
    // Test is now dependent on external Vault service
    // Brittle, slow, and requires Docker
}
```

### After (Simple Setup)

```go
// DO THIS INSTEAD
func TestPhoneEndpoint_NEW_WAY(t *testing.T) {
    ctx := context.Background()
    
    // Simple test setup with no external dependencies
    crypto, _ := encx.NewTestCrypto(t)
    
    // Rest of your test logic remains the same
    // Fast, reliable, no Docker required
}
```

## Best Practices

### 1. Use Interfaces in Your Code

Always depend on `encx.CryptoService` interface, not the concrete `*encx.Crypto` type:

```go
// Good
func NewService(crypto encx.CryptoService) *Service

// Bad  
func NewService(crypto *encx.Crypto) *Service
```

### 2. Choose the Right Testing Tool

- **CryptoServiceMock**: Unit testing business logic
- **NewTestCrypto**: Integration testing encryption behavior
- **TestDataFactory**: When you need predictable encrypted data

### 3. Clean Test Structure

```go
func TestFeature(t *testing.T) {
    // Arrange
    ctx := context.Background()
    crypto, _ := encx.NewTestCrypto(t)
    
    // Act
    result, err := performOperation(crypto, input)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 4. Test Both Encryption and Decryption

```go
func TestRoundTrip(t *testing.T) {
    ctx := context.Background()
    crypto, _ := encx.NewTestCrypto(t)
    
    original := &MyStruct{SensitiveField: "secret"}
    
    // Test encryption
    err := crypto.ProcessStruct(ctx, original)
    require.NoError(t, err)
    assert.Empty(t, original.SensitiveField)
    
    // Test decryption  
    err = crypto.DecryptStruct(ctx, original)
    require.NoError(t, err)
    assert.Equal(t, "secret", original.SensitiveField)
}
```

## Performance Considerations

The test utilities are designed for performance:

- In-memory databases by default
- Minimal setup overhead
- Parallel test friendly
- No Docker containers required

```go
func BenchmarkEncryption(b *testing.B) {
    crypto, _ := encx.NewTestCrypto(b)
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = crypto.EncryptData(ctx, []byte("test"), dek)
    }
}
```

## Troubleshooting

### Common Issues

1. **"Pepper must be 32 bytes"**
   - Ensure custom pepper is exactly 32 bytes
   - Default test pepper is already correct size

2. **Mock expectations not met**
   - Always call `mockCrypto.AssertExpectations(t)`
   - Use `mock.Anything` for parameters you don't care about

3. **Database errors**
   - Test crypto automatically cleans up databases
   - Use `UseRealDatabase: false` for fastest tests

### Getting Help

- Check the `testing_example_test.go` file for comprehensive examples
- All test utilities include proper cleanup
- Mock objects support all interface methods

## Summary

The encx package provides comprehensive testing utilities that eliminate the original testing limitations:

✅ **Mock interface**: `CryptoServiceMock` for unit testing  
✅ **Test utilities**: `TestDataFactory` for predictable data  
✅ **Dependency injection**: Interface-based design supports test doubles  
✅ **Simple setup**: `NewTestCrypto` requires no external dependencies  
✅ **Integration testing**: Full encryption testing without Vault containers  

The original comment about testing limitations is no longer valid. You can now write reliable, fast tests for any encryption scenario without external dependencies.
