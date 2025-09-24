package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hengadev/encx/internal/reliability"
)

func TestReliabilityHealthChecker_CreateCircuitBreakerHealthCheck(t *testing.T) {
	reliabilityManager := reliability.NewReliabilityManager()
	rhc := NewReliabilityHealthChecker(reliabilityManager)

	// Create a reliability service
	config := reliability.DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 1

	// Add the service to the manager manually (since GetOrCreate would create it)
	reliabilityManager.GetOrCreate("test-service", config)

	// Create health check
	healthCheck := rhc.CreateCircuitBreakerHealthCheck("test-service")

	if healthCheck.Name != "circuit_breaker_test-service" {
		t.Errorf("Expected name 'circuit_breaker_test-service', got %s", healthCheck.Name)
	}

	if healthCheck.Critical {
		t.Error("Expected circuit breaker health check to not be critical")
	}

	// Test with healthy service
	ctx := context.Background()
	status, err := healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error for healthy service, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status, got %v", status)
	}
}

func TestReliabilityHealthChecker_CreateCircuitBreakerHealthCheck_Unhealthy(t *testing.T) {
	reliabilityManager := reliability.NewReliabilityManager()
	rhc := NewReliabilityHealthChecker(reliabilityManager)

	// Create a reliability service with low failure threshold
	config := reliability.DefaultReliabilityConfig()
	config.CircuitBreaker.FailureThreshold = 1
	config.Retry.MaxAttempts = 1

	service := reliabilityManager.GetOrCreate("unhealthy-service", config)

	// Make the service unhealthy by causing failures
	ctx := context.Background()
	service.Execute(ctx, func(ctx context.Context) error {
		return errors.New("test error")
	})

	// Create health check
	healthCheck := rhc.CreateCircuitBreakerHealthCheck("unhealthy-service")

	// Test with unhealthy service
	status, err := healthCheck.CheckFunc(ctx)
	if err == nil {
		t.Error("Expected error for unhealthy service")
	}
	if status != StatusDegraded {
		t.Errorf("Expected degraded status, got %v", status)
	}
}

func TestReliabilityHealthChecker_CreateReliabilityOverviewHealthCheck(t *testing.T) {
	reliabilityManager := reliability.NewReliabilityManager()
	rhc := NewReliabilityHealthChecker(reliabilityManager)

	// Create health check
	healthCheck := rhc.CreateReliabilityOverviewHealthCheck()

	if healthCheck.Name != "reliability_overview" {
		t.Errorf("Expected name 'reliability_overview', got %s", healthCheck.Name)
	}

	ctx := context.Background()

	// Test with no services
	status, err := healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with no services, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with no services, got %v", status)
	}

	// Add some services
	config := reliability.DefaultReliabilityConfig()
	reliabilityManager.GetOrCreate("service1", config)
	reliabilityManager.GetOrCreate("service2", config)

	// Test with healthy services
	status, err = healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with healthy services, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with healthy services, got %v", status)
	}
}

func TestReliabilityHealthChecker_CreateRetryStatsHealthCheck(t *testing.T) {
	reliabilityManager := reliability.NewReliabilityManager()
	rhc := NewReliabilityHealthChecker(reliabilityManager)

	// Create a service and generate some retry stats
	config := reliability.DefaultReliabilityConfig()
	config.Retry.MaxAttempts = 3
	service := reliabilityManager.GetOrCreate("retry-test-service", config)

	ctx := context.Background()

	// Execute some operations to generate stats
	service.Execute(ctx, func(ctx context.Context) error {
		return nil // Success
	})

	// Create health check with 50% max failure rate
	healthCheck := rhc.CreateRetryStatsHealthCheck("retry-test-service", 0.5)

	if healthCheck.Name != "retry_stats_retry-test-service" {
		t.Errorf("Expected name 'retry_stats_retry-test-service', got %s", healthCheck.Name)
	}

	// Test with good stats
	status, err := healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with good stats, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with good stats, got %v", status)
	}
}

