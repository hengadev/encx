package monitoring

import (
	"context"
	"fmt"
	"time"
)

// EnhancedObservabilityHook combines structured logging and enhanced metrics
type EnhancedObservabilityHook struct {
	logger            *StructuredLogger
	metricsCollector  *EnhancedMetricsCollector
	enablePerfMetrics bool
	enableSecMetrics  bool
}

// EnhancedObservabilityConfig configures the enhanced observability hook
type EnhancedObservabilityConfig struct {
	Logger                *StructuredLogger
	MetricsCollector      *EnhancedMetricsCollector
	EnablePerfMetrics     bool
	EnableSecurityMetrics bool
}

// NewEnhancedObservabilityHook creates a new enhanced observability hook
func NewEnhancedObservabilityHook(config EnhancedObservabilityConfig) *EnhancedObservabilityHook {
	if config.Logger == nil {
		config.Logger = NewProductionLogger("encx.observability")
	}
	if config.MetricsCollector == nil {
		config.MetricsCollector = NewEnhancedMetricsCollector(EnhancedMetricsConfig{
			Logger: config.Logger,
		})
	}

	return &EnhancedObservabilityHook{
		logger:            config.Logger,
		metricsCollector:  config.MetricsCollector,
		enablePerfMetrics: config.EnablePerfMetrics,
		enableSecMetrics:  config.EnableSecurityMetrics,
	}
}

// OnProcessStart logs and records metrics when processing starts
func (e *EnhancedObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
	// Enhanced structured logging
	fields := map[string]any{
		"operation": operation,
		"phase":     "start",
	}

	// Add metadata to fields
	for k, v := range metadata {
		fields[k] = v
	}

	logger := e.logger.WithContext(ctx).WithFields(fields)
	logger.Info("Crypto operation started")

	// Enhanced metrics
	tags := map[string]string{
		"operation": operation,
	}

	// Extract relevant metadata for tags
	if opType, ok := metadata["operation_type"].(string); ok {
		tags["operation_type"] = opType
	}
	if component, ok := metadata["component"].(string); ok {
		tags["component"] = component
	}

	e.metricsCollector.IncrementCounter("encx.operations.started", tags)
}

// OnProcessComplete logs and records metrics when processing completes
func (e *EnhancedObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	// Enhanced structured logging
	success := err == nil

	fields := map[string]any{
		"operation":   operation,
		"phase":       "complete",
		"duration":    duration.String(),
		"duration_ms": duration.Nanoseconds() / 1000000,
		"success":     success,
	}

	// Add metadata to fields
	for k, v := range metadata {
		fields[k] = v
	}

	if err != nil {
		fields["error"] = err.Error()
		fields["error_type"] = fmt.Sprintf("%T", err)
	}

	logger := e.logger.WithContext(ctx).WithFields(fields)

	if err != nil {
		logger.Error("Crypto operation failed")
	} else {
		logger.Info("Crypto operation completed successfully")
	}

	// Enhanced metrics
	tags := map[string]string{
		"operation": operation,
	}

	// Extract relevant metadata for tags
	if opType, ok := metadata["operation_type"].(string); ok {
		tags["operation_type"] = opType
	}
	if component, ok := metadata["component"].(string); ok {
		tags["component"] = component
	}

	// Record performance metrics
	if e.enablePerfMetrics {
		e.metricsCollector.RecordPerformanceMetric(operation, duration, success, tags)
	}

	// Record completion metrics
	if success {
		e.metricsCollector.IncrementCounter("encx.operations.succeeded", tags)
	} else {
		e.metricsCollector.IncrementCounter("encx.operations.failed", tags)
	}

	// Record duration histogram
	e.metricsCollector.RecordTiming("encx.operations.duration", duration, tags)

	// Record data size if available
	if dataSize, ok := metadata["data_size"].(int64); ok {
		e.metricsCollector.RecordValue("encx.operations.data_size", float64(dataSize), tags)

		// Calculate throughput if successful
		if success && dataSize > 0 {
			throughputMBps := float64(dataSize) / (1024 * 1024) / duration.Seconds()
			e.metricsCollector.RecordValue("encx.operations.throughput_mbps", throughputMBps, tags)
		}
	}
}

