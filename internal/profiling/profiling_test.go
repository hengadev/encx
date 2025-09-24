package profiling

import (
	"context"
	"errors"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewProfiler(t *testing.T) {
	config := DefaultProfilingConfig()
	profiler := NewProfiler(config)

	if profiler == nil {
		t.Error("Expected profiler to be created")
	}

	if profiler.isRunning {
		t.Error("Expected profiler to not be running initially")
	}

	if len(profiler.profileSessions) != 0 {
		t.Error("Expected no profile sessions initially")
	}
}

func TestProfiler_StartStop(t *testing.T) {
	// Use a temporary directory for profiles
	tempDir := t.TempDir()

	config := DefaultProfilingConfig()
	config.OutputDir = tempDir
	config.HTTPEndpoint = "" // Disable HTTP endpoint for testing

	profiler := NewProfiler(config)
	ctx := context.Background()

	// Test start
	err := profiler.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting profiler, got %v", err)
	}

	if !profiler.isRunning {
		t.Error("Expected profiler to be running after start")
	}

	// Test double start
	err = profiler.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting profiler twice")
	}

	// Test stop
	err = profiler.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error stopping profiler, got %v", err)
	}

	if profiler.isRunning {
		t.Error("Expected profiler to not be running after stop")
	}

	// Test double stop (should not error)
	err = profiler.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error stopping profiler twice, got %v", err)
	}
}

func TestProfiler_CPUProfiling(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultProfilingConfig()
	config.OutputDir = tempDir
	config.HTTPEndpoint = ""
	config.ProfileDuration = time.Millisecond * 100

	profiler := NewProfiler(config)
	ctx := context.Background()

	err := profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop(ctx)

	// Start CPU profiling
	err = profiler.StartProfile(ProfileTypeCPU)
	if err != nil {
		t.Errorf("Expected no error starting CPU profile, got %v", err)
	}

	// Check if profile is active
	activeProfiles := profiler.GetActiveProfiles()
	if len(activeProfiles) != 1 {
		t.Errorf("Expected 1 active profile, got %d", len(activeProfiles))
	}

	if _, exists := activeProfiles[ProfileTypeCPU]; !exists {
		t.Error("Expected CPU profile to be active")
	}

	// Wait a bit for some profiling data
	time.Sleep(time.Millisecond * 50)

	// Stop profiling
	err = profiler.StopProfile(ProfileTypeCPU)
	if err != nil {
		t.Errorf("Expected no error stopping CPU profile, got %v", err)
	}

	// Check if profile is no longer active
	activeProfiles = profiler.GetActiveProfiles()
	if len(activeProfiles) != 0 {
		t.Errorf("Expected 0 active profiles, got %d", len(activeProfiles))
	}
}

func TestProfiler_MemoryProfiling(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultProfilingConfig()
	config.OutputDir = tempDir
	config.HTTPEndpoint = ""
	config.ProfileDuration = time.Millisecond * 100

	profiler := NewProfiler(config)
	ctx := context.Background()

	err := profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop(ctx)

	// Start memory profiling
	err = profiler.StartProfile(ProfileTypeMemory)
	if err != nil {
		t.Errorf("Expected no error starting memory profile, got %v", err)
	}

	// Allocate some memory to generate profiling data
	_ = make([]byte, 1024*1024) // 1MB allocation

	// Wait for profile duration
	time.Sleep(time.Millisecond * 150)

	// Check that profile file was created
	history := profiler.GetProfileHistory()
	if session, exists := history[ProfileTypeMemory]; exists {
		if session.FilePath == "" {
			t.Error("Expected profile file path to be set")
		}

		// Check if file exists
		if _, err := os.Stat(session.FilePath); os.IsNotExist(err) {
			t.Error("Expected profile file to exist")
		}
	} else {
		t.Error("Expected memory profile session in history")
	}
}

