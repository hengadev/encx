package serialization

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Benchmark data sets for consistent testing
var (
	testStrings = []string{
		"short",
		"medium length string for testing",
		"this is a much longer string that represents typical user data like email addresses, names, or descriptions that might be encrypted in a real application",
		"",
		"🎉 Unicode string with emojis and special characters: åß∂ƒ©˙∆˚¬…æ",
	}

	testInts = []int64{0, 1, -1, 123456789, -123456789, 9223372036854775807, -9223372036854775808}

	testFloats = []float64{0.0, 1.0, -1.0, 3.14159, -3.14159, 1.7976931348623157e+308, 4.9e-324}

	testBools = []bool{true, false}

	testTimes = []time.Time{
		time.Unix(0, 0),
		time.Now(),
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
)

// BenchmarkSerializeString compares string serialization performance
func BenchmarkSerializeString(b *testing.B) {
	testStr := "test string for benchmark comparison analysis"

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testStr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testStr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSerializeInt compares integer serialization performance
func BenchmarkSerializeInt(b *testing.B) {
	testInt := int64(123456789)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testInt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testInt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSerializeFloat compares float serialization performance
func BenchmarkSerializeFloat(b *testing.B) {
	testFloat := 3.14159265358979323846

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testFloat)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testFloat)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSerializeBool compares boolean serialization performance
func BenchmarkSerializeBool(b *testing.B) {
	testBool := true

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testBool)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testBool)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSerializeTime compares time serialization performance
func BenchmarkSerializeTime(b *testing.B) {
	testTime := time.Now()

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testTime)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testTime)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSerializeBytes compares byte slice serialization performance
func BenchmarkSerializeBytes(b *testing.B) {
	testBytes := make([]byte, 1024) // 1KB test data
	rand.Read(testBytes)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testBytes)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testBytes)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDeserializeString compares string deserialization performance
func BenchmarkDeserializeString(b *testing.B) {
	testStr := "test string for benchmark comparison analysis"
	originalData, _ := Serialize(testStr)
	optimizedData, _ := SerializeOptimized(testStr)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result string
			err := Deserialize(originalData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result string
			err := DeserializeOptimized(optimizedData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDeserializeInt compares integer deserialization performance
func BenchmarkDeserializeInt(b *testing.B) {
	testInt := int64(123456789)
	originalData, _ := Serialize(testInt)
	optimizedData, _ := SerializeOptimized(testInt)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result int64
			err := Deserialize(originalData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result int64
			err := DeserializeOptimized(optimizedData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkRoundTripString compares full round-trip performance for strings
func BenchmarkRoundTripString(b *testing.B) {
	testStr := "test string for complete round-trip benchmark"

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := Serialize(testStr)
			if err != nil {
				b.Fatal(err)
			}
			var result string
			err = Deserialize(data, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := SerializeOptimized(testStr)
			if err != nil {
				b.Fatal(err)
			}
			var result string
			err = DeserializeOptimized(data, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkVariousSizes tests performance across different data sizes
func BenchmarkVariousSizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000, 100000}

	for _, size := range sizes {
		testData := make([]byte, size)
		rand.Read(testData)

		b.Run(fmt.Sprintf("Size_%d_Original", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Serialize(testData)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("Size_%d_Optimized", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := SerializeOptimized(testData)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkBatchSerialization tests batch processing performance
func BenchmarkBatchSerialization(b *testing.B) {
	const batchSize = 1000

	// Create homogeneous string batch
	stringBatch := make([]interface{}, batchSize)
	for i := 0; i < batchSize; i++ {
		stringBatch[i] = fmt.Sprintf("test string %d", i)
	}

	// Create homogeneous int batch
	intBatch := make([]interface{}, batchSize)
	for i := 0; i < batchSize; i++ {
		intBatch[i] = int64(i)
	}

	b.Run("StringBatch_Individual", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, v := range stringBatch {
				_, err := Serialize(v)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("StringBatch_Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeBatch(stringBatch)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("IntBatch_Individual", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, v := range intBatch {
				_, err := Serialize(v)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("IntBatch_Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeBatch(intBatch)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMemoryEfficiency tests memory allocation patterns
func BenchmarkMemoryEfficiency(b *testing.B) {
	testStr := "memory efficiency test string"

	b.Run("Original_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := Serialize(testStr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Optimized_Allocs", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := SerializeOptimized(testStr)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkRealWorldMixed tests mixed data types like in real applications
func BenchmarkRealWorldMixed(b *testing.B) {
	// Simulate real-world data patterns
	userData := []interface{}{
		"user@example.com",           // email
		"John Doe",                   // name
		int64(1234567890),            // user ID
		true,                         // active status
		time.Now(),                   // created timestamp
		[]byte("encrypted password"), // password hash
		3.14159,                      // some float value
		int32(25),                    // age
	}

	b.Run("Mixed_Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, v := range userData {
				_, err := Serialize(v)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("Mixed_Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, v := range userData {
				_, err := SerializeOptimized(v)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("Mixed_Batch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SerializeBatch(userData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkOutputSize compares the size of serialized output
func BenchmarkOutputSize(b *testing.B) {
	testCases := []interface{}{
		"short string",
		"this is a longer string that represents more typical user data",
		int64(123456789),
		3.14159265358979,
		true,
		time.Now(),
		make([]byte, 1000),
	}

	for i, testCase := range testCases {
		b.Run(fmt.Sprintf("Case_%d", i), func(b *testing.B) {
			originalData, _ := Serialize(testCase)
			optimizedData, _ := SerializeOptimized(testCase)

			b.Logf("Original size: %d bytes", len(originalData))
			b.Logf("Optimized size: %d bytes", len(optimizedData))
			b.Logf("Size difference: %d bytes", len(optimizedData)-len(originalData))

			if len(optimizedData) > 0 {
				efficiency := float64(len(originalData)) / float64(len(optimizedData))
				b.Logf("Size efficiency: %.2fx", efficiency)
			}

			// Don't actually benchmark anything here, just report sizes
			b.SkipNow()
		})
	}
}

// BenchmarkConcurrentAccess tests performance under concurrent load
func BenchmarkConcurrentAccess(b *testing.B) {
	testStr := "concurrent access test string"

	b.Run("Original_Concurrent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := Serialize(testStr)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("Optimized_Concurrent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := SerializeOptimized(testStr)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}