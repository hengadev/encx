package monitoring

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNoOpMetricsCollector(t *testing.T) {
	collector := &NoOpMetricsCollector{}

	// Test that all methods can be called without panic
	tags := map[string]string{"test": "value"}

	// Should not panic
	collector.IncrementCounter("test_counter", tags)
	collector.IncrementCounterBy("test_counter", 5, tags)
	collector.SetGauge("test_gauge", 42.5, tags)
	collector.RecordTiming("test_timing", time.Millisecond, tags)
	collector.RecordValue("test_value", 3.14, tags)

	err := collector.Flush()
	assert.NoError(t, err)
}

func TestNewInMemoryMetricsCollector(t *testing.T) {
	collector := NewInMemoryMetricsCollector()

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.counters)
	assert.NotNil(t, collector.gauges)
	assert.NotNil(t, collector.timings)
	assert.NotNil(t, collector.values)
}

func TestInMemoryMetricsCollector_IncrementCounter(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	tags := map[string]string{"env": "test"}

	// Test first increment
	collector.IncrementCounter("requests", tags)

	// Check that counter was created and incremented
	key := collector.keyWithTags("requests", tags)
	assert.Contains(t, collector.counters, key)
	value := atomic.LoadInt64(collector.counters[key])
	assert.Equal(t, int64(1), value)

	// Test second increment
	collector.IncrementCounter("requests", tags)
	value = atomic.LoadInt64(collector.counters[key])
	assert.Equal(t, int64(2), value)
}

func TestInMemoryMetricsCollector_IncrementCounterBy(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	tags := map[string]string{"env": "test", "service": "api"}

	// Test increment by specific value
	collector.IncrementCounterBy("bytes_processed", 1024, tags)

	key := collector.keyWithTags("bytes_processed", tags)
	value := atomic.LoadInt64(collector.counters[key])
	assert.Equal(t, int64(1024), value)

	// Test increment by another value
	collector.IncrementCounterBy("bytes_processed", 512, tags)
	value = atomic.LoadInt64(collector.counters[key])
	assert.Equal(t, int64(1536), value)
}

func TestInMemoryMetricsCollector_SetGauge(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	tags := map[string]string{"region": "us-east-1"}

	// Test setting gauge value
	collector.SetGauge("memory_usage", 75.5, tags)

	key := collector.keyWithTags("memory_usage", tags)
	assert.Contains(t, collector.gauges, key)
	assert.Equal(t, 75.5, collector.gauges[key])

	// Test updating gauge value
	collector.SetGauge("memory_usage", 82.3, tags)
	assert.Equal(t, 82.3, collector.gauges[key])
}

func TestInMemoryMetricsCollector_RecordTiming(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	tags := map[string]string{"operation": "encrypt"}

	duration1 := 150 * time.Millisecond
	duration2 := 200 * time.Millisecond

	// Test recording timings
	collector.RecordTiming("operation_duration", duration1, tags)
	collector.RecordTiming("operation_duration", duration2, tags)

	key := collector.keyWithTags("operation_duration", tags)
	assert.Contains(t, collector.timings, key)
	assert.Len(t, collector.timings[key], 2)
	assert.Contains(t, collector.timings[key], duration1)
	assert.Contains(t, collector.timings[key], duration2)
}

func TestInMemoryMetricsCollector_RecordValue(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	tags := map[string]string{"batch": "1"}

	// Test recording values
	collector.RecordValue("throughput", 1250.5, tags)
	collector.RecordValue("throughput", 1500.2, tags)

	key := collector.keyWithTags("throughput", tags)
	assert.Contains(t, collector.values, key)
	assert.Len(t, collector.values[key], 2)
	assert.Contains(t, collector.values[key], 1250.5)
	assert.Contains(t, collector.values[key], 1500.2)
}

func TestInMemoryMetricsCollector_KeyWithTags(t *testing.T) {
	collector := NewInMemoryMetricsCollector()

	tests := []struct {
		name     string
		metricName string
		tags     map[string]string
		expected string
	}{
		{
			name:       "no tags",
			metricName: "test_metric",
			tags:       nil,
			expected:   "test_metric",
		},
		{
			name:       "empty tags",
			metricName: "test_metric",
			tags:       map[string]string{},
			expected:   "test_metric",
		},
		{
			name:       "single tag",
			metricName: "requests",
			tags:       map[string]string{"env": "prod"},
			expected:   "requests,env=prod",
		},
		{
			name:       "multiple tags",
			metricName: "latency",
			tags:       map[string]string{"env": "prod", "service": "api", "region": "us-east-1"},
			expected:   "latency,env=prod,region=us-east-1,service=api", // should be sorted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.keyWithTags(tt.metricName, tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInMemoryMetricsCollector_Flush(t *testing.T) {
	collector := NewInMemoryMetricsCollector()

	// Add some metrics
	collector.IncrementCounter("test", nil)
	collector.SetGauge("memory", 50.0, nil)

	// Flush should not error
	err := collector.Flush()
	assert.NoError(t, err)

	// Metrics should still be there (flush doesn't clear in-memory)
	assert.Len(t, collector.counters, 1)
	assert.Len(t, collector.gauges, 1)
}

func TestInMemoryMetricsCollector_ConcurrentAccess(t *testing.T) {
	collector := NewInMemoryMetricsCollector()
	tags := map[string]string{"worker": "test"}

	// Test concurrent counter increments
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			collector.IncrementCounter("concurrent_test", tags)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	key := collector.keyWithTags("concurrent_test", tags)
	value := atomic.LoadInt64(collector.counters[key])
	assert.Equal(t, int64(10), value)
}

func TestInMemoryMetricsCollector_DifferentTags(t *testing.T) {
	collector := NewInMemoryMetricsCollector()

	// Same metric name, different tags should create different entries
	collector.IncrementCounter("requests", map[string]string{"env": "prod"})
	collector.IncrementCounter("requests", map[string]string{"env": "dev"})
	collector.IncrementCounter("requests", map[string]string{"env": "prod"}) // increment again

	assert.Len(t, collector.counters, 2)

	prodKey := collector.keyWithTags("requests", map[string]string{"env": "prod"})
	devKey := collector.keyWithTags("requests", map[string]string{"env": "dev"})

	prodValue := atomic.LoadInt64(collector.counters[prodKey])
	devValue := atomic.LoadInt64(collector.counters[devKey])

	assert.Equal(t, int64(2), prodValue)
	assert.Equal(t, int64(1), devValue)
}

func TestInMemoryMetricsCollector_EdgeCases(t *testing.T) {
	collector := NewInMemoryMetricsCollector()

	// Test empty metric name
	collector.IncrementCounter("", nil)
	assert.Contains(t, collector.counters, "")

	// Test nil tags
	collector.SetGauge("test_gauge", 42.0, nil)
	assert.Contains(t, collector.gauges, "test_gauge")

	// Test zero values
	collector.IncrementCounterBy("zero_test", 0, nil)
	value := atomic.LoadInt64(collector.counters["zero_test"])
	assert.Equal(t, int64(0), value)

	// Test negative values
	collector.IncrementCounterBy("negative_test", -5, nil)
	value = atomic.LoadInt64(collector.counters["negative_test"])
	assert.Equal(t, int64(-5), value)

	// Test zero duration
	collector.RecordTiming("zero_duration", 0, nil)
	assert.Contains(t, collector.timings, "zero_duration")
	assert.Equal(t, time.Duration(0), collector.timings["zero_duration"][0])
}