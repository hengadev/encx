package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ReliabilityConfig holds configuration for the reliability service
type ReliabilityConfig struct {
	// CircuitBreakerConfig for circuit breaker behavior
	CircuitBreaker CircuitBreakerConfig
	// RetryConfig for retry behavior
	Retry RetryConfig
	// EnableMetrics enables metrics collection
	EnableMetrics bool
	// MetricsPrefix for metric names
	MetricsPrefix string
}

// DefaultReliabilityConfig returns a default reliability configuration
func DefaultReliabilityConfig() ReliabilityConfig {
	return ReliabilityConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig(),
		Retry:          DefaultRetryConfig(),
		EnableMetrics:  true,
		MetricsPrefix:  "reliability",
	}
}

// ReliabilityService combines circuit breaker and retry policies for fault tolerance
type ReliabilityService struct {
	name           string
	config         ReliabilityConfig
	circuitBreaker *CircuitBreaker
	retryExecutor  *RetryExecutorWithStats
	metrics        *ReliabilityMetrics
	mutex          sync.RWMutex
}

// NewReliabilityService creates a new reliability service
func NewReliabilityService(name string, config ReliabilityConfig) *ReliabilityService {
	// Create circuit breaker with enhanced configuration
	cbConfig := config.CircuitBreaker
	if cbConfig.OnStateChange == nil {
		cbConfig.OnStateChange = func(name string, from, to CircuitState) {
			// Default state change handler
		}
	}

	circuitBreaker := NewCircuitBreaker(name, cbConfig)

	// Create retry executor with statistics
	retryPolicy := NewExponentialBackoffPolicy(config.Retry)
	retryExecutor := NewRetryExecutorWithStats(retryPolicy)

	// Create metrics collector if enabled
	var metrics *ReliabilityMetrics
	if config.EnableMetrics {
		metrics = NewReliabilityMetrics(name, config.MetricsPrefix)
	}

	service := &ReliabilityService{
		name:           name,
		config:         config,
		circuitBreaker: circuitBreaker,
		retryExecutor:  retryExecutor,
		metrics:        metrics,
	}

	// Set up retry callback to update metrics
	if config.EnableMetrics {
		retryExecutor.SetOnRetryCallback(func(attempt int, delay time.Duration, err error) {
			metrics.RecordRetry(attempt, delay, err)
		})
	}

	// Set up circuit breaker state change callback
	cbConfig.OnStateChange = func(cbName string, from, to CircuitState) {
		if config.EnableMetrics {
			metrics.RecordStateChange(from, to)
		}
	}
	circuitBreaker.config.OnStateChange = cbConfig.OnStateChange

	return service
}

// Execute executes an operation with both circuit breaker and retry protection
func (rs *ReliabilityService) Execute(ctx context.Context, operation func(context.Context) error) error {
	startTime := time.Now()

	// Execute with circuit breaker protection
	err := rs.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		// Within circuit breaker, execute with retry logic
		return rs.retryExecutor.ExecuteWithStats(ctx, operation)
	})

	duration := time.Since(startTime)

	// Record metrics
	if rs.metrics != nil {
		if err == nil {
			rs.metrics.RecordSuccess(duration)
		} else {
			rs.metrics.RecordFailure(duration, err)
		}
	}

	return err
}

// ExecuteWithFallback executes an operation with circuit breaker, retry, and fallback
func (rs *ReliabilityService) ExecuteWithFallback(
	ctx context.Context,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	startTime := time.Now()

	// Execute with circuit breaker protection
	err := rs.circuitBreaker.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			// Within circuit breaker, execute with retry logic
			return rs.retryExecutor.ExecuteWithStats(ctx, operation)
		},
		fallback,
	)

	duration := time.Since(startTime)

	// Record metrics
	if rs.metrics != nil {
		if err == nil {
			rs.metrics.RecordSuccess(duration)
		} else {
			rs.metrics.RecordFailure(duration, err)
		}
	}

	return err
}

// GetStats returns comprehensive statistics about the reliability service
func (rs *ReliabilityService) GetStats() ReliabilityStats {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	stats := ReliabilityStats{
		Name:                rs.name,
		CircuitBreakerStats: rs.circuitBreaker.Stats(),
		RetryStats:          rs.retryExecutor.GetStats(),
	}

	if rs.metrics != nil {
		stats.Metrics = rs.metrics.GetMetrics()
	}

	return stats
}

// ResetStats resets all statistics
func (rs *ReliabilityService) ResetStats() {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.retryExecutor.ResetStats()
	if rs.metrics != nil {
		rs.metrics.Reset()
	}
}

// IsHealthy returns true if the service is in a healthy state
func (rs *ReliabilityService) IsHealthy() bool {
	state := rs.circuitBreaker.State()
	return state == StateClosed || state == StateHalfOpen
}

// ReliabilityStats contains comprehensive statistics
type ReliabilityStats struct {
	Name                string                 `json:"name"`
	CircuitBreakerStats CircuitBreakerStats    `json:"circuit_breaker"`
	RetryStats          RetryStats             `json:"retry"`
	Metrics             *ReliabilityMetricsData `json:"metrics,omitempty"`
}

// ReliabilityMetrics collects metrics for reliability operations
type ReliabilityMetrics struct {
	serviceName string
	prefix      string
	data        ReliabilityMetricsData
	mutex       sync.RWMutex
}

// ReliabilityMetricsData contains metric data
type ReliabilityMetricsData struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	CircuitOpenRequests int64         `json:"circuit_open_requests"`
	RetryAttempts       int64         `json:"retry_attempts"`
	TotalLatency        time.Duration `json:"total_latency"`
	AverageLatency      time.Duration `json:"average_latency"`
	StateTransitions    map[string]int64 `json:"state_transitions"`
	LastUpdate          time.Time     `json:"last_update"`
}

