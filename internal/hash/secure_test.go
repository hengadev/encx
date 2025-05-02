package hash

import (
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/hengadev/encx/internal/encxerr"
	"github.com/hengadev/encx/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleSecureHashing(t *testing.T) {
	// Setup test parameters
	argon2Params := &types.Argon2Params{
		Memory:      19456,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	pepper := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	// Test struct with various field types
	type TestStruct struct {
		// String field
		Password     string
		PasswordHash string

		// Numeric fields
		ID     int
		IDHash string

		Count     uint
		CountHash string

		Amount     float64
		AmountHash string

		// Time field
		CreatedAt     time.Time
		CreatedAtHash string

		// Pointer field
		OptionalData     *string
		OptionalDataHash string

		// Zero value field
		EmptyField     string
		EmptyFieldHash string

		// Field without corresponding hashed field
		MissingHashedField string
	}

	t.Run("SuccessfulHashing", func(t *testing.T) {
		// Set up test struct
		optionalData := "optional"
		testObj := TestStruct{
			Password:     "secret123",
			ID:           12345,
			Count:        42,
			Amount:       123.456,
			CreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			OptionalData: &optionalData,
			EmptyField:   "",
		}

		// Test string field
		field := reflect.TypeOf(testObj).Field(0) // Password field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.NotEmpty(t, testObj.PasswordHash)
		assert.Contains(t, testObj.PasswordHash, "$argon2id$")

		// Test int field
		field = reflect.TypeOf(testObj).Field(2) // ID field
		fieldValue = reflect.ValueOf(&testObj).Elem().Field(2)

		err = HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.NotEmpty(t, testObj.IDHash)
		assert.Contains(t, testObj.IDHash, "$argon2id$")

		// Test uint field
		field = reflect.TypeOf(testObj).Field(4) // Count field
		fieldValue = reflect.ValueOf(&testObj).Elem().Field(4)

		err = HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.NotEmpty(t, testObj.CountHash)
		assert.Contains(t, testObj.CountHash, "$argon2id$")

		// Test float field
		field = reflect.TypeOf(testObj).Field(6) // Amount field
		fieldValue = reflect.ValueOf(&testObj).Elem().Field(6)

		err = HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.NotEmpty(t, testObj.AmountHash)
		assert.Contains(t, testObj.AmountHash, "$argon2id$")

		// Test time field
		field = reflect.TypeOf(testObj).Field(8) // CreatedAt field
		fieldValue = reflect.ValueOf(&testObj).Elem().Field(8)

		err = HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.NotEmpty(t, testObj.CreatedAtHash)
		assert.Contains(t, testObj.CreatedAtHash, "$argon2id$")

		// Test pointer field
		field = reflect.TypeOf(testObj).Field(10) // OptionalData field
		fieldValue = reflect.ValueOf(&testObj).Elem().Field(10)

		err = HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.NotEmpty(t, testObj.OptionalDataHash)
		assert.Contains(t, testObj.OptionalDataHash, "$argon2id$")

		// Test zero value field
		field = reflect.TypeOf(testObj).Field(12) // EmptyField field
		fieldValue = reflect.ValueOf(&testObj).Elem().Field(12)

		err = HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.NoError(t, err)
		assert.Empty(t, testObj.EmptyFieldHash, "Zero value fields should result in empty hashed fields")
	})

	t.Run("ArgonError", func(t *testing.T) {
		// Invalid Argon2 params to simulate a hashing error
		invalidParams := &types.Argon2Params{
			Memory:      0, // Invalid value
			Iterations:  0,
			Parallelism: 0,
			SaltLength:  0,
			KeyLength:   0,
		}

		testObj := TestStruct{
			Password: "secret123",
		}

		field := reflect.TypeOf(testObj).Field(0) // Password field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, invalidParams, pepper)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed")
	})
}

func TestHandleSecureHashingErrors(t *testing.T) {
	argon2Params := types.DefaultArgon2Params()
	pepper := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	t.Run("MissingHashedFieldError", func(t *testing.T) {
		testObj := struct {
			Password string
		}{
			Password: "secret123",
		}

		field := reflect.TypeOf(testObj).Field(0) // Password field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.Error(t, err)

		// Check that error wraps the expected base error
		assert.ErrorIs(t, err, encxerr.ErrMissingField)
		// Check that the error message contains expected information
		assert.Contains(t, err.Error(), "PasswordHash")
	})

	t.Run("InvalidFieldTypeError", func(t *testing.T) {
		testObj := struct {
			Password     string
			PasswordHash int // Should be string
		}{
			Password: "secret123",
		}

		field := reflect.TypeOf(testObj).Field(0) // Password field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.Error(t, err)

		// Check that error wraps the expected base error
		assert.ErrorIs(t, err, encxerr.ErrInvalidFieldType)
		// Check that the error message contains expected information
		assert.Contains(t, err.Error(), "PasswordHash")
	})

	t.Run("NilPointerError", func(t *testing.T) {
		type PointerStruct struct {
		}

		testObj := struct {
			Password     *string
			PasswordHash string
		}{
			Password: nil, // Nil pointer
		}

		field := reflect.TypeOf(testObj).Field(0) // Password field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		// Based on your implementation, nil pointers might be treated as zero values
		// or might return an error - adjust this test accordingly

		err := HandleSecure(field, fieldValue, structValue, argon2Params, pepper)

		// Option 1: If nil pointers should be treated as errors
		if errors.Is(err, encxerr.ErrNilPointer) {
			assert.Contains(t, err.Error(), "Password")
			assert.Contains(t, err.Error(), "nil pointer")
		} else {
			// Option 2: If nil pointers should be treated as zero values
			require.NoError(t, err)
			assert.Empty(t, testObj.PasswordHash, "Nil pointers should result in empty hashed fields")
		}
	})

	t.Run("TypeConversionError", func(t *testing.T) {
		// This test requires a custom setup to trigger a type conversion error
		// Let's create a problematic time.Time value that would fail conversion

		// Set up a case that would fail time.Time type assertion
		testObj := struct {
			CreatedAt     any // Type interface{} but contains time.Time
			CreatedAtHash string
			THash         string
		}{
			CreatedAt: "not-a-time-value", // This will fail when trying to convert to time.Time
		}

		// Create a modified field that looks like time.Time but isn't
		field := reflect.TypeOf(struct{ T time.Time }{}).Field(0) // Get time.Time field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)   // But value is interface{}
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.Error(t, err)

		// Check that error wraps the expected base error
		assert.ErrorIs(t, err, encxerr.ErrTypeConversion)
		assert.Contains(t, err.Error(), "time.Time")
	})

	t.Run("UnsupportedTypeError", func(t *testing.T) {
		testObj := struct {
			Complex     complex128
			ComplexHash string
		}{
			Complex: 1 + 2i,
		}

		field := reflect.TypeOf(testObj).Field(0) // Complex field
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, argon2Params, pepper)
		require.Error(t, err)

		// Check that error wraps the expected base error
		assert.ErrorIs(t, err, encxerr.ErrUnsupportedType)
		assert.Contains(t, err.Error(), "Complex")
		assert.Contains(t, err.Error(), "complex128")
	})

	t.Run("OperationFailedError", func(t *testing.T) {
		// Create invalid Argon2 params to force a hashing error
		invalidParams := &types.Argon2Params{
			Memory:      0,
			Iterations:  0,
			Parallelism: 0,
			SaltLength:  0,
			KeyLength:   0,
		}

		testObj := struct {
			Password     string
			PasswordHash string
		}{
			Password: "test",
		}

		field := reflect.TypeOf(testObj).Field(0)
		fieldValue := reflect.ValueOf(&testObj).Elem().Field(0)
		structValue := reflect.ValueOf(&testObj).Elem()

		err := HandleSecure(field, fieldValue, structValue, invalidParams, pepper)
		require.Error(t, err)

		// Check that error wraps the expected base error
		assert.ErrorIs(t, err, encxerr.ErrOperationFailed)
		assert.Contains(t, err.Error(), "Password")
		// assert.Contains(t, err.Error(), "secure_hash")
	})
}

