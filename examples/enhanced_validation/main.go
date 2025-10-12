package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hengadev/encx"
)

func main() {
	fmt.Println("=== Enhanced Input Validation Demo ===")
	fmt.Println("Demonstrating comprehensive configuration validation in ENCX v0.6.0+")
	fmt.Println("Note: This example shows the new environment-based API")
	fmt.Println()

	ctx := context.Background()
	kms := encx.NewSimpleTestKMS()
	secretStore := encx.NewInMemorySecretStore()

	// Example 1: Missing required environment variables
	fmt.Println("1. Testing missing ENCX_KEK_ALIAS environment variable...")
	os.Unsetenv("ENCX_KEK_ALIAS")
	os.Unsetenv("ENCX_PEPPER_ALIAS")
	_, err := encx.NewCryptoFromEnv(ctx, kms, secretStore)
	if err != nil {
		fmt.Printf("❌ Correctly caught missing KEK alias: %v\n", err)
	}
	fmt.Println()

	// Example 2: Invalid KEK alias (contains special characters)
	fmt.Println("2. Testing invalid KEK alias...")
	os.Setenv("ENCX_KEK_ALIAS", "invalid@alias") // Contains invalid character
	os.Setenv("ENCX_PEPPER_ALIAS", "test-service")
	_, err = encx.NewCryptoFromEnv(ctx, kms, secretStore)
	if err != nil {
		fmt.Printf("❌ Correctly caught invalid alias: %v\n", err)
	}
	fmt.Println()

	// Example 3: Valid KEK alias
	fmt.Println("3. Testing valid KEK alias...")
	os.Setenv("ENCX_KEK_ALIAS", "my-app-kek")
	os.Setenv("ENCX_PEPPER_ALIAS", "my-app-service")
	crypto, err := encx.NewCryptoFromEnv(ctx, kms, secretStore)
	if err != nil {
		log.Fatalf("❌ Unexpected error with valid KEK alias: %v", err)
	}
	fmt.Println("✅ Successfully created crypto instance with valid KEK alias!")
	fmt.Printf("   - KEK Alias: %s\n", crypto.GetAlias())
	fmt.Printf("   - Pepper length: %d bytes (auto-generated)\n", len(crypto.GetPepper()))
	fmt.Println()

	// Example 4: Nil KMS service (should fail)
	fmt.Println("4. Testing nil KMS service...")
	_, err = encx.NewCryptoFromEnv(ctx, nil, secretStore)
	if err != nil {
		fmt.Printf("❌ Correctly caught nil KMS service: %v\n", err)
	}
	fmt.Println()

	// Example 5: Invalid Argon2 parameters
	fmt.Println("5. Testing invalid Argon2 parameters...")
	invalidParams := &encx.Argon2Params{
		Memory:      1,  // Too low
		Iterations:  1,  // Too low
		Parallelism: 0,  // Too low
	}
	_, err = encx.NewCryptoFromEnv(ctx, kms, secretStore, encx.WithArgon2Params(invalidParams))
	if err != nil {
		fmt.Printf("❌ Correctly caught invalid Argon2 parameters: %v\n", err)
	}
	fmt.Println()

	// Example 6: Valid configuration with custom Argon2 parameters
	fmt.Println("6. Testing valid configuration with custom Argon2 parameters...")
	validParams := &encx.Argon2Params{
		Memory:      65536,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
	crypto2, err := encx.NewCryptoFromEnv(ctx, kms, secretStore, encx.WithArgon2Params(validParams))
	if err != nil {
		log.Fatalf("❌ Unexpected error with valid configuration: %v", err)
	}
	fmt.Println("✅ Successfully created crypto instance with custom Argon2 parameters!")
	fmt.Printf("   - KEK Alias: %s\n", crypto2.GetAlias())
	fmt.Printf("   - Pepper length: %d bytes (auto-generated)\n", len(crypto2.GetPepper()))
	fmt.Printf("   - Argon2 memory: %d KB\n", crypto2.GetArgon2Params().Memory)
	fmt.Printf("   - Argon2 iterations: %d\n", crypto2.GetArgon2Params().Iterations)
	fmt.Println()

	// Example 7: Test pepper consistency with in-memory store
	fmt.Println("7. Testing pepper consistency with in-memory store...")

	// Create a new secret store to test pepper consistency
	testSecretStore := encx.NewInMemorySecretStore()

	// First instance - should create pepper
	crypto3, err := encx.NewCryptoFromEnv(ctx, kms, testSecretStore)
	if err != nil {
		log.Fatalf("❌ Failed to create first crypto instance: %v", err)
	}
	firstPepper := crypto3.GetPepper()
	fmt.Printf("✅ First instance created with pepper: %x\n", firstPepper[:8]) // Show first 8 bytes

	// Second instance - should load same pepper from the same secret store
	crypto4, err := encx.NewCryptoFromEnv(ctx, kms, testSecretStore)
	if err != nil {
		log.Fatalf("❌ Failed to create second crypto instance: %v", err)
	}
	secondPepper := crypto4.GetPepper()
	fmt.Printf("✅ Second instance loaded pepper: %x\n", secondPepper[:8]) // Show first 8 bytes

	// Verify peppers are the same
	if string(firstPepper) == string(secondPepper) {
		fmt.Println("✅ Pepper persistence working correctly!")
	} else {
		fmt.Println("❌ Pepper persistence failed - different peppers loaded")
	}
	fmt.Println()

	// Example 8: Test different KEK aliases create different instances
	fmt.Println("8. Testing different KEK aliases...")
	os.Setenv("ENCX_KEK_ALIAS", "different-service")
	os.Setenv("ENCX_PEPPER_ALIAS", "different-app-service")
	crypto5, err := encx.NewCryptoFromEnv(ctx, kms, secretStore)
	if err != nil {
		log.Fatalf("❌ Failed to create crypto with different alias: %v", err)
	}
	fmt.Printf("✅ Created crypto with different alias: %s\n", crypto5.GetAlias())
	if crypto5.GetAlias() != crypto.GetAlias() {
		fmt.Println("✅ Different KEK aliases create separate instances!")
	} else {
		fmt.Println("❌ KEK aliases should be different")
	}
	fmt.Println()

	// Reset to original values
	os.Setenv("ENCX_KEK_ALIAS", "my-app-kek")
	os.Setenv("ENCX_PEPPER_ALIAS", "my-app-service")

	fmt.Println("=== All validation checks completed! ===")
	fmt.Println()

	fmt.Println("=== ENCX v0.6.0+ Validation Benefits ===")
	fmt.Println("✅ Environment-based configuration (12-factor app compliant)")
	fmt.Println("✅ Automatic pepper generation and persistence")
	fmt.Println("✅ No manual pepper management required")
	fmt.Println("✅ Early validation of required parameters")
	fmt.Println("✅ Clear, actionable error messages")
	fmt.Println("✅ Microservices-friendly with service identity")
	fmt.Println("✅ Tests KMS connectivity during initialization")
	fmt.Println("✅ Ensures database accessibility and permissions")
	fmt.Println("✅ Simplified API with fewer required options")
}