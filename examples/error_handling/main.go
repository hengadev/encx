package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

func main() {
	// This example demonstrates the new error handling capabilities
	
	fmt.Println("ENCX Error Handling Demonstration")
	fmt.Println("==================================")
	
	// Demonstrate error classification with various error types
	demonstrateErrorClassification()
	
	fmt.Println("\n--- Practical Error Handling Example ---")
	
	// Simulate a wrapped KMS error
	kmsError := fmt.Errorf("network timeout connecting to KMS: %w", encx.ErrKMSUnavailable)
	handleError("KMS Operation", kmsError)
	
	// Simulate a configuration error  
	configError := fmt.Errorf("invalid pepper configuration: %w", encx.ErrInvalidConfiguration)
	handleError("Configuration Validation", configError)
}

// handleError demonstrates how to handle errors using the new classification system
func handleError(operation string, err error) {
	fmt.Printf("Error in %s: %v\n", operation, err)
	
	// Use error classification helpers for precise error handling
	switch {
	case encx.IsRetryableError(err):
		fmt.Println("→ This is a retryable error - implementing retry logic...")
		handleRetryableError(err)
		
	case encx.IsConfigurationError(err):
		fmt.Println("→ This is a configuration error - checking setup...")
		handleConfigurationError(err)
		
	case encx.IsAuthError(err):
		fmt.Println("→ This is an authentication error - refreshing credentials...")
		handleAuthError(err)
		
	case encx.IsOperationError(err):
		fmt.Println("→ This is an operation error - checking data integrity...")
		handleOperationError(err)
		
	case encx.IsValidationError(err):
		fmt.Println("→ This is a validation error - checking input data...")
		handleValidationError(err)
		
	default:
		fmt.Println("→ Unknown error type - logging for investigation...")
		log.Printf("Unclassified error: %v", err)
	}
	
	// Check for specific error types using errors.Is()
	if errors.Is(err, encx.ErrKMSUnavailable) {
		fmt.Println("  Specific: KMS service is unavailable")
	} else if errors.Is(err, encx.ErrAuthenticationFailed) {
		fmt.Println("  Specific: Authentication failed")
	} else if errors.Is(err, encx.ErrInvalidConfiguration) {
		fmt.Println("  Specific: Configuration is invalid")
	}
}

func demonstrateErrorClassification() {
	fmt.Println("\n--- Error Classification Examples ---")
	
	// Create wrapped errors to demonstrate classification
	testErrors := []struct {
		name string
		err  error
	}{
		{"KMS Unavailable", fmt.Errorf("connection failed: %w", encx.ErrKMSUnavailable)},
		{"Auth Failed", fmt.Errorf("vault login: %w", encx.ErrAuthenticationFailed)},
		{"Bad Config", fmt.Errorf("validation: %w", encx.ErrInvalidConfiguration)},
		{"Encrypt Failed", fmt.Errorf("operation: %w", encx.ErrEncryptionFailed)},
		{"Type Error", fmt.Errorf("conversion: %w", encx.ErrTypeConversion)},
	}
	
	for _, test := range testErrors {
		fmt.Printf("\n%s: %v\n", test.name, test.err)
		classifyError(test.err)
	}
}

func classifyError(err error) {
	classifications := []string{}
	
	if encx.IsRetryableError(err) {
		classifications = append(classifications, "Retryable")
	}
	if encx.IsConfigurationError(err) {
		classifications = append(classifications, "Configuration")
	}
	if encx.IsAuthError(err) {
		classifications = append(classifications, "Authentication")
	}
	if encx.IsOperationError(err) {
		classifications = append(classifications, "Operation")
	}
	if encx.IsValidationError(err) {
		classifications = append(classifications, "Validation")
	}
	
	if len(classifications) == 0 {
		fmt.Println("  → No classification matched")
	} else {
		fmt.Printf("  → Classifications: %v\n", classifications)
	}
}

// Error handling strategies
func handleRetryableError(err error) {
	fmt.Println("  Strategy: Implement exponential backoff retry")
	// Implement retry logic here
}

func handleConfigurationError(err error) {
	fmt.Println("  Strategy: Validate configuration and environment")
	// Validate configuration here
}

func handleAuthError(err error) {
	fmt.Println("  Strategy: Refresh credentials and retry")
	// Refresh auth credentials here
}

func handleOperationError(err error) {
	fmt.Println("  Strategy: Check data integrity and key validity")
	// Check operation prerequisites here
}

func handleValidationError(err error) {
	fmt.Println("  Strategy: Validate input data format and types")
	// Validate input data here
}