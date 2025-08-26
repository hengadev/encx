package encx

import (
	"context"
	"sort"
	"sync/atomic"
	"time"
)

// MetricsCollector defines the interface for collecting and reporting metrics
type MetricsCollector interface {
	// Counters
	IncrementCounter(name string, tags map[string]string)
	IncrementCounterBy(name string, value int64, tags map[string]string)
	
	// Gauges  
	SetGauge(name string, value float64, tags map[string]string)
	
	// Histograms/Timing
	RecordTiming(name string, duration time.Duration, tags map[string]string)
	RecordValue(name string, value float64, tags map[string]string)
	
	// Flush any buffered metrics
	Flush() error
}

// ObservabilityHook defines hooks for monitoring encryption operations
type ObservabilityHook interface {
	// Called before processing starts
	OnProcessStart(ctx context.Context, operation string, metadata map[string]interface{})
	
	// Called after processing completes (success or failure)
	OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]interface{})
	
	// Called when errors occur
	OnError(ctx context.Context, operation string, err error, metadata map[string]interface{})
	
	// Called for key operations
	OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]interface{})
}

// NoOpMetricsCollector is a no-op implementation of MetricsCollector
type NoOpMetricsCollector struct{}

func (n *NoOpMetricsCollector) IncrementCounter(name string, tags map[string]string)                    {}
func (n *NoOpMetricsCollector) IncrementCounterBy(name string, value int64, tags map[string]string)    {}
func (n *NoOpMetricsCollector) SetGauge(name string, value float64, tags map[string]string)            {}
func (n *NoOpMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {}
func (n *NoOpMetricsCollector) RecordValue(name string, value float64, tags map[string]string)         {}
func (n *NoOpMetricsCollector) Flush() error                                                            { return nil }

// NoOpObservabilityHook is a no-op implementation of ObservabilityHook
type NoOpObservabilityHook struct{}

func (n *NoOpObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]interface{})   {}
func (n *NoOpObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]interface{}) {}
func (n *NoOpObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]interface{}) {}
func (n *NoOpObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]interface{}) {}

// InMemoryMetricsCollector is a simple in-memory implementation for testing and development
type InMemoryMetricsCollector struct {
	counters   map[string]*int64
	gauges     map[string]*float64
	timings    []TimingMetric
	values     []ValueMetric
}

type TimingMetric struct {
	Name     string
	Duration time.Duration
	Tags     map[string]string
	Time     time.Time
}

type ValueMetric struct {
	Name  string
	Value float64
	Tags  map[string]string
	Time  time.Time
}

// NewInMemoryMetricsCollector creates a new in-memory metrics collector
func NewInMemoryMetricsCollector() *InMemoryMetricsCollector {
	return &InMemoryMetricsCollector{
		counters: make(map[string]*int64),
		gauges:   make(map[string]*float64),
		timings:  make([]TimingMetric, 0),
		values:   make([]ValueMetric, 0),
	}
}

func (m *InMemoryMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	m.IncrementCounterBy(name, 1, tags)
}

func (m *InMemoryMetricsCollector) IncrementCounterBy(name string, value int64, tags map[string]string) {
	key := m.buildKey(name, tags)
	if _, exists := m.counters[key]; !exists {
		m.counters[key] = new(int64)
	}
	atomic.AddInt64(m.counters[key], value)
}

func (m *InMemoryMetricsCollector) SetGauge(name string, value float64, tags map[string]string) {
	key := m.buildKey(name, tags)
	if _, exists := m.gauges[key]; !exists {
		m.gauges[key] = new(float64)
	}
	// Note: This is not atomic, but good enough for in-memory testing
	*m.gauges[key] = value
}

func (m *InMemoryMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	m.timings = append(m.timings, TimingMetric{
		Name:     name,
		Duration: duration,
		Tags:     m.copyTags(tags),
		Time:     time.Now(),
	})
}

func (m *InMemoryMetricsCollector) RecordValue(name string, value float64, tags map[string]string) {
	m.values = append(m.values, ValueMetric{
		Name:  name,
		Value: value,
		Tags:  m.copyTags(tags),
		Time:  time.Now(),
	})
}

func (m *InMemoryMetricsCollector) Flush() error {
	// Nothing to flush for in-memory implementation
	return nil
}

