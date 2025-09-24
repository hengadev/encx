package security

import (
	"context"
	"crypto/subtle"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// SecurityLevel represents the security level for operations
type SecurityLevel int

const (
	SecurityLevelLow SecurityLevel = iota
	SecurityLevelMedium
	SecurityLevelHigh
	SecurityLevelCritical
)

// String returns the string representation of the security level
func (sl SecurityLevel) String() string {
	switch sl {
	case SecurityLevelLow:
		return "LOW"
	case SecurityLevelMedium:
		return "MEDIUM"
	case SecurityLevelHigh:
		return "HIGH"
	case SecurityLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Type        string                 `json:"type"`
	Level       SecurityLevel          `json:"level"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Source      string                 `json:"source,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Outcome     string                 `json:"outcome,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
}

// SecurityAuditor provides security auditing and monitoring capabilities
type SecurityAuditor struct {
	events       []SecurityEvent
	eventHandler func(SecurityEvent)
	maxEvents    int
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(maxEvents int) *SecurityAuditor {
	if maxEvents <= 0 {
		maxEvents = 1000 // Default maximum events
	}

	return &SecurityAuditor{
		events:    make([]SecurityEvent, 0, maxEvents),
		maxEvents: maxEvents,
	}
}

// SetEventHandler sets a custom event handler for security events
func (sa *SecurityAuditor) SetEventHandler(handler func(SecurityEvent)) {
	sa.eventHandler = handler
}

// LogEvent logs a security event
func (sa *SecurityAuditor) LogEvent(event SecurityEvent) {
	event.Timestamp = time.Now().UTC()
	event.Source = sa.getCallerInfo(2)

	// Add to events list
	if len(sa.events) >= sa.maxEvents {
		// Remove oldest event
		sa.events = sa.events[1:]
	}
	sa.events = append(sa.events, event)

	// Call event handler if set
	if sa.eventHandler != nil {
		sa.eventHandler(event)
	}
}

// LogAuthenticationEvent logs an authentication-related event
func (sa *SecurityAuditor) LogAuthenticationEvent(userID, operation, outcome string, level SecurityLevel) {
	event := SecurityEvent{
		Type:      "authentication",
		Level:     level,
		Message:   fmt.Sprintf("Authentication %s for user %s: %s", operation, userID, outcome),
		UserID:    userID,
		Operation: operation,
		Outcome:   outcome,
	}
	sa.LogEvent(event)
}

// LogAuthorizationEvent logs an authorization-related event
func (sa *SecurityAuditor) LogAuthorizationEvent(userID, resource, operation, outcome string, level SecurityLevel) {
	event := SecurityEvent{
		Type:      "authorization",
		Level:     level,
		Message:   fmt.Sprintf("Authorization %s for user %s on resource %s: %s", operation, userID, resource, outcome),
		UserID:    userID,
		Operation: operation,
		Resource:  resource,
		Outcome:   outcome,
	}
	sa.LogEvent(event)
}

// LogCryptoEvent logs a cryptographic operation event
func (sa *SecurityAuditor) LogCryptoEvent(operation, outcome string, duration time.Duration, level SecurityLevel) {
	event := SecurityEvent{
		Type:      "cryptographic",
		Level:     level,
		Message:   fmt.Sprintf("Cryptographic operation %s: %s", operation, outcome),
		Operation: operation,
		Outcome:   outcome,
		Duration:  duration,
	}
	sa.LogEvent(event)
}

// LogDataAccessEvent logs a data access event
func (sa *SecurityAuditor) LogDataAccessEvent(userID, resource, operation, outcome string, level SecurityLevel) {
	event := SecurityEvent{
		Type:      "data_access",
		Level:     level,
		Message:   fmt.Sprintf("Data access %s by user %s on resource %s: %s", operation, userID, resource, outcome),
		UserID:    userID,
		Operation: operation,
		Resource:  resource,
		Outcome:   outcome,
	}
	sa.LogEvent(event)
}

// LogSecurityViolation logs a security violation
func (sa *SecurityAuditor) LogSecurityViolation(violationType, description string, context map[string]interface{}) {
	event := SecurityEvent{
		Type:       "security_violation",
		Level:      SecurityLevelCritical,
		Message:    fmt.Sprintf("Security violation detected: %s - %s", violationType, description),
		Context:    context,
		StackTrace: sa.getStackTrace(),
	}
	sa.LogEvent(event)
}

// GetEvents returns all logged events
func (sa *SecurityAuditor) GetEvents() []SecurityEvent {
	return sa.events
}

// GetEventsByLevel returns events filtered by security level
func (sa *SecurityAuditor) GetEventsByLevel(level SecurityLevel) []SecurityEvent {
	var filtered []SecurityEvent
	for _, event := range sa.events {
		if event.Level == level {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetEventsByType returns events filtered by type
func (sa *SecurityAuditor) GetEventsByType(eventType string) []SecurityEvent {
	var filtered []SecurityEvent
	for _, event := range sa.events {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// getCallerInfo gets information about the caller
func (sa *SecurityAuditor) getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	// Extract just the filename, not the full path
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]
	return fmt.Sprintf("%s:%d", filename, line)
}

// getStackTrace gets the current stack trace
func (sa *SecurityAuditor) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// SecurityValidator provides various security validation utilities
type SecurityValidator struct {
	auditor *SecurityAuditor
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(auditor *SecurityAuditor) *SecurityValidator {
	return &SecurityValidator{
		auditor: auditor,
	}
}

// ValidateKeySize validates cryptographic key sizes
func (sv *SecurityValidator) ValidateKeySize(keySize int, algorithm string) error {
	minKeySizes := map[string]int{
		"AES":      16, // AES-128 minimum
		"AES-128":  16,
		"AES-192":  24,
		"AES-256":  32,
		"HMAC":     16, // Minimum for HMAC
		"RSA":      256, // 2048-bit minimum (256 bytes)
		"ECDSA":    32,  // P-256 minimum
		"ChaCha20": 32,
	}

	minSize, exists := minKeySizes[strings.ToUpper(algorithm)]
	if !exists {
		minSize = 16 // Default minimum
	}

	if keySize < minSize {
		err := fmt.Errorf("insecure key size %d for algorithm %s (minimum: %d)", keySize, algorithm, minSize)
		if sv.auditor != nil {
			sv.auditor.LogSecurityViolation("weak_key", err.Error(), map[string]interface{}{
				"algorithm": algorithm,
				"key_size":  keySize,
				"min_size":  minSize,
			})
		}
		return err
	}

	return nil
}

// ValidatePasswordStrength validates password strength
func (sv *SecurityValidator) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		err := fmt.Errorf("password too short: %d characters (minimum 8)", len(password))
		if sv.auditor != nil {
			sv.auditor.LogSecurityViolation("weak_password", err.Error(), map[string]interface{}{
				"length": len(password),
			})
		}
		return err
	}

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

	missing := []string{}
	if !hasUpper {
		missing = append(missing, "uppercase")
	}
	if !hasLower {
		missing = append(missing, "lowercase")
	}
	if !hasDigit {
		missing = append(missing, "digit")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		err := fmt.Errorf("password missing required character types: %s", strings.Join(missing, ", "))
		if sv.auditor != nil {
			sv.auditor.LogSecurityViolation("weak_password", err.Error(), map[string]interface{}{
				"missing_types": missing,
			})
		}
		return err
	}

	return nil
}

// ValidateTokenFormat validates token format and entropy
func (sv *SecurityValidator) ValidateTokenFormat(token string, minLength int) error {
	if len(token) < minLength {
		err := fmt.Errorf("token too short: %d characters (minimum %d)", len(token), minLength)
		if sv.auditor != nil {
			sv.auditor.LogSecurityViolation("weak_token", err.Error(), map[string]interface{}{
				"length":     len(token),
				"min_length": minLength,
			})
		}
		return err
	}

	// Basic entropy check - ensure it's not all the same character
	if len(token) > 0 {
		firstChar := token[0]
		allSame := true
		for _, char := range token {
			if char != firstChar {
				allSame = false
				break
			}
		}

		if allSame {
			err := fmt.Errorf("token has insufficient entropy (all characters are the same)")
			if sv.auditor != nil {
				sv.auditor.LogSecurityViolation("weak_token", err.Error(), map[string]interface{}{
					"token_length": len(token),
				})
			}
			return err
		}
	}

	return nil
}

// SecurityContext provides security context for operations
type SecurityContext struct {
	UserID        string            `json:"user_id"`
	SessionID     string            `json:"session_id"`
	IPAddress     string            `json:"ip_address"`
	UserAgent     string            `json:"user_agent"`
	Permissions   []string          `json:"permissions"`
	SecurityLevel SecurityLevel     `json:"security_level"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at"`
}

// IsExpired checks if the security context has expired
func (sc *SecurityContext) IsExpired() bool {
	return !sc.ExpiresAt.IsZero() && time.Now().After(sc.ExpiresAt)
}

// HasPermission checks if the context has a specific permission
func (sc *SecurityContext) HasPermission(permission string) bool {
	for _, perm := range sc.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// Validate validates the security context
func (sc *SecurityContext) Validate() error {
	if sc.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if sc.IsExpired() {
		return fmt.Errorf("security context has expired")
	}

	return nil
}

// SecureOperationTracker tracks sensitive operations for audit purposes
type SecureOperationTracker struct {
	operations map[string]*OperationMetrics
	auditor    *SecurityAuditor
}

// OperationMetrics tracks metrics for secure operations
type OperationMetrics struct {
	Count          int64         `json:"count"`
	SuccessCount   int64         `json:"success_count"`
	FailureCount   int64         `json:"failure_count"`
	TotalDuration  time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	LastExecution  time.Time     `json:"last_execution"`
}

// NewSecureOperationTracker creates a new operation tracker
func NewSecureOperationTracker(auditor *SecurityAuditor) *SecureOperationTracker {
	return &SecureOperationTracker{
		operations: make(map[string]*OperationMetrics),
		auditor:    auditor,
	}
}

// TrackOperation tracks a secure operation
func (sot *SecureOperationTracker) TrackOperation(ctx context.Context, operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Update metrics
	if sot.operations[operation] == nil {
		sot.operations[operation] = &OperationMetrics{}
	}

	metrics := sot.operations[operation]
	metrics.Count++
	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.Count)
	metrics.LastExecution = start

	outcome := "success"
	if err != nil {
		metrics.FailureCount++
		outcome = "failure"
	} else {
		metrics.SuccessCount++
	}

	// Log to auditor
	if sot.auditor != nil {
		level := SecurityLevelLow
		if err != nil {
			level = SecurityLevelMedium
		}
		sot.auditor.LogCryptoEvent(operation, outcome, duration, level)
	}

	return err
}

// GetOperationMetrics returns metrics for a specific operation
func (sot *SecureOperationTracker) GetOperationMetrics(operation string) *OperationMetrics {
	return sot.operations[operation]
}

// GetAllMetrics returns metrics for all operations
func (sot *SecureOperationTracker) GetAllMetrics() map[string]*OperationMetrics {
	result := make(map[string]*OperationMetrics)
	for k, v := range sot.operations {
		result[k] = v
	}
	return result
}

// Global security auditor and validator
var (
	globalSecurityAuditor   = NewSecurityAuditor(10000)
	globalSecurityValidator = NewSecurityValidator(globalSecurityAuditor)
	globalOperationTracker  = NewSecureOperationTracker(globalSecurityAuditor)
)

// LogSecurityEvent logs a security event using the global auditor
func LogSecurityEvent(eventType, message string, level SecurityLevel) {
	event := SecurityEvent{
		Type:    eventType,
		Level:   level,
		Message: message,
	}
	globalSecurityAuditor.LogEvent(event)
}

// ValidateKeySize validates a key size using the global validator
func ValidateKeySize(keySize int, algorithm string) error {
	return globalSecurityValidator.ValidateKeySize(keySize, algorithm)
}

// ValidatePasswordStrength validates password strength using the global validator
func ValidatePasswordStrength(password string) error {
	return globalSecurityValidator.ValidatePasswordStrength(password)
}

// TrackSecureOperation tracks a secure operation using the global tracker
func TrackSecureOperation(ctx context.Context, operation string, fn func() error) error {
	return globalOperationTracker.TrackOperation(ctx, operation, fn)
}

// SetGlobalEventHandler sets a global event handler for security events
func SetGlobalEventHandler(handler func(SecurityEvent)) {
	globalSecurityAuditor.SetEventHandler(handler)
}