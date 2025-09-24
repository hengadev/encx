package reliability_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/hengadev/encx"
	"github.com/hengadev/encx/internal/reliability"
)

// ReliabilityIntegrationTestSuite tests reliability features in real scenarios
type ReliabilityIntegrationTestSuite struct {
	suite.Suite
	crypto         *encx.Crypto
	ctx            context.Context
	failingKMS     *FailingTestKMS
	circuitBreaker *reliability.CircuitBreaker
}

// SetupSuite initializes test environment
func (suite *ReliabilityIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create a KMS that can be configured to fail
	suite.failingKMS = NewFailingTestKMS()

	// Create circuit breaker for KMS operations
	cbConfig := reliability.CircuitBreakerConfig{
		Name:           "test-kms",
		MaxRequests:    3,
		Interval:       5 * time.Second,
		Timeout:        10 * time.Second,
		FailureThreshold: 2,
	}

	var err error
	suite.circuitBreaker, err = reliability.NewCircuitBreaker(cbConfig)
	require.NoError(suite.T(), err)

	// Create crypto instance with failing KMS and circuit breaker
	suite.crypto, err = encx.NewCrypto(suite.ctx,
		encx.WithKMSService(suite.failingKMS),
		encx.WithKEKAlias("reliability-test-key"),
		encx.WithPepper([]byte("reliability-test-pepper-32-bytes")),
		encx.WithCircuitBreaker(suite.circuitBreaker),
	)
	require.NoError(suite.T(), err)
}

// TestCircuitBreakerTrip tests that circuit breaker opens after failures
func (suite *ReliabilityIntegrationTestSuite) TestCircuitBreakerTrip() {
	// Configure KMS to fail
	suite.failingKMS.SetShouldFail(true)

	// Generate test data
	testData := []byte("test data for circuit breaker")

	// First attempt - should fail but circuit is closed
	_, err := suite.crypto.GenerateDEK(suite.ctx)
	assert.Error(suite.T(), err, "First DEK generation should fail")
	assert.Equal(suite.T(), reliability.StateClosed, suite.circuitBreaker.State())

	// Second attempt - should fail, circuit still closed
	_, err = suite.crypto.GenerateDEK(suite.ctx)
	assert.Error(suite.T(), err, "Second DEK generation should fail")
	assert.Equal(suite.T(), reliability.StateClosed, suite.circuitBreaker.State())

	// Third attempt - should trip the circuit breaker
	_, err = suite.crypto.GenerateDEK(suite.ctx)
	assert.Error(suite.T(), err, "Third DEK generation should fail and trip circuit")

	// Allow some time for circuit breaker to process
	time.Sleep(100 * time.Millisecond)

	// Circuit should now be open
	assert.Equal(suite.T(), reliability.StateOpen, suite.circuitBreaker.State())

	// Further attempts should fail fast (circuit breaker error)
	_, err = suite.crypto.GenerateDEK(suite.ctx)
	assert.Error(suite.T(), err, "Should fail fast due to open circuit")
	assert.Contains(suite.T(), err.Error(), "circuit breaker is open")
}

// TestCircuitBreakerRecovery tests circuit breaker recovery
func (suite *ReliabilityIntegrationTestSuite) TestCircuitBreakerRecovery() {
	// First trip the circuit breaker
	suite.failingKMS.SetShouldFail(true)

	// Trip the circuit
	for i := 0; i < 3; i++ {
		_, err := suite.crypto.GenerateDEK(suite.ctx)
		assert.Error(suite.T(), err)
	}

	// Wait for circuit breaker to process failures
	time.Sleep(100 * time.Millisecond)
	assert.Equal(suite.T(), reliability.StateOpen, suite.circuitBreaker.State())

	// Wait for timeout to transition to half-open
	time.Sleep(11 * time.Second)
	assert.Equal(suite.T(), reliability.StateHalfOpen, suite.circuitBreaker.State())

	// Fix the KMS
	suite.failingKMS.SetShouldFail(false)

	// Test recovery - should succeed and close circuit
	_, err := suite.crypto.GenerateDEK(suite.ctx)
	assert.NoError(suite.T(), err, "Should succeed in half-open state")

	// Circuit should be closed again
	time.Sleep(100 * time.Millisecond)
	assert.Equal(suite.T(), reliability.StateClosed, suite.circuitBreaker.State())

	// Further operations should work normally
	_, err = suite.crypto.GenerateDEK(suite.ctx)
	assert.NoError(suite.T(), err, "Should work normally after recovery")
}

