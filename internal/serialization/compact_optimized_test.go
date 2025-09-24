package serialization

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOptimizedSerializerCompatibility ensures optimized and original serializers produce compatible output
func TestOptimizedSerializerCompatibility(t *testing.T) {
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"String", "test string"},
		{"EmptyString", ""},
		{"Unicode", "🎉 Unicode: åß∂ƒ"},
		{"Int64", int64(123456789)},
		{"Int64_Negative", int64(-123456789)},
		{"Int64_Max", int64(9223372036854775807)},
		{"Int64_Min", int64(-9223372036854775808)},
		{"Int32", int32(12345)},
		{"Int32_Negative", int32(-12345)},
		{"Int", int(123456)},
		{"Uint64", uint64(123456789)},
		{"Uint32", uint32(12345)},
		{"Uint", uint(123456)},
		{"Bool_True", true},
		{"Bool_False", false},
		{"Time", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
		{"Time_Unix", time.Unix(1609459200, 0)},
		{"Float64", 3.14159265358979323846},
		{"Float64_Negative", -3.14159265358979323846},
		{"Float64_Zero", 0.0},
		{"Float32", float32(3.14159)},
		{"Bytes", []byte("test bytes")},
		{"EmptyBytes", []byte{}},
		{"LargeBytes", make([]byte, 10000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize with both methods
			originalSerialized, err1 := Serialize(tc.value)
			optimizedSerialized, err2 := SerializeOptimized(tc.value)

			// Both should succeed or fail together
			if err1 != nil || err2 != nil {
				require.Equal(t, err1 != nil, err2 != nil, "Both methods should succeed or fail together")
				if err1 != nil {
					return // Both failed as expected
				}
			}

			// Serialized data should be identical
			assert.Equal(t, originalSerialized, optimizedSerialized,
				"Serialized data should be identical between original and optimized methods")

			// Test deserialization compatibility
			testDeserialization(t, tc.value, originalSerialized, optimizedSerialized)
		})
	}
}

// testDeserialization tests that both deserializers can handle data from both serializers
func testDeserialization(t *testing.T, original interface{}, originalData, optimizedData []byte) {
	switch original.(type) {
	case string:
		var result1, result2, result3, result4 string

		// Original data -> Original deserializer
		err := Deserialize(originalData, &result1)
		require.NoError(t, err)
		assert.Equal(t, original, result1)

		// Original data -> Optimized deserializer
		err = DeserializeOptimized(originalData, &result2)
		require.NoError(t, err)
		assert.Equal(t, original, result2)

		// Optimized data -> Original deserializer
		err = Deserialize(optimizedData, &result3)
		require.NoError(t, err)
		assert.Equal(t, original, result3)

		// Optimized data -> Optimized deserializer
		err = DeserializeOptimized(optimizedData, &result4)
		require.NoError(t, err)
		assert.Equal(t, original, result4)

	case int64:
		var result1, result2, result3, result4 int64

		err := Deserialize(originalData, &result1)
		require.NoError(t, err)
		assert.Equal(t, original, result1)

		err = DeserializeOptimized(originalData, &result2)
		require.NoError(t, err)
		assert.Equal(t, original, result2)

		err = Deserialize(optimizedData, &result3)
		require.NoError(t, err)
		assert.Equal(t, original, result3)

		err = DeserializeOptimized(optimizedData, &result4)
		require.NoError(t, err)
		assert.Equal(t, original, result4)

	case bool:
		var result1, result2, result3, result4 bool

		err := Deserialize(originalData, &result1)
		require.NoError(t, err)
		assert.Equal(t, original, result1)

		err = DeserializeOptimized(originalData, &result2)
		require.NoError(t, err)
		assert.Equal(t, original, result2)

		err = Deserialize(optimizedData, &result3)
		require.NoError(t, err)
		assert.Equal(t, original, result3)

		err = DeserializeOptimized(optimizedData, &result4)
		require.NoError(t, err)
		assert.Equal(t, original, result4)

	case time.Time:
		var result1, result2, result3, result4 time.Time

		err := Deserialize(originalData, &result1)
		require.NoError(t, err)
		assert.True(t, original.(time.Time).Equal(result1), "Times should be equal (original -> original)")

		err = DeserializeOptimized(originalData, &result2)
		require.NoError(t, err)
		assert.True(t, original.(time.Time).Equal(result2), "Times should be equal (original -> optimized)")

		err = Deserialize(optimizedData, &result3)
		require.NoError(t, err)
		assert.True(t, original.(time.Time).Equal(result3), "Times should be equal (optimized -> original)")

		err = DeserializeOptimized(optimizedData, &result4)
		require.NoError(t, err)
		assert.True(t, original.(time.Time).Equal(result4), "Times should be equal (optimized -> optimized)")

	case []byte:
		var result1, result2, result3, result4 []byte

		err := Deserialize(originalData, &result1)
		require.NoError(t, err)
		assert.Equal(t, original, result1)

		err = DeserializeOptimized(originalData, &result2)
		require.NoError(t, err)
		assert.Equal(t, original, result2)

		err = Deserialize(optimizedData, &result3)
		require.NoError(t, err)
		assert.Equal(t, original, result3)

		err = DeserializeOptimized(optimizedData, &result4)
		require.NoError(t, err)
		assert.Equal(t, original, result4)
	}
}

