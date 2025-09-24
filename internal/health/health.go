package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	// StatusHealthy indicates the component is healthy
	StatusHealthy HealthStatus = "healthy"
	// StatusUnhealthy indicates the component is unhealthy
	StatusUnhealthy HealthStatus = "unhealthy"
	// StatusDegraded indicates the component is partially healthy
	StatusDegraded HealthStatus = "degraded"
	// StatusUnknown indicates the component status is unknown
	StatusUnknown HealthStatus = "unknown"
)

// HealthCheck represents a health check for a component
type HealthCheck struct {
	Name        string                                     `json:"name"`
	Description string                                     `json:"description"`
	CheckFunc   func(context.Context) (HealthStatus, error) `json:"-"`
	Timeout     time.Duration                              `json:"timeout"`
	Critical    bool                                       `json:"critical"`
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Name        string        `json:"name"`
	Status      HealthStatus  `json:"status"`
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
	Critical    bool          `json:"critical"`
	Details     interface{}   `json:"details,omitempty"`
}

// HealthReport represents the overall health status of the system
type HealthReport struct {
	Status      HealthStatus               `json:"status"`
	Timestamp   time.Time                  `json:"timestamp"`
	Duration    time.Duration              `json:"duration"`
	Version     string                     `json:"version,omitempty"`
	ServiceName string                     `json:"service_name,omitempty"`
	Results     map[string]*HealthResult   `json:"results"`
	Summary     *HealthSummary             `json:"summary"`
}

// HealthSummary provides a summary of health check results
type HealthSummary struct {
	Total       int `json:"total"`
	Healthy     int `json:"healthy"`
	Unhealthy   int `json:"unhealthy"`
	Degraded    int `json:"degraded"`
	Unknown     int `json:"unknown"`
	CriticalFailed int `json:"critical_failed"`
}

// HealthChecker manages and executes health checks
type HealthChecker struct {
	checks      map[string]*HealthCheck
	mutex       sync.RWMutex
	version     string
	serviceName string
	timeout     time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(serviceName, version string) *HealthChecker {
	return &HealthChecker{
		checks:      make(map[string]*HealthCheck),
		serviceName: serviceName,
		version:     version,
		timeout:     time.Second * 30, // Default timeout
	}
}

// SetTimeout sets the default timeout for health checks
func (hc *HealthChecker) SetTimeout(timeout time.Duration) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.timeout = timeout
}

// RegisterCheck registers a health check
func (hc *HealthChecker) RegisterCheck(check *HealthCheck) error {
	if check == nil {
		return fmt.Errorf("health check cannot be nil")
	}
	if check.Name == "" {
		return fmt.Errorf("health check name cannot be empty")
	}
	if check.CheckFunc == nil {
		return fmt.Errorf("health check function cannot be nil")
	}

	// Set default timeout if not specified
	if check.Timeout == 0 {
		check.Timeout = hc.timeout
	}

	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.checks[check.Name] = check
	return nil
}

// UnregisterCheck removes a health check
func (hc *HealthChecker) UnregisterCheck(name string) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	delete(hc.checks, name)
}

// GetCheck returns a health check by name
func (hc *HealthChecker) GetCheck(name string) (*HealthCheck, bool) {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()
	check, exists := hc.checks[name]
	return check, exists
}

