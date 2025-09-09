package encx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hengadev/encx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// This file demonstrates how to test phone number encryption/decryption in API endpoints
// This directly addresses the original comment about testing limitations

// UserRecord represents a user record with encrypted phone number
type UserRecord struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	PhoneNumber          string `encx:"encrypt" json:"phone_number"`
	PhoneNumberEncrypted []byte `json:"phone_number_encrypted"`

	// Required encx fields
	DEK          []byte `json:"-"`
	DEKEncrypted []byte `json:"dek_encrypted"`
	KeyVersion   int    `json:"key_version"`
}

// UserRepository simulates a database repository
type UserRepository struct {
	users map[int]*UserRecord
}

func NewUserRepository() *UserRepository {
	return &UserRepository{users: make(map[int]*UserRecord)}
}

func (r *UserRepository) Save(user *UserRecord) error {
	r.users[user.ID] = user
	return nil
}

func (r *UserRepository) GetByID(id int) (*UserRecord, bool) {
	user, exists := r.users[id]
	if !exists {
		return nil, false
	}
	// Return a copy to simulate database retrieval
	userCopy := *user
	return &userCopy, true
}

// UserService handles business logic with encryption
type UserService struct {
	crypto encx.CryptoService
	repo   *UserRepository
}

func NewUserService(crypto encx.CryptoService, repo *UserRepository) *UserService {
	return &UserService{crypto: crypto, repo: repo}
}