func TestProfiler_ProfileOperation(t *testing.T) {
	config := DefaultProfilingConfig()
	config.HTTPEndpoint = ""
	config.AutoProfile = false // Disable auto-profiling for this test

	profiler := NewProfiler(config)
	ctx := context.Background()

	err := profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop(ctx)

	// Profile a simple operation
	callCount := 0
	err = profiler.ProfileOperation(ctx, "test_operation", func() error {
		callCount++
		time.Sleep(time.Millisecond * 10) // Simulate some work
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error from profiled operation, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}

	stats := profiler.GetStats()
	if stats.OperationCount != 1 {
		t.Errorf("Expected operation count of 1, got %d", stats.OperationCount)
	}

	if stats.AverageLatency == 0 {
		t.Error("Expected average latency to be greater than 0")
	}
}

func TestProfiler_ProfileOperationWithError(t *testing.T) {
	config := DefaultProfilingConfig()
	config.HTTPEndpoint = ""
	config.AutoProfile = false

	profiler := NewProfiler(config)
	ctx := context.Background()

	err := profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop(ctx)

	// Profile an operation that returns an error
	testError := errors.New("test error")
	err = profiler.ProfileOperation(ctx, "failing_operation", func() error {
		return testError
	})

	if err != testError {
		t.Errorf("Expected operation error to be returned, got %v", err)
	}

	stats := profiler.GetStats()
	if stats.OperationCount != 1 {
		t.Errorf("Expected operation count of 1, got %d", stats.OperationCount)
	}
}

func TestProfiler_GetStats(t *testing.T) {
	config := DefaultProfilingConfig()
	config.HTTPEndpoint = ""

	profiler := NewProfiler(config)
	ctx := context.Background()

	stats := profiler.GetStats()
	if stats.IsRunning {
		t.Error("Expected profiler to not be running initially")
	}

	err := profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop(ctx)

	stats = profiler.GetStats()
	if !stats.IsRunning {
		t.Error("Expected profiler to be running after start")
	}

	if stats.ActiveProfiles != 0 {
		t.Errorf("Expected 0 active profiles initially, got %d", stats.ActiveProfiles)
	}

	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 total sessions initially, got %d", stats.TotalSessions)
	}
}

