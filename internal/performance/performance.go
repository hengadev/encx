package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// StructProcessor defines the interface for struct processing operations
type StructProcessor interface {
	ProcessStruct(ctx context.Context, object any) error
	DecryptStruct(ctx context.Context, object any) error
}

// BatchProcessOptions configures batch processing behavior
type BatchProcessOptions struct {
	// MaxConcurrency limits the number of concurrent operations (0 = number of CPUs)
	MaxConcurrency int

	// BatchSize is the number of items to process in each batch (0 = auto-calculate)
	BatchSize int

	// StopOnFirstError determines if processing should stop on the first error
	StopOnFirstError bool

	// EnableProgress enables progress tracking and callbacks
	EnableProgress bool

	// ProgressCallback is called after each item is processed
	ProgressCallback func(processed, total int, item any, err error)
}

// BatchProcessResult contains the results of batch processing
type BatchProcessResult struct {
	// Processed is the number of successfully processed items
	Processed int

	// Failed is the number of failed items
	Failed int

	// Total is the total number of items
	Total int

	// Errors contains all errors encountered (if StopOnFirstError is false)
	Errors []BatchError

	// Duration is the total processing time
	Duration string
}

// BatchError represents an error that occurred during batch processing
type BatchError struct {
	Index int   // Index of the item that failed
	Item  any   // The item that failed to process
	Error error // The error that occurred
}

// ProcessStructsBatch processes multiple structs concurrently with optimized batching
func ProcessStructsBatch(ctx context.Context, processor StructProcessor, structs []any, options ...*BatchProcessOptions) (*BatchProcessResult, error) {
	if len(structs) == 0 {
		return &BatchProcessResult{Total: 0}, nil
	}

	// Apply options
	opts := &BatchProcessOptions{
		MaxConcurrency:   runtime.NumCPU(),
		BatchSize:        calculateOptimalBatchSize(len(structs)),
		StopOnFirstError: false,
		EnableProgress:   false,
	}
	if len(options) > 0 && options[0] != nil {
		if options[0].MaxConcurrency > 0 {
			opts.MaxConcurrency = options[0].MaxConcurrency
		}
		if options[0].BatchSize > 0 {
			opts.BatchSize = options[0].BatchSize
		}
		opts.StopOnFirstError = options[0].StopOnFirstError
		opts.EnableProgress = options[0].EnableProgress
		opts.ProgressCallback = options[0].ProgressCallback
	}

	// Validate inputs
	for i, s := range structs {
		if s == nil {
			return nil, fmt.Errorf("struct at index %d is nil", i)
		}
	}

	start := time.Now().UnixNano()
	defer func() {
		// Force GC after batch processing to clean up temporary allocations
		runtime.GC()
	}()

	result := &BatchProcessResult{
		Total:  len(structs),
		Errors: make([]BatchError, 0),
	}

	// Use semaphore pattern to limit concurrency
	semaphore := make(chan struct{}, opts.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Context for cancellation
	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Process items in batches
	for batchStart := 0; batchStart < len(structs); batchStart += opts.BatchSize {
		batchEnd := batchStart + opts.BatchSize
		if batchEnd > len(structs) {
			batchEnd = len(structs)
		}

		batch := structs[batchStart:batchEnd]

		// Process each item in the batch
		for i, item := range batch {
			select {
			case <-batchCtx.Done():
				// Context cancelled, stop processing
				wg.Wait()
				return result, batchCtx.Err()
			default:
			}

			wg.Add(1)
			go func(itemIndex int, structItem any) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Process the struct
				err := processor.ProcessStruct(batchCtx, structItem)

				// Update results
				mu.Lock()
				globalIndex := batchStart + itemIndex

				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, BatchError{
						Index: globalIndex,
						Item:  structItem,
						Error: err,
					})

					if opts.StopOnFirstError {
						cancel() // Cancel remaining operations
					}
				} else {
					result.Processed++
				}

				// Progress callback
				if opts.EnableProgress && opts.ProgressCallback != nil {
					processedCount := result.Processed + result.Failed
					opts.ProgressCallback(processedCount, result.Total, structItem, err)
				}

				mu.Unlock()
			}(i, item)
		}

		// Check for early termination
		if opts.StopOnFirstError {
			select {
			case <-batchCtx.Done():
				break
			default:
			}
		}
	}

	// Wait for all operations to complete
	wg.Wait()

	end := time.Now().UnixNano()
	result.Duration = fmt.Sprintf("%.2fms", float64(end-start)/1e6)

	// Return error if stop on first error and we have errors
	if opts.StopOnFirstError && len(result.Errors) > 0 {
		return result, result.Errors[0].Error
	}

	return result, nil
}

