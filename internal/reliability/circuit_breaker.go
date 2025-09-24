package reliability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the current state of the circuit breaker
type CircuitState int

const (
	// StateClosed - Normal operation, requests pass through
	StateClosed CircuitState = iota
	// StateOpen - Circuit is open, requests fail fast
	StateOpen
	// StateHalfOpen - Testing state, limited requests allowed
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	// SuccessThreshold is the number of successes needed to close the circuit in half-open state
	SuccessThreshold int
	// Timeout is how long the circuit stays open before transitioning to half-open
	Timeout time.Duration
	// MaxConcurrentRequests is the maximum number of requests allowed in half-open state
	MaxConcurrentRequests int
	// ShouldTrip is a custom function to determine if an error should count as a failure
	ShouldTrip func(error) bool
	// OnStateChange is called when the circuit state changes
	OnStateChange func(name string, from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns a default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               time.Second * 60,
		MaxConcurrentRequests: 1,
		ShouldTrip: func(err error) bool {
			return err != nil
		},
		OnStateChange: func(name string, from, to CircuitState) {
			// Default no-op
		},
	}
}

// CircuitBreaker implements the circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig

	mutex             sync.RWMutex
	state             CircuitState
	generation        int64
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	nextAttemptTime   time.Time
	concurrentRequests int32
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultCircuitBreakerConfig().FailureThreshold
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = DefaultCircuitBreakerConfig().SuccessThreshold
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultCircuitBreakerConfig().Timeout
	}
	if config.MaxConcurrentRequests <= 0 {
		config.MaxConcurrentRequests = DefaultCircuitBreakerConfig().MaxConcurrentRequests
	}
	if config.ShouldTrip == nil {
		config.ShouldTrip = DefaultCircuitBreakerConfig().ShouldTrip
	}
	if config.OnStateChange == nil {
		config.OnStateChange = DefaultCircuitBreakerConfig().OnStateChange
	}

	return &CircuitBreaker{
		name:   name,
		config: config,
		state:  StateClosed,
	}
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.beforeRequest(ctx); err != nil {
		return err
	}

	defer cb.afterRequest()

	err := fn(ctx)
	cb.recordResult(err)

	return err
}

// ExecuteWithFallback executes the function with circuit breaker protection and fallback
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error, fallback func(context.Context) error) error {
	err := cb.Execute(ctx, fn)

	// If circuit is open or request failed, try fallback
	if IsCircuitOpenError(err) || (err != nil && cb.config.ShouldTrip(err)) {
		if fallback != nil {
			return fallback(ctx)
		}
	}

	return err
}

// beforeRequest checks if the request should be allowed
func (cb *CircuitBreaker) beforeRequest(ctx context.Context) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return NewCircuitOpenError(cb.name, cb.nextAttemptTime)
	}

	if state == StateHalfOpen {
		if cb.concurrentRequests >= int32(cb.config.MaxConcurrentRequests) {
			return NewCircuitOpenError(cb.name, cb.nextAttemptTime)
		}
		cb.concurrentRequests++
	}

	cb.generation = generation
	return nil
}

// afterRequest is called after a request completes
func (cb *CircuitBreaker) afterRequest() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == StateHalfOpen {
		cb.concurrentRequests--
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	if cb.config.ShouldTrip(err) {
		cb.onFailure(now)
	} else {
		cb.onSuccess(now)
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure(now time.Time) {
	cb.failureCount++
	cb.lastFailureTime = now

	if cb.state == StateClosed {
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.setState(StateOpen, now)
		}
	} else if cb.state == StateHalfOpen {
		cb.setState(StateOpen, now)
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess(now time.Time) {
	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.setState(StateClosed, now)
		}
	} else if cb.state == StateClosed {
		// Reset failure count on success in closed state
		cb.failureCount = 0
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	prevState := cb.state
	cb.state = state

	switch state {
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
		cb.nextAttemptTime = time.Time{}
	case StateOpen:
		cb.nextAttemptTime = now.Add(cb.config.Timeout)
		cb.successCount = 0
	case StateHalfOpen:
		cb.successCount = 0
		cb.concurrentRequests = 0
	}

	cb.config.OnStateChange(cb.name, prevState, state)
}

// currentState returns the current state, potentially transitioning from open to half-open
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, int64) {
	if cb.state == StateOpen && now.After(cb.nextAttemptTime) {
		cb.setState(StateHalfOpen, now)
	}

	return cb.state, cb.generation
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	state, _ := cb.currentState(time.Now())
	return state
}

// Stats returns statistics about the circuit breaker
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return CircuitBreakerStats{
		Name:               cb.name,
		State:              cb.state,
		FailureCount:       cb.failureCount,
		SuccessCount:       cb.successCount,
		LastFailureTime:    cb.lastFailureTime,
		NextAttemptTime:    cb.nextAttemptTime,
		ConcurrentRequests: int(cb.concurrentRequests),
	}
}

