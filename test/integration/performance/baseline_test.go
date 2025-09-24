package performance_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/hengadev/encx"
	"github.com/hengadev/encx/internal/monitoring"
)

// PerformanceBaselineTestSuite establishes performance baselines for ENCX operations
type PerformanceBaselineTestSuite struct {
	suite.Suite
	crypto         *encx.Crypto
	ctx            context.Context
	metricsCollector *monitoring.InMemoryMetricsCollector
}

// SetupSuite initializes test environment
func (suite *PerformanceBaselineTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create metrics collector for performance tracking
	suite.metricsCollector = monitoring.NewInMemoryMetricsCollector()

	// Create crypto instance with performance monitoring
	var err error
	suite.crypto, err = encx.NewCrypto(suite.ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("performance-test-key"),
		encx.WithPepper([]byte("performance-test-pepper-32-bytes")),
		encx.WithMetricsCollector(suite.metricsCollector),
	)
	require.NoError(suite.T(), err)

	// Warm up the system
	suite.warmUpSystem()
}

// warmUpSystem performs initial operations to warm up caches and connections
func (suite *PerformanceBaselineTestSuite) warmUpSystem() {
	for i := 0; i < 10; i++ {
		dek, _ := suite.crypto.GenerateDEK(suite.ctx)
		testData := []byte("warmup data")
		encrypted, _ := suite.crypto.EncryptData(suite.ctx, testData, dek)
		suite.crypto.DecryptData(suite.ctx, encrypted, dek)
	}
}

// TestDEKGenerationBaseline establishes baseline for DEK generation
func (suite *PerformanceBaselineTestSuite) TestDEKGenerationBaseline() {
	const iterations = 1000
	const expectedMaxLatency = 10 * time.Millisecond // 10ms max per DEK generation

	start := time.Now()

	for i := 0; i < iterations; i++ {
		dek, err := suite.crypto.GenerateDEK(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), dek, 32, "DEK should be 32 bytes")
	}

	duration := time.Since(start)
	avgLatency := duration / iterations

	suite.T().Logf("DEK Generation Baseline:")
	suite.T().Logf("  Total operations: %d", iterations)
	suite.T().Logf("  Total duration: %v", duration)
	suite.T().Logf("  Average latency: %v", avgLatency)
	suite.T().Logf("  Operations/sec: %.0f", float64(iterations)/duration.Seconds())

	// Performance assertions
	assert.Less(suite.T(), avgLatency, expectedMaxLatency,
		"Average DEK generation latency should be under %v", expectedMaxLatency)

	// Throughput should be at least 100 ops/sec
	opsPerSec := float64(iterations) / duration.Seconds()
	assert.Greater(suite.T(), opsPerSec, 100.0, "Should achieve at least 100 DEK generations per second")
}

// TestDataEncryptionBaseline establishes baseline for data encryption
func (suite *PerformanceBaselineTestSuite) TestDataEncryptionBaseline() {
	const iterations = 1000
	const expectedMaxLatency = 5 * time.Millisecond // 5ms max per encryption

	// Generate DEK once
	dek, err := suite.crypto.GenerateDEK(suite.ctx)
	require.NoError(suite.T(), err)

	// Test with different data sizes
	testSizes := []int{16, 64, 256, 1024, 4096} // bytes

	for _, size := range testSizes {
		suite.T().Run(fmt.Sprintf("Size_%d_bytes", size), func(t *testing.T) {
			testData := make([]byte, size)
			_, err := rand.Read(testData)
			require.NoError(t, err)

			start := time.Now()

			for i := 0; i < iterations; i++ {
				encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
				require.NoError(t, err)
				assert.Greater(t, len(encrypted), size, "Encrypted data should be larger than input")
			}

			duration := time.Since(start)
			avgLatency := duration / iterations
			throughputMBps := float64(size*iterations) / (1024*1024) / duration.Seconds()

			t.Logf("Encryption Baseline (%d bytes):", size)
			t.Logf("  Total operations: %d", iterations)
			t.Logf("  Average latency: %v", avgLatency)
			t.Logf("  Throughput: %.2f MB/s", throughputMBps)
			t.Logf("  Operations/sec: %.0f", float64(iterations)/duration.Seconds())

			// Performance assertions
			assert.Less(t, avgLatency, expectedMaxLatency,
				"Encryption latency for %d bytes should be under %v", size, expectedMaxLatency)
		})
	}
}