func TestReliabilityHealthChecker_RegisterAllReliabilityHealthChecks(t *testing.T) {
	reliabilityManager := reliability.NewReliabilityManager()
	rhc := NewReliabilityHealthChecker(reliabilityManager)
	healthChecker := NewHealthChecker("test", "1.0.0")

	// Create some services first
	config := reliability.DefaultReliabilityConfig()
	reliabilityManager.GetOrCreate("service1", config)
	reliabilityManager.GetOrCreate("service2", config)

	// Register health checks
	err := rhc.RegisterAllReliabilityHealthChecks(healthChecker)
	if err != nil {
		t.Errorf("Expected no error registering health checks, got %v", err)
	}

	// Verify checks were registered
	checks := healthChecker.ListChecks()

	// Should have overview check plus circuit breaker and retry checks for each service
	expectedChecks := []string{
		"reliability_overview",
		"circuit_breaker_service1",
		"retry_stats_service1",
		"circuit_breaker_service2",
		"retry_stats_service2",
	}

	if len(checks) != len(expectedChecks) {
		t.Errorf("Expected %d checks, got %d", len(expectedChecks), len(checks))
	}

	for _, expectedCheck := range expectedChecks {
		if _, exists := checks[expectedCheck]; !exists {
			t.Errorf("Expected to find check %s", expectedCheck)
		}
	}
}

func TestCryptoReliabilityHealthChecker_CreateKMSOperationsHealthCheck(t *testing.T) {
	cryptoManager := reliability.NewCryptoReliabilityManager(reliability.DefaultCryptoReliabilityConfig())
	crhc := NewCryptoReliabilityHealthChecker(cryptoManager)

	// Create health check
	healthCheck := crhc.CreateKMSOperationsHealthCheck()

	if healthCheck.Name != "kms_operations_health" {
		t.Errorf("Expected name 'kms_operations_health', got %s", healthCheck.Name)
	}

	if !healthCheck.Critical {
		t.Error("Expected KMS operations health check to be critical")
	}

	ctx := context.Background()

	// Test with no KMS services
	status, err := healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with no KMS services, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with no KMS services, got %v", status)
	}

	// Add a KMS operation
	cryptoManager.ExecuteKMSOperation(ctx, "encrypt", func(ctx context.Context) error {
		return nil
	})

	// Test with healthy KMS service
	status, err = healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with healthy KMS service, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with healthy KMS service, got %v", status)
	}
}

func TestCryptoReliabilityHealthChecker_CreateDatabaseOperationsHealthCheck(t *testing.T) {
	cryptoManager := reliability.NewCryptoReliabilityManager(reliability.DefaultCryptoReliabilityConfig())
	crhc := NewCryptoReliabilityHealthChecker(cryptoManager)

	// Create health check
	healthCheck := crhc.CreateDatabaseOperationsHealthCheck()

	if healthCheck.Name != "database_operations_health" {
		t.Errorf("Expected name 'database_operations_health', got %s", healthCheck.Name)
	}

	if !healthCheck.Critical {
		t.Error("Expected database operations health check to be critical")
	}

	ctx := context.Background()

	// Test with no database services
	status, err := healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with no database services, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with no database services, got %v", status)
	}

	// Add a database operation
	cryptoManager.ExecuteDatabaseOperation(ctx, "query", func(ctx context.Context) error {
		return nil
	})

	// Test with healthy database service
	status, err = healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with healthy database service, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with healthy database service, got %v", status)
	}
}

func TestCryptoReliabilityHealthChecker_CreateNetworkOperationsHealthCheck(t *testing.T) {
	cryptoManager := reliability.NewCryptoReliabilityManager(reliability.DefaultCryptoReliabilityConfig())
	crhc := NewCryptoReliabilityHealthChecker(cryptoManager)

	// Create health check
	healthCheck := crhc.CreateNetworkOperationsHealthCheck()

	if healthCheck.Name != "network_operations_health" {
		t.Errorf("Expected name 'network_operations_health', got %s", healthCheck.Name)
	}

	if healthCheck.Critical {
		t.Error("Expected network operations health check to not be critical")
	}

	ctx := context.Background()

	// Test with no network services
	status, err := healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with no network services, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with no network services, got %v", status)
	}

	// Add a network operation
	cryptoManager.ExecuteNetworkOperation(ctx, "api_call", func(ctx context.Context) error {
		return nil
	})

	// Test with healthy network service
	status, err = healthCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error with healthy network service, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status with healthy network service, got %v", status)
	}
}

func TestCryptoReliabilityHealthChecker_RegisterAllCryptoReliabilityHealthChecks(t *testing.T) {
	cryptoManager := reliability.NewCryptoReliabilityManager(reliability.DefaultCryptoReliabilityConfig())
	crhc := NewCryptoReliabilityHealthChecker(cryptoManager)
	healthChecker := NewHealthChecker("test", "1.0.0")

	// Register health checks
	err := crhc.RegisterAllCryptoReliabilityHealthChecks(healthChecker)
	if err != nil {
		t.Errorf("Expected no error registering crypto health checks, got %v", err)
	}

	// Verify checks were registered
	checks := healthChecker.ListChecks()

	expectedChecks := []string{
		"kms_operations_health",
		"database_operations_health",
		"network_operations_health",
	}

	if len(checks) != len(expectedChecks) {
		t.Errorf("Expected %d checks, got %d", len(expectedChecks), len(checks))
	}

	for _, expectedCheck := range expectedChecks {
		if _, exists := checks[expectedCheck]; !exists {
			t.Errorf("Expected to find check %s", expectedCheck)
		}
	}
}

