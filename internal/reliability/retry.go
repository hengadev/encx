package reliability

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy defines the interface for retry policies
type RetryPolicy interface {
	// NextDelay returns the delay before the next attempt, given the attempt number (0-indexed)
	NextDelay(attempt int) time.Duration
	// ShouldRetry determines if a retry should be attempted based on the error and attempt number
	ShouldRetry(err error, attempt int) bool
	// MaxAttempts returns the maximum number of attempts (including the initial attempt)
	MaxAttempts() int
}

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including initial attempt)
	MaxAttempts int
	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// Multiplier for exponential backoff
	Multiplier float64
	// Jitter adds randomness to delay calculations
	Jitter float64
	// RetryableErrors is a list of error types that should trigger retries
	RetryableErrors []error
	// ShouldRetry is a custom function to determine if an error should trigger a retry
	ShouldRetry func(error, int) bool
	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, delay time.Duration, err error)
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond * 100,
		MaxDelay:     time.Second * 30,
		Multiplier:   2.0,
		Jitter:       0.1,
		ShouldRetry: func(err error, attempt int) bool {
			return err != nil
		},
		OnRetry: func(attempt int, delay time.Duration, err error) {
			// Default no-op
		},
	}
}

// ExponentialBackoffPolicy implements exponential backoff with jitter
type ExponentialBackoffPolicy struct {
	maxAttempts  int
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitter       float64
	shouldRetry  func(error, int) bool
}

// NewExponentialBackoffPolicy creates a new exponential backoff policy
func NewExponentialBackoffPolicy(config RetryConfig) *ExponentialBackoffPolicy {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = DefaultRetryConfig().MaxAttempts
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = DefaultRetryConfig().InitialDelay
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = DefaultRetryConfig().MaxDelay
	}
	if config.Multiplier <= 0 {
		config.Multiplier = DefaultRetryConfig().Multiplier
	}
	if config.Jitter < 0 || config.Jitter > 1 {
		config.Jitter = DefaultRetryConfig().Jitter
	}
	if config.ShouldRetry == nil {
		config.ShouldRetry = DefaultRetryConfig().ShouldRetry
	}

	return &ExponentialBackoffPolicy{
		maxAttempts:  config.MaxAttempts,
		initialDelay: config.InitialDelay,
		maxDelay:     config.MaxDelay,
		multiplier:   config.Multiplier,
		jitter:       config.Jitter,
		shouldRetry:  config.ShouldRetry,
	}
}