// TestOptimizedBatchSerialization tests batch serialization functionality
func TestOptimizedBatchSerialization(t *testing.T) {
	t.Run("HomogeneousStringBatch", func(t *testing.T) {
		values := []interface{}{
			"first string",
			"second string",
			"third string",
		}

		results, err := SerializeBatch(values)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Verify each result
		for i, expected := range []string{"first string", "second string", "third string"} {
			var result string
			err := DeserializeOptimized(results[i], &result)
			require.NoError(t, err)
			assert.Equal(t, expected, result)
		}
	})

	t.Run("HomogeneousIntBatch", func(t *testing.T) {
		values := []interface{}{
			int64(100),
			int64(200),
			int64(300),
		}

		results, err := SerializeBatch(values)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Verify each result
		for i, expected := range []int64{100, 200, 300} {
			var result int64
			err := DeserializeOptimized(results[i], &result)
			require.NoError(t, err)
			assert.Equal(t, expected, result)
		}
	})

	t.Run("HeterogeneousBatch", func(t *testing.T) {
		values := []interface{}{
			"string value",
			int64(123),
			true,
		}

		results, err := SerializeBatch(values)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Verify string
		var strResult string
		err = DeserializeOptimized(results[0], &strResult)
		require.NoError(t, err)
		assert.Equal(t, "string value", strResult)

		// Verify int
		var intResult int64
		err = DeserializeOptimized(results[1], &intResult)
		require.NoError(t, err)
		assert.Equal(t, int64(123), intResult)

		// Verify bool
		var boolResult bool
		err = DeserializeOptimized(results[2], &boolResult)
		require.NoError(t, err)
		assert.Equal(t, true, boolResult)
	})

	t.Run("EmptyBatch", func(t *testing.T) {
		results, err := SerializeBatch([]interface{}{})
		require.NoError(t, err)
		assert.Nil(t, results)
	})
}

// TestGetSerializedSize tests the size prediction functionality
func TestGetSerializedSize(t *testing.T) {
	testCases := []struct {
		name         string
		value        interface{}
		expectedSize int
	}{
		{"String", "hello", 4 + 5},
		{"EmptyString", "", 4 + 0},
		{"Int64", int64(123), 8},
		{"Int32", int32(123), 4},
		{"Bool", true, 1},
		{"Time", time.Now(), 8},
		{"Bytes", []byte("test"), 4 + 4},
		{"EmptyBytes", []byte{}, 4 + 0},
		{"Float64", 3.14, 8},
		{"Float32", float32(3.14), 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			predictedSize := GetSerializedSize(tc.value)
			assert.Equal(t, tc.expectedSize, predictedSize)

			// Verify prediction is accurate
			serialized, err := SerializeOptimized(tc.value)
			require.NoError(t, err)
			assert.Equal(t, predictedSize, len(serialized))
		})
	}
}

// TestOptimizedErrorHandling tests error conditions
func TestOptimizedErrorHandling(t *testing.T) {
	t.Run("UnsupportedType", func(t *testing.T) {
		type CustomStruct struct {
			Field int
		}

		_, err := SerializeOptimized(CustomStruct{Field: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
	})

	t.Run("InsufficientDataString", func(t *testing.T) {
		var result string
		err := DeserializeOptimized([]byte{1, 2}, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient data for string length")
	})

	t.Run("InsufficientDataInt64", func(t *testing.T) {
		var result int64
		err := DeserializeOptimized([]byte{1, 2, 3}, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient data for int64")
	})

	t.Run("UnsupportedTargetType", func(t *testing.T) {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		type CustomType int
		var result CustomType

		err := DeserializeOptimized(data, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported target type")
	})
}

// TestOptimizedFloatSpecialValues tests handling of special float values
func TestOptimizedFloatSpecialValues(t *testing.T) {
	specialValues := []float64{
		math.Inf(1),   // Positive infinity
		math.Inf(-1),  // Negative infinity
		math.NaN(),    // Not a Number
		0.0,           // Positive zero
		math.Copysign(0.0, -1), // Negative zero
		math.SmallestNonzeroFloat64,
		math.MaxFloat64,
	}

	for i, value := range specialValues {
		t.Run(fmt.Sprintf("SpecialFloat_%d", i), func(t *testing.T) {
			// Serialize
			data, err := SerializeOptimized(value)
			require.NoError(t, err)

			// Deserialize
			var result float64
			err = DeserializeOptimized(data, &result)
			require.NoError(t, err)

			// Special handling for NaN
			if math.IsNaN(value) {
				assert.True(t, math.IsNaN(result))
			} else {
				assert.Equal(t, value, result)
			}
		})
	}
}

// TestOptimizedLargeData tests performance with large data sets
func TestOptimizedLargeData(t *testing.T) {
	sizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			// Create test data
			testData := make([]byte, size)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			// Serialize
			serialized, err := SerializeOptimized(testData)
			require.NoError(t, err)
			assert.Equal(t, 4+size, len(serialized)) // 4 bytes length + data

			// Deserialize
			var result []byte
			err = DeserializeOptimized(serialized, &result)
			require.NoError(t, err)
			assert.Equal(t, testData, result)
		})
	}
}

// TestOptimizedConcurrentSafety tests thread safety
func TestOptimizedConcurrentSafety(t *testing.T) {
	const numGoroutines = 100
	const operationsPerGoroutine = 100

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				testValue := fmt.Sprintf("goroutine-%d-op-%d", id, j)

				// Serialize
				data, err := SerializeOptimized(testValue)
				if err != nil {
					errors <- err
					continue
				}

				// Deserialize
				var result string
				err = DeserializeOptimized(data, &result)
				if err != nil {
					errors <- err
					continue
				}

				if result != testValue {
					errors <- fmt.Errorf("value mismatch: expected %s, got %s", testValue, result)
					continue
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
		errorCount++
	}

	assert.Equal(t, 0, errorCount, "All concurrent operations should succeed")
}