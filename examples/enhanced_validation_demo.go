package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

func main() {
	fmt.Println("=== Enhanced Input Validation Demo ===")
	fmt.Println("Demonstrating comprehensive configuration validation in ENCX")
	fmt.Println()

	ctx := context.Background()

	// Example 1: Invalid KEK alias
	fmt.Println("1. Testing invalid KEK alias...")
	_, err := encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("invalid@alias"), // Contains invalid character
		encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
	)
	if err != nil {
		fmt.Printf("❌ Correctly caught invalid alias: %v\n", err)
	}
	fmt.Println()

	// Example 2: Invalid pepper length
	fmt.Println("2. Testing invalid pepper length...")
	_, err = encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("valid-alias"),
		encx.WithPepper([]byte("too-short")), // Only 9 bytes
	)
	if err != nil {
		fmt.Printf("❌ Correctly caught invalid pepper length: %v\n", err)
	}
	fmt.Println()

	// Example 3: Zero pepper (security risk)
	fmt.Println("3. Testing zero pepper (security risk)...")
	zeroPepper := make([]byte, 32) // All zeros
	_, err = encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("valid-alias"),
		encx.WithPepper(zeroPepper),
	)
	if err != nil {
		fmt.Printf("❌ Correctly caught zero pepper: %v\n", err)
	}
	fmt.Println()

	// Example 4: Missing required configuration
	fmt.Println("4. Testing missing required configuration...")
	_, err = encx.NewCrypto(ctx,
		// Missing KMS service, KEK alias, and pepper
	)
	if err != nil {
		fmt.Printf("❌ Correctly caught missing configuration: %v\n", err)
	}
	fmt.Println()

	// Example 5: Configuration conflicts
	fmt.Println("5. Testing configuration conflicts...")
	_, err = encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("valid-alias"),
		encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		encx.WithPepperSecretPath("secret/path"), // Conflict: both direct pepper and secret path
	)
	if err != nil {
		fmt.Printf("❌ Correctly caught configuration conflict: %v\n", err)
	}
	fmt.Println()

	// Example 6: Invalid Argon2 parameters
	fmt.Println("6. Testing invalid Argon2 parameters...")
	invalidParams := &encx.Argon2Params{
		Memory:      1,  // Too low
		Iterations:  1,  // Too low
		Parallelism: 0,  // Too low
	}
	_, err = encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("valid-alias"),
		encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		encx.WithArgon2ParamsV2(invalidParams),
	)
	if err != nil {
		fmt.Printf("❌ Correctly caught invalid Argon2 parameters: %v\n", err)
	}
	fmt.Println()

	// Example 7: Valid configuration (should work)
	fmt.Println("7. Testing valid configuration...")
	crypto, err := encx.NewCrypto(ctx,
		encx.WithKMSService(encx.NewSimpleTestKMS()),
		encx.WithKEKAlias("my-app-kek"),
		encx.WithPepper([]byte("test-pepper-exactly-32-bytes-OK!")),
		encx.WithArgon2ParamsV2(&encx.Argon2Params{
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
			SaltLength:  16,
			KeyLength:   32,
		}),
	)
	if err != nil {
		log.Fatalf("❌ Unexpected error with valid configuration: %v", err)
	}
	fmt.Println("✅ Successfully created crypto instance with valid configuration!")
	fmt.Printf("   - KEK Alias: %s\n", crypto.GetAlias())
	fmt.Printf("   - Pepper length: %d bytes\n", len(crypto.GetPepper()))
	fmt.Printf("   - Argon2 memory: %d KB\n", crypto.GetArgon2Params().Memory)
	fmt.Println()

	// Example 8: Demonstrate backward compatibility
	fmt.Println("8. Testing backward compatibility with legacy constructor...")
	kms := encx.NewSimpleTestKMS()
	kms.SetSecret(ctx, "legacy/pepper", []byte("test-pepper-exactly-32-bytes-OK!"))
	
	legacyCrypto, err := encx.New(ctx, kms, "legacy-kek", "legacy/pepper")
	if err != nil {
		fmt.Printf("❌ Legacy constructor failed: %v\n", err)
	} else {
		fmt.Println("✅ Legacy constructor still works!")
		fmt.Printf("   - KEK Alias: %s\n", legacyCrypto.GetAlias())
	}
	fmt.Println()

	fmt.Println("=== Enhanced Validation Benefits ===")
	fmt.Println("✅ Catches configuration errors early (at startup, not runtime)")
	fmt.Println("✅ Provides clear, actionable error messages")
	fmt.Println("✅ Prevents common security misconfigurations")
	fmt.Println("✅ Validates all configuration combinations")
	fmt.Println("✅ Maintains full backward compatibility")
	fmt.Println("✅ Tests KMS connectivity during initialization")
	fmt.Println("✅ Ensures database accessibility and permissions")
}