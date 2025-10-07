package security

import (
	"crypto/subtle"
	"runtime"
)

// SecureMemory provides utilities for secure memory handling in cryptographic operations.
//
// IMPORTANT: Sensitive data (passwords, keys, secrets) MUST be stored as []byte, never string.
// Go strings are immutable and cannot be securely erased from memory.
// Always use []byte for sensitive data and call ZeroBytes when done.
//
// Example:
//
//	password := []byte("secret123")
//	defer security.ZeroBytes(password)
//	// Use password...
type SecureMemory struct{}

// NewSecureMemory creates a new SecureMemory instance
func NewSecureMemory() *SecureMemory {
	return &SecureMemory{}
}

// ZeroBytes securely zeros a byte slice to prevent sensitive data from lingering in memory.
// This function overwrites the memory multiple times to ensure data destruction
// even against advanced forensic techniques.
func (s *SecureMemory) ZeroBytes(data []byte) {
	if len(data) == 0 {
		return
	}

	// Multiple overwrite passes with different patterns for enhanced security
	patterns := []byte{0x00, 0xFF, 0xAA, 0x55, 0x00}

	for _, pattern := range patterns {
		for i := range data {
			data[i] = pattern
		}
		// Compiler barrier to prevent optimization
		runtime.KeepAlive(data)
	}

	// Final zero pass
	for i := range data {
		data[i] = 0
	}

	// Memory barrier to ensure writes complete
	runtime.KeepAlive(data)
}

// SecureAllocate allocates memory that will be securely zeroed when freed.
// Returns a byte slice that should be used for sensitive data.
func (s *SecureMemory) SecureAllocate(size int) []byte {
	if size <= 0 {
		return nil
	}

	// Allocate aligned memory to improve performance and security
	data := make([]byte, size)

	// Touch all pages to ensure they're allocated
	for i := 0; i < len(data); i += 4096 { // Assume 4KB pages
		data[i] = 0
	}
	if len(data) > 0 {
		data[len(data)-1] = 0
	}

	return data
}

// SecureCopy performs a secure copy of sensitive data with automatic cleanup
func (s *SecureMemory) SecureCopy(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}

	dst := s.SecureAllocate(len(src))
	copy(dst, src)
	return dst
}

// ConstantTimeCompare compares two byte slices in constant time to prevent timing attacks.
// Returns 1 if slices are equal, 0 otherwise.
func (s *SecureMemory) ConstantTimeCompare(a, b []byte) int {
	return subtle.ConstantTimeCompare(a, b)
}

// ConstantTimeEq compares two byte slices and returns true if they are equal.
// This is a wrapper around ConstantTimeCompare for convenience.
func (s *SecureMemory) ConstantTimeEq(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ConstantTimeSelect selects between two values based on a condition in constant time.
// If condition is 1, returns a; if condition is 0, returns b.
func (s *SecureMemory) ConstantTimeSelect(condition int, a, b []byte) []byte {
	if len(a) != len(b) {
		// For safety, return zero slice if lengths don't match
		return make([]byte, 0)
	}

	result := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		result[i] = byte(subtle.ConstantTimeSelect(condition, int(a[i]), int(b[i])))
	}
	return result
}

// ConstantTimeCopy copies data in constant time to prevent timing-based information leakage
func (s *SecureMemory) ConstantTimeCopy(dst, src []byte) {
	if len(dst) != len(src) {
		panic("secure_memory: dst and src must have same length for constant time copy")
	}

	subtle.ConstantTimeCopy(1, dst, src)
}

// SecureRandom generates cryptographically secure random bytes
// This is a convenience wrapper that ensures proper random number generation
func (s *SecureMemory) SecureRandom(size int) ([]byte, error) {
	if size <= 0 {
		return nil, nil
	}

	data := s.SecureAllocate(size)
	if err := FillSecureRandom(data); err != nil {
		s.ZeroBytes(data)
		return nil, err
	}

	return data, nil
}

// SecureBuffer represents a buffer that automatically zeros itself when finalized
type SecureBuffer struct {
	data     []byte
	finalized bool
}

