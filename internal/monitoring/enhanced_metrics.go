package monitoring

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric measurement
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Unit      string            `json:"unit,omitempty"`
}

// MetricsBackend defines the interface for metrics storage/export
type MetricsBackend interface {
	// Export metrics to the backend
	Export(ctx context.Context, metrics []Metric) error

	// Close the backend connection
	Close() error

	// Name returns the backend name for logging
	Name() string
}

// PrometheusMetricsBackend exports metrics in Prometheus format
type PrometheusMetricsBackend struct {
	endpoint string
	client   interface{} // HTTP client would go here
}

// NewPrometheusMetricsBackend creates a new Prometheus backend
func NewPrometheusMetricsBackend(endpoint string) *PrometheusMetricsBackend {
	return &PrometheusMetricsBackend{
		endpoint: endpoint,
	}
}

func (p *PrometheusMetricsBackend) Export(ctx context.Context, metrics []Metric) error {
	// Implementation would export to Prometheus endpoint
	// For now, just log that we would export
	return nil
}

func (p *PrometheusMetricsBackend) Close() error {
	return nil
}

func (p *PrometheusMetricsBackend) Name() string {
	return "prometheus"
}

// StatsD backend for metrics
type StatsDMetricsBackend struct {
	endpoint string
	conn     interface{} // UDP connection would go here
}

func NewStatsDMetricsBackend(endpoint string) *StatsDMetricsBackend {
	return &StatsDMetricsBackend{
		endpoint: endpoint,
	}
}

func (s *StatsDMetricsBackend) Export(ctx context.Context, metrics []Metric) error {
	// Implementation would send to StatsD
	return nil
}

func (s *StatsDMetricsBackend) Close() error {
	return nil
}

func (s *StatsDMetricsBackend) Name() string {
	return "statsd"
}

// FileMetricsBackend writes metrics to a file
type FileMetricsBackend struct {
	filepath string
	file     interface{} // File handle would go here
	mu       sync.Mutex
}

func NewFileMetricsBackend(filepath string) *FileMetricsBackend {
	return &FileMetricsBackend{
		filepath: filepath,
	}
}

func (f *FileMetricsBackend) Export(ctx context.Context, metrics []Metric) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Implementation would write to file
	return nil
}

func (f *FileMetricsBackend) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Close file handle
	return nil
}

func (f *FileMetricsBackend) Name() string {
	return "file"
}

// EnhancedMetricsCollector provides advanced metrics collection with multiple backends
type EnhancedMetricsCollector struct {
	mu           sync.RWMutex
	metrics      []Metric
	backends     []MetricsBackend
	exportTimer  *time.Timer
	exportPeriod time.Duration
	logger       Logger

	// Rate limiting
	rateLimiter map[string]*time.Timer

	// Aggregation windows
	aggregationPeriod time.Duration
	aggregatedMetrics map[string]*AggregatedMetric
}

// AggregatedMetric holds aggregated data for a metric
type AggregatedMetric struct {
	Name       string
	Type       MetricType
	Tags       map[string]string
	Count      int64
	Sum        float64
	Min        float64
	Max        float64
	LastUpdate time.Time
}

// EnhancedMetricsConfig configures the enhanced metrics collector
type EnhancedMetricsConfig struct {
	ExportPeriod      time.Duration
	AggregationPeriod time.Duration
	Backends          []MetricsBackend
	Logger            Logger
	EnableRateLimit   bool
}

// NewEnhancedMetricsCollector creates a new enhanced metrics collector
func NewEnhancedMetricsCollector(config EnhancedMetricsConfig) *EnhancedMetricsCollector {
	if config.ExportPeriod == 0 {
		config.ExportPeriod = 30 * time.Second
	}
	if config.AggregationPeriod == 0 {
		config.AggregationPeriod = 10 * time.Second
	}
	if config.Logger == nil {
		config.Logger = &StandardLogger{}
	}

	collector := &EnhancedMetricsCollector{
		backends:          config.Backends,
		exportPeriod:      config.ExportPeriod,
		aggregationPeriod: config.AggregationPeriod,
		logger:            config.Logger,
		rateLimiter:       make(map[string]*time.Timer),
		aggregatedMetrics: make(map[string]*AggregatedMetric),
	}

	// Start periodic export
	collector.startPeriodicExport()

	return collector
}

