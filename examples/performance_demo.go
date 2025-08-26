package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hengadev/encx"
)

// Demo user struct
type User struct {
	Name             string `encx:"encrypt"`
	NameEncrypted    []byte
	Email            string `encx:"hash_basic"`
	EmailHash        string
	DEK              []byte
	DEKEncrypted     []byte
	KeyVersion       int
}

func main() {
	fmt.Println("=== Performance Optimizations Demo ===")
	fmt.Println("Demonstrating batch processing and performance improvements")
	fmt.Println()

	// Create crypto instance
	ctx := context.Background()
	crypto, err := encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("demo-kek-alias"),
		encx.WithPepper([]byte("demo-pepper-exactly-32-bytes-OK!")),
	)
	if err != nil {
		log.Fatalf("Failed to create crypto instance: %v", err)
	}

	// Demo 1: Show processing statistics
	fmt.Println("1. Processing Statistics:")
	stats := crypto.GetProcessingStats()
	for key, value := range stats {
		fmt.Printf("   %s: %v\n", key, value)
	}
	fmt.Println()

	// Demo 2: Batch size calculation
	fmt.Println("2. Optimal Batch Size Calculations:")
	testSizes := []int{50, 500, 5000, 50000}
	for _, size := range testSizes {
		batchSize := calculateOptimalBatchSize(size)
		fmt.Printf("   %d items → batch size: %d\n", size, batchSize)
	}
	fmt.Println()

	// Demo 3: Create test data for performance comparison
	fmt.Println("3. Performance Comparison (Sequential vs Batch):")
	itemCounts := []int{10, 50, 100}
	
	for _, count := range itemCounts {
		fmt.Printf("\n   Testing with %d items:\n", count)
		
		// Create test data for sequential processing
		sequentialUsers := make([]*User, count)
		for i := 0; i < count; i++ {
			sequentialUsers[i] = &User{
				Name:  fmt.Sprintf("SeqUser%d", i),
				Email: fmt.Sprintf("sequser%d@example.com", i),
			}
		}
		
		// Sequential processing
		start := time.Now()
		sequentialErrors := 0
		for _, user := range sequentialUsers {
			if err := crypto.ProcessStruct(ctx, user); err != nil {
				sequentialErrors++
			}
		}
		sequentialDuration := time.Since(start)
		
		// Create test data for batch processing
		batchUsers := make([]any, count)
		for i := 0; i < count; i++ {
			batchUsers[i] = &User{
				Name:  fmt.Sprintf("BatchUser%d", i),
				Email: fmt.Sprintf("batchuser%d@example.com", i),
			}
		}
		
		// Batch processing
		start = time.Now()
		result, err := crypto.ProcessStructsBatch(ctx, batchUsers, &encx.BatchProcessOptions{
			MaxConcurrency: 4,
			BatchSize:     calculateOptimalBatchSize(count),
		})
		batchDuration := time.Since(start)
		
		// Display results
		fmt.Printf("     Sequential: %v (%d errors)\n", sequentialDuration, sequentialErrors)
		
		if err != nil {
			fmt.Printf("     Batch: %v (failed: %v)\n", batchDuration, err)
		} else {
			fmt.Printf("     Batch: %v (processed: %d, failed: %d)\n", 
				batchDuration, result.Processed, result.Failed)
			
			if batchDuration < sequentialDuration {
				speedup := float64(sequentialDuration) / float64(batchDuration)
				fmt.Printf("     🚀 Batch was %.2fx faster!\n", speedup)
			} else if result.Failed == 0 {
				fmt.Printf("     ℹ️ Sequential faster (expected for small datasets)\n")
			}
		}
	}

	fmt.Println()
	fmt.Println("4. Batch Processing Options Demo:")
	
	// Create a small test dataset
	users := make([]any, 5)
	for i := 0; i < 5; i++ {
		users[i] = &User{
			Name:  fmt.Sprintf("ProgressUser%d", i),
			Email: fmt.Sprintf("progress%d@example.com", i),
		}
	}
	
	// Demo with progress callback
	fmt.Printf("   Processing %d items with progress tracking:\n", len(users))
	options := &encx.BatchProcessOptions{
		MaxConcurrency:   2,
		BatchSize:        2,
		StopOnFirstError: false,
		EnableProgress:   true,
		ProgressCallback: func(processed, total int, item any, err error) {
			if err != nil {
				fmt.Printf("     [%d/%d] ❌ Error: %v\n", processed, total, err)
			} else {
				fmt.Printf("     [%d/%d] ✅ Processed successfully\n", processed, total)
			}
		},
	}
	
	result, err := crypto.ProcessStructsBatch(ctx, users, options)
	if err != nil {
		fmt.Printf("   Final result: Error - %v\n", err)
	} else {
		fmt.Printf("   Final result: %d processed, %d failed in %s\n", 
			result.Processed, result.Failed, result.Duration)
	}

	fmt.Println()
	fmt.Println("=== Performance Benefits ===")
	fmt.Println("✅ Concurrent processing utilizes multiple CPU cores")
	fmt.Println("✅ Batch processing reduces setup/teardown overhead")
	fmt.Println("✅ Configurable concurrency limits prevent resource exhaustion")
	fmt.Println("✅ Progress tracking for long-running operations")
	fmt.Println("✅ Error handling options (stop on first vs collect all)")
	fmt.Println("✅ Automatic memory management and garbage collection")
	fmt.Println("✅ Context cancellation support for graceful shutdown")
}

// Helper function (normally this would be internal)
func calculateOptimalBatchSize(totalItems int) int {
	if totalItems <= 100 {
		return totalItems
	}
	
	// Use a simple calculation for demo
	batchSize := totalItems / 10
	if batchSize < 10 {
		batchSize = 10
	} else if batchSize > 1000 {
		batchSize = 1000
	}
	
	return batchSize
}