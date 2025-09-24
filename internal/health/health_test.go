package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHealthChecker_RegisterCheck(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "test-check",
		Description: "A test health check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Timeout: time.Second * 5,
	}

	err := checker.RegisterCheck(check)
	if err != nil {
		t.Errorf("Expected no error registering check, got %v", err)
	}

	// Test retrieving the check
	retrievedCheck, exists := checker.GetCheck("test-check")
	if !exists {
		t.Error("Expected to find registered check")
	}
	if retrievedCheck.Name != "test-check" {
		t.Errorf("Expected check name 'test-check', got %s", retrievedCheck.Name)
	}
}

func TestHealthChecker_RegisterCheck_InvalidInputs(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	// Test nil check
	err := checker.RegisterCheck(nil)
	if err == nil {
		t.Error("Expected error for nil check")
	}

	// Test empty name
	check := &HealthCheck{
		Name: "",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}
	err = checker.RegisterCheck(check)
	if err == nil {
		t.Error("Expected error for empty name")
	}

	// Test nil check function
	check = &HealthCheck{
		Name:      "test",
		CheckFunc: nil,
	}
	err = checker.RegisterCheck(check)
	if err == nil {
		t.Error("Expected error for nil check function")
	}
}

func TestHealthChecker_CheckHealth_Success(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check1 := &HealthCheck{
		Name:        "check1",
		Description: "First check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: true,
	}

	check2 := &HealthCheck{
		Name:        "check2",
		Description: "Second check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: false,
	}

	checker.RegisterCheck(check1)
	checker.RegisterCheck(check2)

	ctx := context.Background()
	report := checker.CheckHealth(ctx)

	if report.Status != StatusHealthy {
		t.Errorf("Expected overall status to be healthy, got %v", report.Status)
	}

	if len(report.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(report.Results))
	}

	if report.Summary.Total != 2 {
		t.Errorf("Expected total count of 2, got %d", report.Summary.Total)
	}

	if report.Summary.Healthy != 2 {
		t.Errorf("Expected healthy count of 2, got %d", report.Summary.Healthy)
	}
}

func TestHealthChecker_CheckHealth_WithFailures(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check1 := &HealthCheck{
		Name:        "critical-check",
		Description: "Critical check that fails",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusUnhealthy, errors.New("critical failure")
		},
		Critical: true,
	}

	check2 := &HealthCheck{
		Name:        "non-critical-check",
		Description: "Non-critical check that succeeds",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: false,
	}

	checker.RegisterCheck(check1)
	checker.RegisterCheck(check2)

	ctx := context.Background()
	report := checker.CheckHealth(ctx)

	// Should be unhealthy because critical check failed
	if report.Status != StatusUnhealthy {
		t.Errorf("Expected overall status to be unhealthy, got %v", report.Status)
	}

	if report.Summary.CriticalFailed != 1 {
		t.Errorf("Expected 1 critical failure, got %d", report.Summary.CriticalFailed)
	}

	if report.Summary.Healthy != 1 {
		t.Errorf("Expected 1 healthy check, got %d", report.Summary.Healthy)
	}

	if report.Summary.Unhealthy != 1 {
		t.Errorf("Expected 1 unhealthy check, got %d", report.Summary.Unhealthy)
	}
}

func TestHealthChecker_CheckHealth_Degraded(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check1 := &HealthCheck{
		Name:        "healthy-check",
		Description: "Healthy check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: true,
	}

	check2 := &HealthCheck{
		Name:        "degraded-check",
		Description: "Degraded check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusDegraded, nil
		},
		Critical: false,
	}

	checker.RegisterCheck(check1)
	checker.RegisterCheck(check2)

	ctx := context.Background()
	report := checker.CheckHealth(ctx)

	// Should be degraded because one check is degraded
	if report.Status != StatusDegraded {
		t.Errorf("Expected overall status to be degraded, got %v", report.Status)
	}

	if report.Summary.Degraded != 1 {
		t.Errorf("Expected 1 degraded check, got %d", report.Summary.Degraded)
	}
}

func TestHealthChecker_CheckSingle(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "single-check",
		Description: "A single check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	checker.RegisterCheck(check)

	ctx := context.Background()
	result, err := checker.CheckSingle(ctx, "single-check")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %v", result.Status)
	}

	// Test non-existent check
	_, err = checker.CheckSingle(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent check")
	}
}