// GetCounterValue returns the current value of a counter
func (m *InMemoryMetricsCollector) GetCounterValue(name string, tags map[string]string) int64 {
	key := m.buildKey(name, tags)
	if counter, exists := m.counters[key]; exists {
		return atomic.LoadInt64(counter)
	}
	return 0
}

// GetGaugeValue returns the current value of a gauge
func (m *InMemoryMetricsCollector) GetGaugeValue(name string, tags map[string]string) float64 {
	key := m.buildKey(name, tags)
	if gauge, exists := m.gauges[key]; exists {
		return *gauge
	}
	return 0
}

// GetTimings returns all recorded timing metrics
func (m *InMemoryMetricsCollector) GetTimings() []TimingMetric {
	return append([]TimingMetric(nil), m.timings...)
}

// GetValues returns all recorded value metrics
func (m *InMemoryMetricsCollector) GetValues() []ValueMetric {
	return append([]ValueMetric(nil), m.values...)
}

// GetAllCounterKeys returns all counter keys for iteration
func (m *InMemoryMetricsCollector) GetAllCounterKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	for key := range m.counters {
		keys[key] = struct{}{}
	}
	return keys
}

// GetCounterValueByKey returns the counter value for a full key
func (m *InMemoryMetricsCollector) GetCounterValueByKey(key string) int64 {
	if counter, exists := m.counters[key]; exists {
		return atomic.LoadInt64(counter)
	}
	return 0
}

func (m *InMemoryMetricsCollector) buildKey(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}
	
	// Sort tags to ensure deterministic key generation
	var keys []string
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	key := name
	for _, k := range keys {
		key += "," + k + ":" + tags[k]
	}
	return key
}

func (m *InMemoryMetricsCollector) copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}
	
	copied := make(map[string]string, len(tags))
	for k, v := range tags {
		copied[k] = v
	}
	return copied
}

// StandardObservabilityHook is a comprehensive observability hook implementation
type StandardObservabilityHook struct {
	metrics MetricsCollector
}

// NewStandardObservabilityHook creates a new standard observability hook
func NewStandardObservabilityHook(metrics MetricsCollector) *StandardObservabilityHook {
	if metrics == nil {
		metrics = &NoOpMetricsCollector{}
	}
	
	return &StandardObservabilityHook{
		metrics: metrics,
	}
}

func (h *StandardObservabilityHook) OnProcessStart(ctx context.Context, operation string, metadata map[string]interface{}) {
	tags := h.buildTags(metadata)
	tags["operation"] = operation
	
	h.metrics.IncrementCounter("encx.process.started", tags)
}

func (h *StandardObservabilityHook) OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]interface{}) {
	tags := h.buildTags(metadata)
	tags["operation"] = operation
	
	if err != nil {
		tags["status"] = "error"
		h.metrics.IncrementCounter("encx.process.failed", tags)
	} else {
		tags["status"] = "success"
		h.metrics.IncrementCounter("encx.process.completed", tags)
	}
	
	h.metrics.RecordTiming("encx.process.duration", duration, tags)
}

func (h *StandardObservabilityHook) OnError(ctx context.Context, operation string, err error, metadata map[string]interface{}) {
	tags := h.buildTags(metadata)
	tags["operation"] = operation
	tags["error_type"] = getErrorType(err)
	
	h.metrics.IncrementCounter("encx.errors", tags)
}

func (h *StandardObservabilityHook) OnKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]interface{}) {
	tags := h.buildTags(metadata)
	tags["operation"] = operation
	tags["key_alias"] = keyAlias
	
	h.metrics.IncrementCounter("encx.key_operations", tags)
	h.metrics.SetGauge("encx.key_version", float64(keyVersion), map[string]string{
		"key_alias": keyAlias,
	})
}

func (h *StandardObservabilityHook) buildTags(metadata map[string]interface{}) map[string]string {
	tags := make(map[string]string)
	
	if metadata != nil {
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				tags[k] = str
			}
		}
	}
	
	return tags
}

func getErrorType(err error) string {
	if err == nil {
		return "none"
	}
	
	// You could implement more sophisticated error type detection here
	// For now, just return the basic error type
	return "general_error"
}