package monitoring

import (
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

// NoOpMetricsCollector is a no-op implementation of MetricsCollector
type NoOpMetricsCollector struct{}

func (n *NoOpMetricsCollector) IncrementCounter(name string, tags map[string]string)                {}
func (n *NoOpMetricsCollector) IncrementCounterBy(name string, value int64, tags map[string]string) {}
func (n *NoOpMetricsCollector) SetGauge(name string, value float64, tags map[string]string)         {}
func (n *NoOpMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
}
func (n *NoOpMetricsCollector) RecordValue(name string, value float64, tags map[string]string) {}
func (n *NoOpMetricsCollector) Flush() error                                                   { return nil }

// InMemoryMetricsCollector is an in-memory implementation for testing
type InMemoryMetricsCollector struct {
	counters map[string]*int64
	gauges   map[string]float64
	timings  map[string][]time.Duration
	values   map[string][]float64
}

// NewInMemoryMetricsCollector creates a new in-memory metrics collector
func NewInMemoryMetricsCollector() *InMemoryMetricsCollector {
	return &InMemoryMetricsCollector{
		counters: make(map[string]*int64),
		gauges:   make(map[string]float64),
		timings:  make(map[string][]time.Duration),
		values:   make(map[string][]float64),
	}
}

func (m *InMemoryMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	key := m.keyWithTags(name, tags)
	if _, exists := m.counters[key]; !exists {
		var counter int64 = 0
		m.counters[key] = &counter
	}
	atomic.AddInt64(m.counters[key], 1)
}

func (m *InMemoryMetricsCollector) IncrementCounterBy(name string, value int64, tags map[string]string) {
	key := m.keyWithTags(name, tags)
	if _, exists := m.counters[key]; !exists {
		var counter int64 = 0
		m.counters[key] = &counter
	}
	atomic.AddInt64(m.counters[key], value)
}

func (m *InMemoryMetricsCollector) SetGauge(name string, value float64, tags map[string]string) {
	key := m.keyWithTags(name, tags)
	m.gauges[key] = value
}

func (m *InMemoryMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	key := m.keyWithTags(name, tags)
	m.timings[key] = append(m.timings[key], duration)
}

func (m *InMemoryMetricsCollector) RecordValue(name string, value float64, tags map[string]string) {
	key := m.keyWithTags(name, tags)
	m.values[key] = append(m.values[key], value)
}

func (m *InMemoryMetricsCollector) Flush() error {
	return nil
}

func (m *InMemoryMetricsCollector) keyWithTags(name string, tags map[string]string) string {
	if len(tags) == 0 {
		return name
	}

	// Sort tags for consistent key generation
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := name
	for _, k := range keys {
		result += "," + k + "=" + tags[k]
	}

	return result
}

// GetCounter returns the value of a counter
func (m *InMemoryMetricsCollector) GetCounter(name string, tags map[string]string) int64 {
	key := m.keyWithTags(name, tags)
	if _, exists := m.counters[key]; !exists {
		return 0
	}
	return atomic.LoadInt64(m.counters[key])
}

// GetGauge returns the value of a gauge
func (m *InMemoryMetricsCollector) GetGauge(name string, tags map[string]string) float64 {
	key := m.keyWithTags(name, tags)
	return m.gauges[key]
}

// GetTimings returns all recorded timings
func (m *InMemoryMetricsCollector) GetTimings(name string, tags map[string]string) []time.Duration {
	key := m.keyWithTags(name, tags)
	return m.timings[key]
}

// GetValues returns all recorded values
func (m *InMemoryMetricsCollector) GetValues(name string, tags map[string]string) []float64 {
	key := m.keyWithTags(name, tags)
	return m.values[key]
}

// Reset clears all metrics
func (m *InMemoryMetricsCollector) Reset() {
	m.counters = make(map[string]*int64)
	m.gauges = make(map[string]float64)
	m.timings = make(map[string][]time.Duration)
	m.values = make(map[string][]float64)
}

