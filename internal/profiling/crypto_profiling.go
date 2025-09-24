package profiling

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// CryptoProfiler provides crypto-specific performance profiling
type CryptoProfiler struct {
	profiler *Profiler

	// Crypto operation metrics
	encryptionMetrics map[string]*OperationMetrics
	decryptionMetrics map[string]*OperationMetrics
	keyMetrics        map[string]*OperationMetrics
	hashingMetrics    map[string]*OperationMetrics

	mutex sync.RWMutex
}

// OperationMetrics tracks metrics for specific crypto operations
type OperationMetrics struct {
	TotalCalls      int64         `json:"total_calls"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	ErrorCount      int64         `json:"error_count"`
	LastCall        time.Time     `json:"last_call"`

	// Memory metrics
	TotalAllocs uint64 `json:"total_allocs"`
	TotalBytes  uint64 `json:"total_bytes"`

	// Performance thresholds exceeded
	SlowCallsCount int64         `json:"slow_calls_count"`
	SlowThreshold  time.Duration `json:"slow_threshold"`
}

// CryptoProfilingConfig holds configuration for crypto profiling
type CryptoProfilingConfig struct {
	// Enable profiling for different crypto operations
	EnableEncryptionProfiling    bool
	EnableDecryptionProfiling    bool
	EnableKeyOperationsProfiling bool
	EnableHashingProfiling       bool

	// Performance thresholds
	EncryptionSlowThreshold   time.Duration
	DecryptionSlowThreshold   time.Duration
	KeyOperationSlowThreshold time.Duration
	HashingSlowThreshold      time.Duration

	// Memory thresholds for triggering profiling
	MemoryAllocationThreshold uint64

	// Auto-profiling triggers
	AutoProfileSlowOperations bool
	AutoProfileHighMemory     bool

	// Sample rates (0.0 = never, 1.0 = always)
	OperationSampleRate float64
	MemorySampleRate    float64
}

// DefaultCryptoProfilingConfig returns default crypto profiling configuration
func DefaultCryptoProfilingConfig() CryptoProfilingConfig {
	return CryptoProfilingConfig{
		EnableEncryptionProfiling:    true,
		EnableDecryptionProfiling:    true,
		EnableKeyOperationsProfiling: true,
		EnableHashingProfiling:       true,

		EncryptionSlowThreshold:   time.Millisecond * 100,
		DecryptionSlowThreshold:   time.Millisecond * 100,
		KeyOperationSlowThreshold: time.Millisecond * 500,
		HashingSlowThreshold:      time.Millisecond * 10,

		MemoryAllocationThreshold: 1024 * 1024, // 1MB

		AutoProfileSlowOperations: true,
		AutoProfileHighMemory:     true,

		OperationSampleRate: 0.1, // Sample 10% of operations
		MemorySampleRate:    1.0, // Always track memory
	}
}

// NewCryptoProfiler creates a new crypto profiler
func NewCryptoProfiler(profiler *Profiler) *CryptoProfiler {
	return &CryptoProfiler{
		profiler:          profiler,
		encryptionMetrics: make(map[string]*OperationMetrics),
		decryptionMetrics: make(map[string]*OperationMetrics),
		keyMetrics:        make(map[string]*OperationMetrics),
		hashingMetrics:    make(map[string]*OperationMetrics),
	}
}

// ProfileEncryption profiles an encryption operation
func (cp *CryptoProfiler) ProfileEncryption(ctx context.Context, operationName string, dataSize int64, operation func() error) error {
	return cp.profileOperation(ctx, "encryption", operationName, dataSize, operation, cp.encryptionMetrics)
}

// ProfileDecryption profiles a decryption operation
func (cp *CryptoProfiler) ProfileDecryption(ctx context.Context, operationName string, dataSize int64, operation func() error) error {
	return cp.profileOperation(ctx, "decryption", operationName, dataSize, operation, cp.decryptionMetrics)
}

// ProfileKeyOperation profiles a key management operation
func (cp *CryptoProfiler) ProfileKeyOperation(ctx context.Context, operationName string, operation func() error) error {
	return cp.profileOperation(ctx, "key_operation", operationName, 0, operation, cp.keyMetrics)
}

// ProfileHashing profiles a hashing operation
func (cp *CryptoProfiler) ProfileHashing(ctx context.Context, operationName string, dataSize int64, operation func() error) error {
	return cp.profileOperation(ctx, "hashing", operationName, dataSize, operation, cp.hashingMetrics)
}

// profileOperation is the core profiling logic for crypto operations
func (cp *CryptoProfiler) profileOperation(ctx context.Context, category, operationName string, dataSize int64, operation func() error, metricsMap map[string]*OperationMetrics) error {

	// Pre-operation memory stats
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	startTime := time.Now()

	// Execute the operation
	err := operation()

	duration := time.Since(startTime)

	// Post-operation memory stats
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory allocation for this operation
	allocsIncrease := memAfter.Mallocs - memBefore.Mallocs
	bytesIncrease := memAfter.TotalAlloc - memBefore.TotalAlloc

	// Update metrics
	cp.updateOperationMetrics(metricsMap, operationName, duration, err != nil, allocsIncrease, bytesIncrease, dataSize)

	// Check if we should trigger profiling based on performance
	cp.checkProfilingTriggers(category, operationName, duration, bytesIncrease)

	return err
}

// updateOperationMetrics updates metrics for an operation
func (cp *CryptoProfiler) updateOperationMetrics(metricsMap map[string]*OperationMetrics, operationName string, duration time.Duration, hasError bool, allocs, bytes uint64, dataSize int64) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	metrics, exists := metricsMap[operationName]
	if !exists {
		metrics = &OperationMetrics{
			MinDuration:   duration,
			MaxDuration:   duration,
			SlowThreshold: cp.getSlowThresholdForOperation(operationName),
		}
		metricsMap[operationName] = metrics
	}

	// Update counters
	metrics.TotalCalls++
	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.TotalCalls)
	metrics.LastCall = time.Now()

	// Update duration bounds
	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	// Update memory metrics
	metrics.TotalAllocs += allocs
	metrics.TotalBytes += bytes

	// Update error count
	if hasError {
		metrics.ErrorCount++
	}

	// Check if this is a slow call
	if duration > metrics.SlowThreshold {
		metrics.SlowCallsCount++
	}
}

// checkProfilingTriggers checks if we should trigger profiling based on operation performance
func (cp *CryptoProfiler) checkProfilingTriggers(category, operationName string, duration time.Duration, memoryIncrease uint64) {
	config := DefaultCryptoProfilingConfig() // In real implementation, this would come from the profiler config

	// Check if operation was slow and should trigger CPU profiling
	if config.AutoProfileSlowOperations {
		var threshold time.Duration
		switch category {
		case "encryption":
			threshold = config.EncryptionSlowThreshold
		case "decryption":
			threshold = config.DecryptionSlowThreshold
		case "key_operation":
			threshold = config.KeyOperationSlowThreshold
		case "hashing":
			threshold = config.HashingSlowThreshold
		default:
			threshold = time.Millisecond * 100
		}

		if duration > threshold {
			go cp.profiler.StartProfile(ProfileTypeCPU)
		}
	}

	// Check if memory allocation was high and should trigger memory profiling
	if config.AutoProfileHighMemory && memoryIncrease > config.MemoryAllocationThreshold {
		go cp.profiler.StartProfile(ProfileTypeMemory)
	}
}

// getSlowThresholdForOperation returns the slow threshold for a specific operation
func (cp *CryptoProfiler) getSlowThresholdForOperation(operationName string) time.Duration {
	// This is a simplified version - in real implementation, this would be configurable
	switch {
	case contains(operationName, "encrypt"):
		return time.Millisecond * 100
	case contains(operationName, "decrypt"):
		return time.Millisecond * 100
	case contains(operationName, "key") || contains(operationName, "kms"):
		return time.Millisecond * 500
	case contains(operationName, "hash"):
		return time.Millisecond * 10
	default:
		return time.Millisecond * 50
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexSubstring(s, substr) >= 0))
}

// indexSubstring finds the index of substring in string
func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetEncryptionMetrics returns encryption metrics
func (cp *CryptoProfiler) GetEncryptionMetrics() map[string]*OperationMetrics {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.copyMetrics(cp.encryptionMetrics)
}

// GetDecryptionMetrics returns decryption metrics
func (cp *CryptoProfiler) GetDecryptionMetrics() map[string]*OperationMetrics {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.copyMetrics(cp.decryptionMetrics)
}

// GetKeyMetrics returns key operation metrics
func (cp *CryptoProfiler) GetKeyMetrics() map[string]*OperationMetrics {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.copyMetrics(cp.keyMetrics)
}

// GetHashingMetrics returns hashing metrics
func (cp *CryptoProfiler) GetHashingMetrics() map[string]*OperationMetrics {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.copyMetrics(cp.hashingMetrics)
}

// GetAllMetrics returns all crypto operation metrics
func (cp *CryptoProfiler) GetAllMetrics() CryptoMetricsReport {
	return CryptoMetricsReport{
		Encryption:    cp.GetEncryptionMetrics(),
		Decryption:    cp.GetDecryptionMetrics(),
		KeyOperations: cp.GetKeyMetrics(),
		Hashing:       cp.GetHashingMetrics(),
		Timestamp:     time.Now(),
	}
}

// CryptoMetricsReport contains all crypto metrics
type CryptoMetricsReport struct {
	Encryption    map[string]*OperationMetrics `json:"encryption"`
	Decryption    map[string]*OperationMetrics `json:"decryption"`
	KeyOperations map[string]*OperationMetrics `json:"key_operations"`
	Hashing       map[string]*OperationMetrics `json:"hashing"`
	Timestamp     time.Time                    `json:"timestamp"`
}

// copyMetrics creates a deep copy of metrics map
func (cp *CryptoProfiler) copyMetrics(source map[string]*OperationMetrics) map[string]*OperationMetrics {
	copy := make(map[string]*OperationMetrics)
	for key, metrics := range source {
		metricsCopy := *metrics
		copy[key] = &metricsCopy
	}
	return copy
}

// ResetMetrics resets all crypto operation metrics
func (cp *CryptoProfiler) ResetMetrics() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	cp.encryptionMetrics = make(map[string]*OperationMetrics)
	cp.decryptionMetrics = make(map[string]*OperationMetrics)
	cp.keyMetrics = make(map[string]*OperationMetrics)
	cp.hashingMetrics = make(map[string]*OperationMetrics)
}

// GetTopSlowOperations returns the top N slowest operations across all categories
func (cp *CryptoProfiler) GetTopSlowOperations(n int) []SlowOperationInfo {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	var operations []SlowOperationInfo

	// Collect all operations
	for name, metrics := range cp.encryptionMetrics {
		operations = append(operations, SlowOperationInfo{
			Category:        "encryption",
			Name:            name,
			AverageDuration: metrics.AverageDuration,
			MaxDuration:     metrics.MaxDuration,
			SlowCallsCount:  metrics.SlowCallsCount,
			TotalCalls:      metrics.TotalCalls,
		})
	}

	for name, metrics := range cp.decryptionMetrics {
		operations = append(operations, SlowOperationInfo{
			Category:        "decryption",
			Name:            name,
			AverageDuration: metrics.AverageDuration,
			MaxDuration:     metrics.MaxDuration,
			SlowCallsCount:  metrics.SlowCallsCount,
			TotalCalls:      metrics.TotalCalls,
		})
	}

	for name, metrics := range cp.keyMetrics {
		operations = append(operations, SlowOperationInfo{
			Category:        "key_operations",
			Name:            name,
			AverageDuration: metrics.AverageDuration,
			MaxDuration:     metrics.MaxDuration,
			SlowCallsCount:  metrics.SlowCallsCount,
			TotalCalls:      metrics.TotalCalls,
		})
	}

	for name, metrics := range cp.hashingMetrics {
		operations = append(operations, SlowOperationInfo{
			Category:        "hashing",
			Name:            name,
			AverageDuration: metrics.AverageDuration,
			MaxDuration:     metrics.MaxDuration,
			SlowCallsCount:  metrics.SlowCallsCount,
			TotalCalls:      metrics.TotalCalls,
		})
	}

	// Sort by average duration (descending)
	sortSlowOperations(operations)

	// Return top N
	if n > len(operations) {
		n = len(operations)
	}

	return operations[:n]
}

// SlowOperationInfo contains information about slow operations
type SlowOperationInfo struct {
	Category        string        `json:"category"`
	Name            string        `json:"name"`
	AverageDuration time.Duration `json:"average_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	SlowCallsCount  int64         `json:"slow_calls_count"`
	TotalCalls      int64         `json:"total_calls"`
}