func TestRegisterReliabilityHealthChecks(t *testing.T) {
	// Clear global health checker
	globalHealthChecker = nil
	globalHealthCheckerOnce = sync.Once{}

	reliabilityManager := reliability.NewReliabilityManager()

	// Create some services
	config := reliability.DefaultReliabilityConfig()
	reliabilityManager.GetOrCreate("global-service", config)

	// Register health checks
	err := RegisterReliabilityHealthChecks(reliabilityManager)
	if err != nil {
		t.Errorf("Expected no error registering global reliability health checks, got %v", err)
	}

	// Verify checks were registered with global health checker
	globalChecker := GetGlobalHealthChecker()
	checks := globalChecker.ListChecks()

	if len(checks) == 0 {
		t.Error("Expected some checks to be registered")
	}

	// Should have at least the overview check
	if _, exists := checks["reliability_overview"]; !exists {
		t.Error("Expected to find reliability_overview check")
	}
}

func TestRegisterCryptoReliabilityHealthChecks(t *testing.T) {
	// Clear global health checker
	globalHealthChecker = nil
	globalHealthCheckerOnce = sync.Once{}

	cryptoManager := reliability.NewCryptoReliabilityManager(reliability.DefaultCryptoReliabilityConfig())

	// Register health checks
	err := RegisterCryptoReliabilityHealthChecks(cryptoManager)
	if err != nil {
		t.Errorf("Expected no error registering global crypto health checks, got %v", err)
	}

	// Verify checks were registered with global health checker
	globalChecker := GetGlobalHealthChecker()
	checks := globalChecker.ListChecks()

	expectedChecks := []string{
		"kms_operations_health",
		"database_operations_health",
		"network_operations_health",
	}

	for _, expectedCheck := range expectedChecks {
		if _, exists := checks[expectedCheck]; !exists {
			t.Errorf("Expected to find check %s", expectedCheck)
		}
	}
}

func TestIntegration_HealthAndReliability(t *testing.T) {
	// Clear global health checker
	globalHealthChecker = nil
	globalHealthCheckerOnce = sync.Once{}

	// Create managers
	reliabilityManager := reliability.NewReliabilityManager()
	cryptoManager := reliability.NewCryptoReliabilityManager(reliability.DefaultCryptoReliabilityConfig())

	// Register health checks
	err := RegisterReliabilityHealthChecks(reliabilityManager)
	if err != nil {
		t.Errorf("Failed to register reliability health checks: %v", err)
	}

	err = RegisterCryptoReliabilityHealthChecks(cryptoManager)
	if err != nil {
		t.Errorf("Failed to register crypto reliability health checks: %v", err)
	}

	// Execute some operations to create services
	ctx := context.Background()

	// General reliability operations
	service := reliabilityManager.GetOrCreate("integration-test", reliability.DefaultReliabilityConfig())
	service.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Crypto operations
	cryptoManager.ExecuteKMSOperation(ctx, "integration-encrypt", func(ctx context.Context) error {
		return nil
	})

	cryptoManager.ExecuteDatabaseOperation(ctx, "integration-query", func(ctx context.Context) error {
		return nil
	})

	// Check overall health
	globalChecker := GetGlobalHealthChecker()
	report := globalChecker.CheckHealth(ctx)

	if report.Status != StatusHealthy {
		t.Errorf("Expected overall healthy status, got %v", report.Status)
	}

	// Verify we have multiple health checks running
	if len(report.Results) < 3 {
		t.Errorf("Expected at least 3 health check results, got %d", len(report.Results))
	}

	// Create an HTTP endpoint and test it
	endpoint := NewHealthEndpoint(globalChecker)

	// Test integration through HTTP
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	endpoint.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected HTTP status 200, got %d", w.Code)
	}

	var httpReport HealthReport
	if err := json.Unmarshal(w.Body.Bytes(), &httpReport); err != nil {
		t.Errorf("Failed to unmarshal HTTP response: %v", err)
	}

	if httpReport.Status != StatusHealthy {
		t.Errorf("Expected HTTP report status healthy, got %v", httpReport.Status)
	}
}