func TestHealthChecker_Timeout(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "slow-check",
		Description: "A slow check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			select {
			case <-time.After(time.Millisecond * 100):
				return StatusHealthy, nil
			case <-ctx.Done():
				return StatusUnhealthy, ctx.Err()
			}
		},
		Timeout: time.Millisecond * 50, // Shorter than the check duration
	}

	checker.RegisterCheck(check)

	ctx := context.Background()
	result, err := checker.CheckSingle(ctx, "slow-check")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// The check should have timed out and returned unhealthy
	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status unhealthy due to timeout, got %v", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message for timeout")
	}
}

func TestHealthEndpoint_HandleHealth(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "api-check",
		Description: "API health check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	checker.RegisterCheck(check)
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	endpoint.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report HealthReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if report.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %v", report.Status)
	}

	if len(report.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(report.Results))
	}
}

func TestHealthEndpoint_HandleHealth_Unhealthy(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "failing-check",
		Description: "A failing check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusUnhealthy, errors.New("check failed")
		},
		Critical: true,
	}

	checker.RegisterCheck(check)
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	endpoint.handleHealth(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var report HealthReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if report.Status != StatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %v", report.Status)
	}
}

func TestHealthEndpoint_HandleLiveness(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	endpoint.handleLiveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestHealthEndpoint_HandleReadiness(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	// Add a critical check that passes
	check := &HealthCheck{
		Name:        "critical-check",
		Description: "Critical readiness check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: true,
	}

	checker.RegisterCheck(check)
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	endpoint.handleReadiness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", response["status"])
	}
}

func TestHealthEndpoint_HandleReadiness_NotReady(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	// Add a critical check that fails
	check := &HealthCheck{
		Name:        "critical-failing-check",
		Description: "Critical check that fails",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusUnhealthy, errors.New("critical failure")
		},
		Critical: true,
	}

	checker.RegisterCheck(check)
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	endpoint.handleReadiness(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "not_ready" {
		t.Errorf("Expected status 'not_ready', got %v", response["status"])
	}
}

func TestHealthEndpoint_HandleSingleCheck(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "individual-check",
		Description: "Individual health check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	checker.RegisterCheck(check)
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/check/individual-check", nil)
	w := httptest.NewRecorder()

	endpoint.handleSingleCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result HealthResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if result.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %v", result.Status)
	}

	if result.Name != "individual-check" {
		t.Errorf("Expected check name 'individual-check', got %s", result.Name)
	}
}

