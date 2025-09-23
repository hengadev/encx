package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
	"github.com/hengadev/encx/internal/serialization"
)

// User struct demonstrating combined encx tags
// The companion encrypted/hashed fields are auto-generated in UserEncx
type User struct {
	// Email: encrypt for secure storage, hash for lookup/indexing
	Email    string `encx:"encrypt,hash_basic"`

	// Password: hash securely, also encrypt for backup/recovery scenarios
	Password string `encx:"hash_secure,encrypt"`

	// Name: only encrypt (no hashing needed)
	Name     string `encx:"encrypt"`

	// Phone: only hash for basic lookups
	Phone    string `encx:"hash_basic"`
}

func main() {
	ctx := context.Background()

	// Create test crypto instance
	crypto, err := encx.NewTestCrypto(nil)
	if err != nil {
		log.Fatalf("Failed to create crypto instance: %v", err)
	}

	// Create user with sensitive data
	user := &User{
		Email:    "user@example.com",
		Password: "super_secret_password",
		Name:     "John Doe",
		Phone:    "+1-555-0123",
	}

	fmt.Println("=== Combined Tags Demo (Code Generation API) ===")
	fmt.Printf("Original User:\n")
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Password: %s\n", user.Password)
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Phone: %s\n", user.Phone)
	fmt.Println()

	// Process the struct using generated code (encrypt + hash operations)
	userEncx, err := ProcessUserEncx(ctx, crypto, user)
	if err != nil {
		log.Fatalf("Failed to process struct: %v", err)
	}

	fmt.Printf("After ProcessUserEncx (Generated Code):\n")
	fmt.Printf("  EmailEncrypted: %d bytes\n", len(userEncx.EmailEncrypted))
	fmt.Printf("  EmailHash: %s\n", userEncx.EmailHash)
	fmt.Println()

	fmt.Printf("  PasswordHashSecure: %s\n", userEncx.PasswordHashSecure[:50]+"...")
	fmt.Printf("  PasswordEncrypted: %d bytes\n", len(userEncx.PasswordEncrypted))
	fmt.Println()

	fmt.Printf("  NameEncrypted: %d bytes\n", len(userEncx.NameEncrypted))
	fmt.Println()

	fmt.Printf("  PhoneHash: %s\n", userEncx.PhoneHash)
	fmt.Println()

	fmt.Printf("  DEKEncrypted: %d bytes\n", len(userEncx.DEKEncrypted))
	fmt.Printf("  KeyVersion: %d\n", userEncx.KeyVersion)
	fmt.Println()

	// Demonstrate use cases for combined tags
	fmt.Println("=== Use Cases for Combined Tags ===")
	fmt.Println("1. Email encrypted for privacy, hashed for fast user lookup")
	fmt.Println("2. Password hashed for authentication, encrypted for recovery")
	fmt.Println("3. Name encrypted for privacy protection")
	fmt.Println("4. Phone hashed only for duplicate detection")
	fmt.Println()

	// Demonstrate lookup capability
	fmt.Println("=== Lookup Demonstration ===")
	testEmail := "user@example.com"

	// Important: Must serialize the value the same way as generated code
	testEmailBytes, err := serialization.Serialize(testEmail)
	if err != nil {
		log.Fatalf("Failed to serialize test email: %v", err)
	}
	testEmailHash := crypto.HashBasic(ctx, testEmailBytes)

	if testEmailHash == userEncx.EmailHash {
		fmt.Printf("✅ Email lookup successful! Hash matches for: %s\n", testEmail)
	} else {
		fmt.Printf("❌ Email lookup failed\n")
	}

	// Demonstrate decryption capability
	fmt.Println("\n=== Decryption Demonstration ===")
	decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
	if err != nil {
		log.Fatalf("Failed to decrypt struct: %v", err)
	}

	fmt.Printf("After DecryptUserEncx (Generated Code):\n")
	fmt.Printf("  Email: %s (restored)\n", decryptedUser.Email)
	fmt.Printf("  Password: %s (restored)\n", decryptedUser.Password)
	fmt.Printf("  Name: %s (restored)\n", decryptedUser.Name)
	fmt.Printf("  Phone: %s (from hash lookup)\n", user.Phone) // Original value preserved
	fmt.Println()

	fmt.Println("✅ Combined tags demo completed successfully!")
}