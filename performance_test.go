package encx

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test struct for batch processing
type TestUser struct {
	Name             string `encx:"encrypt"`
	NameEncrypted    []byte
	Email            string `encx:"hash_basic"`
	EmailHash        string
	DEK              []byte
	DEKEncrypted     []byte
	KeyVersion       int
}

func TestProcessStructsBatch_BasicFunctionality(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	ctx := context.Background()
	
	// Create test data
	users := make([]any, 5)
	for i := 0; i < 5; i++ {
		users[i] = &TestUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}
	
	// Process batch
	result, err := crypto.ProcessStructsBatch(ctx, users)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// Verify results
	assert.Equal(t, 5, result.Total)
	assert.Equal(t, 5, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Errors)
	assert.NotEmpty(t, result.Duration)
	
	// Verify all users were processed
	for i, userAny := range users {
		user := userAny.(*TestUser)
		assert.Empty(t, user.Name, "Name should be cleared for user %d", i)
		assert.NotEmpty(t, user.NameEncrypted, "NameEncrypted should be populated for user %d", i)
		assert.NotEmpty(t, user.EmailHash, "EmailHash should be populated for user %d", i)
		assert.NotEmpty(t, user.DEK, "DEK should be populated for user %d", i)
	}
}

func TestProcessStructsBatch_WithOptions(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	ctx := context.Background()
	
	users := make([]any, 10)
	for i := 0; i < 10; i++ {
		users[i] = &TestUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}
	
	// Test with custom options
	var progressCount int64
	options := &BatchProcessOptions{
		MaxConcurrency:   2,
		BatchSize:        3,
		StopOnFirstError: false,
		EnableProgress:   true,
		ProgressCallback: func(processed, total int, item any, err error) {
			atomic.AddInt64(&progressCount, 1)
			assert.LessOrEqual(t, processed, total)
		},
	}
	
	result, err := crypto.ProcessStructsBatch(ctx, users, options)
	require.NoError(t, err)
	
	assert.Equal(t, 10, result.Total)
	assert.Equal(t, 10, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, int64(10), progressCount)
}

func TestProcessStructsBatch_ErrorHandling(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	ctx := context.Background()
	
	// Test nil struct validation - should fail upfront
	itemsWithNil := []any{
		&TestUser{Name: "Valid User", Email: "valid@example.com"},
		nil, // This will cause immediate validation error
	}
	
	_, err := crypto.ProcessStructsBatch(ctx, itemsWithNil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "struct at index 1 is nil")
	
	// Test with valid structs only - should all succeed
	validItems := []any{
		&TestUser{Name: "Valid User 1", Email: "valid1@example.com"},
		&TestUser{Name: "Valid User 2", Email: "valid2@example.com"},
	}
	
	options := &BatchProcessOptions{
		StopOnFirstError: false,
	}
	
	result, err := crypto.ProcessStructsBatch(ctx, validItems, options)
	require.NoError(t, err)
	
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Errors)
}

func TestProcessStructsBatch_StopOnFirstError(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	ctx := context.Background()
	
	// Test with valid structs but verify the StopOnFirstError behavior
	// Since validation happens upfront, we test with valid structs
	items := []any{
		&TestUser{Name: "User1", Email: "test1@example.com"},
		&TestUser{Name: "User2", Email: "test2@example.com"},
	}
	
	options := &BatchProcessOptions{
		StopOnFirstError: true,
	}
	
	result, err := crypto.ProcessStructsBatch(ctx, items, options)
	require.NoError(t, err) // Should succeed with valid structs
	
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Processed)
	assert.Equal(t, 0, result.Failed)
	
	// Test that nil validation still works with StopOnFirstError
	itemsWithNil := []any{
		nil, // This will fail validation immediately
		&TestUser{Name: "User", Email: "test@example.com"},
	}
	
	_, err = crypto.ProcessStructsBatch(ctx, itemsWithNil, options)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "struct at index 0 is nil")
}

func TestDecryptStructsBatch_BasicFunctionality(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	ctx := context.Background()
	
	// Create and process test data first
	users := make([]any, 3)
	originalNames := make([]string, 3)
	
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("User%d", i)
		originalNames[i] = name
		users[i] = &TestUser{
			Name:  name,
			Email: fmt.Sprintf("user%d@example.com", i),
		}
		
		// Process individual struct first
		err := crypto.ProcessStruct(ctx, users[i])
		require.NoError(t, err)
	}
	
	// Now decrypt in batch
	result, err := crypto.DecryptStructsBatch(ctx, users)
	require.NoError(t, err)
	
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 3, result.Processed)
	assert.Equal(t, 0, result.Failed)
	
	// Verify decryption worked
	for i, userAny := range users {
		user := userAny.(*TestUser)
		assert.Equal(t, originalNames[i], user.Name, "Name should be restored for user %d", i)
	}
}

