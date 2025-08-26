package encx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryMetricsCollector(t *testing.T) {
	collector := NewInMemoryMetricsCollector()

	// Test counters
	collector.IncrementCounter("test.counter", map[string]string{"tag1": "value1"})
	collector.IncrementCounterBy("test.counter", 5, map[string]string{"tag1": "value1"})
	
	assert.Equal(t, int64(6), collector.GetCounterValue("test.counter", map[string]string{"tag1": "value1"}))
	assert.Equal(t, int64(0), collector.GetCounterValue("test.counter", map[string]string{"tag1": "value2"}))

	// Test gauges
	collector.SetGauge("test.gauge", 42.5, map[string]string{"tag1": "value1"})
	assert.Equal(t, 42.5, collector.GetGaugeValue("test.gauge", map[string]string{"tag1": "value1"}))

	// Test timings
	duration := 100 * time.Millisecond
	collector.RecordTiming("test.timing", duration, map[string]string{"operation": "test"})
	
	timings := collector.GetTimings()
	require.Len(t, timings, 1)
	assert.Equal(t, "test.timing", timings[0].Name)
	assert.Equal(t, duration, timings[0].Duration)
	assert.Equal(t, "test", timings[0].Tags["operation"])

	// Test values
	collector.RecordValue("test.value", 123.45, map[string]string{"type": "example"})
	
	values := collector.GetValues()
	require.Len(t, values, 1)
	assert.Equal(t, "test.value", values[0].Name)
	assert.Equal(t, 123.45, values[0].Value)
	assert.Equal(t, "example", values[0].Tags["type"])
}

func TestStandardObservabilityHook(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	hook := NewStandardObservabilityHook(collector)
	ctx := context.Background()

	// Test process start
	metadata := map[string]interface{}{
		"operation_type": "test",
		"test_id":        "123",
	}
	hook.OnProcessStart(ctx, "TestOperation", metadata)
	assert.Equal(t, int64(1), collector.GetCounterValue("encx.process.started", map[string]string{
		"operation":      "TestOperation",
		"operation_type": "test",
		"test_id":        "123",
	}))

	// Test successful completion
	duration := 50 * time.Millisecond
	hook.OnProcessComplete(ctx, "TestOperation", duration, nil, metadata)
	
	assert.Equal(t, int64(1), collector.GetCounterValue("encx.process.completed", map[string]string{
		"operation":      "TestOperation",
		"operation_type": "test",
		"test_id":        "123",
		"status":         "success",
	}))
	
	timings := collector.GetTimings()
	require.Len(t, timings, 1)
	assert.Equal(t, "encx.process.duration", timings[0].Name)
	assert.Equal(t, duration, timings[0].Duration)

	// Test error handling
	testErr := assert.AnError
	hook.OnError(ctx, "TestOperation", testErr, metadata)
	
	assert.Equal(t, int64(1), collector.GetCounterValue("encx.errors", map[string]string{
		"operation":      "TestOperation",
		"operation_type": "test",
		"test_id":        "123",
		"error_type":     "general_error",
	}))

	// Test key operations
	hook.OnKeyOperation(ctx, "rotate", "test-alias", 2, metadata)
	
	assert.Equal(t, int64(1), collector.GetCounterValue("encx.key_operations", map[string]string{
		"operation":      "rotate",
		"key_alias":      "test-alias",
		"operation_type": "test",
		"test_id":        "123",
	}))
	
	assert.Equal(t, 2.0, collector.GetGaugeValue("encx.key_version", map[string]string{
		"key_alias": "test-alias",
	}))
}

