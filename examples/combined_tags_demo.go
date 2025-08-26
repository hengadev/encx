package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

// User struct demonstrating combined encx tags
type User struct {
	// Email: encrypt for secure storage, hash for lookup/indexing
	Email             string `encx:"encrypt,hash_basic"`
	EmailEncrypted    []byte // Stores encrypted email
	EmailHash         string // Stores hashed email for lookups

	// Password: hash securely, also encrypt for backup/recovery scenarios
	Password          string `encx:"hash_secure,encrypt"`
	PasswordHash      string // Stores Argon2id hashed password
	PasswordEncrypted []byte // Stores encrypted password

	// Name: only encrypt (no hashing needed)
	Name              string `encx:"encrypt"`
	NameEncrypted     []byte

	// Phone: only hash for basic lookups
	Phone             string `encx:"hash_basic"`
	PhoneHash         string

	// Required fields for encx
	DEK               []byte // Data Encryption Key
	DEKEncrypted      []byte // Encrypted DEK
	KeyVersion        int    // Key version for rotation
}

func main() {
	// Create test crypto instance using SimpleTestKMS
	crypto, kms := encx.NewTestCryptoWithSimpleKMS(nil)
	_ = kms // Suppress unused variable warning

	// Create user with sensitive data
	user := &User{
		Email:    "user@example.com",
		Password: "super_secret_password",
		Name:     "John Doe",
		Phone:    "+1-555-0123",
	}

	fmt.Println("=== Combined Tags Demo ===")
	fmt.Printf("Original User:\n")
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Password: %s\n", user.Password)
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Phone: %s\n", user.Phone)
	fmt.Println()

	// Process the struct (encrypt + hash operations)
	ctx := context.Background()
	if err := crypto.ProcessStruct(ctx, user); err != nil {
		log.Fatalf("Failed to process struct: %v", err)
	}

	fmt.Printf("After ProcessStruct:\n")
	fmt.Printf("  Email: '%s' (cleared)\n", user.Email)
	fmt.Printf("  EmailEncrypted: %d bytes\n", len(user.EmailEncrypted))
	fmt.Printf("  EmailHash: %s\n", user.EmailHash)
	fmt.Println()

	fmt.Printf("  Password: '%s' (cleared)\n", user.Password)
	fmt.Printf("  PasswordHash: %s\n", user.PasswordHash[:50]+"...")
	fmt.Printf("  PasswordEncrypted: %d bytes\n", len(user.PasswordEncrypted))
	fmt.Println()

	fmt.Printf("  Name: '%s' (cleared)\n", user.Name)
	fmt.Printf("  NameEncrypted: %d bytes\n", len(user.NameEncrypted))
	fmt.Println()

	fmt.Printf("  Phone: '%s' (preserved - hash only)\n", user.Phone)
	fmt.Printf("  PhoneHash: %s\n", user.PhoneHash)
	fmt.Println()

	fmt.Printf("  DEK: %d bytes\n", len(user.DEK))
	fmt.Printf("  DEKEncrypted: %d bytes\n", len(user.DEKEncrypted))
	fmt.Printf("  KeyVersion: %d\n", user.KeyVersion)
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
	testEmailHash := crypto.HashBasic(ctx, []byte(testEmail))
	
	if testEmailHash == user.EmailHash {
		fmt.Printf("✅ Email lookup successful! Hash matches for: %s\n", testEmail)
	} else {
		fmt.Printf("❌ Email lookup failed\n")
	}

	// Demonstrate decryption capability
	fmt.Println("\n=== Decryption Demonstration ===")
	if err := crypto.DecryptStruct(ctx, user); err != nil {
		log.Fatalf("Failed to decrypt struct: %v", err)
	}

	fmt.Printf("After DecryptStruct:\n")
	fmt.Printf("  Email: %s (restored)\n", user.Email)
	fmt.Printf("  Password: %s (restored)\n", user.Password)
	fmt.Printf("  Name: %s (restored)\n", user.Name)
	fmt.Printf("  Phone: %s (unchanged)\n", user.Phone)
	fmt.Println()

	fmt.Println("✅ Combined tags demo completed successfully!")
}