func TestCryptoProfiler_ProfileEncryption(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()
	operationName := "aes256_encrypt"
	dataSize := int64(1024)

	callCount := 0
	err := cryptoProfiler.ProfileEncryption(ctx, operationName, dataSize, func() error {
		callCount++
		// Simulate encryption work
		time.Sleep(time.Microsecond * 100)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Check metrics
	encryptionMetrics := cryptoProfiler.GetEncryptionMetrics()
	if len(encryptionMetrics) != 1 {
		t.Errorf("Expected 1 encryption metric, got %d", len(encryptionMetrics))
	}

	metrics, exists := encryptionMetrics[operationName]
	if !exists {
		t.Error("Expected to find encryption metrics for operation")
	}

	if metrics.TotalCalls != 1 {
		t.Errorf("Expected 1 total call, got %d", metrics.TotalCalls)
	}

	if metrics.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", metrics.ErrorCount)
	}
}

func TestCryptoProfiler_ProfileDecryption(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()
	operationName := "aes256_decrypt"
	dataSize := int64(1024)

	err := cryptoProfiler.ProfileDecryption(ctx, operationName, dataSize, func() error {
		// Simulate decryption work
		time.Sleep(time.Microsecond * 100)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check metrics
	decryptionMetrics := cryptoProfiler.GetDecryptionMetrics()
	if len(decryptionMetrics) != 1 {
		t.Errorf("Expected 1 decryption metric, got %d", len(decryptionMetrics))
	}

	metrics, exists := decryptionMetrics[operationName]
	if !exists {
		t.Error("Expected to find decryption metrics for operation")
	}

	if metrics.TotalCalls != 1 {
		t.Errorf("Expected 1 total call, got %d", metrics.TotalCalls)
	}
}

func TestCryptoProfiler_ProfileKeyOperation(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()
	operationName := "generate_dek"

	err := cryptoProfiler.ProfileKeyOperation(ctx, operationName, func() error {
		// Simulate key operation work
		time.Sleep(time.Millisecond * 10)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check metrics
	keyMetrics := cryptoProfiler.GetKeyMetrics()
	if len(keyMetrics) != 1 {
		t.Errorf("Expected 1 key metric, got %d", len(keyMetrics))
	}

	metrics, exists := keyMetrics[operationName]
	if !exists {
		t.Error("Expected to find key metrics for operation")
	}

	if metrics.TotalCalls != 1 {
		t.Errorf("Expected 1 total call, got %d", metrics.TotalCalls)
	}
}

func TestCryptoProfiler_ProfileHashing(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()
	operationName := "sha256_hash"
	dataSize := int64(512)

	err := cryptoProfiler.ProfileHashing(ctx, operationName, dataSize, func() error {
		// Simulate hashing work
		time.Sleep(time.Microsecond * 50)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check metrics
	hashingMetrics := cryptoProfiler.GetHashingMetrics()
	if len(hashingMetrics) != 1 {
		t.Errorf("Expected 1 hashing metric, got %d", len(hashingMetrics))
	}

	metrics, exists := hashingMetrics[operationName]
	if !exists {
		t.Error("Expected to find hashing metrics for operation")
	}

	if metrics.TotalCalls != 1 {
		t.Errorf("Expected 1 total call, got %d", metrics.TotalCalls)
	}
}

func TestCryptoProfiler_WithErrors(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()
	operationName := "failing_encrypt"
	testError := errors.New("encryption failed")

	err := cryptoProfiler.ProfileEncryption(ctx, operationName, 1024, func() error {
		return testError
	})

	if err != testError {
		t.Errorf("Expected operation error to be returned, got %v", err)
	}

	// Check that error was recorded in metrics
	encryptionMetrics := cryptoProfiler.GetEncryptionMetrics()
	metrics, exists := encryptionMetrics[operationName]
	if !exists {
		t.Error("Expected to find encryption metrics for failing operation")
	}

	if metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", metrics.ErrorCount)
	}

	if metrics.TotalCalls != 1 {
		t.Errorf("Expected 1 total call, got %d", metrics.TotalCalls)
	}
}

func TestCryptoProfiler_GetAllMetrics(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()

	// Execute operations in each category
	cryptoProfiler.ProfileEncryption(ctx, "test_encrypt", 1024, func() error { return nil })
	cryptoProfiler.ProfileDecryption(ctx, "test_decrypt", 1024, func() error { return nil })
	cryptoProfiler.ProfileKeyOperation(ctx, "test_key_gen", func() error { return nil })
	cryptoProfiler.ProfileHashing(ctx, "test_hash", 512, func() error { return nil })

	// Get all metrics
	report := cryptoProfiler.GetAllMetrics()

	if len(report.Encryption) != 1 {
		t.Errorf("Expected 1 encryption metric, got %d", len(report.Encryption))
	}

	if len(report.Decryption) != 1 {
		t.Errorf("Expected 1 decryption metric, got %d", len(report.Decryption))
	}

	if len(report.KeyOperations) != 1 {
		t.Errorf("Expected 1 key operation metric, got %d", len(report.KeyOperations))
	}

	if len(report.Hashing) != 1 {
		t.Errorf("Expected 1 hashing metric, got %d", len(report.Hashing))
	}

	if report.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestCryptoProfiler_GetTopSlowOperations(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()

	// Execute operations with different durations
	cryptoProfiler.ProfileEncryption(ctx, "fast_encrypt", 1024, func() error {
		time.Sleep(time.Microsecond * 10)
		return nil
	})

	cryptoProfiler.ProfileEncryption(ctx, "slow_encrypt", 1024, func() error {
		time.Sleep(time.Millisecond * 10)
		return nil
	})

	cryptoProfiler.ProfileDecryption(ctx, "medium_decrypt", 1024, func() error {
		time.Sleep(time.Microsecond * 100)
		return nil
	})

	// Get top slow operations
	slowOps := cryptoProfiler.GetTopSlowOperations(3)

	if len(slowOps) != 3 {
		t.Errorf("Expected 3 slow operations, got %d", len(slowOps))
	}

	// The slowest should be first (slow_encrypt)
	if slowOps[0].Name != "slow_encrypt" {
		t.Errorf("Expected slowest operation to be 'slow_encrypt', got %s", slowOps[0].Name)
	}

	// Check that they're sorted by duration (descending)
	if len(slowOps) >= 2 && slowOps[0].AverageDuration < slowOps[1].AverageDuration {
		t.Error("Expected operations to be sorted by average duration (descending)")
	}
}

func TestCryptoProfiler_GetMemoryIntensiveOperations(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()

	// Execute operations that allocate different amounts of memory
	cryptoProfiler.ProfileEncryption(ctx, "memory_encrypt", 1024, func() error {
		// Allocate some memory
		_ = make([]byte, 1024*10)
		runtime.GC() // Force GC to ensure memory stats are updated
		return nil
	})

	cryptoProfiler.ProfileDecryption(ctx, "simple_decrypt", 512, func() error {
		// Minimal memory allocation
		return nil
	})

	// Get memory intensive operations
	memOps := cryptoProfiler.GetMemoryIntensiveOperations(2)

	if len(memOps) != 2 {
		t.Errorf("Expected 2 memory operations, got %d", len(memOps))
	}

	// Check that the results contain expected operations
	found := false
	for _, op := range memOps {
		if op.Name == "memory_encrypt" || op.Name == "simple_decrypt" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find memory_encrypt or simple_decrypt operations")
	}
}

func TestCryptoProfiler_ResetMetrics(t *testing.T) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)

	ctx := context.Background()

	// Execute some operations
	cryptoProfiler.ProfileEncryption(ctx, "test_encrypt", 1024, func() error { return nil })
	cryptoProfiler.ProfileDecryption(ctx, "test_decrypt", 1024, func() error { return nil })

	// Verify metrics exist
	encMetrics := cryptoProfiler.GetEncryptionMetrics()
	decMetrics := cryptoProfiler.GetDecryptionMetrics()

	if len(encMetrics) == 0 {
		t.Error("Expected encryption metrics before reset")
	}
	if len(decMetrics) == 0 {
		t.Error("Expected decryption metrics before reset")
	}

	// Reset metrics
	cryptoProfiler.ResetMetrics()

	// Verify metrics are cleared
	encMetrics = cryptoProfiler.GetEncryptionMetrics()
	decMetrics = cryptoProfiler.GetDecryptionMetrics()

	if len(encMetrics) != 0 {
		t.Errorf("Expected no encryption metrics after reset, got %d", len(encMetrics))
	}
	if len(decMetrics) != 0 {
		t.Errorf("Expected no decryption metrics after reset, got %d", len(decMetrics))
	}
}

func TestGlobalProfiler(t *testing.T) {
	// Clear any existing global profiler
	globalProfiler = nil
	globalProfilerOnce = sync.Once{}

	profiler := GetGlobalProfiler()
	if profiler == nil {
		t.Error("Expected global profiler to be created")
	}

	// Test that subsequent calls return the same instance
	profiler2 := GetGlobalProfiler()
	if profiler != profiler2 {
		t.Error("Expected same global profiler instance")
	}
}

func TestGlobalCryptoProfiler(t *testing.T) {
	// Clear any existing global crypto profiler
	globalCryptoProfiler = nil
	globalCryptoProfilerOnce = sync.Once{}

	cryptoProfiler := GetGlobalCryptoProfiler()
	if cryptoProfiler == nil {
		t.Error("Expected global crypto profiler to be created")
	}

	// Test that subsequent calls return the same instance
	cryptoProfiler2 := GetGlobalCryptoProfiler()
	if cryptoProfiler != cryptoProfiler2 {
		t.Error("Expected same global crypto profiler instance")
	}
}

func TestGlobalProfilingFunctions(t *testing.T) {
	// Clear globals
	globalProfiler = nil
	globalProfilerOnce = sync.Once{}
	globalCryptoProfiler = nil
	globalCryptoProfilerOnce = sync.Once{}

	ctx := context.Background()

	// Test global crypto profiling functions
	err := ProfileGlobalEncryption(ctx, "global_encrypt", 1024, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error from global encryption profiling, got %v", err)
	}

	err = ProfileGlobalDecryption(ctx, "global_decrypt", 1024, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error from global decryption profiling, got %v", err)
	}

	err = ProfileGlobalKeyOperation(ctx, "global_key_gen", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error from global key operation profiling, got %v", err)
	}

	err = ProfileGlobalHashing(ctx, "global_hash", 512, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error from global hashing profiling, got %v", err)
	}

	// Verify that operations were recorded
	cryptoProfiler := GetGlobalCryptoProfiler()
	allMetrics := cryptoProfiler.GetAllMetrics()

	if len(allMetrics.Encryption) == 0 {
		t.Error("Expected global encryption metrics")
	}
	if len(allMetrics.Decryption) == 0 {
		t.Error("Expected global decryption metrics")
	}
	if len(allMetrics.KeyOperations) == 0 {
		t.Error("Expected global key operation metrics")
	}
	if len(allMetrics.Hashing) == 0 {
		t.Error("Expected global hashing metrics")
	}
}

// Benchmark tests
func BenchmarkCryptoProfiler_ProfileEncryption(b *testing.B) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cryptoProfiler.ProfileEncryption(ctx, "benchmark_encrypt", 1024, func() error {
			return nil
		})
	}
}

func BenchmarkCryptoProfiler_ProfileDecryption(b *testing.B) {
	profiler := NewProfiler(DefaultProfilingConfig())
	cryptoProfiler := NewCryptoProfiler(profiler)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cryptoProfiler.ProfileDecryption(ctx, "benchmark_decrypt", 1024, func() error {
			return nil
		})
	}
}

func BenchmarkProfiler_ProfileOperation(b *testing.B) {
	config := DefaultProfilingConfig()
	config.AutoProfile = false
	profiler := NewProfiler(config)
	ctx := context.Background()

	profiler.Start(ctx)
	defer profiler.Stop(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.ProfileOperation(ctx, "benchmark_operation", func() error {
			return nil
		})
	}
}