// NextDelay calculates the delay for the next retry attempt
func (p *ExponentialBackoffPolicy) NextDelay(attempt int) time.Duration {
	if attempt < 0 {
		return 0
	}

	// Calculate exponential backoff
	delay := float64(p.initialDelay) * math.Pow(p.multiplier, float64(attempt))

	// Apply maximum delay limit
	if delay > float64(p.maxDelay) {
		delay = float64(p.maxDelay)
	}

	// Add jitter to prevent thundering herd
	if p.jitter > 0 {
		jitterRange := delay * p.jitter
		jitterOffset := (rand.Float64() - 0.5) * 2 * jitterRange
		delay += jitterOffset
	}

	// Ensure delay is non-negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// ShouldRetry determines if a retry should be attempted
func (p *ExponentialBackoffPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= p.maxAttempts-1 { // -1 because attempt is 0-indexed
		return false
	}
	return p.shouldRetry(err, attempt)
}

// MaxAttempts returns the maximum number of attempts
func (p *ExponentialBackoffPolicy) MaxAttempts() int {
	return p.maxAttempts
}

// FixedDelayPolicy implements a fixed delay between retries
type FixedDelayPolicy struct {
	maxAttempts int
	delay       time.Duration
	shouldRetry func(error, int) bool
}

// NewFixedDelayPolicy creates a new fixed delay policy
func NewFixedDelayPolicy(maxAttempts int, delay time.Duration, shouldRetry func(error, int) bool) *FixedDelayPolicy {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if delay <= 0 {
		delay = time.Second
	}
	if shouldRetry == nil {
		shouldRetry = func(err error, attempt int) bool {
			return err != nil
		}
	}

	return &FixedDelayPolicy{
		maxAttempts: maxAttempts,
		delay:       delay,
		shouldRetry: shouldRetry,
	}
}

// NextDelay returns the fixed delay
func (p *FixedDelayPolicy) NextDelay(attempt int) time.Duration {
	return p.delay
}

// ShouldRetry determines if a retry should be attempted
func (p *FixedDelayPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= p.maxAttempts-1 {
		return false
	}
	return p.shouldRetry(err, attempt)
}

// MaxAttempts returns the maximum number of attempts
func (p *FixedDelayPolicy) MaxAttempts() int {
	return p.maxAttempts
}

// LinearBackoffPolicy implements linear backoff
type LinearBackoffPolicy struct {
	maxAttempts  int
	initialDelay time.Duration
	increment    time.Duration
	maxDelay     time.Duration
	shouldRetry  func(error, int) bool
}

// NewLinearBackoffPolicy creates a new linear backoff policy
func NewLinearBackoffPolicy(maxAttempts int, initialDelay, increment, maxDelay time.Duration, shouldRetry func(error, int) bool) *LinearBackoffPolicy {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if initialDelay <= 0 {
		initialDelay = time.Millisecond * 100
	}
	if increment <= 0 {
		increment = time.Millisecond * 100
	}
	if maxDelay <= 0 {
		maxDelay = time.Second * 30
	}
	if shouldRetry == nil {
		shouldRetry = func(err error, attempt int) bool {
			return err != nil
		}
	}

	return &LinearBackoffPolicy{
		maxAttempts:  maxAttempts,
		initialDelay: initialDelay,
		increment:    increment,
		maxDelay:     maxDelay,
		shouldRetry:  shouldRetry,
	}
}

// NextDelay calculates the delay for linear backoff
func (p *LinearBackoffPolicy) NextDelay(attempt int) time.Duration {
	delay := p.initialDelay + time.Duration(attempt)*p.increment
	if delay > p.maxDelay {
		delay = p.maxDelay
	}
	return delay
}

// ShouldRetry determines if a retry should be attempted
func (p *LinearBackoffPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= p.maxAttempts-1 {
		return false
	}
	return p.shouldRetry(err, attempt)
}

// MaxAttempts returns the maximum number of attempts
func (p *LinearBackoffPolicy) MaxAttempts() int {
	return p.maxAttempts
}

// RetryExecutor handles retry logic for operations
type RetryExecutor struct {
	policy  RetryPolicy
	onRetry func(attempt int, delay time.Duration, err error)
}

// NewRetryExecutor creates a new retry executor with the given policy
func NewRetryExecutor(policy RetryPolicy) *RetryExecutor {
	return &RetryExecutor{
		policy: policy,
		onRetry: func(attempt int, delay time.Duration, err error) {
			// Default no-op
		},
	}
}

// SetOnRetryCallback sets a callback function to be called before each retry
func (r *RetryExecutor) SetOnRetryCallback(callback func(attempt int, delay time.Duration, err error)) {
	r.onRetry = callback
}

// Execute executes the given operation with retry logic
func (r *RetryExecutor) Execute(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < r.policy.MaxAttempts(); attempt++ {
		// Check context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the operation
		err := operation(ctx)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !r.policy.ShouldRetry(err, attempt) {
			break
		}

		// If this is the last attempt, don't delay
		if attempt >= r.policy.MaxAttempts()-1 {
			break
		}

		// Calculate delay and wait
		delay := r.policy.NextDelay(attempt)
		r.onRetry(attempt+1, delay, err)

		// Wait for the calculated delay or until context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// ExecuteWithFallback executes the operation with retry logic and a fallback function
func (r *RetryExecutor) ExecuteWithFallback(
	ctx context.Context,
	operation func(context.Context) error,
	fallback func(context.Context, error) error,
) error {
	err := r.Execute(ctx, operation)
	if err != nil && fallback != nil {
		return fallback(ctx, err)
	}
	return err
}

// RetryStats contains statistics about retry operations
type RetryStats struct {
	TotalAttempts    int           `json:"total_attempts"`
	SuccessfulRetries int          `json:"successful_retries"`
	FailedRetries    int           `json:"failed_retries"`
	TotalDelay       time.Duration `json:"total_delay"`
	LastError        string        `json:"last_error,omitempty"`
}

// RetryExecutorWithStats extends RetryExecutor with statistics collection
type RetryExecutorWithStats struct {
	*RetryExecutor
	stats RetryStats
}

// NewRetryExecutorWithStats creates a new retry executor with statistics
func NewRetryExecutorWithStats(policy RetryPolicy) *RetryExecutorWithStats {
	executor := NewRetryExecutor(policy)
	statsExecutor := &RetryExecutorWithStats{
		RetryExecutor: executor,
	}

	// Set callback to collect statistics
	executor.SetOnRetryCallback(func(attempt int, delay time.Duration, err error) {
		statsExecutor.stats.TotalAttempts = attempt
		statsExecutor.stats.TotalDelay += delay
		if err != nil {
			statsExecutor.stats.LastError = err.Error()
		}
	})

	return statsExecutor
}

// ExecuteWithStats executes the operation and updates statistics
func (r *RetryExecutorWithStats) ExecuteWithStats(ctx context.Context, operation func(context.Context) error) error {
	r.stats = RetryStats{} // Reset stats

	err := r.Execute(ctx, operation)
	if err == nil && r.stats.TotalAttempts > 0 {
		r.stats.SuccessfulRetries++
	} else if err != nil {
		r.stats.FailedRetries++
		r.stats.LastError = err.Error()
	}

	return err
}

// GetStats returns the current retry statistics
func (r *RetryExecutorWithStats) GetStats() RetryStats {
	return r.stats
}

// ResetStats resets the statistics
func (r *RetryExecutorWithStats) ResetStats() {
	r.stats = RetryStats{}
}

// RetryableOperation is a convenience function for executing operations with retry
func RetryableOperation(
	ctx context.Context,
	operation func(context.Context) error,
	config RetryConfig,
) error {
	policy := NewExponentialBackoffPolicy(config)
	executor := NewRetryExecutor(policy)

	if config.OnRetry != nil {
		executor.SetOnRetryCallback(config.OnRetry)
	}

	return executor.Execute(ctx, operation)
}

// RetryableOperationWithFallback executes an operation with retry and fallback
func RetryableOperationWithFallback(
	ctx context.Context,
	operation func(context.Context) error,
	fallback func(context.Context, error) error,
	config RetryConfig,
) error {
	policy := NewExponentialBackoffPolicy(config)
	executor := NewRetryExecutor(policy)

	if config.OnRetry != nil {
		executor.SetOnRetryCallback(config.OnRetry)
	}

	return executor.ExecuteWithFallback(ctx, operation, fallback)
}

// Common retry error predicates

// IsTemporaryError checks if an error is temporary and should be retried
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for temporary interface
	if temp, ok := err.(interface{ Temporary() bool }); ok {
		return temp.Temporary()
	}

	// Check for timeout errors
	if timeout, ok := err.(interface{ Timeout() bool }); ok {
		return timeout.Timeout()
	}

	return false
}

// IsNetworkError checks if an error is network-related
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"no route to host",
		"network is unreachable",
		"timeout",
		"temporary failure",
	}

	for _, netErr := range networkErrors {
		if contains(errStr, netErr) {
			return true
		}
	}

	return false
}

// IsRetryableStatusCode checks if an HTTP status code is retryable
func IsRetryableStatusCode(statusCode int) bool {
	retryableCodes := []int{
		408, // Request Timeout
		429, // Too Many Requests
		500, // Internal Server Error
		502, // Bad Gateway
		503, // Service Unavailable
		504, // Gateway Timeout
	}

	for _, code := range retryableCodes {
		if statusCode == code {
			return true
		}
	}

	return false
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(s) > len(substr) &&
		 (s[:len(substr)] == substr ||
		  s[len(s)-len(substr):] == substr ||
		  indexSubstring(s, substr) >= 0))
}

// Simple substring search helper
func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Global retry executor instances for convenience
var (
	defaultRetryExecutor = NewRetryExecutor(NewExponentialBackoffPolicy(DefaultRetryConfig()))
)

// Retry executes an operation with default retry policy
func Retry(ctx context.Context, operation func(context.Context) error) error {
	return defaultRetryExecutor.Execute(ctx, operation)
}

// RetryWithConfig executes an operation with custom retry configuration
func RetryWithConfig(ctx context.Context, operation func(context.Context) error, config RetryConfig) error {
	return RetryableOperation(ctx, operation, config)
}