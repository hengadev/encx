package monitoring

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ObservabilityHook defines hooks for monitoring encryption operations
type ObservabilityHook interface {
	// Called before processing starts
	OnProcessStart(ctx context.Context, operation string, metadata map[string]any)

	// Called after processing completes (success or failure)
	OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any)

	// Called when errors occur
	OnError(ctx context.Context, operation string, err error, metadata map[string]any)

	// Called for key operations
	OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any)
}

// NoOpObservabilityHook is a no-op implementation of ObservabilityHook
type NoOpObservabilityHook struct{}

func (n *NoOpObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
}
func (n *NoOpObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
}
func (n *NoOpObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
}
func (n *NoOpObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
}

// LoggingObservabilityHook logs all operations
type LoggingObservabilityHook struct {
	logger Logger
}

// Logger defines the interface for logging
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// StandardLogger wraps the standard log package
type StandardLogger struct{}

func (s *StandardLogger) Info(msg string, args ...any) {
	log.Printf("[INFO] "+msg, args...)
}

func (s *StandardLogger) Error(msg string, args ...any) {
	log.Printf("[ERROR] "+msg, args...)
}

func (s *StandardLogger) Debug(msg string, args ...any) {
	log.Printf("[DEBUG] "+msg, args...)
}

// NewLoggingObservabilityHook creates a new logging observability hook
func NewLoggingObservabilityHook(logger Logger) *LoggingObservabilityHook {
	if logger == nil {
		logger = &StandardLogger{}
	}
	return &LoggingObservabilityHook{
		logger: logger,
	}
}

func (l *LoggingObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
	l.logger.Info("Operation started: %s, metadata: %v", operation, metadata)
}

func (l *LoggingObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	if err != nil {
		l.logger.Error("Operation failed: %s, duration: %v, error: %v, metadata: %v", operation, duration, err, metadata)
	} else {
		l.logger.Info("Operation completed: %s, duration: %v, metadata: %v", operation, duration, metadata)
	}
}

func (l *LoggingObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
	l.logger.Error("Operation error: %s, error: %v, metadata: %v", operation, err, metadata)
}

func (l *LoggingObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
	l.logger.Info("Key operation: %s, alias: %s, version: %d, metadata: %v", operation, keyAlias, keyVersion, metadata)
}

// MetricsObservabilityHook collects metrics for operations
type MetricsObservabilityHook struct {
	collector MetricsCollector
}

// NewMetricsObservabilityHook creates a new metrics observability hook
func NewMetricsObservabilityHook(collector MetricsCollector) *MetricsObservabilityHook {
	if collector == nil {
		collector = &NoOpMetricsCollector{}
	}
	return &MetricsObservabilityHook{
		collector: collector,
	}
}

func (m *MetricsObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
	tags := map[string]string{"operation": operation}
	if opType, ok := metadata["operation_type"].(string); ok {
		tags["operation_type"] = opType
	}
	m.collector.IncrementCounter("encx.process.started", tags)
}

func (m *MetricsObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	tags := map[string]string{"operation": operation}
	if opType, ok := metadata["operation_type"].(string); ok {
		tags["operation_type"] = opType
	}

	if err != nil {
		tags["status"] = "error"
		m.collector.IncrementCounter("encx.process.failed", tags)
	} else {
		tags["status"] = "success"
		m.collector.IncrementCounter("encx.process.succeeded", tags)
	}

	m.collector.RecordTiming("encx.process.duration", duration, tags)
}

func (m *MetricsObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
	tags := map[string]string{
		"operation": operation,
		"error":     fmt.Sprintf("%T", err),
	}
	m.collector.IncrementCounter("encx.errors", tags)
}

func (m *MetricsObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
	tags := map[string]string{
		"operation":   operation,
		"key_alias":   keyAlias,
		"key_version": fmt.Sprintf("%d", keyVersion),
	}
	m.collector.IncrementCounter("encx.key_operations", tags)
}

// CompositeObservabilityHook combines multiple hooks
type CompositeObservabilityHook struct {
	hooks []ObservabilityHook
}

// NewCompositeObservabilityHook creates a new composite hook
func NewCompositeObservabilityHook(hooks ...ObservabilityHook) *CompositeObservabilityHook {
	return &CompositeObservabilityHook{
		hooks: hooks,
	}
}

func (c *CompositeObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]any) {
	for _, hook := range c.hooks {
		hook.OnProcessStart(ctx, operation, metadata)
	}
}

func (c *CompositeObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	for _, hook := range c.hooks {
		hook.OnProcessComplete(ctx, operation, duration, err, metadata)
	}
}

func (c *CompositeObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]any) {
	for _, hook := range c.hooks {
		hook.OnError(ctx, operation, err, metadata)
	}
}

func (c *CompositeObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
	for _, hook := range c.hooks {
		hook.OnKeyOperation(ctx, operation, keyAlias, keyVersion, metadata)
	}
}