func TestMonitoringConfiguration(t *testing.T) {
	ctx := context.Background()
	collector := NewInMemoryMetricsCollector()
	tempDir := t.TempDir()

	// Test with standard monitoring
	crypto, err := NewCrypto(ctx,
		WithKMSService(NewSimpleTestKMS()),
		WithKEKAlias("test-monitoring-alias"),
		WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		WithDatabasePath(tempDir+"/monitoring_test.db"),
		WithStandardMonitoring(collector),
	)
	require.NoError(t, err)

	// Verify monitoring is configured
	assert.NotNil(t, crypto.metricsCollector)
	assert.NotNil(t, crypto.observabilityHook)

	// Test ProcessStruct with monitoring
	user := &TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	err = crypto.ProcessStruct(ctx, user)
	require.NoError(t, err)

	// Verify metrics were recorded
	processStartedValue := collector.GetCounterValue("encx.process.started", map[string]string{
		"operation":      "ProcessStruct",
		"operation_type": "struct_processing", 
		"struct_type":    "*encx.TestUser",
	})
	
	processCompletedValue := collector.GetCounterValue("encx.process.completed", map[string]string{
		"operation":      "ProcessStruct",
		"operation_type": "struct_processing",
		"struct_type":    "*encx.TestUser", 
		"status":         "success",
	})
	
	assert.Greater(t, processStartedValue, int64(0))
	assert.Greater(t, processCompletedValue, int64(0))
	
	timings := collector.GetTimings()
	assert.NotEmpty(t, timings)
	
	// Find ProcessStruct timing
	var foundTiming bool
	for _, timing := range timings {
		if timing.Name == "encx.process.duration" && timing.Tags["operation"] == "ProcessStruct" {
			foundTiming = true
			assert.Greater(t, timing.Duration, time.Duration(0))
			break
		}
	}
	assert.True(t, foundTiming, "Should have recorded ProcessStruct timing")
}

func TestMonitoringWithCustomCollector(t *testing.T) {
	ctx := context.Background()
	collector := NewInMemoryMetricsCollector()
	tempDir := t.TempDir()

	// Test with individual monitoring components
	crypto, err := NewCrypto(ctx,
		WithKMSService(NewSimpleTestKMS()),
		WithKEKAlias("test-custom-alias"),
		WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		WithDatabasePath(tempDir+"/custom_test.db"),
		WithMetricsCollector(collector),
		WithObservabilityHook(NewStandardObservabilityHook(collector)),
	)
	require.NoError(t, err)

	// Test key rotation monitoring
	err = crypto.RotateKEK(ctx)
	require.NoError(t, err)

	// Debug what keys were actually created
	t.Logf("All counters after rotation: %+v", collector.counters)
	
	// Verify key operation metrics with correct tags
	assert.Greater(t, collector.GetCounterValue("encx.process.started", map[string]string{
		"key_alias": "test-custom-alias",
		"operation": "RotateKEK",
		"operation_type": "key_rotation",
	}), int64(0))
	
	assert.Greater(t, collector.GetCounterValue("encx.key_operations", map[string]string{
		"key_alias": "test-custom-alias",
		"operation":  "rotate",
		"operation_type": "key_rotation",
	}), int64(0))
	
	// Verify key version gauge was updated
	assert.Greater(t, collector.GetGaugeValue("encx.key_version", map[string]string{
		"key_alias": "test-custom-alias",
	}), 1.0)
}

func TestNoOpMonitoring(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Test with default (no-op) monitoring
	crypto, err := NewCrypto(ctx,
		WithKMSService(NewSimpleTestKMS()),
		WithKEKAlias("test-noop-alias"),
		WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		WithDatabasePath(tempDir+"/noop_test.db"),
	)
	require.NoError(t, err)

	// Should have no-op implementations
	assert.IsType(t, &NoOpMetricsCollector{}, crypto.metricsCollector)
	assert.IsType(t, &NoOpObservabilityHook{}, crypto.observabilityHook)

	// Operations should still work fine
	user := &TestUser{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}

	err = crypto.ProcessStruct(ctx, user)
	require.NoError(t, err)

	// Verify encryption worked
	assert.Empty(t, user.Name)
	assert.NotEmpty(t, user.NameEncrypted)
	assert.NotEmpty(t, user.EmailHash)
}