// TestDataDecryptionBaseline establishes baseline for data decryption
func (suite *PerformanceBaselineTestSuite) TestDataDecryptionBaseline() {
	const iterations = 1000

	// Generate DEK and test data
	dek, err := suite.crypto.GenerateDEK(suite.ctx)
	require.NoError(suite.T(), err)

	testSizes := []int{16, 64, 256, 1024, 4096}

	for _, size := range testSizes {
		suite.T().Run(fmt.Sprintf("Size_%d_bytes", size), func(t *testing.T) {
			testData := make([]byte, size)
			_, err := rand.Read(testData)
			require.NoError(t, err)

			// Pre-encrypt data
			encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
			require.NoError(t, err)

			start := time.Now()

			for i := 0; i < iterations; i++ {
				decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
				require.NoError(t, err)
				assert.Equal(t, testData, decrypted)
			}

			duration := time.Since(start)
			avgLatency := duration / iterations
			throughputMBps := float64(size*iterations) / (1024*1024) / duration.Seconds()

			t.Logf("Decryption Baseline (%d bytes):", size)
			t.Logf("  Average latency: %v", avgLatency)
			t.Logf("  Throughput: %.2f MB/s", throughputMBps)
		})
	}
}

// TestHashingBaseline establishes baseline for hashing operations
func (suite *PerformanceBaselineTestSuite) TestHashingBaseline() {
	const iterations = 10000 // More iterations since hashing is faster
	const expectedMaxLatency = 1 * time.Millisecond

	testData := "test@example.com"

	// Test basic hash for searching
	start := time.Now()
	for i := 0; i < iterations; i++ {
		hash, err := suite.crypto.HashForSearch(suite.ctx, fmt.Sprintf("%s%d", testData, i))
		require.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), hash)
	}
	searchHashDuration := time.Since(start)

	// Test secure hash
	start = time.Now()
	for i := 0; i < iterations; i++ {
		hash, err := suite.crypto.HashSecure(suite.ctx, fmt.Sprintf("%s%d", testData, i))
		require.NoError(suite.T(), err)
		assert.NotEmpty(suite.T(), hash)
	}
	secureHashDuration := time.Since(start)

	suite.T().Logf("Hash Baseline:")
	suite.T().Logf("  Search hash avg latency: %v", searchHashDuration/iterations)
	suite.T().Logf("  Secure hash avg latency: %v", secureHashDuration/iterations)
	suite.T().Logf("  Search hash ops/sec: %.0f", float64(iterations)/searchHashDuration.Seconds())
	suite.T().Logf("  Secure hash ops/sec: %.0f", float64(iterations)/secureHashDuration.Seconds())

	// Performance assertions
	assert.Less(suite.T(), searchHashDuration/iterations, expectedMaxLatency,
		"Search hash latency should be under %v", expectedMaxLatency)
	assert.Less(suite.T(), secureHashDuration/iterations, expectedMaxLatency*5, // Secure hash can be slower
		"Secure hash latency should be under %v", expectedMaxLatency*5)
}

// TestConcurrentPerformance tests performance under concurrent load
func (suite *PerformanceBaselineTestSuite) TestConcurrentPerformance() {
	const numGoroutines = runtime.NumCPU()
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	results := make(chan time.Duration, numGoroutines*operationsPerGoroutine)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				opStart := time.Now()

				// Perform complete encrypt/decrypt cycle
				dek, err := suite.crypto.GenerateDEK(suite.ctx)
				if err != nil {
					suite.T().Errorf("DEK generation failed: %v", err)
					continue
				}

				testData := []byte(fmt.Sprintf("concurrent-test-g%d-op%d", goroutineID, j))
				encrypted, err := suite.crypto.EncryptData(suite.ctx, testData, dek)
				if err != nil {
					suite.T().Errorf("Encryption failed: %v", err)
					continue
				}

				decrypted, err := suite.crypto.DecryptData(suite.ctx, encrypted, dek)
				if err != nil {
					suite.T().Errorf("Decryption failed: %v", err)
					continue
				}

				if string(decrypted) != string(testData) {
					suite.T().Errorf("Data mismatch in concurrent test")
					continue
				}

				results <- time.Since(opStart)
			}
		}(i)
	}

	wg.Wait()
	close(results)

	totalDuration := time.Since(start)
	totalOps := numGoroutines * operationsPerGoroutine

	// Collect latency statistics
	var latencies []time.Duration
	for latency := range results {
		latencies = append(latencies, latency)
	}

	// Calculate statistics
	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency = time.Hour // Start with a very high value

	for _, lat := range latencies {
		totalLatency += lat
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}

	avgLatency := totalLatency / time.Duration(len(latencies))
	throughput := float64(totalOps) / totalDuration.Seconds()

	suite.T().Logf("Concurrent Performance Baseline:")
	suite.T().Logf("  Goroutines: %d", numGoroutines)
	suite.T().Logf("  Operations per goroutine: %d", operationsPerGoroutine)
	suite.T().Logf("  Total operations: %d", totalOps)
	suite.T().Logf("  Total duration: %v", totalDuration)
	suite.T().Logf("  Average latency: %v", avgLatency)
	suite.T().Logf("  Min latency: %v", minLatency)
	suite.T().Logf("  Max latency: %v", maxLatency)
	suite.T().Logf("  Operations/sec: %.0f", throughput)

	// Performance assertions for concurrent workload
	assert.Greater(suite.T(), throughput, 50.0, "Should achieve at least 50 ops/sec under concurrent load")
	assert.Less(suite.T(), avgLatency, 100*time.Millisecond, "Average latency should be under 100ms")
	assert.Less(suite.T(), maxLatency, 500*time.Millisecond, "Max latency should be under 500ms")
}