// IncrementCounter increments a counter metric
func (e *EnhancedMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	e.IncrementCounterBy(name, 1, tags)
}

// IncrementCounterBy increments a counter by a specific value
func (e *EnhancedMetricsCollector) IncrementCounterBy(name string, value int64, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     float64(value),
		Tags:      copyTags(tags),
		Timestamp: time.Now().UTC(),
		Unit:      "count",
	}

	e.recordMetric(metric)
}

// SetGauge sets a gauge metric value
func (e *EnhancedMetricsCollector) SetGauge(name string, value float64, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Tags:      copyTags(tags),
		Timestamp: time.Now().UTC(),
	}

	e.recordMetric(metric)
}

// RecordTiming records a timing/duration metric
func (e *EnhancedMetricsCollector) RecordTiming(name string, duration time.Duration, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     float64(duration.Nanoseconds()) / 1e6, // Convert to milliseconds
		Tags:      copyTags(tags),
		Timestamp: time.Now().UTC(),
		Unit:      "ms",
	}

	e.recordMetric(metric)
}

// RecordValue records a value metric
func (e *EnhancedMetricsCollector) RecordValue(name string, value float64, tags map[string]string) {
	metric := Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Tags:      copyTags(tags),
		Timestamp: time.Now().UTC(),
	}

	e.recordMetric(metric)
}

// RecordBusinessMetric records a business-specific metric
func (e *EnhancedMetricsCollector) RecordBusinessMetric(name string, value float64, unit string, tags map[string]string) {
	metric := Metric{
		Name:      fmt.Sprintf("encx.business.%s", name),
		Type:      MetricTypeGauge,
		Value:     value,
		Tags:      copyTags(tags),
		Timestamp: time.Now().UTC(),
		Unit:      unit,
	}

	e.recordMetric(metric)
}

// RecordPerformanceMetric records performance-related metrics
func (e *EnhancedMetricsCollector) RecordPerformanceMetric(operation string, duration time.Duration, success bool, tags map[string]string) {
	if tags == nil {
		tags = make(map[string]string)
	}
	tags["operation"] = operation
	if success {
		tags["status"] = "success"
	} else {
		tags["status"] = "error"
	}

	// Record duration
	e.RecordTiming(fmt.Sprintf("encx.performance.%s.duration", operation), duration, tags)

	// Record success/failure count
	statusTags := copyTags(tags)
	e.IncrementCounter(fmt.Sprintf("encx.performance.%s.total", operation), statusTags)
}

// RecordSecurityMetric records security-related metrics
func (e *EnhancedMetricsCollector) RecordSecurityMetric(event string, severity string, tags map[string]string) {
	if tags == nil {
		tags = make(map[string]string)
	}
	tags["event"] = event
	tags["severity"] = severity

	e.IncrementCounter("encx.security.events", tags)
}

// recordMetric internal method to record a metric
func (e *EnhancedMetricsCollector) recordMetric(metric Metric) {
	// Rate limiting check
	if e.shouldRateLimit(metric) {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Store raw metric
	e.metrics = append(e.metrics, metric)

	// Update aggregated metrics
	e.updateAggregatedMetric(metric)
}

// shouldRateLimit checks if the metric should be rate limited
func (e *EnhancedMetricsCollector) shouldRateLimit(metric Metric) bool {
	// Simple rate limiting: allow at most one metric per second per unique key
	key := e.metricKey(metric)
	e.mu.RLock()
	timer, exists := e.rateLimiter[key]
	e.mu.RUnlock()

	if exists && timer != nil {
		// Rate limited
		return true
	}

	// Set rate limit timer
	e.mu.Lock()
	e.rateLimiter[key] = time.AfterFunc(time.Second, func() {
		e.mu.Lock()
		delete(e.rateLimiter, key)
		e.mu.Unlock()
	})
	e.mu.Unlock()

	return false
}

// metricKey creates a unique key for a metric
func (e *EnhancedMetricsCollector) metricKey(metric Metric) string {
	var parts []string
	parts = append(parts, metric.Name)

	// Sort tags for consistent key generation
	if len(metric.Tags) > 0 {
		keys := make([]string, 0, len(metric.Tags))
		for k := range metric.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s:%s", k, metric.Tags[k]))
		}
	}

	return strings.Join(parts, "|")
}