// ListChecks returns all registered health checks
func (hc *HealthChecker) ListChecks() map[string]*HealthCheck {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	checks := make(map[string]*HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	return checks
}

// CheckHealth executes all registered health checks
func (hc *HealthChecker) CheckHealth(ctx context.Context) *HealthReport {
	startTime := time.Now()

	hc.mutex.RLock()
	checks := make(map[string]*HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mutex.RUnlock()

	results := make(map[string]*HealthResult)
	var wg sync.WaitGroup
	resultMutex := sync.Mutex{}

	// Execute all health checks concurrently
	for name, check := range checks {
		wg.Add(1)
		go func(name string, check *HealthCheck) {
			defer wg.Done()
			result := hc.executeCheck(ctx, name, check)

			resultMutex.Lock()
			results[name] = result
			resultMutex.Unlock()
		}(name, check)
	}

	wg.Wait()

	// Calculate overall status and summary
	summary := hc.calculateSummary(results)
	overallStatus := hc.calculateOverallStatus(results)

	return &HealthReport{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Duration:    time.Since(startTime),
		Version:     hc.version,
		ServiceName: hc.serviceName,
		Results:     results,
		Summary:     summary,
	}
}

// CheckSingle executes a single health check by name
func (hc *HealthChecker) CheckSingle(ctx context.Context, name string) (*HealthResult, error) {
	check, exists := hc.GetCheck(name)
	if !exists {
		return nil, fmt.Errorf("health check '%s' not found", name)
	}

	return hc.executeCheck(ctx, name, check), nil
}

// executeCheck executes a single health check
func (hc *HealthChecker) executeCheck(ctx context.Context, name string, check *HealthCheck) *HealthResult {
	startTime := time.Now()

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	result := &HealthResult{
		Name:      name,
		Timestamp: startTime,
		Critical:  check.Critical,
	}

	// Execute the health check
	status, err := check.CheckFunc(checkCtx)
	result.Duration = time.Since(startTime)
	result.Status = status

	if err != nil {
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Health check failed: %v", err)
		if result.Status == StatusHealthy {
			result.Status = StatusUnhealthy
		}
	}

	return result
}

// calculateSummary calculates the summary of health check results
func (hc *HealthChecker) calculateSummary(results map[string]*HealthResult) *HealthSummary {
	summary := &HealthSummary{}

	for _, result := range results {
		summary.Total++
		switch result.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusUnhealthy:
			summary.Unhealthy++
			if result.Critical {
				summary.CriticalFailed++
			}
		case StatusDegraded:
			summary.Degraded++
		case StatusUnknown:
			summary.Unknown++
		}
	}

	return summary
}

// calculateOverallStatus determines the overall system health status
func (hc *HealthChecker) calculateOverallStatus(results map[string]*HealthResult) HealthStatus {
	if len(results) == 0 {
		return StatusUnknown
	}

	hasUnhealthy := false
	hasDegraded := false
	hasCriticalFailures := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
			if result.Critical {
				hasCriticalFailures = true
			}
		case StatusDegraded:
			hasDegraded = true
		case StatusUnknown:
			if result.Critical {
				hasCriticalFailures = true
			}
		}
	}

	// If any critical check fails, system is unhealthy
	if hasCriticalFailures {
		return StatusUnhealthy
	}

	// If any non-critical check is unhealthy or any is degraded
	if hasUnhealthy || hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// HealthEndpoint provides HTTP endpoints for health checks
type HealthEndpoint struct {
	checker *HealthChecker
	mux     *http.ServeMux
}

// NewHealthEndpoint creates a new health endpoint
func NewHealthEndpoint(checker *HealthChecker) *HealthEndpoint {
	endpoint := &HealthEndpoint{
		checker: checker,
		mux:     http.NewServeMux(),
	}

	endpoint.setupRoutes()
	return endpoint
}

// setupRoutes sets up the HTTP routes for health endpoints
func (he *HealthEndpoint) setupRoutes() {
	he.mux.HandleFunc("/health", he.handleHealth)
	he.mux.HandleFunc("/health/live", he.handleLiveness)
	he.mux.HandleFunc("/health/ready", he.handleReadiness)
	he.mux.HandleFunc("/health/check/", he.handleSingleCheck)
}

// ServeHTTP implements http.Handler
func (he *HealthEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	he.mux.ServeHTTP(w, r)
}

