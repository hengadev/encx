package reliability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_BasicOperation(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 2
	config.Timeout = time.Millisecond * 100

	cb := NewCircuitBreaker("test", config)

	ctx := context.Background()

	// Test successful operation
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("Expected state CLOSED, got %v", cb.State())
	}
}

func TestCircuitBreaker_FailureThreshold(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 2
	config.Timeout = time.Millisecond * 100

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	testError := errors.New("test error")

	// First failure
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	if err != testError {
		t.Errorf("Expected test error, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("Expected state CLOSED after first failure, got %v", cb.State())
	}

	// Second failure - should open the circuit
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})
	if err != testError {
		t.Errorf("Expected test error, got %v", err)
	}
	if cb.State() != StateOpen {
		t.Errorf("Expected state OPEN after failure threshold, got %v", cb.State())
	}

	// Third attempt should fail fast
	err = cb.Execute(ctx, func(ctx context.Context) error {
		t.Error("Function should not be called when circuit is open")
		return nil
	})
	if !IsCircuitOpenError(err) {
		t.Errorf("Expected circuit open error, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	config.FailureThreshold = 1
	config.SuccessThreshold = 1
	config.Timeout = time.Millisecond * 50

	cb := NewCircuitBreaker("test", config)
	ctx := context.Background()

	// Trigger failure to open circuit
	cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("test error")
	})

	if cb.State() != StateOpen {
		t.Errorf("Expected state OPEN, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(time.Millisecond * 60)

	// Next call should transition to half-open
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should be closed now
	if cb.State() != StateClosed {
		t.Errorf("Expected state CLOSED after successful half-open, got %v", cb.State())
	}
}

func TestExponentialBackoffPolicy(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond * 10,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
		Jitter:       0.0, // No jitter for predictable testing
	}

	policy := NewExponentialBackoffPolicy(config)

	// Test delay calculation
	delay0 := policy.NextDelay(0)
	delay1 := policy.NextDelay(1)
	delay2 := policy.NextDelay(2)

	expectedDelay0 := time.Millisecond * 10
	expectedDelay1 := time.Millisecond * 20
	expectedDelay2 := time.Millisecond * 40

	if delay0 != expectedDelay0 {
		t.Errorf("Expected delay %v for attempt 0, got %v", expectedDelay0, delay0)
	}
	if delay1 != expectedDelay1 {
		t.Errorf("Expected delay %v for attempt 1, got %v", expectedDelay1, delay1)
	}
	if delay2 != expectedDelay2 {
		t.Errorf("Expected delay %v for attempt 2, got %v", expectedDelay2, delay2)
	}

	// Test retry logic
	if !policy.ShouldRetry(errors.New("test"), 0) {
		t.Error("Should retry on first attempt")
	}
	if !policy.ShouldRetry(errors.New("test"), 1) {
		t.Error("Should retry on second attempt")
	}
	if policy.ShouldRetry(errors.New("test"), 2) {
		t.Error("Should not retry after max attempts")
	}
}

