package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedObservabilityHook(t *testing.T) {
	// Create test logger and metrics collector
	logger := NewDevelopmentLogger("test")
	metricsCollector := NewEnhancedMetricsCollector(EnhancedMetricsConfig{
		ExportPeriod: time.Hour, // Long period to avoid automatic exports during tests
		Logger:       logger,
	})

	// Create enhanced observability hook
	hook := NewEnhancedObservabilityHook(EnhancedObservabilityConfig{
		Logger:                logger,
		MetricsCollector:      metricsCollector,
		EnablePerfMetrics:     true,
		EnableSecurityMetrics: true,
	})

	ctx := context.Background()
	operation := "test_encrypt"
	metadata := map[string]any{
		"operation_type": "encryption",
		"component":      "crypto",
		"data_size":      int64(1024),
	}

	t.Run("OnProcessStart", func(t *testing.T) {
		hook.OnProcessStart(ctx, operation, metadata)

		// Check that started counter was incremented
		aggregated := metricsCollector.GetAggregatedMetrics()
		found := false
		for _, metric := range aggregated {
			if metric.Name == "encx.operations.started" {
				found = true
				assert.Equal(t, int64(1), metric.Count)
			}
		}
		assert.True(t, found, "Started counter should be incremented")
	})

	t.Run("OnProcessComplete_Success", func(t *testing.T) {
		duration := 100 * time.Millisecond
		hook.OnProcessComplete(ctx, operation, duration, nil, metadata)

		// Check metrics
		aggregated := metricsCollector.GetAggregatedMetrics()

		// Should have success counter
		foundSuccess := false
		foundDuration := false
		foundThroughput := false

		for _, metric := range aggregated {
			switch metric.Name {
			case "encx.operations.succeeded":
				foundSuccess = true
				assert.Equal(t, int64(1), metric.Count)
			case "encx.operations.duration":
				foundDuration = true
				assert.Equal(t, int64(1), metric.Count)
			case "encx.operations.throughput_mbps":
				foundThroughput = true
				assert.True(t, metric.Sum > 0)
			}
		}

		assert.True(t, foundSuccess, "Success counter should be incremented")
		assert.True(t, foundDuration, "Duration should be recorded")
		assert.True(t, foundThroughput, "Throughput should be recorded")
	})

	t.Run("OnProcessComplete_Failure", func(t *testing.T) {
		duration := 50 * time.Millisecond
		testErr := errors.New("test error")
		hook.OnProcessComplete(ctx, operation, duration, testErr, metadata)

		// Check that failure counter was incremented
		aggregated := metricsCollector.GetAggregatedMetrics()
		found := false
		for _, metric := range aggregated {
			if metric.Name == "encx.operations.failed" {
				found = true
				assert.True(t, metric.Count >= 1)
			}
		}
		assert.True(t, found, "Failed counter should be incremented")
	})

	t.Run("OnError", func(t *testing.T) {
		testErr := errors.New("authentication failed")
		hook.OnError(ctx, operation, testErr, metadata)

		// Check that error counter was incremented
		aggregated := metricsCollector.GetAggregatedMetrics()
		foundError := false
		foundSecurity := false

		for _, metric := range aggregated {
			switch metric.Name {
			case "encx.errors.total":
				foundError = true
				assert.True(t, metric.Count >= 1)
			case "encx.security.events":
				foundSecurity = true
				assert.True(t, metric.Count >= 1)
			}
		}

		assert.True(t, foundError, "Error counter should be incremented")
		assert.True(t, foundSecurity, "Security event should be recorded")
	})

	t.Run("OnKeyOperation", func(t *testing.T) {
		hook.OnKeyOperation(ctx, "encrypt_dek", "test-key", 1, metadata)

		// Check that key operation counter was incremented
		aggregated := metricsCollector.GetAggregatedMetrics()
		found := false
		for _, metric := range aggregated {
			if metric.Name == "encx.key_operations.total" {
				found = true
				assert.True(t, metric.Count >= 1)
			}
		}
		assert.True(t, found, "Key operation counter should be incremented")
	})

	// Clean up
	require.NoError(t, hook.Stop())
}

