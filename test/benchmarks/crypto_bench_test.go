package benchmarks

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/hengadev/encx"
)

// Global variables to prevent compiler optimizations
var (
	result          []byte
	decryptedResult []byte
	err             error
)

// BenchmarkCryptoOperations benchmarks core cryptographic operations
func BenchmarkCryptoOperations(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	// Test data of various sizes
	testSizes := []int{
		16,     // 16 bytes - small data (UUID, short strings)
		64,     // 64 bytes - medium data (email, usernames)
		256,    // 256 bytes - larger data (addresses, descriptions)
		1024,   // 1KB - large data (documents, JSON objects)
		4096,   // 4KB - very large data (large JSON, small files)
		16384,  // 16KB - extra large data
	}

	for _, size := range testSizes {
		testData := make([]byte, size)
		rand.Read(testData)

		b.Run("GenerateDEK", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err = crypto.GenerateDEK()
			}
		})

		b.Run("EncryptData_"+sizeLabel(size), func(b *testing.B) {
			dek, _ := crypto.GenerateDEK()
			b.ResetTimer()
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				result, err = crypto.EncryptData(ctx, testData, dek)
			}
		})

		b.Run("DecryptData_"+sizeLabel(size), func(b *testing.B) {
			dek, _ := crypto.GenerateDEK()
			encrypted, _ := crypto.EncryptData(ctx, testData, dek)
			b.ResetTimer()
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				decryptedResult, err = crypto.DecryptData(ctx, encrypted, dek)
			}
		})

		b.Run("EncryptDEK", func(b *testing.B) {
			dek, _ := crypto.GenerateDEK()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err = crypto.EncryptDEK(ctx, dek)
			}
		})

		b.Run("DecryptDEK", func(b *testing.B) {
			dek, _ := crypto.GenerateDEK()
			encryptedDEK, _ := crypto.EncryptDEK(ctx, dek)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decryptedResult, err = crypto.DecryptDEKWithVersion(ctx, encryptedDEK, 1)
			}
		})
	}
}

// BenchmarkHashingOperations benchmarks hashing operations
func BenchmarkHashingOperations(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testSizes := []int{16, 64, 256, 1024}

	for _, size := range testSizes {
		testData := make([]byte, size)
		rand.Read(testData)

		b.Run("HashBasic_"+sizeLabel(size), func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				resultStr := crypto.HashBasic(ctx, testData)
				result = []byte(resultStr)
			}
		})

		b.Run("HashSecure_"+sizeLabel(size), func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				resultStr, err := crypto.HashSecure(ctx, testData)
				result = []byte(resultStr)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent crypto operations
func BenchmarkConcurrentOperations(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testData := make([]byte, 256)
	rand.Read(testData)
	dek, _ := crypto.GenerateDEK()

	b.Run("ConcurrentEncrypt", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := crypto.EncryptData(ctx, testData, dek)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	encrypted, _ := crypto.EncryptData(ctx, testData, dek)
	b.Run("ConcurrentDecrypt", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := crypto.DecryptData(ctx, encrypted, dek)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	b.Run("ConcurrentHashBasic", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = crypto.HashBasic(ctx, testData)
			}
		})
	})

	b.Run("ConcurrentHashSecure", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := crypto.HashSecure(ctx, testData)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})
}

// BenchmarkMemoryUsage benchmarks memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	testData := make([]byte, 1024)
	rand.Read(testData)

	b.Run("EncryptAllocations", func(b *testing.B) {
		dek, _ := crypto.GenerateDEK()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err = crypto.EncryptData(ctx, testData, dek)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("HashBasicAllocations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resultStr := crypto.HashBasic(ctx, testData)
			result = []byte(resultStr)
		}
	})

	b.Run("HashSecureAllocations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resultStr, err := crypto.HashSecure(ctx, testData)
			result = []byte(resultStr)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// BenchmarkThroughput measures data throughput for different operations
func BenchmarkThroughput(b *testing.B) {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(b)
	if err != nil {
		b.Fatalf("Failed to create test crypto: %v", err)
	}

	// Large data sizes for throughput testing
	sizes := []int{
		1024,     // 1KB
		10240,    // 10KB
		102400,   // 100KB
		1048576,  // 1MB
	}

	for _, size := range sizes {
		testData := make([]byte, size)
		rand.Read(testData)
		dek, _ := crypto.GenerateDEK()

		b.Run("EncryptThroughput_"+sizeLabel(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := crypto.EncryptData(ctx, testData, dek)
				if err != nil {
					b.Error(err)
				}
			}
		})

		encrypted, _ := crypto.EncryptData(ctx, testData, dek)
		b.Run("DecryptThroughput_"+sizeLabel(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := crypto.DecryptData(ctx, encrypted, dek)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

// sizeLabel converts byte size to human-readable label
func sizeLabel(size int) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%dB", size)
	case size < 1024*1024:
		return fmt.Sprintf("%dKB", size/1024)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%dMB", size/(1024*1024))
	default:
		return fmt.Sprintf("%dGB", size/(1024*1024*1024))
	}
}

// Helper to ensure we use the results (prevent optimization)
func init() {
	_ = result
	_ = decryptedResult
	_ = err
}