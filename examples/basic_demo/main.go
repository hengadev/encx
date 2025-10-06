package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

// User demonstrates basic encryption usage
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	ctx := context.Background()

	// Create a test crypto instance (use proper KMS in production)
	crypto, err := encx.NewTestCrypto(nil)
	if err != nil {
		log.Fatal("Failed to create crypto service:", err)
	}

	fmt.Println("=== Basic ENCX Demo ===")
	fmt.Println("This demo shows basic manual encryption/decryption operations.")
	fmt.Println()

	// Create sample data
	user := &User{
		ID:       1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "mySecretPassword123",
	}

	fmt.Printf("Original data:\n")
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Password: %s\n", user.Password)
	fmt.Println()

	// Manual encryption
	fmt.Println("Encrypting sensitive data...")

	// Generate a DEK
	dek, err := crypto.GenerateDEK()
	if err != nil {
		log.Fatal("Failed to generate DEK:", err)
	}

	// Encrypt password
	encryptedPassword, err := crypto.EncryptData(ctx, []byte(user.Password), dek)
	if err != nil {
		log.Fatal("Failed to encrypt password:", err)
	}

	// Hash email for searchability
	emailHash := crypto.HashBasic(ctx, []byte(user.Email))

	fmt.Printf("After encryption:\n")
	fmt.Printf("  Name: %s (not encrypted)\n", user.Name)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Email hash: %s...\n", emailHash[:16])
	fmt.Printf("  Password: [ENCRYPTED] (%d bytes)\n", len(encryptedPassword))
	fmt.Println()

	// Manual decryption
	fmt.Println("Decrypting data...")
	decryptedPassword, err := crypto.DecryptData(ctx, encryptedPassword, dek)
	if err != nil {
		log.Fatal("Failed to decrypt password:", err)
	}

	fmt.Printf("Decrypted password: %s\n", string(decryptedPassword))
	fmt.Println()

	fmt.Println("✅ Basic demo completed successfully!")
	fmt.Println("For more advanced examples, check the context7/ directory.")
}
