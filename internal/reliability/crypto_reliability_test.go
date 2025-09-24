package reliability

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCryptoReliabilityManager_KMSOperations(t *testing.T) {
	config := DefaultCryptoReliabilityConfig()
	config.KMSOperations.CircuitBreaker.FailureThreshold = 2
	config.KMSOperations.Retry.MaxAttempts = 2

	manager := NewCryptoReliabilityManager(config)
	ctx := context.Background()

	// Test successful KMS operation
	callCount := 0
	err := manager.ExecuteKMSOperation(ctx, "encrypt", func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Check service health
	if !manager.IsOperationHealthy("kms", "encrypt") {
		t.Error("KMS encrypt operation should be healthy")
	}
}

func TestCryptoReliabilityManager_DatabaseOperations(t *testing.T) {
	config := DefaultCryptoReliabilityConfig()
	config.DatabaseOperations.CircuitBreaker.FailureThreshold = 2
	config.DatabaseOperations.Retry.MaxAttempts = 3

	manager := NewCryptoReliabilityManager(config)
	ctx := context.Background()

	testError := errors.New("database connection error")

	// Test database operation with retry
	callCount := 0
	err := manager.ExecuteDatabaseOperation(ctx, "query", func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return testError
		}
		return nil // Success on third attempt
	})

	if err != nil {
		t.Errorf("Expected no error after retries, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestCryptoReliabilityManager_NetworkOperations(t *testing.T) {
	config := DefaultCryptoReliabilityConfig()
	config.NetworkOperations.CircuitBreaker.FailureThreshold = 1
	config.NetworkOperations.CircuitBreaker.Timeout = time.Millisecond * 50

	manager := NewCryptoReliabilityManager(config)
	ctx := context.Background()

	testError := errors.New("network timeout")

	// Test network operation failure and circuit opening
	err1 := manager.ExecuteNetworkOperation(ctx, "api_call", func(ctx context.Context) error {
		return testError
	})
	if err1 != testError {
		t.Errorf("Expected test error, got %v", err1)
	}

	// Second call should trigger circuit breaker
	err2 := manager.ExecuteNetworkOperation(ctx, "api_call", func(ctx context.Context) error {
		t.Error("Function should not be called when circuit is open")
		return nil
	})
	if !IsCircuitOpenError(err2) {
		t.Errorf("Expected circuit open error, got %v", err2)
	}

	// Check service health
	if manager.IsOperationHealthy("network", "api_call") {
		t.Error("Network api_call operation should not be healthy when circuit is open")
	}
}

func TestCryptoReliabilityManager_WithFallback(t *testing.T) {
	config := DefaultCryptoReliabilityConfig()
	config.KMSOperations.CircuitBreaker.FailureThreshold = 1
	config.KMSOperations.Retry.MaxAttempts = 1

	manager := NewCryptoReliabilityManager(config)
	ctx := context.Background()

	testError := errors.New("kms unavailable")
	fallbackCalled := false

	err := manager.ExecuteKMSOperationWithFallback(ctx, "decrypt",
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

func TestCryptoReliabilityManager_ExecuteOperation(t *testing.T) {
	manager := NewCryptoReliabilityManager(DefaultCryptoReliabilityConfig())
	ctx := context.Background()

	// Test KMS operation type
	kmsCallCount := 0
	err := manager.ExecuteOperation(ctx, KMSOperation, "sign", func(ctx context.Context) error {
		kmsCallCount++
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error for KMS operation, got %v", err)
	}
	if kmsCallCount != 1 {
		t.Errorf("Expected 1 KMS call, got %d", kmsCallCount)
	}

	// Test Database operation type
	dbCallCount := 0
	err = manager.ExecuteOperation(ctx, DatabaseOperation, "insert", func(ctx context.Context) error {
		dbCallCount++
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error for database operation, got %v", err)
	}
	if dbCallCount != 1 {
		t.Errorf("Expected 1 database call, got %d", dbCallCount)
	}

	// Test Network operation type
	networkCallCount := 0
	err = manager.ExecuteOperation(ctx, NetworkOperation, "request", func(ctx context.Context) error {
		networkCallCount++
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error for network operation, got %v", err)
	}
	if networkCallCount != 1 {
		t.Errorf("Expected 1 network call, got %d", networkCallCount)
	}
}

func TestReliabilityWrapper(t *testing.T) {
	config := DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 2
	config.Retry.MaxAttempts = 3

	wrapper := NewReliabilityWrapperWithConfig("test-wrapper", config)
	ctx := context.Background()

	// Test successful operation
	callCount := 0
	err := wrapper.Wrap(ctx, func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Check wrapper health
	if !wrapper.IsHealthy() {
		t.Error("Wrapper should be healthy after successful operation")
	}

	// Check stats
	stats := wrapper.GetStats()
	if stats.Name != "test-wrapper" {
		t.Errorf("Expected service name 'test-wrapper', got %s", stats.Name)
	}
}

func TestReliabilityWrapper_WithFallback(t *testing.T) {
	config := DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 1
	config.Retry.MaxAttempts = 1

	wrapper := NewReliabilityWrapperWithConfig("fallback-test", config)
	ctx := context.Background()

	testError := errors.New("operation failed")
	fallbackCalled := false

	err := wrapper.WrapWithFallback(ctx,
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

func TestGlobalCryptoReliabilityFunctions(t *testing.T) {
	ctx := context.Background()

	// Test global execute crypto operation
	kmsCallCount := 0
	err := ExecuteCryptoOperation(ctx, KMSOperation, "global_test", func(ctx context.Context) error {
		kmsCallCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if kmsCallCount != 1 {
		t.Errorf("Expected 1 call, got %d", kmsCallCount)
	}

	// Test global execute with fallback
	fallbackCalled := false
	err = ExecuteCryptoOperationWithFallback(ctx, DatabaseOperation, "global_fallback_test",
		func(ctx context.Context) error {
			return errors.New("test error")
		},
		func(ctx context.Context) error {
			fallbackCalled = true
			return nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error with fallback, got %v", err)
	}
	if !fallbackCalled {
		t.Error("Expected fallback to be called")
	}

	// Test global stats retrieval
	stats := GetCryptoReliabilityStats()
	if len(stats) == 0 {
		t.Error("Expected at least some statistics")
	}

	// Test global health checks
	healthyServices := GetHealthyCryptoServices()
	if len(healthyServices) == 0 {
		t.Error("Expected at least some healthy services")
	}

	unhealthyServices := GetUnhealthyCryptoServices()
	// This might be empty, which is fine
	t.Logf("Healthy services: %v", healthyServices)
	t.Logf("Unhealthy services: %v", unhealthyServices)
}

func TestCryptoReliabilityConfig_Defaults(t *testing.T) {
	config := DefaultCryptoReliabilityConfig()

	// Verify KMS config is most restrictive
	if config.KMSOperations.CircuitBreaker.FailureThreshold >= config.DatabaseOperations.CircuitBreaker.FailureThreshold {
		t.Error("KMS operations should have lower failure threshold than database operations")
	}

	if config.KMSOperations.Retry.MaxAttempts >= config.DatabaseOperations.Retry.MaxAttempts {
		t.Error("KMS operations should have fewer max attempts than database operations")
	}

	// Verify Network config is most tolerant
	if config.NetworkOperations.CircuitBreaker.FailureThreshold <= config.KMSOperations.CircuitBreaker.FailureThreshold {
		t.Error("Network operations should have higher failure threshold than KMS operations")
	}

	if config.NetworkOperations.Retry.MaxAttempts < config.KMSOperations.Retry.MaxAttempts {
		t.Error("Network operations should have at least as many max attempts as KMS operations")
	}
}

func TestCryptoReliabilityManager_Stats(t *testing.T) {
	manager := NewCryptoReliabilityManager(DefaultCryptoReliabilityConfig())
	ctx := context.Background()

	// Execute some operations to generate stats
	manager.ExecuteKMSOperation(ctx, "test1", func(ctx context.Context) error {
		return nil
	})

	manager.ExecuteDatabaseOperation(ctx, "test2", func(ctx context.Context) error {
		return nil
	})

	manager.ExecuteNetworkOperation(ctx, "test3", func(ctx context.Context) error {
		return errors.New("test error")
	})

	// Get stats
	allStats := manager.GetAllStats()
	if len(allStats) != 3 {
		t.Errorf("Expected 3 service statistics, got %d", len(allStats))
	}

	// Check healthy services
	healthyServices := manager.GetHealthyServices()
	if len(healthyServices) < 2 {
		t.Errorf("Expected at least 2 healthy services, got %d", len(healthyServices))
	}

	// Check unhealthy services (network operation should have failed)
	unhealthyServices := manager.GetUnhealthyServices()
	// The network operation might not have triggered circuit breaker yet
	t.Logf("Healthy: %v, Unhealthy: %v", healthyServices, unhealthyServices)
}

// Benchmark tests
func BenchmarkCryptoReliabilityManager_KMSOperation(b *testing.B) {
	manager := NewCryptoReliabilityManager(DefaultCryptoReliabilityConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ExecuteKMSOperation(ctx, "benchmark", func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkReliabilityWrapper_Wrap(b *testing.B) {
	wrapper := NewReliabilityWrapper("benchmark")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapper.Wrap(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkGlobalExecuteCryptoOperation(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteCryptoOperation(ctx, KMSOperation, "benchmark", func(ctx context.Context) error {
			return nil
		})
	}
}