package security

import (
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"
)

// SecureRandomGenerator provides cryptographically secure random number generation
type SecureRandomGenerator struct {
	reader io.Reader
	mutex  sync.Mutex
}

// NewSecureRandomGenerator creates a new secure random generator
func NewSecureRandomGenerator() *SecureRandomGenerator {
	return &SecureRandomGenerator{
		reader: rand.Reader,
	}
}

// Read generates secure random bytes
func (srg *SecureRandomGenerator) Read(b []byte) (int, error) {
	srg.mutex.Lock()
	defer srg.mutex.Unlock()

	n, err := srg.reader.Read(b)
	if err != nil {
		return n, fmt.Errorf("secure random generation failed: %w", err)
	}

	return n, nil
}

// Generate generates a slice of secure random bytes
func (srg *SecureRandomGenerator) Generate(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	data := make([]byte, size)
	_, err := srg.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GenerateKey generates a cryptographic key of specified size
func (srg *SecureRandomGenerator) GenerateKey(keySize int) ([]byte, error) {
	// Validate key size for common cryptographic algorithms
	validKeySizes := map[int]bool{
		16: true, // AES-128
		24: true, // AES-192
		32: true, // AES-256
		64: true, // Large keys (e.g., HMAC)
	}

	if !validKeySizes[keySize] && keySize < 16 {
		return nil, fmt.Errorf("insecure key size: %d bytes (minimum 16 bytes)", keySize)
	}

	return srg.Generate(keySize)
}

// GenerateNonce generates a cryptographically secure nonce
func (srg *SecureRandomGenerator) GenerateNonce(size int) ([]byte, error) {
	if size < 12 { // GCM recommends 12 bytes minimum
		return nil, fmt.Errorf("nonce size too small: %d bytes (minimum 12 bytes)", size)
	}

	return srg.Generate(size)
}

// GenerateSalt generates a cryptographically secure salt
func (srg *SecureRandomGenerator) GenerateSalt(size int) ([]byte, error) {
	if size < 16 { // Minimum recommended salt size
		return nil, fmt.Errorf("salt size too small: %d bytes (minimum 16 bytes)", size)
	}

	return srg.Generate(size)
}

// GeneratePassword generates a secure random password with specified character sets
func (srg *SecureRandomGenerator) GeneratePassword(length int, includeSpecial bool) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password too short: %d characters (minimum 8)", length)
	}

	// Character sets
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	special := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	charset := lowercase + uppercase + digits
	if includeSpecial {
		charset += special
	}

	password := make([]byte, length)
	randomBytes, err := srg.Generate(length)
	if err != nil {
		return "", err
	}

	for i, b := range randomBytes {
		password[i] = charset[int(b)%len(charset)]
	}

	// Ensure password has at least one character from each required set
	if length >= 4 {
		// Force inclusion of required character types
		minPositions, err := srg.Generate(4)
		if err != nil {
			return "", err
		}

		password[int(minPositions[0])%length] = lowercase[int(minPositions[0])%len(lowercase)]
		password[int(minPositions[1])%length] = uppercase[int(minPositions[1])%len(uppercase)]
		password[int(minPositions[2])%length] = digits[int(minPositions[2])%len(digits)]

		if includeSpecial && length > 4 {
			password[int(minPositions[3])%length] = special[int(minPositions[3])%len(special)]
		}
	}

	return string(password), nil
}

// EntropyPool provides additional entropy source for critical operations
type EntropyPool struct {
	pool   []byte
	mutex  sync.RWMutex
	size   int
	filled int
}

// NewEntropyPool creates a new entropy pool
func NewEntropyPool(size int) *EntropyPool {
	if size < 1024 {
		size = 1024 // Minimum pool size
	}

	return &EntropyPool{
		pool: make([]byte, size),
		size: size,
	}
}

// Fill fills the entropy pool with random data
func (ep *EntropyPool) Fill() error {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	_, err := rand.Read(ep.pool)
	if err != nil {
		return fmt.Errorf("failed to fill entropy pool: %w", err)
	}

	ep.filled = len(ep.pool)
	return nil
}

