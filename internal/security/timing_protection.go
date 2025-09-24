package security

import (
	"crypto/subtle"
	"time"
)

// TimingProtection provides utilities to protect against timing attacks
type TimingProtection struct {
	baseDelay time.Duration
}

// NewTimingProtection creates a new timing protection instance
func NewTimingProtection(baseDelay time.Duration) *TimingProtection {
	if baseDelay <= 0 {
		baseDelay = time.Microsecond * 100 // Default 100μs
	}

	return &TimingProtection{
		baseDelay: baseDelay,
	}
}

// ConstantTimeOperation executes an operation in constant time by adding delays
func (tp *TimingProtection) ConstantTimeOperation(operation func() error) error {
	start := time.Now()
	err := operation()
	elapsed := time.Since(start)

	// Add delay to normalize timing
	if elapsed < tp.baseDelay {
		time.Sleep(tp.baseDelay - elapsed)
	}

	return err
}

// ConstantTimeSelect selects between two operations based on condition
func (tp *TimingProtection) ConstantTimeSelect(condition bool, trueOp, falseOp func() error) error {
	start := time.Now()

	var err error
	if condition {
		err = trueOp()
		// Execute dummy operation to maintain timing
		tp.dummyOperation()
	} else {
		tp.dummyOperation()
		err = falseOp()
	}

	elapsed := time.Since(start)
	if elapsed < tp.baseDelay {
		time.Sleep(tp.baseDelay - elapsed)
	}

	return err
}

// dummyOperation performs a dummy computation to maintain constant timing
func (tp *TimingProtection) dummyOperation() {
	// Perform some computation that takes similar time to real operations
	dummy := make([]byte, 32)
	for i := 0; i < len(dummy); i++ {
		dummy[i] = byte(i * 7) // Simple computation
	}
	// Prevent compiler optimization
	_ = dummy
}

// SecureStringComparison provides constant-time string comparison utilities
type SecureStringComparison struct{}

// NewSecureStringComparison creates a new secure string comparison instance
func NewSecureStringComparison() *SecureStringComparison {
	return &SecureStringComparison{}
}

// CompareStrings compares two strings in constant time
func (ssc *SecureStringComparison) CompareStrings(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// CompareHashes compares two hash values in constant time
func (ssc *SecureStringComparison) CompareHashes(expected, actual string) bool {
	// Ensure both hashes are the same length to prevent length-based timing attacks
	if len(expected) != len(actual) {
		// Perform a dummy comparison to maintain timing
		dummy := make([]byte, len(expected)+len(actual))
		subtle.ConstantTimeCompare(dummy[:len(expected)], dummy[len(expected):])
		return false
	}

	return ssc.CompareStrings(expected, actual)
}

// CompareTokens compares authentication tokens in constant time
func (ssc *SecureStringComparison) CompareTokens(expected, actual string) bool {
	return ssc.CompareHashes(expected, actual)
}

// SideChannelProtection provides protection against side-channel attacks
type SideChannelProtection struct {
	noise []byte
}

// NewSideChannelProtection creates a new side-channel protection instance
func NewSideChannelProtection() *SideChannelProtection {
	// Generate noise data for masking operations
	noise, _ := GenerateSecureRandom(1024)

	return &SideChannelProtection{
		noise: noise,
	}
}

// MaskOperation performs an operation with side-channel masking
func (scp *SideChannelProtection) MaskOperation(data []byte, operation func([]byte) []byte) []byte {
	if len(data) == 0 {
		return operation(data)
	}

	// Create mask from noise
	mask := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		mask[i] = scp.noise[i%len(scp.noise)]
	}

	// Apply mask
	maskedData := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		maskedData[i] = data[i] ^ mask[i]
	}

	// Perform operation on masked data
	maskedResult := operation(maskedData)

	// Remove mask from result
	result := make([]byte, len(maskedResult))
	for i := 0; i < len(maskedResult); i++ {
		result[i] = maskedResult[i] ^ mask[i%len(mask)]
	}

	return result
}

// BlindOperation performs blinded computation to prevent side-channel attacks
func (scp *SideChannelProtection) BlindOperation(data []byte, operation func([]byte) ([]byte, error)) ([]byte, error) {
	if len(data) == 0 {
		return operation(data)
	}

	// Generate blinding factor
	blindingFactor, err := GenerateSecureRandom(len(data))
	if err != nil {
		return nil, err
	}

	// Blind the data
	blindedData := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		blindedData[i] = data[i] ^ blindingFactor[i]
	}

	// Perform operation on blinded data
	blindedResult, err := operation(blindedData)
	if err != nil {
		return nil, err
	}

	// Unblind the result
	result := make([]byte, len(blindedResult))
	for i := 0; i < len(blindedResult); i++ {
		result[i] = blindedResult[i] ^ blindingFactor[i%len(blindingFactor)]
	}

	return result, nil
}