// TestRetryPolicyWithExponentialBackoff tests retry behavior
func (suite *ReliabilityIntegrationTestSuite) TestRetryPolicyWithExponentialBackoff() {
	// Create retry policy
	retryPolicy := reliability.RetryPolicy{
		MaxAttempts:    4,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}

	// Configure KMS to fail first 2 attempts, then succeed
	suite.failingKMS.SetFailureCount(2)

	start := time.Now()

	// Execute with retry
	var dek []byte
	err := retryPolicy.Execute(suite.ctx, func(ctx context.Context) error {
		var genErr error
		dek, genErr = suite.crypto.GenerateDEK(ctx)
		return genErr
	})

	duration := time.Since(start)

	// Should succeed after retries
	assert.NoError(suite.T(), err, "Should succeed after retries")
	assert.NotEmpty(suite.T(), dek, "DEK should be generated")

	// Should have taken time due to retries and backoff
	assert.Greater(suite.T(), duration, 200*time.Millisecond, "Should take time due to retries")
	assert.Less(suite.T(), duration, 5*time.Second, "Should not take too long")

	// Verify actual retry count
	assert.Equal(suite.T(), 3, suite.failingKMS.GetAttemptCount(), "Should have made 3 attempts total")
}

// TestConcurrentReliabilityFeatures tests reliability under concurrent load
func (suite *ReliabilityIntegrationTestSuite) TestConcurrentReliabilityFeatures() {
	const numGoroutines = 20
	const operationsPerGoroutine = 10

	// Reset KMS state
	suite.failingKMS.Reset()
	suite.failingKMS.SetShouldFail(false)

	results := make(chan error, numGoroutines*operationsPerGoroutine)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Generate DEK
				dek, err := suite.crypto.GenerateDEK(suite.ctx)
				if err != nil {
					results <- fmt.Errorf("goroutine %d operation %d: %w", goroutineID, j, err)
					continue
				}

				// Test data encryption
				testData := []byte(fmt.Sprintf("data-g%d-op%d", goroutineID, j))
				encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
				if err != nil {
					results <- fmt.Errorf("goroutine %d operation %d encrypt: %w", goroutineID, j, err)
					continue
				}

				// Test data decryption
				decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
				if err != nil {
					results <- fmt.Errorf("goroutine %d operation %d decrypt: %w", goroutineID, j, err)
					continue
				}

				if string(decrypted) != string(testData) {
					results <- fmt.Errorf("goroutine %d operation %d: data mismatch", goroutineID, j)
					continue
				}

				results <- nil
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Check all results
	successCount := 0
	failureCount := 0
	for err := range results {
		if err != nil {
			suite.T().Logf("Operation failed: %v", err)
			failureCount++
		} else {
			successCount++
		}
	}

	// All operations should succeed under normal conditions
	assert.Equal(suite.T(), numGoroutines*operationsPerGoroutine, successCount)
	assert.Equal(suite.T(), 0, failureCount)
}

// TestFailureIsolation tests that failures in one component don't affect others
func (suite *ReliabilityIntegrationTestSuite) TestFailureIsolation() {
	// Test that KMS failures don't prevent hash operations
	suite.failingKMS.SetShouldFail(true)

	// KMS operations should fail
	_, err := suite.crypto.GenerateDEK(suite.ctx)
	assert.Error(suite.T(), err, "KMS operations should fail")

	// But hash operations should still work (they don't use KMS)
	hash, err := suite.crypto.HashForSearch(suite.ctx, "test@example.com")
	assert.NoError(suite.T(), err, "Hash operations should still work")
	assert.NotEmpty(suite.T(), hash, "Hash should be generated")

	// Secure hash should also work
	secureHash, err := suite.crypto.HashSecure(suite.ctx, "sensitive-data")
	assert.NoError(suite.T(), err, "Secure hash operations should still work")
	assert.NotEmpty(suite.T(), secureHash, "Secure hash should be generated")
}