func TestStructuredLogger(t *testing.T) {
	t.Run("Development Logger", func(t *testing.T) {
		logger := NewDevelopmentLogger("test")
		assert.NotNil(t, logger)

		// Test logging methods don't panic
		assert.NotPanics(t, func() {
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")
		})
	})

	t.Run("Production Logger", func(t *testing.T) {
		logger := NewProductionLogger("test")
		assert.NotNil(t, logger)

		// Test logging methods don't panic
		assert.NotPanics(t, func() {
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")
		})
	})

	t.Run("WithFields", func(t *testing.T) {
		logger := NewDevelopmentLogger("test")
		fields := map[string]any{
			"user_id":    "123",
			"request_id": "req-456",
		}

		newLogger := logger.WithFields(fields)
		assert.NotNil(t, newLogger)
		assert.NotEqual(t, logger, newLogger) // Should be different instances
	})

	t.Run("WithContext", func(t *testing.T) {
		logger := NewDevelopmentLogger("test")
		ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
		ctx = context.WithValue(ctx, "user_id", "user-456")

		contextLogger := logger.WithContext(ctx)
		assert.NotNil(t, contextLogger)

		// Should not panic
		assert.NotPanics(t, func() {
			contextLogger.Info("message with context")
		})
	})

	t.Run("LogCryptoOperation", func(t *testing.T) {
		logger := NewDevelopmentLogger("test")
		ctx := context.Background()

		metadata := map[string]any{
			"data_size": 1024,
			"algorithm": "AES-256-GCM",
		}

		// Test successful operation
		assert.NotPanics(t, func() {
			logger.LogCryptoOperation(ctx, "encrypt", 100*time.Millisecond, nil, metadata)
		})

		// Test failed operation
		testErr := errors.New("encryption failed")
		assert.NotPanics(t, func() {
			logger.LogCryptoOperation(ctx, "encrypt", 50*time.Millisecond, testErr, metadata)
		})
	})

	t.Run("LogSecurityEvent", func(t *testing.T) {
		logger := NewDevelopmentLogger("test")
		ctx := context.Background()

		metadata := map[string]any{
			"source_ip": "192.168.1.1",
			"user_id":   "suspicious-user",
		}

		// Test different severity levels
		severities := []string{"low", "medium", "high", "critical"}

		for _, severity := range severities {
			assert.NotPanics(t, func() {
				logger.LogSecurityEvent(ctx, "failed_authentication", severity, metadata)
			})
		}
	})
}

func TestEnhancedMetricsCollector(t *testing.T) {
	t.Run("Basic Metrics Operations", func(t *testing.T) {
		collector := NewEnhancedMetricsCollector(EnhancedMetricsConfig{
			ExportPeriod: time.Hour, // Long period to avoid automatic exports
		})

		tags := map[string]string{
			"operation": "test",
			"component": "crypto",
		}

		// Test counter
		collector.IncrementCounter("test.counter", tags)
		collector.IncrementCounterBy("test.counter.by", 5, tags)

		// Test gauge
		collector.SetGauge("test.gauge", 42.5, tags)

		// Test timing
		collector.RecordTiming("test.timing", 100*time.Millisecond, tags)

		// Test value
		collector.RecordValue("test.value", 123.45, tags)

		// Check aggregated metrics
		aggregated := collector.GetAggregatedMetrics()
		assert.True(t, len(aggregated) > 0, "Should have aggregated metrics")

		require.NoError(t, collector.Stop())
	})

	t.Run("Performance Metrics", func(t *testing.T) {
		collector := NewEnhancedMetricsCollector(EnhancedMetricsConfig{
			ExportPeriod: time.Hour,
		})

		tags := map[string]string{"operation": "encrypt"}

		// Test successful operation
		collector.RecordPerformanceMetric("encrypt", 100*time.Millisecond, true, tags)

		// Test failed operation
		collector.RecordPerformanceMetric("encrypt", 50*time.Millisecond, false, tags)

		aggregated := collector.GetAggregatedMetrics()

		// Count total performance metrics (success + failure will be separate due to different tags)
		totalCount := int64(0)
		durationCount := int64(0)

		for _, metric := range aggregated {
			if metric.Name == "encx.performance.encrypt.total" {
				totalCount += metric.Count
			}
			if metric.Name == "encx.performance.encrypt.duration" {
				durationCount += metric.Count
			}
		}

		assert.True(t, totalCount >= 2, "Should have recorded at least 2 performance total metrics (success + failure)")
		assert.True(t, durationCount >= 2, "Should have recorded at least 2 performance duration metrics")

		require.NoError(t, collector.Stop())
	})

	t.Run("Security Metrics", func(t *testing.T) {
		collector := NewEnhancedMetricsCollector(EnhancedMetricsConfig{
			ExportPeriod: time.Hour,
		})

		tags := map[string]string{
			"event":    "failed_auth",
			"severity": "high",
		}

		collector.RecordSecurityMetric("failed_auth", "high", tags)

		aggregated := collector.GetAggregatedMetrics()
		found := false
		for _, metric := range aggregated {
			if metric.Name == "encx.security.events" {
				found = true
				assert.True(t, metric.Count >= 1)
			}
		}
		assert.True(t, found, "Security metrics should be recorded")

		require.NoError(t, collector.Stop())
	})

	t.Run("Business Metrics", func(t *testing.T) {
		collector := NewEnhancedMetricsCollector(EnhancedMetricsConfig{
			ExportPeriod: time.Hour,
		})

		tags := map[string]string{"tenant": "test-tenant"}

		collector.RecordBusinessMetric("active_users", 150.0, "count", tags)

		aggregated := collector.GetAggregatedMetrics()
		found := false
		for _, metric := range aggregated {
			if metric.Name == "encx.business.active_users" {
				found = true
				assert.Equal(t, 150.0, metric.Sum)
			}
		}
		assert.True(t, found, "Business metrics should be recorded")

		require.NoError(t, collector.Stop())
	})
}