// NewReliabilityMetrics creates a new metrics collector
func NewReliabilityMetrics(serviceName, prefix string) *ReliabilityMetrics {
	return &ReliabilityMetrics{
		serviceName: serviceName,
		prefix:      prefix,
		data: ReliabilityMetricsData{
			StateTransitions: make(map[string]int64),
		},
	}
}

// RecordSuccess records a successful operation
func (rm *ReliabilityMetrics) RecordSuccess(duration time.Duration) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.data.TotalRequests++
	rm.data.SuccessfulRequests++
	rm.data.TotalLatency += duration
	rm.data.AverageLatency = rm.data.TotalLatency / time.Duration(rm.data.TotalRequests)
	rm.data.LastUpdate = time.Now()
}

// RecordFailure records a failed operation
func (rm *ReliabilityMetrics) RecordFailure(duration time.Duration, err error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.data.TotalRequests++
	rm.data.FailedRequests++
	rm.data.TotalLatency += duration
	rm.data.AverageLatency = rm.data.TotalLatency / time.Duration(rm.data.TotalRequests)
	rm.data.LastUpdate = time.Now()

	// Check if it's a circuit open error
	if IsCircuitOpenError(err) {
		rm.data.CircuitOpenRequests++
	}
}

// RecordRetry records a retry attempt
func (rm *ReliabilityMetrics) RecordRetry(attempt int, delay time.Duration, err error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.data.RetryAttempts++
	rm.data.LastUpdate = time.Now()
}

// RecordStateChange records a circuit breaker state change
func (rm *ReliabilityMetrics) RecordStateChange(from, to CircuitState) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	transitionKey := fmt.Sprintf("%s_to_%s", from.String(), to.String())
	rm.data.StateTransitions[transitionKey]++
	rm.data.LastUpdate = time.Now()
}

// GetMetrics returns the current metrics data
func (rm *ReliabilityMetrics) GetMetrics() *ReliabilityMetricsData {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// Create a copy to avoid race conditions
	data := rm.data
	data.StateTransitions = make(map[string]int64)
	for k, v := range rm.data.StateTransitions {
		data.StateTransitions[k] = v
	}

	return &data
}

// Reset resets all metrics
func (rm *ReliabilityMetrics) Reset() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.data = ReliabilityMetricsData{
		StateTransitions: make(map[string]int64),
		LastUpdate:       time.Now(),
	}
}

// ReliabilityManager manages multiple reliability services
type ReliabilityManager struct {
	services map[string]*ReliabilityService
	mutex    sync.RWMutex
}

// NewReliabilityManager creates a new reliability manager
func NewReliabilityManager() *ReliabilityManager {
	return &ReliabilityManager{
		services: make(map[string]*ReliabilityService),
	}
}

// GetOrCreate gets an existing service or creates a new one
func (rm *ReliabilityManager) GetOrCreate(name string, config ReliabilityConfig) *ReliabilityService {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if service, exists := rm.services[name]; exists {
		return service
	}

	service := NewReliabilityService(name, config)
	rm.services[name] = service
	return service
}

// Get retrieves a service by name
func (rm *ReliabilityManager) Get(name string) (*ReliabilityService, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	service, exists := rm.services[name]
	return service, exists
}

// Remove removes a service from the manager
func (rm *ReliabilityManager) Remove(name string) bool {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.services[name]; exists {
		delete(rm.services, name)
		return true
	}
	return false
}

// GetAllStats returns statistics for all managed services
func (rm *ReliabilityManager) GetAllStats() map[string]ReliabilityStats {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	stats := make(map[string]ReliabilityStats)
	for name, service := range rm.services {
		stats[name] = service.GetStats()
	}
	return stats
}

// GetHealthyServices returns a list of services that are currently healthy
func (rm *ReliabilityManager) GetHealthyServices() []string {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var healthy []string
	for name, service := range rm.services {
		if service.IsHealthy() {
			healthy = append(healthy, name)
		}
	}
	return healthy
}

// GetUnhealthyServices returns a list of services that are currently unhealthy
func (rm *ReliabilityManager) GetUnhealthyServices() []string {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var unhealthy []string
	for name, service := range rm.services {
		if !service.IsHealthy() {
			unhealthy = append(unhealthy, name)
		}
	}
	return unhealthy
}

// Clear removes all services from the manager
func (rm *ReliabilityManager) Clear() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.services = make(map[string]*ReliabilityService)
}

// Global reliability manager for convenience
var globalReliabilityManager = NewReliabilityManager()

// GetReliabilityService gets or creates a reliability service with default configuration
func GetReliabilityService(name string) *ReliabilityService {
	return globalReliabilityManager.GetOrCreate(name, DefaultReliabilityConfig())
}

// GetReliabilityServiceWithConfig gets or creates a reliability service with custom configuration
func GetReliabilityServiceWithConfig(name string, config ReliabilityConfig) *ReliabilityService {
	return globalReliabilityManager.GetOrCreate(name, config)
}

// ExecuteReliably executes an operation with default reliability configuration
func ExecuteReliably(ctx context.Context, serviceName string, operation func(context.Context) error) error {
	service := GetReliabilityService(serviceName)
	return service.Execute(ctx, operation)
}

// ExecuteReliablyWithConfig executes an operation with custom reliability configuration
func ExecuteReliablyWithConfig(
	ctx context.Context,
	serviceName string,
	config ReliabilityConfig,
	operation func(context.Context) error,
) error {
	service := GetReliabilityServiceWithConfig(serviceName, config)
	return service.Execute(ctx, operation)
}