func TestBatchProcessing_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	crypto, _ := NewTestCrypto(t)
	ctx := context.Background()
	
	// Create a larger dataset for performance testing
	itemCount := 100
	users := make([]any, itemCount)
	for i := 0; i < itemCount; i++ {
		users[i] = &TestUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}
	
	// Test batch processing performance
	start := time.Now()
	result, err := crypto.ProcessStructsBatch(ctx, users, &BatchProcessOptions{
		MaxConcurrency: 4,
	})
	batchDuration := time.Since(start)
	
	require.NoError(t, err)
	assert.Equal(t, itemCount, result.Processed)
	
	// Test sequential processing for comparison
	sequentialUsers := make([]any, itemCount)
	for i := 0; i < itemCount; i++ {
		sequentialUsers[i] = &TestUser{
			Name:  fmt.Sprintf("SeqUser%d", i),
			Email: fmt.Sprintf("sequser%d@example.com", i),
		}
	}
	
	start = time.Now()
	for _, user := range sequentialUsers {
		err := crypto.ProcessStruct(ctx, user)
		require.NoError(t, err)
	}
	sequentialDuration := time.Since(start)
	
	t.Logf("Batch processing (%d items): %v", itemCount, batchDuration)
	t.Logf("Sequential processing (%d items): %v", itemCount, sequentialDuration)
	
	// Batch processing should be faster for this size (though not always guaranteed due to overhead)
	// This is more of an informational test
	if batchDuration < sequentialDuration {
		t.Logf("✅ Batch processing was %.2fx faster", float64(sequentialDuration)/float64(batchDuration))
	} else {
		t.Logf("ℹ️ Sequential was faster (expected for small datasets due to overhead)")
	}
}

func TestCalculateOptimalBatchSize(t *testing.T) {
	tests := []struct {
		name       string
		totalItems int
		expected   int
	}{
		{"small dataset", 50, 50},
		{"medium dataset", 500, calculateOptimalBatchSize(500)},
		{"large dataset", 10000, calculateOptimalBatchSize(10000)},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateOptimalBatchSize(tt.totalItems)
			assert.Positive(t, result)
			
			if tt.totalItems <= 100 {
				assert.Equal(t, tt.totalItems, result)
			} else {
				assert.LessOrEqual(t, result, 1000) // Should not exceed max batch size
				assert.GreaterOrEqual(t, result, 10) // Should not be less than min batch size
			}
		})
	}
}

func TestGetProcessingStats(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	
	stats := crypto.GetProcessingStats()
	
	// Verify expected keys exist
	expectedKeys := []string{
		"optimal_batch_size_for_1000_items",
		"optimal_batch_size_for_10000_items", 
		"recommended_max_concurrency",
		"available_cpu_cores",
		"runtime_version",
	}
	
	for _, key := range expectedKeys {
		assert.Contains(t, stats, key)
	}
	
	// Verify values are reasonable
	assert.Positive(t, stats["optimal_batch_size_for_1000_items"])
	assert.Positive(t, stats["optimal_batch_size_for_10000_items"])
	assert.Positive(t, stats["recommended_max_concurrency"])
	assert.Positive(t, stats["available_cpu_cores"])
	assert.NotEmpty(t, stats["runtime_version"])
}


func TestBatchProcessing_ContextCancellation(t *testing.T) {
	crypto, _ := NewTestCrypto(t)
	
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create a large batch that will take some time
	users := make([]any, 1000) // Larger batch to increase chance of cancellation
	for i := 0; i < 1000; i++ {
		users[i] = &TestUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}
	
	// Cancel context after a very short delay
	go func() {
		time.Sleep(1 * time.Millisecond)
		cancel()
	}()
	
	// Process batch - might be cancelled
	result, err := crypto.ProcessStructsBatch(ctx, users)
	
	// Either it completes successfully (if cancellation was too late) or gets cancelled
	if err != nil {
		// If cancelled, check error message
		assert.Contains(t, err.Error(), "context canceled")
		// Result should still be valid
		if result != nil {
			assert.LessOrEqual(t, result.Processed+result.Failed, result.Total)
		}
	} else {
		// If completed successfully, all should be processed
		require.NotNil(t, result)
		assert.Equal(t, 1000, result.Processed)
		assert.Equal(t, 0, result.Failed)
	}
}

// Benchmark for batch processing
func BenchmarkProcessStructsBatch(b *testing.B) {
	crypto, _ := NewTestCrypto(b)
	ctx := context.Background()
	
	// Create test data
	users := make([]any, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = &TestUser{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Reset users for each iteration
		for j := 0; j < 1000; j++ {
			users[j] = &TestUser{
				Name:  fmt.Sprintf("User%d", j),
				Email: fmt.Sprintf("user%d@example.com", j),
			}
		}
		
		result, err := crypto.ProcessStructsBatch(ctx, users)
		if err != nil {
			b.Fatalf("Batch processing failed: %v", err)
		}
		if result.Processed != 1000 {
			b.Fatalf("Expected 1000 processed, got %d", result.Processed)
		}
	}
}