func TestMetricsBackends(t *testing.T) {
	t.Run("Prometheus Backend", func(t *testing.T) {
		backend := NewPrometheusMetricsBackend("http://localhost:9090")
		assert.Equal(t, "prometheus", backend.Name())

		ctx := context.Background()
		metrics := []Metric{
			{
				Name:      "test.counter",
				Type:      MetricTypeCounter,
				Value:     1.0,
				Timestamp: time.Now(),
			},
		}

		// Should not error (no-op implementation)
		assert.NoError(t, backend.Export(ctx, metrics))
		assert.NoError(t, backend.Close())
	})

	t.Run("StatsD Backend", func(t *testing.T) {
		backend := NewStatsDMetricsBackend("localhost:8125")
		assert.Equal(t, "statsd", backend.Name())

		ctx := context.Background()
		metrics := []Metric{
			{
				Name:      "test.gauge",
				Type:      MetricTypeGauge,
				Value:     42.5,
				Timestamp: time.Now(),
			},
		}

		// Should not error (no-op implementation)
		assert.NoError(t, backend.Export(ctx, metrics))
		assert.NoError(t, backend.Close())
	})

	t.Run("File Backend", func(t *testing.T) {
		backend := NewFileMetricsBackend("/tmp/test-metrics.log")
		assert.Equal(t, "file", backend.Name())

		ctx := context.Background()
		metrics := []Metric{
			{
				Name:      "test.histogram",
				Type:      MetricTypeHistogram,
				Value:     100.0,
				Unit:      "ms",
				Timestamp: time.Now(),
			},
		}

		// Should not error (no-op implementation)
		assert.NoError(t, backend.Export(ctx, metrics))
		assert.NoError(t, backend.Close())
	})
}

// Benchmark the enhanced metrics collector
func BenchmarkEnhancedMetricsCollector(b *testing.B) {
	collector := NewEnhancedMetricsCollector(EnhancedMetricsConfig{
		ExportPeriod: time.Hour, // Disable automatic export
	})
	defer collector.Stop()

	tags := map[string]string{
		"operation": "benchmark",
		"component": "test",
	}

	b.Run("IncrementCounter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collector.IncrementCounter("bench.counter", tags)
		}
	})

	b.Run("RecordTiming", func(b *testing.B) {
		duration := 100 * time.Millisecond
		for i := 0; i < b.N; i++ {
			collector.RecordTiming("bench.timing", duration, tags)
		}
	})

	b.Run("SetGauge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collector.SetGauge("bench.gauge", float64(i), tags)
		}
	})
}

func TestIsSecurityError(t *testing.T) {
	hook := &EnhancedObservabilityHook{}

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "authentication error",
			err:      errors.New("authentication failed"),
			expected: true,
		},
		{
			name:     "unauthorized error",
			err:      errors.New("unauthorized access"),
			expected: true,
		},
		{
			name:     "decryption failed",
			err:      errors.New("decryption failed"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hook.isSecurityError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}