// TestMemoryUsageBaseline measures memory usage patterns
func (suite *PerformanceBaselineTestSuite) TestMemoryUsageBaseline() {
	const iterations = 1000

	// Force garbage collection before test
	runtime.GC()

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform operations
	dek, _ := suite.crypto.GenerateDEK(suite.ctx)
	testData := make([]byte, 1024) // 1KB test data

	for i := 0; i < iterations; i++ {
		encrypted, _ := suite.crypto.EncryptData(suite.ctx, testData, dek)
		suite.crypto.DecryptData(suite.ctx, encrypted, dek)
	}

	// Force garbage collection after test
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocsDelta := m2.TotalAlloc - m1.TotalAlloc
	allocsPerOp := allocsDelta / iterations

	suite.T().Logf("Memory Usage Baseline:")
	suite.T().Logf("  Total memory allocated: %d bytes", allocsDelta)
	suite.T().Logf("  Memory per operation: %d bytes", allocsPerOp)
	suite.T().Logf("  Heap objects: %d", m2.HeapObjects)
	suite.T().Logf("  GC cycles: %d", m2.NumGC-m1.NumGC)

	// Memory usage assertions (these may need tuning based on actual usage)
	assert.Less(suite.T(), allocsPerOp, uint64(10*1024), "Memory per operation should be under 10KB")
}

// TestLoadTestBaseline runs a sustained load test
func (suite *PerformanceBaselineTestSuite) TestLoadTestBaseline() {
	const duration = 30 * time.Second
	const targetRPS = 100 // requests per second

	ctx, cancel := context.WithTimeout(suite.ctx, duration)
	defer cancel()

	var operationCount int64
	var errorCount int64
	var wg sync.WaitGroup

	// Launch multiple workers
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ticker := time.NewTicker(time.Second / time.Duration(targetRPS/numWorkers))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Perform operation
					dek, err := suite.crypto.GenerateDEK(ctx)
					if err != nil {
						errorCount++
						continue
					}

					testData := []byte(fmt.Sprintf("load-test-worker-%d-%d", workerID, operationCount))
					encrypted, err := suite.crypto.EncryptData(ctx, testData, dek)
					if err != nil {
						errorCount++
						continue
					}

					_, err = suite.crypto.DecryptData(ctx, encrypted, dek)
					if err != nil {
						errorCount++
						continue
					}

					operationCount++
				}
			}
		}(i)
	}

	wg.Wait()

	actualRPS := float64(operationCount) / duration.Seconds()
	errorRate := float64(errorCount) / float64(operationCount+errorCount) * 100

	suite.T().Logf("Load Test Baseline:")
	suite.T().Logf("  Duration: %v", duration)
	suite.T().Logf("  Target RPS: %d", targetRPS)
	suite.T().Logf("  Actual RPS: %.0f", actualRPS)
	suite.T().Logf("  Total operations: %d", operationCount)
	suite.T().Logf("  Error count: %d", errorCount)
	suite.T().Logf("  Error rate: %.2f%%", errorRate)

	// Performance assertions for sustained load
	assert.Greater(suite.T(), actualRPS, float64(targetRPS)*0.8, "Should achieve at least 80%% of target RPS")
	assert.Less(suite.T(), errorRate, 1.0, "Error rate should be under 1%%")
}

// TestSuite entry point
func TestPerformanceBaselineSuite(t *testing.T) {
	suite.Run(t, new(PerformanceBaselineTestSuite))
}