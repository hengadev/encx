package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hengadev/encx"
)

// Demo user struct with different field types
type User struct {
	Name           string `encx:"encrypt"`
	NameEncrypted  []byte
	Email          string `encx:"hash_basic"`
	EmailHash      string
	Phone          string `encx:"encrypt,hash_basic"`
	PhoneEncrypted []byte
	PhoneHash      string
	DEK            []byte
	DEKEncrypted   []byte
	KeyVersion     int
}

func main() {
	fmt.Println("=== Monitoring and Observability Demo ===")
	fmt.Println("Demonstrating metrics collection and observability hooks in ENCX")
	fmt.Println()

	ctx := context.Background()

	// Create an in-memory metrics collector for demonstration
	metricsCollector := encx.NewInMemoryMetricsCollector()

	// Create crypto instance with monitoring enabled
	crypto, err := encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("demo-monitoring-kek"),
		encx.WithPepper([]byte("demo-pepper-exactly-32-bytes-OK!")),
		encx.WithStandardMonitoring(metricsCollector), // Enable monitoring
	)
	if err != nil {
		log.Fatalf("Failed to create crypto instance: %v", err)
	}

	// Demo 1: Process multiple structs to generate metrics
	fmt.Println("1. Processing Structs with Monitoring:")
	users := []*User{
		{Name: "Alice Johnson", Email: "alice@example.com", Phone: "+1-555-0101"},
		{Name: "Bob Smith", Email: "bob@example.com", Phone: "+1-555-0102"},
		{Name: "Charlie Brown", Email: "charlie@example.com", Phone: "+1-555-0103"},
	}

	for i, user := range users {
		start := time.Now()
		err := crypto.ProcessStruct(ctx, user)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("   ❌ User %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✅ User %d processed in %v\n", i+1, duration)
		}
	}
	fmt.Println()

	// Demo 2: Key rotation monitoring
	fmt.Println("2. Key Rotation Monitoring:")
	start := time.Now()
	err = crypto.RotateKEK(ctx)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("   ❌ Key rotation failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Key rotated successfully in %v\n", duration)
	}
	fmt.Println()

	// Demo 3: Display collected metrics
	fmt.Println("3. Collected Metrics:")
	fmt.Println()

	// Show counters
	fmt.Println("   📊 Counters:")
	counterMetrics := []struct {
		name string
		tags map[string]string
	}{
		{
			name: "encx.process.started",
			tags: map[string]string{"operation": "ProcessStruct"},
		},
		{
			name: "encx.process.completed",
			tags: map[string]string{"operation": "ProcessStruct", "status": "success"},
		},
		{
			name: "encx.key_operations",
			tags: map[string]string{"operation": "rotate"},
		},
	}

	for _, metric := range counterMetrics {
		// Since we can't match partial tags easily, let's find the full metric name
		found := false
		for fullKey := range metricsCollector.GetAllCounterKeys() {
			if containsAllTags(fullKey, metric.name, metric.tags) {
				value := metricsCollector.GetCounterValueByKey(fullKey)
				fmt.Printf("     %s: %d\n", fullKey, value)
				found = true
			}
		}
		if !found {
			fmt.Printf("     %s (with tags %v): 0 (not found)\n", metric.name, metric.tags)
		}
	}
	fmt.Println()

	// Show timing metrics
	fmt.Println("   ⏱️  Timing Metrics:")
	timings := metricsCollector.GetTimings()
	operationTimes := make(map[string][]time.Duration)

	for _, timing := range timings {
		if operation, ok := timing.Tags["operation"]; ok {
			operationTimes[operation] = append(operationTimes[operation], timing.Duration)
		}
	}

	for operation, durations := range operationTimes {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))
			fmt.Printf("     %s: %d calls, avg duration: %v\n", operation, len(durations), avg)
		}
	}
	fmt.Println()

	// Show gauges
	fmt.Println("   📈 Gauges:")
	keyVersion := metricsCollector.GetGaugeValue("encx.key_version", map[string]string{
		"key_alias": "demo-monitoring-kek",
	})
	if keyVersion > 0 {
		fmt.Printf("     Current key version: %.0f\n", keyVersion)
	}
	fmt.Println()

	// Demo 4: Demonstrate batch processing monitoring
	fmt.Println("4. Batch Processing Monitoring:")
	batchUsers := make([]any, 5)
	for i := 0; i < 5; i++ {
		batchUsers[i] = &User{
			Name:  fmt.Sprintf("BatchUser%d", i),
			Email: fmt.Sprintf("batch%d@example.com", i),
			Phone: fmt.Sprintf("+1-555-0%03d", 200+i),
		}
	}

	start = time.Now()
	result, err := crypto.ProcessStructsBatch(ctx, batchUsers, &encx.BatchProcessOptions{
		MaxConcurrency:   2,
		BatchSize:        3,
		StopOnFirstError: false,
		EnableProgress:   true,
		ProgressCallback: func(processed, total int, item any, err error) {
			if err != nil {
				fmt.Printf("     [%d/%d] ❌ Error processing item: %v\n", processed, total, err)
			} else {
				fmt.Printf("     [%d/%d] ✅ Item processed successfully\n", processed, total)
			}
		},
	})

	duration = time.Since(start)
	if err != nil {
		fmt.Printf("   ❌ Batch processing failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Batch processed: %d succeeded, %d failed in %v\n",
			result.Processed, result.Failed, duration)
	}
	fmt.Println()

	// Demo 5: Show final metrics summary
	fmt.Println("5. Final Metrics Summary:")
	fmt.Printf("   📊 Total operations tracked: %d\n", len(timings))

	totalStructsProcessed := int64(0)
	for fullKey := range metricsCollector.GetAllCounterKeys() {
		if containsTag(fullKey, "operation:ProcessStruct") && containsTag(fullKey, "status:success") {
			totalStructsProcessed += metricsCollector.GetCounterValueByKey(fullKey)
		}
	}
	fmt.Printf("   ✅ Total structs successfully processed: %d\n", totalStructsProcessed)

	keyRotations := metricsCollector.GetCounterValue("encx.key_operations", map[string]string{
		"operation": "rotate",
	})
	// If exact match fails, try to find it
	if keyRotations == 0 {
		for fullKey := range metricsCollector.GetAllCounterKeys() {
			if containsTag(fullKey, "operation:rotate") {
				keyRotations += metricsCollector.GetCounterValueByKey(fullKey)
			}
		}
	}
	fmt.Printf("   🔄 Key rotations performed: %d\n", keyRotations)

	fmt.Println()
	fmt.Println("=== Monitoring Benefits ===")
	fmt.Println("✅ Real-time visibility into cryptographic operations")
	fmt.Println("✅ Performance monitoring with detailed timing metrics")
	fmt.Println("✅ Error tracking and operational health insights")
	fmt.Println("✅ Key rotation and lifecycle management monitoring")
	fmt.Println("✅ Batch processing efficiency metrics")
	fmt.Println("✅ Configurable monitoring with custom collectors")
	fmt.Println("✅ Zero-overhead no-op implementations for production")
}

// Helper functions for the demo (these would be methods on the collector in a real implementation)

func containsAllTags(fullKey, metricName string, tags map[string]string) bool {
	if !containsMetricName(fullKey, metricName) {
		return false
	}
	for k, v := range tags {
		if !containsTag(fullKey, k+":"+v) {
			return false
		}
	}
	return true
}

func containsMetricName(fullKey, metricName string) bool {
	return len(fullKey) >= len(metricName) && fullKey[:len(metricName)] == metricName
}

func containsTag(fullKey, tag string) bool {
	return len(fullKey) > len(tag) &&
		(fullKey[len(fullKey)-len(tag):] == tag ||
			fmt.Sprintf(",%s,", tag) != "" &&
				(fullKey == tag || fullKey[:len(tag)+1] == tag+"," ||
					fullKey[len(fullKey)-len(tag)-1:] == ","+tag ||
					len(fullKey) > len(tag)+1 && fullKey[len(fullKey)-len(tag)-1:] == ","+tag))
}