// Extract extracts random bytes from the entropy pool
func (ep *EntropyPool) Extract(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid extract size: %d", size)
	}

	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	if ep.filled < size {
		// Refill pool if insufficient entropy
		if err := ep.refill(); err != nil {
			return nil, err
		}
	}

	// Extract from random position
	startPos := 0
	if ep.filled > size {
		posBytes := make([]byte, 4)
		_, err := rand.Read(posBytes)
		if err != nil {
			return nil, err
		}
		startPos = int(uint32(posBytes[0])<<24|uint32(posBytes[1])<<16|uint32(posBytes[2])<<8|uint32(posBytes[3])) % (ep.filled - size)
	}

	result := make([]byte, size)
	copy(result, ep.pool[startPos:startPos+size])

	// Mark extracted bytes as used by XORing with new random data
	newData := make([]byte, size)
	rand.Read(newData)
	for i := 0; i < size; i++ {
		ep.pool[startPos+i] ^= newData[i]
	}

	ep.filled -= size
	if ep.filled < ep.size/4 {
		// Async refill when pool is getting low
		go ep.Fill()
	}

	return result, nil
}

// refill is an internal method to refill the entropy pool
func (ep *EntropyPool) refill() error {
	_, err := rand.Read(ep.pool)
	if err != nil {
		return err
	}
	ep.filled = len(ep.pool)
	return nil
}

// RandomnessTest performs basic randomness quality tests
type RandomnessTest struct {
	samples []byte
	size    int
}

// NewRandomnessTest creates a new randomness test with sample data
func NewRandomnessTest(data []byte) *RandomnessTest {
	return &RandomnessTest{
		samples: data,
		size:    len(data),
	}
}

// FrequencyTest performs a basic frequency test for randomness
func (rt *RandomnessTest) FrequencyTest() float64 {
	if rt.size == 0 {
		return 0.0
	}

	ones := 0
	for _, b := range rt.samples {
		for i := 0; i < 8; i++ {
			if (b>>i)&1 == 1 {
				ones++
			}
		}
	}

	totalBits := rt.size * 8
	frequency := float64(ones) / float64(totalBits)

	// Perfect randomness would have frequency close to 0.5
	return frequency
}

// SerialTest performs a basic serial correlation test
func (rt *RandomnessTest) SerialTest() float64 {
	if rt.size < 2 {
		return 0.0
	}

	correlations := 0
	for i := 0; i < rt.size-1; i++ {
		if rt.samples[i] == rt.samples[i+1] {
			correlations++
		}
	}

	// Lower correlation indicates better randomness
	return float64(correlations) / float64(rt.size-1)
}

// Global secure random generator
var globalSecureRandom = NewSecureRandomGenerator()

// FillSecureRandom fills a byte slice with cryptographically secure random data
func FillSecureRandom(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	_, err := globalSecureRandom.Read(data)
	return err
}

// GenerateSecureRandom generates cryptographically secure random bytes
func GenerateSecureRandom(size int) ([]byte, error) {
	return globalSecureRandom.Generate(size)
}

// GenerateSecureKey generates a cryptographic key
func GenerateSecureKey(keySize int) ([]byte, error) {
	return globalSecureRandom.GenerateKey(keySize)
}

// GenerateSecureNonce generates a cryptographically secure nonce
func GenerateSecureNonce(size int) ([]byte, error) {
	return globalSecureRandom.GenerateNonce(size)
}

// GenerateSecureSalt generates a cryptographically secure salt
func GenerateSecureSalt(size int) ([]byte, error) {
	return globalSecureRandom.GenerateSalt(size)
}

// GenerateSecurePassword generates a secure random password
func GenerateSecurePassword(length int, includeSpecial bool) (string, error) {
	return globalSecureRandom.GeneratePassword(length, includeSpecial)
}

// SecureRandomSource provides a time-seeded random source for additional entropy
type SecureRandomSource struct {
	lastTime time.Time
	counter  uint64
	mutex    sync.Mutex
}

// NewSecureRandomSource creates a new secure random source
func NewSecureRandomSource() *SecureRandomSource {
	return &SecureRandomSource{
		lastTime: time.Now(),
		counter:  0,
	}
}

// Seed provides additional entropy based on timing and counter
func (srs *SecureRandomSource) Seed() []byte {
	srs.mutex.Lock()
	defer srs.mutex.Unlock()

	now := time.Now()
	timeDiff := now.Sub(srs.lastTime).Nanoseconds()
	srs.lastTime = now
	srs.counter++

	// Combine timing and counter for additional entropy
	seed := make([]byte, 16)

	// Time-based entropy
	timeBytes := uint64(timeDiff)
	for i := 0; i < 8; i++ {
		seed[i] = byte(timeBytes >> (i * 8))
	}

	// Counter-based entropy
	for i := 0; i < 8; i++ {
		seed[i+8] = byte(srs.counter >> (i * 8))
	}

	return seed
}