// CacheTimingProtection protects against cache timing attacks
type CacheTimingProtection struct {
	lookupTable [][]byte
	tableSize   int
}

// NewCacheTimingProtection creates protection against cache timing attacks
func NewCacheTimingProtection(tableSize int) *CacheTimingProtection {
	if tableSize <= 0 {
		tableSize = 256 // Default table size
	}

	// Pre-populate lookup table to warm cache
	lookupTable := make([][]byte, tableSize)
	for i := 0; i < tableSize; i++ {
		lookupTable[i], _ = GenerateSecureRandom(32)
	}

	return &CacheTimingProtection{
		lookupTable: lookupTable,
		tableSize:   tableSize,
	}
}

// SecureLookup performs a lookup that accesses all table entries to prevent cache timing
func (ctp *CacheTimingProtection) SecureLookup(index int) []byte {
	if index < 0 || index >= ctp.tableSize {
		return nil
	}

	result := make([]byte, 32)

	// Access all entries to prevent cache timing
	for i := 0; i < ctp.tableSize; i++ {
		// Use constant time select to choose the correct entry
		condition := subtle.ConstantTimeEq(int32(i), int32(index))
		for j := 0; j < 32; j++ {
			result[j] = byte(subtle.ConstantTimeSelect(condition, int(ctp.lookupTable[i][j]), int(result[j])))
		}
	}

	return result
}

// PowerAnalysisProtection protects against power analysis attacks
type PowerAnalysisProtection struct {
	dummyOperations int
}

// NewPowerAnalysisProtection creates protection against power analysis attacks
func NewPowerAnalysisProtection(dummyOps int) *PowerAnalysisProtection {
	if dummyOps <= 0 {
		dummyOps = 10 // Default number of dummy operations
	}

	return &PowerAnalysisProtection{
		dummyOperations: dummyOps,
	}
}

// ProtectedComputation performs computation with power analysis protection
func (pap *PowerAnalysisProtection) ProtectedComputation(data []byte, computation func([]byte) []byte) []byte {
	// Perform dummy operations before real computation
	dummy := make([]byte, len(data))
	for i := 0; i < pap.dummyOperations; i++ {
		copy(dummy, data)
		// Perform dummy computation
		for j := 0; j < len(dummy); j++ {
			dummy[j] ^= byte(i * j)
		}
	}

	// Perform real computation
	result := computation(data)

	// Perform dummy operations after real computation
	for i := 0; i < pap.dummyOperations; i++ {
		copy(dummy, result)
		// Perform dummy computation
		for j := 0; j < len(dummy); j++ {
			dummy[j] ^= byte(i * j)
		}
	}

	// Clear dummy data
	ZeroBytes(dummy)

	return result
}

// SecureComparison provides various secure comparison utilities
type SecureComparison struct {
	timing *TimingProtection
}

// NewSecureComparison creates a new secure comparison instance
func NewSecureComparison() *SecureComparison {
	return &SecureComparison{
		timing: NewTimingProtection(time.Microsecond * 50),
	}
}

// ComparePasswords compares passwords securely
func (sc *SecureComparison) ComparePasswords(expected, actual string) bool {
	var result bool
	sc.timing.ConstantTimeOperation(func() error {
		result = subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
		return nil
	})
	return result
}

// CompareAPIKeys compares API keys securely
func (sc *SecureComparison) CompareAPIKeys(expected, actual string) bool {
	return sc.ComparePasswords(expected, actual)
}

// CompareSessionTokens compares session tokens securely
func (sc *SecureComparison) CompareSessionTokens(expected, actual string) bool {
	return sc.ComparePasswords(expected, actual)
}

// Global instances for convenience
var (
	globalTimingProtection      = NewTimingProtection(time.Microsecond * 100)
	globalSecureComparison      = NewSecureComparison()
	globalSideChannelProtection = NewSideChannelProtection()
)

// ProtectAgainstTiming performs an operation with timing attack protection
func ProtectAgainstTiming(operation func() error) error {
	return globalTimingProtection.ConstantTimeOperation(operation)
}

// SecureCompareStrings compares strings in constant time
func SecureCompareStrings(a, b string) bool {
	return globalSecureComparison.ComparePasswords(a, b)
}

// SecureCompareBytes compares byte slices in constant time
func SecureCompareBytes(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// MaskSensitiveOperation performs an operation with side-channel masking
func MaskSensitiveOperation(data []byte, operation func([]byte) []byte) []byte {
	return globalSideChannelProtection.MaskOperation(data, operation)
}