// sortSlowOperations sorts operations by average duration (descending)
func sortSlowOperations(operations []SlowOperationInfo) {
	// Simple bubble sort - in production, use sort.Slice
	n := len(operations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if operations[j].AverageDuration < operations[j+1].AverageDuration {
				operations[j], operations[j+1] = operations[j+1], operations[j]
			}
		}
	}
}

// GetMemoryIntensiveOperations returns operations that use the most memory
func (cp *CryptoProfiler) GetMemoryIntensiveOperations(n int) []MemoryOperationInfo {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	var operations []MemoryOperationInfo

	// Collect memory information from all operations
	for name, metrics := range cp.encryptionMetrics {
		if metrics.TotalCalls > 0 {
			operations = append(operations, MemoryOperationInfo{
				Category:      "encryption",
				Name:          name,
				TotalBytes:    metrics.TotalBytes,
				TotalAllocs:   metrics.TotalAllocs,
				AverageBytes:  metrics.TotalBytes / uint64(metrics.TotalCalls),
				AverageAllocs: metrics.TotalAllocs / uint64(metrics.TotalCalls),
			})
		}
	}

	for name, metrics := range cp.decryptionMetrics {
		if metrics.TotalCalls > 0 {
			operations = append(operations, MemoryOperationInfo{
				Category:      "decryption",
				Name:          name,
				TotalBytes:    metrics.TotalBytes,
				TotalAllocs:   metrics.TotalAllocs,
				AverageBytes:  metrics.TotalBytes / uint64(metrics.TotalCalls),
				AverageAllocs: metrics.TotalAllocs / uint64(metrics.TotalCalls),
			})
		}
	}

	for name, metrics := range cp.keyMetrics {
		if metrics.TotalCalls > 0 {
			operations = append(operations, MemoryOperationInfo{
				Category:      "key_operations",
				Name:          name,
				TotalBytes:    metrics.TotalBytes,
				TotalAllocs:   metrics.TotalAllocs,
				AverageBytes:  metrics.TotalBytes / uint64(metrics.TotalCalls),
				AverageAllocs: metrics.TotalAllocs / uint64(metrics.TotalCalls),
			})
		}
	}

	for name, metrics := range cp.hashingMetrics {
		if metrics.TotalCalls > 0 {
			operations = append(operations, MemoryOperationInfo{
				Category:      "hashing",
				Name:          name,
				TotalBytes:    metrics.TotalBytes,
				TotalAllocs:   metrics.TotalAllocs,
				AverageBytes:  metrics.TotalBytes / uint64(metrics.TotalCalls),
				AverageAllocs: metrics.TotalAllocs / uint64(metrics.TotalCalls),
			})
		}
	}

	// Sort by total bytes (descending)
	sortMemoryOperations(operations)

	// Return top N
	if n > len(operations) {
		n = len(operations)
	}

	return operations[:n]
}