func TestEncodeSecureHash(t *testing.T) {

	// Setup valid params
	validParams := types.DefaultArgon2Params()
	nonZeroPepper := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	zeroPepper := [16]byte{}
	t.Run("SuccessfulHashing", func(t *testing.T) {
		result, err := encodeSecureHash("password123", validParams, nonZeroPepper)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "$argon2id$")
	})
	t.Run("InvalidParams", func(t *testing.T) {
		invalidParams := &types.Argon2Params{
			Memory:      1024, // Too low
			Iterations:  1,    // Too low
			Parallelism: 1,
			SaltLength:  8,  // Too short
			KeyLength:   16, // Too short
		}

		_, err := encodeSecureHash("password123", invalidParams, nonZeroPepper)
		assert.Error(t, err)
	})
	t.Run("ZeroPepper", func(t *testing.T) {
		_, err := encodeSecureHash("password123", validParams, zeroPepper)
		assert.Error(t, err)
		assert.ErrorIs(t, ErrUninitializedPepper, err)
	})
	t.Run("OutputFormat", func(t *testing.T) {
		result, err := encodeSecureHash("password123", validParams, nonZeroPepper)
		assert.NoError(t, err)

		// Check format using regex
		pattern := `^\$argon2id\$v=\d+\$m=\d+,t=\d+,p=\d+\$[A-Za-z0-9+/]+=*\$[A-Za-z0-9+/]+=*$`
		matched, err := regexp.MatchString(pattern, result)
		assert.NoError(t, err)
		assert.True(t, matched)
	})
	t.Run("DifferentInputsDifferentHashes", func(t *testing.T) {
		hash1, err := encodeSecureHash("password123", validParams, nonZeroPepper)
		assert.NoError(t, err)

		hash2, err := encodeSecureHash("password124", validParams, nonZeroPepper)
		assert.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})
	t.Run("RandomSalt", func(t *testing.T) {
		hash1, err := encodeSecureHash("password123", validParams, nonZeroPepper)
		assert.NoError(t, err)

		hash2, err := encodeSecureHash("password123", validParams, nonZeroPepper)
		assert.NoError(t, err)

		assert.NotEqual(t, hash1, hash2, "Same input should produce different hashes due to random salt")
	})
}

// TODO: add a benchmark for this
func BenchmarkEncodeSecureHash(b *testing.B) {

}