// updateAggregatedMetric updates the aggregated metrics
func (e *EnhancedMetricsCollector) updateAggregatedMetric(metric Metric) {
	key := e.metricKey(metric)

	agg, exists := e.aggregatedMetrics[key]
	if !exists {
		agg = &AggregatedMetric{
			Name:       metric.Name,
			Type:       metric.Type,
			Tags:       copyTags(metric.Tags),
			Count:      0,
			Sum:        0,
			Min:        metric.Value,
			Max:        metric.Value,
			LastUpdate: metric.Timestamp,
		}
		e.aggregatedMetrics[key] = agg
	}

	// Update aggregated values
	agg.Count++
	agg.Sum += metric.Value
	if metric.Value < agg.Min {
		agg.Min = metric.Value
	}
	if metric.Value > agg.Max {
		agg.Max = metric.Value
	}
	agg.LastUpdate = metric.Timestamp
}

// Flush exports all metrics to backends
func (e *EnhancedMetricsCollector) Flush() error {
	e.mu.Lock()
	metrics := make([]Metric, len(e.metrics))
	copy(metrics, e.metrics)
	e.metrics = e.metrics[:0] // Clear metrics
	e.mu.Unlock()

	return e.exportMetrics(context.Background(), metrics)
}

// exportMetrics exports metrics to all configured backends
func (e *EnhancedMetricsCollector) exportMetrics(ctx context.Context, metrics []Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	var errors []string

	for _, backend := range e.backends {
		if err := backend.Export(ctx, metrics); err != nil {
			errMsg := fmt.Sprintf("failed to export to %s: %v", backend.Name(), err)
			errors = append(errors, errMsg)
			e.logger.Error("Metrics export failed: %s", errMsg)
		} else {
			e.logger.Debug("Exported %d metrics to %s", len(metrics), backend.Name())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("metrics export errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// startPeriodicExport starts the periodic metrics export
func (e *EnhancedMetricsCollector) startPeriodicExport() {
	e.exportTimer = time.AfterFunc(e.exportPeriod, func() {
		if err := e.Flush(); err != nil {
			e.logger.Error("Periodic metrics export failed: %v", err)
		}
		e.startPeriodicExport() // Schedule next export
	})
}

// Stop stops the metrics collector and exports remaining metrics
func (e *EnhancedMetricsCollector) Stop() error {
	if e.exportTimer != nil {
		e.exportTimer.Stop()
	}

	// Final flush
	if err := e.Flush(); err != nil {
		e.logger.Error("Final metrics flush failed: %v", err)
	}

	// Close all backends
	for _, backend := range e.backends {
		if err := backend.Close(); err != nil {
			e.logger.Error("Failed to close backend %s: %v", backend.Name(), err)
		}
	}

	return nil
}

// GetAggregatedMetrics returns current aggregated metrics
func (e *EnhancedMetricsCollector) GetAggregatedMetrics() map[string]*AggregatedMetric {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]*AggregatedMetric)
	for k, v := range e.aggregatedMetrics {
		// Copy the aggregated metric
		result[k] = &AggregatedMetric{
			Name:       v.Name,
			Type:       v.Type,
			Tags:       copyTags(v.Tags),
			Count:      v.Count,
			Sum:        v.Sum,
			Min:        v.Min,
			Max:        v.Max,
			LastUpdate: v.LastUpdate,
		}
	}

	return result
}

// Helper function to copy tags map
func copyTags(tags map[string]string) map[string]string {
	if tags == nil {
		return nil
	}

	result := make(map[string]string, len(tags))
	for k, v := range tags {
		result[k] = v
	}
	return result
}