// MemoryOperationInfo contains information about memory usage of operations
type MemoryOperationInfo struct {
	Category      string `json:"category"`
	Name          string `json:"name"`
	TotalBytes    uint64 `json:"total_bytes"`
	TotalAllocs   uint64 `json:"total_allocs"`
	AverageBytes  uint64 `json:"average_bytes"`
	AverageAllocs uint64 `json:"average_allocs"`
}

// sortMemoryOperations sorts operations by total bytes used (descending)
func sortMemoryOperations(operations []MemoryOperationInfo) {
	// Simple bubble sort - in production, use sort.Slice
	n := len(operations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if operations[j].TotalBytes < operations[j+1].TotalBytes {
				operations[j], operations[j+1] = operations[j+1], operations[j]
			}
		}
	}
}

// Global crypto profiler
var globalCryptoProfiler *CryptoProfiler
var globalCryptoProfilerOnce sync.Once

// GetGlobalCryptoProfiler returns the global crypto profiler instance
func GetGlobalCryptoProfiler() *CryptoProfiler {
	globalCryptoProfilerOnce.Do(func() {
		globalCryptoProfiler = NewCryptoProfiler(GetGlobalProfiler())
	})
	return globalCryptoProfiler
}

// Helper functions for global crypto profiling

// ProfileGlobalEncryption profiles an encryption operation using the global crypto profiler
func ProfileGlobalEncryption(ctx context.Context, operationName string, dataSize int64, operation func() error) error {
	return GetGlobalCryptoProfiler().ProfileEncryption(ctx, operationName, dataSize, operation)
}

// ProfileGlobalDecryption profiles a decryption operation using the global crypto profiler
func ProfileGlobalDecryption(ctx context.Context, operationName string, dataSize int64, operation func() error) error {
	return GetGlobalCryptoProfiler().ProfileDecryption(ctx, operationName, dataSize, operation)
}

// ProfileGlobalKeyOperation profiles a key operation using the global crypto profiler
func ProfileGlobalKeyOperation(ctx context.Context, operationName string, operation func() error) error {
	return GetGlobalCryptoProfiler().ProfileKeyOperation(ctx, operationName, operation)
}

// ProfileGlobalHashing profiles a hashing operation using the global crypto profiler
func ProfileGlobalHashing(ctx context.Context, operationName string, dataSize int64, operation func() error) error {
	return GetGlobalCryptoProfiler().ProfileHashing(ctx, operationName, dataSize, operation)
}

