package hash

import (
	"reflect"
	"testing"
	"time"

	"github.com/hengadev/encx/internal/encxerr"

	"github.com/stretchr/testify/assert"
)

// Test structures
type StringTest struct {
	Username     string
	UsernameHash string
}

type IntTest struct {
	UserID     int
	UserIDHash string
}

type FloatTest struct {
	Score     float64
	ScoreHash string
}

type TimeTest struct {
	Timestamp     time.Time
	TimestampHash string
}

type UnsupportedTest struct {
	Complex     complex128
	ComplexHash string
}

type InvalidTargetTest struct {
	Username     string
	UsernameHash int // Wrong type, should be string
}

type MissingTargetTest struct {
	Username string
	// UsernameHash is missing
}

func TestHandleBasicHashing(t *testing.T) {
	// Test cases for success scenarios
	t.Run("String field hashing", func(t *testing.T) {
		// Create test struct
		test := &StringTest{Username: "JohnDoe"}

		// Get reflection values
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0) // Username field
		fieldValue := val.Field(0)   // Value of Username

		assert.NoError(t, HandleBasic(field, fieldValue, val))
		assert.Equal(t, test.UsernameHash, Basic("johndoe")) // Lowercased
	})

	t.Run("Int field hashing", func(t *testing.T) {
		test := &IntTest{UserID: 12345}
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0)
		fieldValue := val.Field(0)

		assert.NoError(t, HandleBasic(field, fieldValue, val))
		assert.Equal(t, test.UserIDHash, Basic("12345"))
	})

	t.Run("Float field hashing", func(t *testing.T) {
		test := &FloatTest{Score: 98.76}
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0)
		fieldValue := val.Field(0)

		assert.NoError(t, HandleBasic(field, fieldValue, val))
		assert.Equal(t, test.ScoreHash, Basic("98.76"))
	})

	t.Run("Time field hashing", func(t *testing.T) {
		testTime := time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC)
		test := &TimeTest{Timestamp: testTime}
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0)
		fieldValue := val.Field(0)

		assert.NoError(t, HandleBasic(field, fieldValue, val))
		assert.Equal(t, test.TimestampHash, Basic(testTime.Format(time.RFC3339)))
	})

	// Test cases for error scenarios
	t.Run("Unsupported type", func(t *testing.T) {
		test := &UnsupportedTest{Complex: complex(1, 2)}
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0)
		fieldValue := val.Field(0)

		err := HandleBasic(field, fieldValue, val)

		assert.Error(t, err)
		assert.ErrorIs(t, err, encxerr.ErrUnsupportedType)
	})

	t.Run("Invalid target field type", func(t *testing.T) {
		test := &InvalidTargetTest{Username: "JohnDoe"}
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0)
		fieldValue := val.Field(0)

		err := HandleBasic(field, fieldValue, val)

		assert.Error(t, err)
		assert.ErrorIs(t, err, encxerr.ErrInvalidFieldType)
	})

	t.Run("Missing target field", func(t *testing.T) {
		test := &MissingTargetTest{Username: "JohnDoe"}
		val := reflect.ValueOf(test).Elem()
		field := val.Type().Field(0)
		fieldValue := val.Field(0)

		err := HandleBasic(field, fieldValue, val)

		assert.Error(t, err)
		assert.ErrorIs(t, err, encxerr.ErrMissingField)
	})
}

func BenchmarkHandleBasicHashing(b *testing.B) {
	test := &StringTest{Username: "JohnDoe"}
	val := reflect.ValueOf(test).Elem()
	field := val.Type().Field(0)
	fieldValue := val.Field(0)

	for i := 0; i < b.N; i++ {
		HandleBasic(field, fieldValue, val)
	}
}

func TestHashBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "Basic hash",
			input: "hello",
			// Expected value is SHA-256 hash of "hello"
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:  "Case insensitivity",
			input: "HELLO",
			// This should produce the same hash as "hello" due to ToLower()
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:  "Empty string",
			input: "",
			// SHA-256 hash of empty string
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:  "Special characters",
			input: "Hello, World!@#$%^&*()",
			// SHA-256 hash of "hello, world!@#$%^&*()"
			expected: "25212de46c7b244004ee3777d9a70af1405b83da0424eda78fd854168793546e",
		},
		{
			name:  "Unicode characters",
			input: "こんにちは世界",
			// SHA-256 hash of these Unicode characters
			expected: "c6a304536826fb57e1b1896fcd8c91693a746233ae6a286dc85a65c8ae1f416f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Basic(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// A benchmark to measure performance
func BenchmarkHashBasic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Basic("hello world this is a benchmark test string")
	}
}