// TestHealthCheckIntegration tests integration with health monitoring
func (suite *ReliabilityIntegrationTestSuite) TestHealthCheckIntegration() {
	// Test health check when everything is working
	suite.failingKMS.SetShouldFail(false)

	healthStatus := suite.crypto.HealthCheck(suite.ctx)
	assert.True(suite.T(), healthStatus.Overall, "Health check should pass when KMS is working")
	assert.True(suite.T(), healthStatus.KMS, "KMS health should be true")
	assert.True(suite.T(), healthStatus.Crypto, "Crypto health should be true")

	// Test health check when KMS is failing
	suite.failingKMS.SetShouldFail(true)

	healthStatus = suite.crypto.HealthCheck(suite.ctx)
	assert.False(suite.T(), healthStatus.Overall, "Overall health should fail when KMS is down")
	assert.False(suite.T(), healthStatus.KMS, "KMS health should be false")
	// Crypto operations not using KMS should still be healthy
	assert.True(suite.T(), healthStatus.Crypto, "Basic crypto health should still be true")
}

// TestSuite entry point
func TestReliabilityIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ReliabilityIntegrationTestSuite))
}

// FailingTestKMS is a test KMS that can be configured to fail
type FailingTestKMS struct {
	mu           sync.RWMutex
	shouldFail   bool
	failureCount int
	attemptCount int
	maxFailures  int
}

// NewFailingTestKMS creates a new FailingTestKMS
func NewFailingTestKMS() *FailingTestKMS {
	return &FailingTestKMS{}
}

// SetShouldFail configures whether operations should fail
func (f *FailingTestKMS) SetShouldFail(fail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shouldFail = fail
}

// SetFailureCount sets number of times to fail before succeeding
func (f *FailingTestKMS) SetFailureCount(count int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.maxFailures = count
	f.failureCount = 0
	f.attemptCount = 0
}

// GetAttemptCount returns the total number of attempts made
func (f *FailingTestKMS) GetAttemptCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.attemptCount
}

// Reset resets the failure state
func (f *FailingTestKMS) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shouldFail = false
	f.failureCount = 0
	f.attemptCount = 0
	f.maxFailures = 0
}

// EncryptDEK implements KMS interface with configurable failures
func (f *FailingTestKMS) EncryptDEK(ctx context.Context, keyID string, dek []byte) ([]byte, error) {
	f.mu.Lock()
	f.attemptCount++
	shouldFailNow := f.shouldFail || (f.maxFailures > 0 && f.failureCount < f.maxFailures)
	if shouldFailNow && f.maxFailures > 0 {
		f.failureCount++
	}
	f.mu.Unlock()

	if shouldFailNow {
		return nil, fmt.Errorf("simulated KMS failure (attempt %d)", f.attemptCount)
	}

	// Simulate successful encryption
	return append([]byte("encrypted:"), dek...), nil
}

// DecryptDEK implements KMS interface with configurable failures
func (f *FailingTestKMS) DecryptDEK(ctx context.Context, keyID string, encryptedDEK []byte) ([]byte, error) {
	f.mu.Lock()
	f.attemptCount++
	shouldFailNow := f.shouldFail || (f.maxFailures > 0 && f.failureCount < f.maxFailures)
	if shouldFailNow && f.maxFailures > 0 {
		f.failureCount++
	}
	f.mu.Unlock()

	if shouldFailNow {
		return nil, fmt.Errorf("simulated KMS failure (attempt %d)", f.attemptCount)
	}

	// Simulate successful decryption by removing "encrypted:" prefix
	if len(encryptedDEK) > 10 && string(encryptedDEK[:10]) == "encrypted:" {
		return encryptedDEK[10:], nil
	}

	return nil, fmt.Errorf("invalid encrypted DEK format")
}

// GenerateDataKey implements KMS interface with configurable failures
func (f *FailingTestKMS) GenerateDataKey(ctx context.Context, keyID string) ([]byte, []byte, error) {
	f.mu.Lock()
	f.attemptCount++
	shouldFailNow := f.shouldFail || (f.maxFailures > 0 && f.failureCount < f.maxFailures)
	if shouldFailNow && f.maxFailures > 0 {
		f.failureCount++
	}
	f.mu.Unlock()

	if shouldFailNow {
		return nil, nil, fmt.Errorf("simulated KMS failure (attempt %d)", f.attemptCount)
	}

	// Generate a test DEK
	dek := make([]byte, 32)
	for i := range dek {
		dek[i] = byte(i % 256)
	}

	encryptedDEK := append([]byte("encrypted:"), dek...)
	return dek, encryptedDEK, nil
}