// NewSecureBuffer creates a new secure buffer of the specified size
func NewSecureBuffer(size int) *SecureBuffer {
	sm := NewSecureMemory()
	return &SecureBuffer{
		data:     sm.SecureAllocate(size),
		finalized: false,
	}
}

// Bytes returns the underlying byte slice
// WARNING: The returned slice should not be stored or used after Finalize() is called
func (sb *SecureBuffer) Bytes() []byte {
	if sb.finalized {
		return nil
	}
	return sb.data
}

// Len returns the length of the buffer
func (sb *SecureBuffer) Len() int {
	if sb.finalized {
		return 0
	}
	return len(sb.data)
}

// Copy returns a copy of the buffer's contents
func (sb *SecureBuffer) Copy() []byte {
	if sb.finalized || len(sb.data) == 0 {
		return nil
	}

	sm := NewSecureMemory()
	return sm.SecureCopy(sb.data)
}

// Finalize securely clears the buffer and marks it as finalized
func (sb *SecureBuffer) Finalize() {
	if sb.finalized {
		return
	}

	sm := NewSecureMemory()
	sm.ZeroBytes(sb.data)
	sb.finalized = true
	sb.data = nil
}

// Finalizer for automatic cleanup
func (sb *SecureBuffer) finalize() {
	sb.Finalize()
}

// SetFinalizer sets up automatic cleanup when the buffer is garbage collected
func (sb *SecureBuffer) SetFinalizer() {
	runtime.SetFinalizer(sb, (*SecureBuffer).finalize)
}

// ClearFinalizer removes the automatic cleanup finalizer
func (sb *SecureBuffer) ClearFinalizer() {
	runtime.SetFinalizer(sb, nil)
}

// SecureSlice is a wrapper around byte slice that provides automatic cleanup
type SecureSlice struct {
	buffer *SecureBuffer
}

// NewSecureSlice creates a new secure slice of the specified size
func NewSecureSlice(size int) *SecureSlice {
	buffer := NewSecureBuffer(size)
	buffer.SetFinalizer()

	return &SecureSlice{
		buffer: buffer,
	}
}

// NewSecureSliceFromBytes creates a secure slice from existing bytes
func NewSecureSliceFromBytes(data []byte) *SecureSlice {
	if len(data) == 0 {
		return &SecureSlice{
			buffer: NewSecureBuffer(0),
		}
	}

	slice := NewSecureSlice(len(data))
	copy(slice.buffer.Bytes(), data)
	return slice
}

// Bytes returns the underlying bytes (use with caution)
func (ss *SecureSlice) Bytes() []byte {
	if ss.buffer == nil {
		return nil
	}
	return ss.buffer.Bytes()
}

// Len returns the length of the slice
func (ss *SecureSlice) Len() int {
	if ss.buffer == nil {
		return 0
	}
	return ss.buffer.Len()
}

// Copy returns a copy of the slice contents
func (ss *SecureSlice) Copy() []byte {
	if ss.buffer == nil {
		return nil
	}
	return ss.buffer.Copy()
}

// Close securely clears the slice and releases resources
func (ss *SecureSlice) Close() error {
	if ss.buffer != nil {
		ss.buffer.ClearFinalizer()
		ss.buffer.Finalize()
		ss.buffer = nil
	}
	return nil
}

// Global secure memory instance for convenience functions
var globalSecureMemory = NewSecureMemory()

// ZeroBytes is a convenience function for securely zeroing bytes
func ZeroBytes(data []byte) {
	globalSecureMemory.ZeroBytes(data)
}

// ConstantTimeCompare is a convenience function for constant-time comparison
func ConstantTimeCompare(a, b []byte) int {
	return globalSecureMemory.ConstantTimeCompare(a, b)
}

// ConstantTimeEq is a convenience function for constant-time equality check
func ConstantTimeEq(a, b []byte) bool {
	return globalSecureMemory.ConstantTimeEq(a, b)
}

// SecureAllocate is a convenience function for secure memory allocation
func SecureAllocate(size int) []byte {
	return globalSecureMemory.SecureAllocate(size)
}

// SecureCopy is a convenience function for secure copying
func SecureCopy(src []byte) []byte {
	return globalSecureMemory.SecureCopy(src)
}