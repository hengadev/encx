package benchmarks

import (
	"context"
	"crypto/rand"
	"runtime"
	"testing"

	"github.com/hengadev/encx"
)

// BenchmarkMemoryAllocations measures memory allocations for different operations
func BenchmarkMemoryAllocations(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testData := make([]byte, 1024)
	rand.Read(testData)

	b.Run("EncryptDataAllocations", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.EncryptData(ctx, testData, dek)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("DecryptDataAllocations", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		encrypted, _ := crypto.EncryptData(ctx, testData, dek)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.DecryptData(ctx, encrypted, dek)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("HashBasicAllocations", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = crypto.HashBasic(ctx, testData)
		}
	})

	b.Run("HashSecureAllocations", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.HashSecure(ctx, testData)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("GenerateDEKAllocations", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.GenerateDEK()
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("EncryptDEKAllocations", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.EncryptDEK(ctx, dek)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("DecryptDEKAllocations", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		encryptedDEK, _ := crypto.EncryptDEK(ctx, dek)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.DecryptDEKWithVersion(ctx, encryptedDEK, 1)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// BenchmarkMemoryPressure tests performance under memory pressure
func BenchmarkMemoryPressure(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	// Create large test data to simulate memory pressure
	largeData := make([]byte, 1024*1024) // 1MB
	rand.Read(largeData)

	b.Run("LargeDataEncryption", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.EncryptData(ctx, largeData, dek)
			if err != nil {
				b.Error(err)
			}
			// Force GC to simulate memory pressure
			if i%10 == 0 {
				runtime.GC()
			}
		}
	})

	b.Run("LargeDataDecryption", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		encrypted, _ := crypto.EncryptData(ctx, largeData, dek)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.DecryptData(ctx, encrypted, dek)
			if err != nil {
				b.Error(err)
			}
			// Force GC to simulate memory pressure
			if i%10 == 0 {
				runtime.GC()
			}
		}
	})
}

// BenchmarkMemoryGrowth measures memory growth over many operations
func BenchmarkMemoryGrowth(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testData := make([]byte, 256)
	rand.Read(testData)

	b.Run("EncryptionMemoryGrowth", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := crypto.EncryptData(ctx, testData, dek)
			if err != nil {
				b.Error(err)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Report memory growth
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})

	b.Run("HashingMemoryGrowth", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = crypto.HashBasic(ctx, testData)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Report memory growth
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})
}

// BenchmarkSerializationMemory benchmarks serialization memory usage
func BenchmarkSerializationMemory(b *testing.B) {
	// Test different data types to see serialization memory overhead
	testCases := map[string]interface{}{
		"string":    "test string value for serialization benchmark",
		"int64":     int64(1234567890),
		"bool":      true,
		"bytes":     make([]byte, 256),
	}

	for name, value := range testCases {
		b.Run("Serialize_"+name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Note: Direct access to serialization would require importing internal package
				// This is a placeholder for when serialization is exposed or testable
				_ = value
			}
		})
	}
}

// BenchmarkConcurrentMemoryUsage tests memory usage under concurrent access
func BenchmarkConcurrentMemoryUsage(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testData := make([]byte, 512)
	rand.Read(testData)
	dek, _ := crypto.GenerateDEK()

	b.Run("ConcurrentEncryptMemory", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := crypto.EncryptData(ctx, testData, dek)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	b.Run("ConcurrentHashMemory", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = crypto.HashBasic(ctx, testData)
			}
		})
	})
}

// BenchmarkMemoryLeaks attempts to detect potential memory leaks
func BenchmarkMemoryLeaks(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory leak test in short mode")
	}

	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testData := make([]byte, 1024)
	rand.Read(testData)

	b.Run("DetectMemoryLeaks", func(b *testing.B) {
		// Baseline memory measurement
		runtime.GC()
		var baseline runtime.MemStats
		runtime.ReadMemStats(&baseline)

		// Perform many operations
		dek, _ := crypto.GenerateDEK()
		for i := 0; i < 1000; i++ {
			encrypted, err := crypto.EncryptData(ctx, testData, dek)
			if err != nil {
				b.Error(err)
				continue
			}

			_, err = crypto.DecryptData(ctx, encrypted, dek)
			if err != nil {
				b.Error(err)
			}

			_ = crypto.HashBasic(ctx, testData)
		}

		// Force GC and measure again
		runtime.GC()
		runtime.GC() // Call twice to ensure cleanup
		var final runtime.MemStats
		runtime.ReadMemStats(&final)

		memoryGrowth := final.Alloc - baseline.Alloc
		if memoryGrowth > 1024*1024 { // More than 1MB growth
			b.Logf("WARNING: Potential memory leak detected. Memory growth: %d bytes", memoryGrowth)
		}

		b.ReportMetric(float64(memoryGrowth), "bytes_leaked")
	})
}