// OnError logs and records metrics when errors occur
func (e *EnhancedObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
	// Enhanced structured logging with security focus
	fields := map[string]any{
		"operation":  operation,
		"phase":      "error",
		"error":      err.Error(),
		"error_type": fmt.Sprintf("%T", err),
	}

	// Add metadata to fields
	for k, v := range metadata {
		fields[k] = v
	}

	// Check if this is a security-relevant error
	isSecurityError := e.isSecurityError(err)
	if isSecurityError {
		fields["security_event"] = true
		fields["severity"] = "high"
	}

	logger := e.logger.WithContext(ctx).WithFields(fields)

	if isSecurityError {
		logger.LogSecurityEvent(ctx, "crypto_operation_error", "high", fields)
	} else {
		logger.Error("Crypto operation error occurred")
	}

	// Enhanced metrics
	tags := map[string]string{
		"operation":  operation,
		"error_type": fmt.Sprintf("%T", err),
	}

	// Extract relevant metadata for tags
	if opType, ok := metadata["operation_type"].(string); ok {
		tags["operation_type"] = opType
	}

	e.metricsCollector.IncrementCounter("encx.errors.total", tags)

	// Security metrics
	if e.enableSecMetrics && isSecurityError {
		e.metricsCollector.RecordSecurityMetric("crypto_error", "high", tags)
	}
}

// OnKeyOperation logs and records metrics for key operations
func (e *EnhancedObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
	// Enhanced structured logging for key operations
	fields := map[string]any{
		"operation":   operation,
		"key_alias":   keyAlias,
		"key_version": keyVersion,
		"phase":       "key_operation",
	}

	// Add metadata to fields
	for k, v := range metadata {
		fields[k] = v
	}

	logger := e.logger.WithContext(ctx).WithFields(fields)
	logger.LogKeyOperation(ctx, operation, keyAlias, keyVersion, metadata)

	// Enhanced metrics
	tags := map[string]string{
		"operation":   operation,
		"key_alias":   keyAlias,
		"key_version": fmt.Sprintf("%d", keyVersion),
	}

	e.metricsCollector.IncrementCounter("encx.key_operations.total", tags)

	// Security metrics for key operations
	if e.enableSecMetrics {
		e.metricsCollector.RecordSecurityMetric("key_operation", "medium", tags)
	}
}

// OnCacheOperation logs cache-related operations
func (e *EnhancedObservabilityHook) OnCacheOperation(ctx context.Context, operation string, hit bool, duration time.Duration, metadata map[string]any) {
	fields := map[string]any{
		"operation": operation,
		"cache_hit": hit,
		"duration":  duration.String(),
		"phase":     "cache_operation",
	}

	// Add metadata to fields
	for k, v := range metadata {
		fields[k] = v
	}

	logger := e.logger.WithContext(ctx).WithFields(fields)
	logger.Debug("Cache operation performed")

	// Cache metrics
	tags := map[string]string{
		"operation": operation,
		"hit":       fmt.Sprintf("%t", hit),
	}

	e.metricsCollector.IncrementCounter("encx.cache.operations", tags)
	e.metricsCollector.RecordTiming("encx.cache.duration", duration, tags)
}

// OnDatabaseOperation logs database-related operations
func (e *EnhancedObservabilityHook) OnDatabaseOperation(ctx context.Context, operation string, table string, duration time.Duration, err error, metadata map[string]any) {
	success := err == nil

	fields := map[string]any{
		"operation": operation,
		"table":     table,
		"duration":  duration.String(),
		"success":   success,
		"phase":     "database_operation",
	}

	if err != nil {
		fields["error"] = err.Error()
		fields["error_type"] = fmt.Sprintf("%T", err)
	}

	// Add metadata to fields
	for k, v := range metadata {
		fields[k] = v
	}

	logger := e.logger.WithContext(ctx).WithFields(fields)

	if err != nil {
		logger.Error("Database operation failed")
	} else {
		logger.Debug("Database operation completed")
	}

	// Database metrics
	tags := map[string]string{
		"operation": operation,
		"table":     table,
		"success":   fmt.Sprintf("%t", success),
	}

	e.metricsCollector.IncrementCounter("encx.database.operations", tags)
	e.metricsCollector.RecordTiming("encx.database.duration", duration, tags)
}

// isSecurityError determines if an error is security-relevant
func (e *EnhancedObservabilityHook) isSecurityError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()

	// Check for common security error patterns
	securityPatterns := []string{
		"authentication",
		"authorization",
		"permission",
		"access denied",
		"unauthorized",
		"invalid key",
		"decryption failed",
		"signature verification",
		"certificate",
		"timeout",
		"rate limit",
	}

	for _, pattern := range securityPatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	s = fmt.Sprintf("%s", s) // Convert to string and lowercase
	substr = fmt.Sprintf("%s", substr)

	// Simple case-insensitive contains check
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetMetricsCollector returns the metrics collector for external use
func (e *EnhancedObservabilityHook) GetMetricsCollector() *EnhancedMetricsCollector {
	return e.metricsCollector
}

// GetLogger returns the structured logger for external use
func (e *EnhancedObservabilityHook) GetLogger() *StructuredLogger {
	return e.logger
}

// Stop gracefully stops the observability hook
func (e *EnhancedObservabilityHook) Stop() error {
	e.logger.Info("Stopping enhanced observability hook")

	if e.metricsCollector != nil {
		return e.metricsCollector.Stop()
	}

	return nil
}

