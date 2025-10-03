package security

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureMemory(t *testing.T) {
	sm := NewSecureMemory()

	t.Run("ZeroBytes", func(t *testing.T) {
		data := []byte{0x41, 0x42, 0x43, 0x44} // "ABCD"
		sm.ZeroBytes(data)

		// Verify all bytes are zeroed
		for i, b := range data {
			assert.Equal(t, byte(0), b, "Byte at position %d should be zero", i)
		}
	})

	t.Run("ZeroBytes_EmptySlice", func(t *testing.T) {
		var data []byte
		// Should not panic
		assert.NotPanics(t, func() {
			sm.ZeroBytes(data)
		})
	})

	t.Run("SecureAllocate", func(t *testing.T) {
		size := 128
		data := sm.SecureAllocate(size)
		assert.Len(t, data, size)
		assert.NotNil(t, data)

		// Verify we can write to all positions
		for i := range data {
			data[i] = byte(i % 256)
		}
	})

	t.Run("SecureAllocate_ZeroSize", func(t *testing.T) {
		data := sm.SecureAllocate(0)
		assert.Nil(t, data)
	})

	t.Run("SecureCopy", func(t *testing.T) {
		original := []byte{0x01, 0x02, 0x03, 0x04}
		copied := sm.SecureCopy(original)

		assert.Equal(t, original, copied)
		assert.NotSame(t, &original[0], &copied[0]) // Different memory addresses

		// Modify original to ensure they're separate
		original[0] = 0xFF
		assert.NotEqual(t, original[0], copied[0])
	})

	t.Run("ConstantTimeCompare", func(t *testing.T) {
		a := []byte{0x01, 0x02, 0x03}
		b := []byte{0x01, 0x02, 0x03}
		c := []byte{0x01, 0x02, 0x04}

		assert.Equal(t, 1, sm.ConstantTimeCompare(a, b))
		assert.Equal(t, 0, sm.ConstantTimeCompare(a, c))
	})

	t.Run("ConstantTimeEq", func(t *testing.T) {
		a := []byte{0x01, 0x02, 0x03}
		b := []byte{0x01, 0x02, 0x03}
		c := []byte{0x01, 0x02, 0x04}

		assert.True(t, sm.ConstantTimeEq(a, b))
		assert.False(t, sm.ConstantTimeEq(a, c))
	})

	t.Run("ConstantTimeSelect", func(t *testing.T) {
		a := []byte{0x01, 0x02}
		b := []byte{0x03, 0x04}

		result1 := sm.ConstantTimeSelect(1, a, b)
		assert.Equal(t, a, result1)

		result2 := sm.ConstantTimeSelect(0, a, b)
		assert.Equal(t, b, result2)
	})

	t.Run("ConstantTimeSelect_DifferentLengths", func(t *testing.T) {
		a := []byte{0x01}
		b := []byte{0x03, 0x04}

		result := sm.ConstantTimeSelect(1, a, b)
		assert.Empty(t, result) // Should return empty slice for safety
	})
}

func TestSecureBuffer(t *testing.T) {
	t.Run("NewSecureBuffer", func(t *testing.T) {
		size := 64
		buffer := NewSecureBuffer(size)
		defer buffer.Finalize()

		assert.Equal(t, size, buffer.Len())
		assert.NotNil(t, buffer.Bytes())
		assert.Len(t, buffer.Bytes(), size)
	})

	t.Run("SecureBuffer_WriteAndRead", func(t *testing.T) {
		buffer := NewSecureBuffer(10)
		defer buffer.Finalize()

		data := buffer.Bytes()
		for i := range data {
			data[i] = byte(i)
		}

		// Verify data was written
		for i, b := range buffer.Bytes() {
			assert.Equal(t, byte(i), b)
		}
	})

	t.Run("SecureBuffer_Copy", func(t *testing.T) {
		buffer := NewSecureBuffer(5)
		defer buffer.Finalize()

		// Write test data
		data := buffer.Bytes()
		for i := range data {
			data[i] = byte(i + 1)
		}

		copied := buffer.Copy()
		assert.Equal(t, buffer.Bytes(), copied)
		assert.NotSame(t, &buffer.Bytes()[0], &copied[0])
	})

	t.Run("SecureBuffer_Finalize", func(t *testing.T) {
		buffer := NewSecureBuffer(5)

		// Write test data
		data := buffer.Bytes()
		for i := range data {
			data[i] = byte(i + 1)
		}

		buffer.Finalize()

		// After finalize, should return nil/empty
		assert.Nil(t, buffer.Bytes())
		assert.Equal(t, 0, buffer.Len())
		assert.Nil(t, buffer.Copy())
	})
}