// handleHealth handles the main health check endpoint
func (he *HealthEndpoint) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	report := he.checker.CheckHealth(ctx)

	w.Header().Set("Content-Type", "application/json")

	// Set appropriate HTTP status code based on health status
	switch report.Status {
	case StatusHealthy:
		w.WriteHeader(http.StatusOK)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK) // Still OK, but degraded
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	case StatusUnknown:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleLiveness handles the liveness probe endpoint (simpler check)
func (he *HealthEndpoint) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Liveness is simpler - just check if the service is running
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"message":   "Service is alive",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleReadiness handles the readiness probe endpoint
func (he *HealthEndpoint) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// For readiness, only check critical components
	report := he.checker.CheckHealth(ctx)

	// Filter to only critical checks for readiness
	criticalHealthy := true
	for _, result := range report.Results {
		if result.Critical && result.Status != StatusHealthy {
			criticalHealthy = false
			break
		}
	}

	response := map[string]interface{}{
		"timestamp": time.Now(),
		"critical_checks_passing": criticalHealthy,
	}

	w.Header().Set("Content-Type", "application/json")

	if criticalHealthy {
		response["status"] = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response["status"] = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// handleSingleCheck handles individual health check endpoints
func (he *HealthEndpoint) handleSingleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract check name from URL path
	path := r.URL.Path
	if len(path) <= len("/health/check/") {
		http.Error(w, "Check name required", http.StatusBadRequest)
		return
	}

	checkName := path[len("/health/check/"):]
	if checkName == "" {
		http.Error(w, "Check name required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := he.checker.CheckSingle(ctx, checkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Set status code based on health check result
	switch result.Status {
	case StatusHealthy:
		w.WriteHeader(http.StatusOK)
	case StatusDegraded:
		w.WriteHeader(http.StatusOK)
	case StatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	case StatusUnknown:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(result)
}

// Common health check functions

// DatabaseHealthCheck creates a health check for database connectivity
func DatabaseHealthCheck(name, description string, pingFunc func(context.Context) error) *HealthCheck {
	return &HealthCheck{
		Name:        name,
		Description: description,
		Critical:    true,
		Timeout:     time.Second * 5,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			if err := pingFunc(ctx); err != nil {
				return StatusUnhealthy, err
			}
			return StatusHealthy, nil
		},
	}
}

// KMSHealthCheck creates a health check for KMS connectivity
func KMSHealthCheck(name, description string, checkFunc func(context.Context) error) *HealthCheck {
	return &HealthCheck{
		Name:        name,
		Description: description,
		Critical:    true,
		Timeout:     time.Second * 10,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			if err := checkFunc(ctx); err != nil {
				return StatusUnhealthy, err
			}
			return StatusHealthy, nil
		},
	}
}

// DiskSpaceHealthCheck creates a health check for disk space
func DiskSpaceHealthCheck(name, path string, threshold float64) *HealthCheck {
	return &HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("Check disk space on %s", path),
		Critical:    false,
		Timeout:     time.Second * 5,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			// This would be implemented with actual disk space checking
			// For now, return healthy as placeholder
			return StatusHealthy, nil
		},
	}
}

// MemoryHealthCheck creates a health check for memory usage
func MemoryHealthCheck(name string, threshold float64) *HealthCheck {
	return &HealthCheck{
		Name:        name,
		Description: "Check memory usage",
		Critical:    false,
		Timeout:     time.Second * 2,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			// This would be implemented with actual memory checking
			// For now, return healthy as placeholder
			return StatusHealthy, nil
		},
	}
}

// CircuitBreakerHealthCheck creates a health check for circuit breaker status
func CircuitBreakerHealthCheck(name, description string, isHealthyFunc func() bool) *HealthCheck {
	return &HealthCheck{
		Name:        name,
		Description: description,
		Critical:    false,
		Timeout:     time.Second * 1,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			if isHealthyFunc() {
				return StatusHealthy, nil
			}
			return StatusDegraded, fmt.Errorf("circuit breaker is open")
		},
	}
}

// Global health checker instance
var globalHealthChecker *HealthChecker
var globalHealthCheckerOnce sync.Once

// GetGlobalHealthChecker returns the global health checker instance
func GetGlobalHealthChecker() *HealthChecker {
	globalHealthCheckerOnce.Do(func() {
		globalHealthChecker = NewHealthChecker("encx-service", "1.0.0")
	})
	return globalHealthChecker
}

// RegisterGlobalCheck registers a check with the global health checker
func RegisterGlobalCheck(check *HealthCheck) error {
	return GetGlobalHealthChecker().RegisterCheck(check)
}

// CheckGlobalHealth executes all global health checks
func CheckGlobalHealth(ctx context.Context) *HealthReport {
	return GetGlobalHealthChecker().CheckHealth(ctx)
}