// CircuitBreakerStats contains statistics about a circuit breaker
type CircuitBreakerStats struct {
	Name               string        `json:"name"`
	State              CircuitState  `json:"state"`
	FailureCount       int           `json:"failure_count"`
	SuccessCount       int           `json:"success_count"`
	LastFailureTime    time.Time     `json:"last_failure_time,omitempty"`
	NextAttemptTime    time.Time     `json:"next_attempt_time,omitempty"`
	ConcurrentRequests int           `json:"concurrent_requests"`
}

// CircuitOpenError is returned when the circuit breaker is open
type CircuitOpenError struct {
	CircuitName     string    `json:"circuit_name"`
	NextAttemptTime time.Time `json:"next_attempt_time"`
}

// NewCircuitOpenError creates a new circuit open error
func NewCircuitOpenError(circuitName string, nextAttemptTime time.Time) *CircuitOpenError {
	return &CircuitOpenError{
		CircuitName:     circuitName,
		NextAttemptTime: nextAttemptTime,
	}
}

// Error implements the error interface
func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("circuit breaker '%s' is open, next attempt allowed at %s",
		e.CircuitName, e.NextAttemptTime.Format(time.RFC3339))
}

// IsCircuitOpenError checks if an error is a circuit open error
func IsCircuitOpenError(err error) bool {
	var circuitErr *CircuitOpenError
	return errors.As(err, &circuitErr)
}

// CircuitBreakerRegistry manages multiple circuit breakers
type CircuitBreakerRegistry struct {
	mutex    sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerRegistry creates a new circuit breaker registry
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (r *CircuitBreakerRegistry) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if breaker, exists := r.breakers[name]; exists {
		return breaker
	}

	breaker := NewCircuitBreaker(name, config)
	r.breakers[name] = breaker
	return breaker
}

// Get retrieves a circuit breaker by name
func (r *CircuitBreakerRegistry) Get(name string) (*CircuitBreaker, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	breaker, exists := r.breakers[name]
	return breaker, exists
}

// Remove removes a circuit breaker from the registry
func (r *CircuitBreakerRegistry) Remove(name string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.breakers[name]; exists {
		delete(r.breakers, name)
		return true
	}
	return false
}

// AllStats returns statistics for all registered circuit breakers
func (r *CircuitBreakerRegistry) AllStats() map[string]CircuitBreakerStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for name, breaker := range r.breakers {
		stats[name] = breaker.Stats()
	}
	return stats
}

// Clear removes all circuit breakers from the registry
func (r *CircuitBreakerRegistry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.breakers = make(map[string]*CircuitBreaker)
}

// Global circuit breaker registry for convenience
var globalRegistry = NewCircuitBreakerRegistry()

// GetCircuitBreaker gets or creates a circuit breaker with default configuration
func GetCircuitBreaker(name string) *CircuitBreaker {
	return globalRegistry.GetOrCreate(name, DefaultCircuitBreakerConfig())
}

// GetCircuitBreakerWithConfig gets or creates a circuit breaker with custom configuration
func GetCircuitBreakerWithConfig(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return globalRegistry.GetOrCreate(name, config)
}