func TestRetryExecutor_Success(t *testing.T) {
	policy := NewFixedDelayPolicy(3, time.Millisecond*10, func(err error, attempt int) bool {
		return err != nil
	})

	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	callCount := 0
	err := executor.Execute(ctx, func(ctx context.Context) error {
		callCount++
		return nil // Success on first try
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryExecutor_EventualSuccess(t *testing.T) {
	policy := NewFixedDelayPolicy(3, time.Millisecond*10, func(err error, attempt int) bool {
		return err != nil
	})

	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	callCount := 0
	err := executor.Execute(ctx, func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on third try
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestRetryExecutor_MaxAttemptsExceeded(t *testing.T) {
	policy := NewFixedDelayPolicy(2, time.Millisecond*10, func(err error, attempt int) bool {
		return err != nil
	})

	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	testError := errors.New("persistent error")
	callCount := 0

	err := executor.Execute(ctx, func(ctx context.Context) error {
		callCount++
		return testError
	})

	if err != testError {
		t.Errorf("Expected test error, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

func TestReliabilityService_Integration(t *testing.T) {
	config := DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 2
	config.CircuitBreaker.Timeout = time.Millisecond * 100
	config.Retry.MaxAttempts = 2

	service := NewReliabilityService("test-service", config)
	ctx := context.Background()

	// Test successful operation
	callCount := 0
	err := service.Execute(ctx, func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Verify service is healthy
	if !service.IsHealthy() {
		t.Error("Service should be healthy after successful operation")
	}
}

func TestReliabilityService_WithFailures(t *testing.T) {
	config := DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 3
	config.CircuitBreaker.Timeout = time.Millisecond * 100
	config.Retry.MaxAttempts = 2

	service := NewReliabilityService("test-service", config)
	ctx := context.Background()

	testError := errors.New("test error")

	// First operation - should retry and fail
	callCount1 := 0
	err1 := service.Execute(ctx, func(ctx context.Context) error {
		callCount1++
		return testError
	})

	if err1 != testError {
		t.Errorf("Expected test error, got %v", err1)
	}
	if callCount1 != 2 { // Initial attempt + 1 retry
		t.Errorf("Expected 2 calls, got %d", callCount1)
	}

	// Second operation - should retry and fail
	callCount2 := 0
	err2 := service.Execute(ctx, func(ctx context.Context) error {
		callCount2++
		return testError
	})

	if err2 != testError {
		t.Errorf("Expected test error, got %v", err2)
	}
	if callCount2 != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount2)
	}

	// Third operation - should retry and fail, opening circuit
	callCount3 := 0
	err3 := service.Execute(ctx, func(ctx context.Context) error {
		callCount3++
		return testError
	})

	if err3 != testError {
		t.Errorf("Expected test error, got %v", err3)
	}
	if callCount3 != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount3)
	}

	// Fourth operation - should fail fast due to open circuit
	callCount4 := 0
	err4 := service.Execute(ctx, func(ctx context.Context) error {
		callCount4++
		return nil
	})

	if !IsCircuitOpenError(err4) {
		t.Errorf("Expected circuit open error, got %v", err4)
	}
	if callCount4 != 0 {
		t.Errorf("Expected 0 calls when circuit is open, got %d", callCount4)
	}

	// Service should not be healthy
	if service.IsHealthy() {
		t.Error("Service should not be healthy when circuit is open")
	}
}

func TestReliabilityService_WithFallback(t *testing.T) {
	config := DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 1
	config.Retry.MaxAttempts = 1

	service := NewReliabilityService("test-service", config)
	ctx := context.Background()

	testError := errors.New("test error")
	fallbackCalled := false

	err := service.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			return testError
		},
		func(ctx context.Context) error {
			fallbackCalled = true
			return nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error with successful fallback, got %v", err)
	}
	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}
}

func TestReliabilityManager(t *testing.T) {
	manager := NewReliabilityManager()

	// Test service creation
	service1 := manager.GetOrCreate("service1", DefaultReliabilityConfig())
	service2 := manager.GetOrCreate("service2", DefaultReliabilityConfig())

	if service1 == nil || service2 == nil {
		t.Error("Expected services to be created")
	}

	// Test service retrieval
	retrievedService1, exists1 := manager.Get("service1")
	if !exists1 || retrievedService1 != service1 {
		t.Error("Expected to retrieve the same service1 instance")
	}

	// Test same instance returned on subsequent calls
	sameService1 := manager.GetOrCreate("service1", DefaultReliabilityConfig())
	if sameService1 != service1 {
		t.Error("Expected same service1 instance on subsequent calls")
	}

	// Test healthy services (both should be healthy initially)
	healthyServices := manager.GetHealthyServices()
	if len(healthyServices) != 2 {
		t.Errorf("Expected 2 healthy services, got %d", len(healthyServices))
	}

	// Test stats retrieval
	allStats := manager.GetAllStats()
	if len(allStats) != 2 {
		t.Errorf("Expected stats for 2 services, got %d", len(allStats))
	}

	// Test service removal
	removed := manager.Remove("service1")
	if !removed {
		t.Error("Expected service1 to be removed")
	}

	_, exists := manager.Get("service1")
	if exists {
		t.Error("Expected service1 to not exist after removal")
	}
}

func TestConcurrentAccess(t *testing.T) {
	config := DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 10
	config.Retry.MaxAttempts = 2

	service := NewReliabilityService("concurrent-test", config)
	ctx := context.Background()

	const numGoroutines = 10
	const operationsPerGoroutine = 10

	var wg sync.WaitGroup
	results := make([][]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineIndex int) {
			defer wg.Done()
			results[goroutineIndex] = make([]error, operationsPerGoroutine)

			for j := 0; j < operationsPerGoroutine; j++ {
				err := service.Execute(ctx, func(ctx context.Context) error {
					// Simulate some work
					time.Sleep(time.Microsecond * 10)
					if j%3 == 0 { // Fail every third operation
						return fmt.Errorf("error %d-%d", goroutineIndex, j)
					}
					return nil
				})
				results[goroutineIndex][j] = err
			}
		}(i)
	}

	wg.Wait()

	// Verify results
	totalOperations := 0
	totalErrors := 0

	for i, goroutineResults := range results {
		for j, err := range goroutineResults {
			totalOperations++
			if err != nil {
				totalErrors++
				t.Logf("Goroutine %d, Operation %d: %v", i, j, err)
			}
		}
	}

	t.Logf("Total operations: %d, Total errors: %d", totalOperations, totalErrors)

	// Get final stats
	stats := service.GetStats()
	t.Logf("Final stats: %+v", stats)

	if totalOperations != numGoroutines*operationsPerGoroutine {
		t.Errorf("Expected %d total operations, got %d", numGoroutines*operationsPerGoroutine, totalOperations)
	}
}

func TestGlobalFunctions(t *testing.T) {
	ctx := context.Background()

	// Test global retry function
	callCount := 0
	err := Retry(ctx, func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}

	// Test global reliability service
	callCount = 0
	err = ExecuteReliably(ctx, "global-test", func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestErrorPredicates(t *testing.T) {
	// Test IsTemporaryError
	tempErr := &testTemporaryError{temporary: true}
	if !IsTemporaryError(tempErr) {
		t.Error("Expected temporary error to be detected")
	}

	nonTempErr := errors.New("permanent error")
	if IsTemporaryError(nonTempErr) {
		t.Error("Expected non-temporary error to not be detected as temporary")
	}

	// Test IsNetworkError
	networkErr := errors.New("connection refused")
	if !IsNetworkError(networkErr) {
		t.Error("Expected network error to be detected")
	}

	nonNetworkErr := errors.New("validation error")
	if IsNetworkError(nonNetworkErr) {
		t.Error("Expected non-network error to not be detected as network error")
	}

	// Test IsRetryableStatusCode
	if !IsRetryableStatusCode(503) {
		t.Error("Expected 503 to be retryable")
	}
	if IsRetryableStatusCode(200) {
		t.Error("Expected 200 to not be retryable")
	}
}

// Test helper types
type testTemporaryError struct {
	temporary bool
	timeout   bool
}

func (e *testTemporaryError) Error() string {
	return "test temporary error"
}

func (e *testTemporaryError) Temporary() bool {
	return e.temporary
}

func (e *testTemporaryError) Timeout() bool {
	return e.timeout
}

// Benchmark tests
func BenchmarkCircuitBreaker_ClosedState(b *testing.B) {
	cb := NewCircuitBreaker("benchmark", DefaultCircuitBreakerConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkRetryExecutor_NoRetries(b *testing.B) {
	policy := NewFixedDelayPolicy(3, time.Millisecond, func(err error, attempt int) bool {
		return err != nil
	})
	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkReliabilityService_Success(b *testing.B) {
	service := NewReliabilityService("benchmark", DefaultReliabilityConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}