// DecryptStructsBatch decrypts multiple structs concurrently
func DecryptStructsBatch(ctx context.Context, processor StructProcessor, structs []any, options ...*BatchProcessOptions) (*BatchProcessResult, error) {
	if len(structs) == 0 {
		return &BatchProcessResult{Total: 0}, nil
	}

	// Apply options (same as ProcessStructsBatch)
	opts := &BatchProcessOptions{
		MaxConcurrency:   runtime.NumCPU(),
		BatchSize:        calculateOptimalBatchSize(len(structs)),
		StopOnFirstError: false,
		EnableProgress:   false,
	}
	if len(options) > 0 && options[0] != nil {
		if options[0].MaxConcurrency > 0 {
			opts.MaxConcurrency = options[0].MaxConcurrency
		}
		if options[0].BatchSize > 0 {
			opts.BatchSize = options[0].BatchSize
		}
		opts.StopOnFirstError = options[0].StopOnFirstError
		opts.EnableProgress = options[0].EnableProgress
		opts.ProgressCallback = options[0].ProgressCallback
	}

	start := time.Now().UnixNano()
	defer runtime.GC() // Clean up after batch processing

	result := &BatchProcessResult{
		Total:  len(structs),
		Errors: make([]BatchError, 0),
	}

	// Use worker pool pattern for decryption
	semaphore := make(chan struct{}, opts.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Process all items concurrently
	for i, item := range structs {
		if item == nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{
				Index: i,
				Item:  item,
				Error: fmt.Errorf("struct at index %d is nil", i),
			})
			continue
		}

		wg.Add(1)
		go func(itemIndex int, structItem any) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Decrypt the struct
			err := processor.DecryptStruct(batchCtx, structItem)

			// Update results
			mu.Lock()
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, BatchError{
					Index: itemIndex,
					Item:  structItem,
					Error: err,
				})

				if opts.StopOnFirstError {
					cancel()
				}
			} else {
				result.Processed++
			}

			// Progress callback
			if opts.EnableProgress && opts.ProgressCallback != nil {
				processedCount := result.Processed + result.Failed
				opts.ProgressCallback(processedCount, result.Total, structItem, err)
			}

			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	end := time.Now().UnixNano()
	result.Duration = fmt.Sprintf("%.2fms", float64(end-start)/1e6)

	if opts.StopOnFirstError && len(result.Errors) > 0 {
		return result, result.Errors[0].Error
	}

	return result, nil
}

// calculateOptimalBatchSize calculates an optimal batch size based on the total number of items
func calculateOptimalBatchSize(totalItems int) int {
	if totalItems <= 100 {
		return totalItems // Process all at once for small datasets
	}

	// For larger datasets, use a batch size that balances memory usage and processing efficiency
	cpuCount := runtime.NumCPU()

	// Aim for 2-4 batches per CPU core
	targetBatches := cpuCount * 3
	batchSize := totalItems / targetBatches

	// Ensure reasonable bounds
	if batchSize < 10 {
		batchSize = 10
	} else if batchSize > 1000 {
		batchSize = 1000
	}

	return batchSize
}

// GetProcessingStats returns performance statistics
func GetProcessingStats() map[string]interface{} {
	return map[string]interface{}{
		"optimal_batch_size_for_1000_items":  calculateOptimalBatchSize(1000),
		"optimal_batch_size_for_10000_items": calculateOptimalBatchSize(10000),
		"recommended_max_concurrency":        runtime.NumCPU(),
		"available_cpu_cores":                runtime.NumCPU(),
		"runtime_version":                    runtime.Version(),
	}
}
