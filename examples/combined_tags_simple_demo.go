package main

import (
	"fmt"

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
	// Create a simple crypto instance similar to what we see in tests
	fmt.Println("=== Combined Tags Demo ===")
	fmt.Println("This demo would show combined encx tags in action.")
	fmt.Println("Combined tags allow both encryption AND hashing of the same field.")
	fmt.Println()

	// Example struct definition
	fmt.Println("Example struct with combined tags:")
	fmt.Println(`
type User struct {
    // Email: encrypt for secure storage, hash for lookup/indexing
    Email             string ` + "`encx:\"encrypt,hash_basic\"`" + `
    EmailEncrypted    []byte // Stores encrypted email
    EmailHash         string // Stores hashed email for lookups

    // Password: hash securely, also encrypt for backup/recovery scenarios  
    Password          string ` + "`encx:\"hash_secure,encrypt\"`" + `
    PasswordHash      string // Stores Argon2id hashed password
    PasswordEncrypted []byte // Stores encrypted password

    // Required fields for encx
    DEK               []byte // Data Encryption Key
    DEKEncrypted      []byte // Encrypted DEK
    KeyVersion        int    // Key version for rotation
}`)

	fmt.Println()
	fmt.Println("=== Benefits of Combined Tags ===")
	fmt.Println("1. Email: encrypt for privacy + hash for fast lookups")
	fmt.Println("2. Password: secure hash for auth + encrypt for recovery")
	fmt.Println("3. Reduces redundant code and ensures consistent processing")
	fmt.Println("4. Each tag requires its own companion field")
	fmt.Println()

	// Demonstrate validation
	fmt.Println("=== Validation Example ===")
	user := &User{}
	
	// Use encx validation (this will work without crypto instance)
	if err := encx.ValidateStruct(user); err != nil {
		fmt.Printf("❌ Validation failed (expected for empty struct):\n%v\n", err)
	} else {
		fmt.Println("✅ Struct validation passed!")
	}

	fmt.Println()
	fmt.Println("✅ Combined tags are now supported in encx!")
	fmt.Println("🔍 Use the validate-tags utility to check your structs:")
	fmt.Println("   go run ./cmd/validate-tags/main.go -v")
}