func TestSecureSlice(t *testing.T) {
	t.Run("NewSecureSlice", func(t *testing.T) {
		slice := NewSecureSlice(10)
		defer slice.Close()

		assert.Equal(t, 10, slice.Len())
		assert.NotNil(t, slice.Bytes())
	})

	t.Run("NewSecureSliceFromBytes", func(t *testing.T) {
		original := []byte{1, 2, 3, 4, 5}
		slice := NewSecureSliceFromBytes(original)
		defer slice.Close()

		assert.Equal(t, len(original), slice.Len())
		assert.Equal(t, original, slice.Bytes())
	})

	t.Run("SecureSlice_Close", func(t *testing.T) {
		slice := NewSecureSlice(5)
		assert.NotNil(t, slice.Bytes())

		err := slice.Close()
		assert.NoError(t, err)

		// After close, should return nil
		assert.Nil(t, slice.Bytes())
		assert.Equal(t, 0, slice.Len())
	})
}

func TestSecureRandomGenerator(t *testing.T) {
	srg := NewSecureRandomGenerator()

	t.Run("Generate", func(t *testing.T) {
		size := 32
		data, err := srg.Generate(size)
		require.NoError(t, err)
		assert.Len(t, data, size)

		// Generate another and ensure they're different
		data2, err := srg.Generate(size)
		require.NoError(t, err)
		assert.NotEqual(t, data, data2)
	})

	t.Run("Generate_InvalidSize", func(t *testing.T) {
		data, err := srg.Generate(0)
		assert.Error(t, err)
		assert.Nil(t, data)

		data, err = srg.Generate(-1)
		assert.Error(t, err)
		assert.Nil(t, data)
	})

	t.Run("GenerateKey", func(t *testing.T) {
		validSizes := []int{16, 24, 32, 64}

		for _, size := range validSizes {
			key, err := srg.GenerateKey(size)
			require.NoError(t, err, "Size: %d", size)
			assert.Len(t, key, size)
		}
	})

	t.Run("GenerateKey_InvalidSize", func(t *testing.T) {
		key, err := srg.GenerateKey(8) // Too small
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("GenerateNonce", func(t *testing.T) {
		nonce, err := srg.GenerateNonce(12)
		require.NoError(t, err)
		assert.Len(t, nonce, 12)
	})

	t.Run("GenerateNonce_TooSmall", func(t *testing.T) {
		nonce, err := srg.GenerateNonce(8) // Too small
		assert.Error(t, err)
		assert.Nil(t, nonce)
	})

	t.Run("GenerateSalt", func(t *testing.T) {
		salt, err := srg.GenerateSalt(16)
		require.NoError(t, err)
		assert.Len(t, salt, 16)
	})

	t.Run("GeneratePassword", func(t *testing.T) {
		password, err := srg.GeneratePassword(12, true)
		require.NoError(t, err)
		assert.Len(t, password, 12)

		// Should contain different character types
		hasUpper := false
		hasLower := false
		hasDigit := false
		hasSpecial := false

		for _, char := range password {
			switch {
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= '0' && char <= '9':
				hasDigit = true
			default:
				hasSpecial = true
			}
		}

		assert.True(t, hasUpper, "Password should contain uppercase letters")
		assert.True(t, hasLower, "Password should contain lowercase letters")
		assert.True(t, hasDigit, "Password should contain digits")
		assert.True(t, hasSpecial, "Password should contain special characters")
	})

	t.Run("GeneratePassword_TooShort", func(t *testing.T) {
		password, err := srg.GeneratePassword(4, false)
		assert.Error(t, err)
		assert.Empty(t, password)
	})
}

func TestTimingProtection(t *testing.T) {
	tp := NewTimingProtection(time.Millisecond * 10)

	t.Run("ConstantTimeOperation", func(t *testing.T) {
		executed := false
		start := time.Now()

		err := tp.ConstantTimeOperation(func() error {
			executed = true
			time.Sleep(time.Millisecond * 2) // Shorter than base delay
			return nil
		})

		elapsed := time.Since(start)
		assert.NoError(t, err)
		assert.True(t, executed)
		// Should take at least the base delay
		assert.GreaterOrEqual(t, elapsed, time.Millisecond*10)
	})

	t.Run("ConstantTimeOperation_WithError", func(t *testing.T) {
		testErr := errors.New("test error")

		err := tp.ConstantTimeOperation(func() error {
			return testErr
		})

		assert.Equal(t, testErr, err)
	})

	t.Run("ConstantTimeSelect", func(t *testing.T) {
		trueExecuted := false
		falseExecuted := false

		// Test true condition
		err := tp.ConstantTimeSelect(true,
			func() error {
				trueExecuted = true
				return nil
			},
			func() error {
				falseExecuted = true
				return errors.New("false operation")
			})

		assert.NoError(t, err)
		assert.True(t, trueExecuted)
		assert.False(t, falseExecuted)

		// Reset flags
		trueExecuted = false
		falseExecuted = false

		// Test false condition
		testErr := errors.New("false error")
		err = tp.ConstantTimeSelect(false,
			func() error {
				trueExecuted = true
				return errors.New("true operation")
			},
			func() error {
				falseExecuted = true
				return testErr
			})

		assert.Equal(t, testErr, err)
		assert.False(t, trueExecuted)
		assert.True(t, falseExecuted)
	})
}

func TestSecureStringComparison(t *testing.T) {
	ssc := NewSecureStringComparison()

	t.Run("CompareStrings", func(t *testing.T) {
		assert.True(t, ssc.CompareStrings("hello", "hello"))
		assert.False(t, ssc.CompareStrings("hello", "world"))
		assert.False(t, ssc.CompareStrings("hello", "hello2"))
	})

	t.Run("CompareHashes", func(t *testing.T) {
		hash1 := "a1b2c3d4e5f6"
		hash2 := "a1b2c3d4e5f6"
		hash3 := "f6e5d4c3b2a1"

		assert.True(t, ssc.CompareHashes(hash1, hash2))
		assert.False(t, ssc.CompareHashes(hash1, hash3))
		assert.False(t, ssc.CompareHashes(hash1, "short"))
	})

	t.Run("CompareTokens", func(t *testing.T) {
		token1 := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
		token2 := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
		token3 := "differentTokenHere"

		assert.True(t, ssc.CompareTokens(token1, token2))
		assert.False(t, ssc.CompareTokens(token1, token3))
	})
}

func TestSecurityAuditor(t *testing.T) {
	auditor := NewSecurityAuditor(100)

	t.Run("LogEvent", func(t *testing.T) {
		event := SecurityEvent{
			Type:    "test",
			Level:   SecurityLevelMedium,
			Message: "Test event",
		}

		auditor.LogEvent(event)
		events := auditor.GetEvents()

		assert.Len(t, events, 1)
		assert.Equal(t, "test", events[0].Type)
		assert.Equal(t, SecurityLevelMedium, events[0].Level)
		assert.Equal(t, "Test event", events[0].Message)
		assert.False(t, events[0].Timestamp.IsZero())
	})

	t.Run("LogAuthenticationEvent", func(t *testing.T) {
		auditor.LogAuthenticationEvent("user123", "login", "success", SecurityLevelLow)
		events := auditor.GetEventsByType("authentication")

		assert.Len(t, events, 1)
		assert.Equal(t, "user123", events[0].UserID)
		assert.Equal(t, "login", events[0].Operation)
		assert.Equal(t, "success", events[0].Outcome)
	})

	t.Run("LogSecurityViolation", func(t *testing.T) {
		context := map[string]interface{}{
			"ip_address": "192.168.1.100",
			"attempts":   5,
		}

		auditor.LogSecurityViolation("brute_force", "Multiple failed login attempts", context)
		events := auditor.GetEventsByLevel(SecurityLevelCritical)

		assert.Len(t, events, 1)
		assert.Equal(t, "security_violation", events[0].Type)
		assert.Equal(t, SecurityLevelCritical, events[0].Level)
		assert.Contains(t, events[0].Message, "brute_force")
		assert.NotNil(t, events[0].Context)
		assert.NotEmpty(t, events[0].StackTrace)
	})

	t.Run("EventHandler", func(t *testing.T) {
		handlerCalled := false
		var handlerEvent SecurityEvent

		auditor.SetEventHandler(func(event SecurityEvent) {
			handlerCalled = true
			handlerEvent = event
		})

		event := SecurityEvent{
			Type:    "handler_test",
			Level:   SecurityLevelHigh,
			Message: "Handler test event",
		}

		auditor.LogEvent(event)

		assert.True(t, handlerCalled)
		assert.Equal(t, "handler_test", handlerEvent.Type)
		assert.Equal(t, SecurityLevelHigh, handlerEvent.Level)
	})

	t.Run("MaxEvents", func(t *testing.T) {
		smallAuditor := NewSecurityAuditor(2)

		// Add 3 events
		for i := 0; i < 3; i++ {
			smallAuditor.LogEvent(SecurityEvent{
				Type:    "overflow_test",
				Level:   SecurityLevelLow,
				Message: "Event " + string(rune('1'+i)),
			})
		}

		events := smallAuditor.GetEvents()
		assert.Len(t, events, 2) // Should only keep the last 2
		assert.Equal(t, "Event 2", events[0].Message)
		assert.Equal(t, "Event 3", events[1].Message)
	})
}

func TestSecurityValidator(t *testing.T) {
	auditor := NewSecurityAuditor(100)
	validator := NewSecurityValidator(auditor)

	t.Run("ValidateKeySize_Valid", func(t *testing.T) {
		assert.NoError(t, validator.ValidateKeySize(32, "AES-256"))
		assert.NoError(t, validator.ValidateKeySize(16, "AES-128"))
		assert.NoError(t, validator.ValidateKeySize(256, "RSA"))
	})

	t.Run("ValidateKeySize_Invalid", func(t *testing.T) {
		err := validator.ValidateKeySize(8, "AES")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insecure key size")

		// Should log security violation
		violations := auditor.GetEventsByType("security_violation")
		assert.Len(t, violations, 1)
		assert.Contains(t, violations[0].Message, "weak_key")
	})

	t.Run("ValidatePasswordStrength_Valid", func(t *testing.T) {
		assert.NoError(t, validator.ValidatePasswordStrength("SecureP@ss1"))
		assert.NoError(t, validator.ValidatePasswordStrength("MyP@ssw0rd!"))
	})

	t.Run("ValidatePasswordStrength_TooShort", func(t *testing.T) {
		err := validator.ValidatePasswordStrength("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})

	t.Run("ValidatePasswordStrength_MissingTypes", func(t *testing.T) {
		err := validator.ValidatePasswordStrength("alllowercase")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required character types")
	})

	t.Run("ValidateTokenFormat", func(t *testing.T) {
		assert.NoError(t, validator.ValidateTokenFormat("valid-token-123", 10))

		err := validator.ValidateTokenFormat("short", 10)
		assert.Error(t, err)

		err = validator.ValidateTokenFormat("aaaaaaaaaa", 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient entropy")
	})
}

func TestSecurityContext(t *testing.T) {
	t.Run("IsExpired", func(t *testing.T) {
		ctx := &SecurityContext{
			UserID:    "user123",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.False(t, ctx.IsExpired())

		ctx.ExpiresAt = time.Now().Add(-time.Hour)
		assert.True(t, ctx.IsExpired())
	})

	t.Run("HasPermission", func(t *testing.T) {
		ctx := &SecurityContext{
			UserID:      "user123",
			Permissions: []string{"read", "write", "admin"},
		}

		assert.True(t, ctx.HasPermission("read"))
		assert.True(t, ctx.HasPermission("admin"))
		assert.False(t, ctx.HasPermission("delete"))
	})

	t.Run("Validate", func(t *testing.T) {
		ctx := &SecurityContext{
			UserID:    "user123",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		assert.NoError(t, ctx.Validate())

		ctx.UserID = ""
		assert.Error(t, ctx.Validate())

		ctx.UserID = "user123"
		ctx.ExpiresAt = time.Now().Add(-time.Hour)
		assert.Error(t, ctx.Validate())
	})
}

func TestGlobalFunctions(t *testing.T) {
	t.Run("ZeroBytes", func(t *testing.T) {
		data := []byte{1, 2, 3, 4}
		ZeroBytes(data)
		for _, b := range data {
			assert.Equal(t, byte(0), b)
		}
	})

	t.Run("ConstantTimeCompare", func(t *testing.T) {
		a := []byte{1, 2, 3}
		b := []byte{1, 2, 3}
		c := []byte{1, 2, 4}

		assert.Equal(t, 1, ConstantTimeCompare(a, b))
		assert.Equal(t, 0, ConstantTimeCompare(a, c))
	})

	t.Run("ConstantTimeEq", func(t *testing.T) {
		a := []byte{1, 2, 3}
		b := []byte{1, 2, 3}
		c := []byte{1, 2, 4}

		assert.True(t, ConstantTimeEq(a, b))
		assert.False(t, ConstantTimeEq(a, c))
	})

	t.Run("SecureAllocate", func(t *testing.T) {
		data := SecureAllocate(64)
		assert.Len(t, data, 64)
	})

	t.Run("FillSecureRandom", func(t *testing.T) {
		data := make([]byte, 32)
		err := FillSecureRandom(data)
		assert.NoError(t, err)

		// Verify it's not all zeros (highly unlikely with real random data)
		allZeros := true
		for _, b := range data {
			if b != 0 {
				allZeros = false
				break
			}
		}
		assert.False(t, allZeros)
	})

	t.Run("GenerateSecureRandom", func(t *testing.T) {
		data, err := GenerateSecureRandom(32)
		assert.NoError(t, err)
		assert.Len(t, data, 32)
	})
}

func TestRandomnessQuality(t *testing.T) {
	// Generate test data
	data, err := GenerateSecureRandom(1000)
	require.NoError(t, err)

	rt := NewRandomnessTest(data)

	t.Run("FrequencyTest", func(t *testing.T) {
		frequency := rt.FrequencyTest()
		// For good randomness, frequency should be close to 0.5
		assert.InDelta(t, 0.5, frequency, 0.1, "Frequency should be close to 0.5 for random data")
	})

	t.Run("SerialTest", func(t *testing.T) {
		correlation := rt.SerialTest()
		// For good randomness, serial correlation should be low
		assert.Less(t, correlation, 0.1, "Serial correlation should be low for random data")
	})
}

// Benchmark tests
func BenchmarkSecureMemory(b *testing.B) {
	sm := NewSecureMemory()

	b.Run("ZeroBytes", func(b *testing.B) {
		data := make([]byte, 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sm.ZeroBytes(data)
		}
	})

	b.Run("ConstantTimeCompare", func(b *testing.B) {
		a := make([]byte, 32)
		b32 := make([]byte, 32)
		rand.Read(a)
		rand.Read(b32)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sm.ConstantTimeCompare(a, b32)
		}
	})

	b.Run("SecureAllocate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := sm.SecureAllocate(256)
			_ = data
		}
	})
}

func BenchmarkSecureRandom(b *testing.B) {
	srg := NewSecureRandomGenerator()

	b.Run("Generate32Bytes", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := srg.Generate(32)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GenerateKey", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := srg.GenerateKey(32)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}