func TestHealthEndpoint_HandleSingleCheck_NotFound(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodGet, "/health/check/non-existent", nil)
	w := httptest.NewRecorder()

	endpoint.handleSingleCheck(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHealthEndpoint_MethodNotAllowed(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")
	endpoint := NewHealthEndpoint(checker)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	endpoint.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCommonHealthChecks(t *testing.T) {
	// Test DatabaseHealthCheck
	dbCheck := DatabaseHealthCheck("test-db", "Test database", func(ctx context.Context) error {
		return nil // Simulate successful ping
	})

	if dbCheck.Name != "test-db" {
		t.Errorf("Expected name 'test-db', got %s", dbCheck.Name)
	}

	if !dbCheck.Critical {
		t.Error("Expected database check to be critical")
	}

	ctx := context.Background()
	status, err := dbCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status, got %v", status)
	}

	// Test KMSHealthCheck
	kmsCheck := KMSHealthCheck("test-kms", "Test KMS", func(ctx context.Context) error {
		return nil // Simulate successful check
	})

	if kmsCheck.Name != "test-kms" {
		t.Errorf("Expected name 'test-kms', got %s", kmsCheck.Name)
	}

	if !kmsCheck.Critical {
		t.Error("Expected KMS check to be critical")
	}

	status, err = kmsCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status, got %v", status)
	}

	// Test CircuitBreakerHealthCheck
	cbCheck := CircuitBreakerHealthCheck("test-cb", "Test circuit breaker", func() bool {
		return true // Simulate healthy circuit breaker
	})

	if cbCheck.Critical {
		t.Error("Expected circuit breaker check to not be critical")
	}

	status, err = cbCheck.CheckFunc(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if status != StatusHealthy {
		t.Errorf("Expected healthy status, got %v", status)
	}

	// Test circuit breaker unhealthy
	cbCheckUnhealthy := CircuitBreakerHealthCheck("test-cb-unhealthy", "Test unhealthy circuit breaker", func() bool {
		return false // Simulate unhealthy circuit breaker
	})

	status, err = cbCheckUnhealthy.CheckFunc(ctx)
	if err == nil {
		t.Error("Expected error for unhealthy circuit breaker")
	}
	if status != StatusDegraded {
		t.Errorf("Expected degraded status, got %v", status)
	}
}

func TestGlobalHealthChecker(t *testing.T) {
	// Clear any existing checks from global instance
	globalHealthChecker = nil
	globalHealthCheckerOnce = sync.Once{}

	checker := GetGlobalHealthChecker()
	if checker == nil {
		t.Error("Expected global health checker to be created")
	}

	// Test that subsequent calls return the same instance
	checker2 := GetGlobalHealthChecker()
	if checker != checker2 {
		t.Error("Expected same global health checker instance")
	}

	// Test global check registration
	check := &HealthCheck{
		Name:        "global-test-check",
		Description: "Global test check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	err := RegisterGlobalCheck(check)
	if err != nil {
		t.Errorf("Expected no error registering global check, got %v", err)
	}

	// Test global health check
	ctx := context.Background()
	report := CheckGlobalHealth(ctx)

	if len(report.Results) == 0 {
		t.Error("Expected at least one result from global health check")
	}

	if _, exists := report.Results["global-test-check"]; !exists {
		t.Error("Expected to find global-test-check in results")
	}
}

func TestHealthChecker_UnregisterCheck(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "temporary-check",
		Description: "A temporary check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	// Register and verify it exists
	checker.RegisterCheck(check)
	_, exists := checker.GetCheck("temporary-check")
	if !exists {
		t.Error("Expected check to exist after registration")
	}

	// Unregister and verify it's gone
	checker.UnregisterCheck("temporary-check")
	_, exists = checker.GetCheck("temporary-check")
	if exists {
		t.Error("Expected check to not exist after unregistration")
	}
}

func TestHealthChecker_ListChecks(t *testing.T) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check1 := &HealthCheck{
		Name:        "check1",
		Description: "First check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	check2 := &HealthCheck{
		Name:        "check2",
		Description: "Second check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	checker.RegisterCheck(check1)
	checker.RegisterCheck(check2)

	checks := checker.ListChecks()
	if len(checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(checks))
	}

	if _, exists := checks["check1"]; !exists {
		t.Error("Expected to find check1 in list")
	}

	if _, exists := checks["check2"]; !exists {
		t.Error("Expected to find check2 in list")
	}
}

// Benchmark tests
func BenchmarkHealthChecker_CheckHealth_SingleCheck(b *testing.B) {
	checker := NewHealthChecker("test-service", "1.0.0")

	check := &HealthCheck{
		Name:        "benchmark-check",
		Description: "Benchmark check",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
	}

	checker.RegisterCheck(check)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.CheckHealth(ctx)
	}
}

func BenchmarkHealthChecker_CheckHealth_MultipleChecks(b *testing.B) {
	checker := NewHealthChecker("test-service", "1.0.0")

	// Register multiple checks
	for i := 0; i < 10; i++ {
		check := &HealthCheck{
			Name:        fmt.Sprintf("benchmark-check-%d", i),
			Description: fmt.Sprintf("Benchmark check %d", i),
			CheckFunc: func(ctx context.Context) (HealthStatus, error) {
				return StatusHealthy, nil
			},
		}
		checker.RegisterCheck(check)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.CheckHealth(ctx)
	}
}

func TestHealthEndpoint_Integration(t *testing.T) {
	checker := NewHealthChecker("integration-service", "1.0.0")

	// Register multiple checks
	healthyCheck := &HealthCheck{
		Name:        "healthy-service",
		Description: "Always healthy service",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: false,
	}

	criticalCheck := &HealthCheck{
		Name:        "critical-service",
		Description: "Critical service",
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			return StatusHealthy, nil
		},
		Critical: true,
	}

	checker.RegisterCheck(healthyCheck)
	checker.RegisterCheck(criticalCheck)

	endpoint := NewHealthEndpoint(checker)

	// Test full integration through HTTP server
	server := httptest.NewServer(endpoint)
	defer server.Close()

	// Test main health endpoint
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test liveness endpoint
	resp, err = http.Get(server.URL + "/health/live")
	if err != nil {
		t.Fatalf("Failed to get liveness endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected liveness status 200, got %d", resp.StatusCode)
	}

	// Test readiness endpoint
	resp, err = http.Get(server.URL + "/health/ready")
	if err != nil {
		t.Fatalf("Failed to get readiness endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected readiness status 200, got %d", resp.StatusCode)
	}

	// Test individual check endpoint
	resp, err = http.Get(server.URL + "/health/check/healthy-service")
	if err != nil {
		t.Fatalf("Failed to get individual check endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected individual check status 200, got %d", resp.StatusCode)
	}
}