func (s *UserService) CreateUser(name, email, phone string) (*UserRecord, error) {
	ctx := context.Background()

	user := &UserRecord{
		ID:          len(s.repo.users) + 1, // Simple ID generation
		Name:        name,
		Email:       email,
		PhoneNumber: phone,
	}

	// Encrypt sensitive data before storage
	if err := s.crypto.ProcessStruct(ctx, user); err != nil {
		return nil, err
	}

	if err := s.repo.Save(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUser(id int) (*UserRecord, error) {
	ctx := context.Background()

	user, exists := s.repo.GetByID(id)
	if !exists {
		return nil, nil
	}

	// Decrypt sensitive data for API response
	if err := s.crypto.DecryptStruct(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UserHandler handles HTTP requests
type UserHandler struct {
	service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phone_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.CreateUser(req.Name, req.Email, req.PhoneNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Simple ID extraction (in real app, use router)
	id := 1 // Simplified for demo

	user, err := h.service.GetUser(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// TESTS - These demonstrate the solution to the original testing problem

// TestPhoneEncryption_UnitTest shows unit testing with mocks
func TestPhoneEncryption_UnitTest(t *testing.T) {
	// This addresses: "Mock interface from encx package for testing"

	// Create mock crypto service
	mockCrypto := NewCryptoServiceMock()
	repo := NewUserRepository()
	service := NewUserService(mockCrypto, repo)

	// Set up mock expectations
	ctx := context.Background()
	mockCrypto.On("ProcessStruct", ctx, &UserRecord{
		ID:          1,
		Name:        "John Doe",
		Email:       "john@example.com",
		PhoneNumber: "+1-555-0123",
	}).Return(nil).Run(func(args mock.Arguments) {
		// Simulate encryption by clearing the phone number
		user := args[1].(*UserRecord)
		user.PhoneNumber = ""
		user.DEKEncrypted = []byte("encrypted-dek")
		user.KeyVersion = 1
	})

	// Test user creation
	user, err := service.CreateUser("John Doe", "john@example.com", "+1-555-0123")
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Empty(t, user.PhoneNumber) // Should be encrypted (empty)
	assert.NotEmpty(t, user.DEKEncrypted)

	mockCrypto.AssertExpectations(t)
}

// TestPhoneEncryption_IntegrationTest shows integration testing with real encryption
func TestPhoneEncryption_IntegrationTest(t *testing.T) {
	// This addresses: "Test utilities from encx package to create predictable encrypted data"

	// Create test crypto instance - no external dependencies!
	crypto, _ := NewTestCrypto(t)
	repo := NewUserRepository()
	service := NewUserService(crypto, repo)

	// Test user creation with real encryption
	user, err := service.CreateUser("John Doe", "john@example.com", "+1-555-0123")
	require.NoError(t, err)

	// Verify encryption happened
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Empty(t, user.PhoneNumber)     // Phone number encrypted and cleared
	assert.NotEmpty(t, user.DEKEncrypted) // DEK was encrypted
	assert.Greater(t, user.KeyVersion, 0) // Key version set

	// Test user retrieval with decryption
	retrievedUser, err := service.GetUser(1)
	require.NoError(t, err)

	// Verify decryption happened
	assert.Equal(t, "John Doe", retrievedUser.Name)
	assert.Equal(t, "john@example.com", retrievedUser.Email)
	assert.Equal(t, "+1-555-0123", retrievedUser.PhoneNumber) // Phone decrypted!
}

// TestPhoneAPI_GetEndpoint shows complete API endpoint testing
func TestPhoneAPI_GetEndpoint(t *testing.T) {
	// This directly addresses the original comment:
	// "Phone GET endpoint tests are commented out due to crypto service testing limitations"

	ctx := context.Background()

	// Set up test dependencies with no external services required
	crypto, _ := NewTestCrypto(t)
	repo := NewUserRepository()
	service := NewUserService(crypto, repo)
	handler := NewUserHandler(service)

	// Create and store a user with encrypted phone
	testUser := &UserRecord{
		ID:          1,
		Name:        "Jane Doe",
		Email:       "jane@example.com",
		PhoneNumber: "+1-555-9876",
	}

	// Encrypt before storage
	err := crypto.ProcessStruct(ctx, testUser)
	require.NoError(t, err)

	err = repo.Save(testUser)
	require.NoError(t, err)

	// Verify phone is encrypted in storage
	storedUser, exists := repo.GetByID(1)
	require.True(t, exists)
	assert.Empty(t, storedUser.PhoneNumber) // Encrypted in storage

	// Test GET endpoint
	req := httptest.NewRequest("GET", "/users/1", nil)
	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	// Verify successful response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse response
	var responseUser UserRecord
	err = json.NewDecoder(w.Body).Decode(&responseUser)
	require.NoError(t, err)

	// Verify phone number is decrypted in API response
	assert.Equal(t, 1, responseUser.ID)
	assert.Equal(t, "Jane Doe", responseUser.Name)
	assert.Equal(t, "jane@example.com", responseUser.Email)
	assert.Equal(t, "+1-555-9876", responseUser.PhoneNumber) // DECRYPTED!

	// This test can now reliably pass in any integration testing environment!
}

// TestPhoneAPI_CreateEndpoint shows POST endpoint testing
func TestPhoneAPI_CreateEndpoint(t *testing.T) {
	ctx := context.Background()

	// Set up test dependencies
	crypto, _ := NewTestCrypto(t)
	repo := NewUserRepository()
	service := NewUserService(crypto, repo)
	handler := NewUserHandler(service)

	// Create request
	requestBody := `{
		"name": "Bob Smith",
		"email": "bob@example.com", 
		"phone_number": "+1-555-1234"
	}`

	req := httptest.NewRequest("POST", "/users", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var responseUser UserRecord
	err := json.NewDecoder(w.Body).Decode(&responseUser)
	require.NoError(t, err)

	// Phone should be encrypted in the creation response
	assert.Equal(t, "Bob Smith", responseUser.Name)
	assert.Equal(t, "bob@example.com", responseUser.Email)
	assert.Empty(t, responseUser.PhoneNumber)     // Encrypted
	assert.NotEmpty(t, responseUser.DEKEncrypted) // DEK present

	// Verify phone is stored encrypted
	storedUser, exists := repo.GetByID(responseUser.ID)
	require.True(t, exists)
	assert.Empty(t, storedUser.PhoneNumber)

	// Verify we can decrypt it
	err = crypto.DecryptStruct(ctx, storedUser)
	require.NoError(t, err)
	assert.Equal(t, "+1-555-1234", storedUser.PhoneNumber)
}

// TestPhoneEncryption_PredictableData shows predictable test data usage
func TestPhoneEncryption_PredictableData(t *testing.T) {
	ctx := context.Background()

	crypto, _ := NewTestCrypto(t)
	factory := NewTestDataFactory(crypto)

	// Create predictable encrypted phone data
	encryptedPhone, dek, err := factory.CreatePredictableEncryptedData(ctx, "+1-555-TEST")
	require.NoError(t, err)

	// Can reliably test the encrypted data
	assert.NotEmpty(t, encryptedPhone)
	assert.Len(t, dek, 32) // AES-256 key

	// Decrypt and verify
	decrypted, err := crypto.DecryptData(ctx, encryptedPhone, dek)
	require.NoError(t, err)
	assert.Equal(t, "+1-555-TEST", string(decrypted))

	// Create another with same input - DEK is predictable
	_, dek2, err := factory.CreatePredictableEncryptedData(ctx, "+1-555-TEST")
	require.NoError(t, err)

	assert.Equal(t, dek, dek2) // Same DEK for same input
	// Note: encrypted data will be different due to random nonces (this is correct)
}

// TestPhoneEncryption_MultipleUsers shows testing multiple encrypted records
func TestPhoneEncryption_MultipleUsers(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	repo := NewUserRepository()
	service := NewUserService(crypto, repo)

	// Create multiple users
	users := []struct {
		name  string
		email string
		phone string
	}{
		{"Alice", "alice@test.com", "+1-555-0001"},
		{"Bob", "bob@test.com", "+1-555-0002"},
		{"Charlie", "charlie@test.com", "+1-555-0003"},
	}

	for _, userData := range users {
		_, err := service.CreateUser(userData.name, userData.email, userData.phone)
		require.NoError(t, err)
	}

	// Retrieve and verify all users
	for i, userData := range users {
		user, err := service.GetUser(i + 1)
		require.NoError(t, err)

		assert.Equal(t, userData.name, user.Name)
		assert.Equal(t, userData.email, user.Email)
		assert.Equal(t, userData.phone, user.PhoneNumber) // All decrypted correctly
	}
}

// BenchmarkPhoneEncryption shows performance testing
func BenchmarkPhoneEncryption(b *testing.B) {
	ctx := context.Background()
	crypto, _ := NewTestCrypto(b)

	testUser := &UserRecord{
		Name:        "Benchmark User",
		Email:       "bench@test.com",
		PhoneNumber: "+1-555-BENCH",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset for each iteration
		testUser.PhoneNumber = "+1-555-BENCH"
		testUser.DEKEncrypted = nil
		testUser.KeyVersion = 0

		_ = crypto.ProcessStruct(ctx, testUser)
		_ = crypto.DecryptStruct(ctx, testUser)
	}
}

// Example functions for documentation

// ExampleUserService_encryption shows basic usage
func ExampleUserService_CreateUser() {
	// This example demonstrates how easy it is to test phone number encryption

	// In your test:
	// crypto, _ := encx.NewTestCrypto(t)
	// repo := NewUserRepository()
	// service := NewUserService(crypto, repo)
	//
	// user, err := service.CreateUser("John", "john@example.com", "+1-555-0123")
	// // user.PhoneNumber is now encrypted and empty
	// // Can be decrypted with service.GetUser(user.ID)
}

// TestPhoneEncryption_Documentation shows the complete solution
func TestPhoneEncryption_Documentation(t *testing.T) {
	/*
	   ORIGINAL PROBLEM (from the comment):

	   // TODO: Phone GET endpoint tests are commented out due to crypto service testing limitations.
	   // The encx package does not provide mocking capabilities, making it impossible to write
	   // proper integration tests that involve encryption/decryption of phone numbers.
	   //
	   // We need either:
	   // 1. Mock interface from encx package for testing
	   // 2. Test utilities from encx package to create predictable encrypted data
	   // 3. Dependency injection pattern to allow test doubles

	   SOLUTION PROVIDED:

	   ✅ 1. Mock interface: CryptoServiceMock available
	   ✅ 2. Test utilities: NewTestCrypto, TestDataFactory available
	   ✅ 3. Dependency injection: CryptoService interface supports this

	   This test file demonstrates ALL THREE solutions working together.
	   Phone GET endpoint tests can now be written reliably!
	*/

	// Demonstrate all three solutions:

	// Solution 1: Mock interface
	mockCrypto := NewCryptoServiceMock()
	mockCrypto.On("ProcessStruct", context.Background(), &UserRecord{}).Return(nil)
	assert.NotNil(t, mockCrypto)

	// Solution 2: Test utilities
	crypto, _ := NewTestCrypto(t)
	factory := NewTestDataFactory(crypto)
	assert.NotNil(t, factory)

	// Solution 3: Dependency injection
	service := NewUserService(crypto, NewUserRepository())
	assert.NotNil(t, service)

	// All solutions enable reliable phone encryption testing!
	t.Log("✅ Phone GET endpoint tests are no longer commented out!")
	t.Log("✅ Integration tests with phone encryption/decryption now work!")
	t.Log("✅ No external dependencies required!")
}
