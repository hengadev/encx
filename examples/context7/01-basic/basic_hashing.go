// Package basic demonstrates secure hashing patterns with encx
//
// Context7 Tags: hashing, password-security, user-lookup, data-integrity, golang-crypto
// Complexity: Beginner
// Use Case: Fast lookups, password security, data verification

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

// UserLookup demonstrates basic hashing for fast user searches
// Use this pattern when you need to find users quickly but don't need
// to decrypt the original data
type UserLookup struct {
	// Basic information
	ID       int    `json:"id"`
	Username string `json:"username"`

	// Hashed fields for fast lookup
	Email        string `encx:"hash_basic" json:"email"`
	EmailHash    string `json:"email_hash"` // For database queries

	Phone        string `encx:"hash_basic" json:"phone"`
	PhoneHash    string `json:"phone_hash"`

	// Required fields
	DEK          []byte `json:"-"`
	DEKEncrypted []byte `json:"dek_encrypted"`
	KeyVersion   int    `json:"key_version"`
}

// SecureAccount demonstrates secure password hashing
// Use this pattern for authentication where you never need to
// decrypt the original password
type SecureAccount struct {
	// Basic information
	Username string `json:"username"`
	Email    string `json:"email"`

	// Secure password hashing (cannot be decrypted)
	Password          string `encx:"hash_secure" json:"-"`
	PasswordHashSecure string `json:"password_hash"` // For authentication

	// Account security questions (also secure hash)
	SecurityAnswer1          string `encx:"hash_secure" json:"-"`
	SecurityAnswer1HashSecure string `json:"security_answer1_hash"`

	SecurityAnswer2          string `encx:"hash_secure" json:"-"`
	SecurityAnswer2HashSecure string `json:"security_answer2_hash"`

	// Required fields
	DEK          []byte `json:"-"`
	DEKEncrypted []byte `json:"dek_encrypted"`
	KeyVersion   int    `json:"key_version"`
}

// ProductCatalog demonstrates basic hashing for product identification
type ProductCatalog struct {
	// Public information
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`

	// Hashed fields for internal tracking
	SKU         string `encx:"hash_basic" json:"sku"`
	SKUHash     string `json:"sku_hash"`

	Barcode     string `encx:"hash_basic" json:"barcode"`
	BarcodeHash string `json:"barcode_hash"`

	// Required fields
	DEK          []byte `json:"-"`
	DEKEncrypted []byte `json:"dek_encrypted"`
	KeyVersion   int    `json:"key_version"`
}

func main() {
	ctx := context.Background()

	// Create crypto instance
	crypto, err := encx.NewTestCrypto(nil)
	if err != nil {
		log.Fatal("Failed to create crypto service:", err)
	}

	// Example 1: Basic hashing for user lookup
	fmt.Println("=== Basic Hashing for User Lookup ===")

	user := &UserLookup{
		ID:       1,
		Username: "john_doe",
		Email:    "john.doe@example.com",
		Phone:    "+1-555-123-4567",
	}

	fmt.Printf("Original user data:\n")
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Printf("  Phone: %s\n", user.Phone)

	// Hash the user data
	if err := crypto.ProcessStruct(ctx, user); err != nil {
		log.Fatal("Failed to hash user data:", err)
	}

	fmt.Printf("\nAfter hashing:\n")
	fmt.Printf("  Email: '%s' (cleared)\n", user.Email)
	fmt.Printf("  Phone: '%s' (cleared)\n", user.Phone)
	fmt.Printf("  Email hash: %s\n", user.EmailHash)
	fmt.Printf("  Phone hash: %s\n", user.PhoneHash)

	// Demonstrate user lookup by email
	fmt.Println("\n=== User Lookup Simulation ===")

	// To find a user by email, hash the search term
	searchUser := &UserLookup{Email: "john.doe@example.com"}
	if err := crypto.ProcessStruct(ctx, searchUser); err != nil {
		log.Fatal("Failed to hash search email:", err)
	}

	fmt.Printf("Search for email 'john.doe@example.com':\n")
	fmt.Printf("  Search hash: %s\n", searchUser.EmailHash)
	fmt.Printf("  Matches user: %t\n", searchUser.EmailHash == user.EmailHash)

	fmt.Println()

	// Example 2: Secure password hashing
	fmt.Println("=== Secure Password Hashing ===")

	account := &SecureAccount{
		Username:        "secure_user",
		Email:          "secure@example.com",
		Password:       "MyVerySecurePassword123!",
		SecurityAnswer1: "Fluffy", // Pet's name
		SecurityAnswer2: "Main Street", // Street you grew up on
	}

	fmt.Printf("Original account data:\n")
	fmt.Printf("  Password: %s\n", account.Password)
	fmt.Printf("  Security Answer 1: %s\n", account.SecurityAnswer1)
	fmt.Printf("  Security Answer 2: %s\n", account.SecurityAnswer2)

	// Hash the sensitive data
	if err := crypto.ProcessStruct(ctx, account); err != nil {
		log.Fatal("Failed to hash account data:", err)
	}

	fmt.Printf("\nAfter secure hashing:\n")
	fmt.Printf("  Password: '%s' (cleared, cannot be recovered)\n", account.Password)
	fmt.Printf("  Security Answer 1: '%s' (cleared)\n", account.SecurityAnswer1)
	fmt.Printf("  Security Answer 2: '%s' (cleared)\n", account.SecurityAnswer2)
	fmt.Printf("  Password hash: %s...\n", account.PasswordHashSecure[:30])

	// Demonstrate password verification
	fmt.Println("\n=== Password Verification ===")

	// Test correct password
	correctPassword := "MyVerySecurePassword123!"
	isValid, err := crypto.CompareSecureHashAndValue(ctx, correctPassword, account.PasswordHashSecure)
	if err != nil {
		log.Fatal("Failed to verify password:", err)
	}
	fmt.Printf("Password '%s' is valid: %t\n", correctPassword, isValid)

	// Test incorrect password
	wrongPassword := "WrongPassword!"
	isValid, err = crypto.CompareSecureHashAndValue(ctx, wrongPassword, account.PasswordHashSecure)
	if err != nil {
		log.Fatal("Failed to verify password:", err)
	}
	fmt.Printf("Password '%s' is valid: %t\n", wrongPassword, isValid)

	fmt.Println()

	// Example 3: Product catalog hashing
	fmt.Println("=== Product Catalog Hashing ===")

	product := &ProductCatalog{
		Name:        "Wireless Headphones",
		Description: "High-quality wireless headphones with noise cancellation",
		Price:       199.99,
		SKU:         "WH-2024-001",
		Barcode:     "1234567890123",
	}

	fmt.Printf("Original product:\n")
	fmt.Printf("  SKU: %s\n", product.SKU)
	fmt.Printf("  Barcode: %s\n", product.Barcode)

	// Hash the identifiers
	if err := crypto.ProcessStruct(ctx, product); err != nil {
		log.Fatal("Failed to hash product data:", err)
	}

	fmt.Printf("\nAfter hashing:\n")
	fmt.Printf("  SKU: '%s' (cleared)\n", product.SKU)
	fmt.Printf("  Barcode: '%s' (cleared)\n", product.Barcode)
	fmt.Printf("  SKU hash: %s\n", product.SKUHash)
	fmt.Printf("  Barcode hash: %s\n", product.BarcodeHash)
}

// Practical usage patterns for hashing

// Pattern 1: User authentication
func AuthenticateUser(crypto *encx.Crypto, username, password string) (bool, error) {
	ctx := context.Background()

	// In production, you'd load the user from database by username
	// For demo, we'll create a mock user
	storedPasswordHash := "stored_hash_from_database"

	// Verify the provided password against stored hash
	isValid, err := crypto.CompareSecureHashAndValue(ctx, password, storedPasswordHash)
	if err != nil {
		return false, fmt.Errorf("password verification failed: %w", err)
	}

	return isValid, nil
}

// Pattern 2: User registration with password hashing
func RegisterUser(crypto *encx.Crypto, username, email, password string) (*SecureAccount, error) {
	ctx := context.Background()

	account := &SecureAccount{
		Username: username,
		Email:    email,
		Password: password,
	}

	// Hash the password securely
	if err := crypto.ProcessStruct(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Now account.PasswordHashSecure contains the hashed password
	// Store account in database with the hash, not the original password
	return account, nil
}

// Pattern 3: Find user by email hash
func FindUserByEmail(crypto *encx.Crypto, email string) (string, error) {
	ctx := context.Background()

	// Create temporary user to generate search hash
	searchUser := &UserLookup{Email: email}
	if err := crypto.ProcessStruct(ctx, searchUser); err != nil {
		return "", fmt.Errorf("failed to hash email: %w", err)
	}

	// Use the hash for database query
	// In production: SELECT * FROM users WHERE email_hash = ?
	searchHash := searchUser.EmailHash
	fmt.Printf("Database query would use hash: %s\n", searchHash)

	return searchHash, nil
}

// Pattern 4: Product lookup by SKU
func FindProductBySKU(crypto *encx.Crypto, sku string) (*ProductCatalog, error) {
	ctx := context.Background()

	// Generate hash for SKU lookup
	searchProduct := &ProductCatalog{SKU: sku}
	if err := crypto.ProcessStruct(ctx, searchProduct); err != nil {
		return nil, fmt.Errorf("failed to hash SKU: %w", err)
	}

	// Query database by hash
	// SELECT * FROM products WHERE sku_hash = ?
	fmt.Printf("Looking up product with SKU hash: %s\n", searchProduct.SKUHash)

	// Return found product (mock)
	return &ProductCatalog{
		Name:    "Found Product",
		SKUHash: searchProduct.SKUHash,
	}, nil
}

// Pattern 5: Security question verification
func VerifySecurityAnswer(crypto *encx.Crypto, userAnswer, storedHash string) (bool, error) {
	ctx := context.Background()

	// Verify the answer against stored hash
	isValid, err := crypto.CompareSecureHashAndValue(ctx, userAnswer, storedHash)
	if err != nil {
		return false, fmt.Errorf("security answer verification failed: %w", err)
	}

	return isValid, nil
}

// Pattern 6: Bulk user lookup
func FindUsersByPhones(crypto *encx.Crypto, phoneNumbers []string) ([]string, error) {
	ctx := context.Background()

	var hashes []string
	for _, phone := range phoneNumbers {
		user := &UserLookup{Phone: phone}
		if err := crypto.ProcessStruct(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to hash phone %s: %w", phone, err)
		}
		hashes = append(hashes, user.PhoneHash)
	}

	// Query database with multiple hashes
	// SELECT * FROM users WHERE phone_hash IN (?, ?, ?)
	fmt.Printf("Bulk lookup for %d phone numbers\n", len(hashes))

	return hashes, nil
}

/*
Key Concepts Demonstrated:

1. **Basic Hashing**: Use `encx:"hash_basic"` for fast lookups (SHA-256)
2. **Secure Hashing**: Use `encx:"hash_secure"` for passwords (Argon2id)
3. **Companion Fields**: Hash fields need companion `*Hash string` fields
4. **Data Clearing**: Original data is cleared after hashing for security
5. **Search Strategy**: Hash search terms to match stored hashes
6. **Password Verification**: Use CompareSecureHashAndValue for auth

Hash Types Comparison:

hash_basic:
- Algorithm: SHA-256
- Use Case: Fast lookups, user search, product identification
- Performance: Very fast
- Security: Good for non-sensitive identifiers
- Reversible: No

hash_secure:
- Algorithm: Argon2id with salt and pepper
- Use Case: Passwords, security questions, sensitive identifiers
- Performance: Intentionally slow (security feature)
- Security: Extremely high
- Reversible: No

When to Use Each Pattern:

Basic Hashing (hash_basic):
✅ User email/phone lookup
✅ Product SKU/barcode search
✅ Non-sensitive identifiers
✅ Fast search requirements

Secure Hashing (hash_secure):
✅ Password storage
✅ Security questions
✅ SSN, credit card numbers (for verification only)
✅ Any data that should never be decrypted

Database Schema for Hashing:

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE,

    -- Hash columns for search
    email_hash VARCHAR(64) UNIQUE,
    phone_hash VARCHAR(64),
    password_hash TEXT NOT NULL,

    -- Encryption metadata
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL
);

-- Indexes for fast searching
CREATE INDEX idx_users_email_hash ON users (email_hash);
CREATE INDEX idx_users_phone_hash ON users (phone_hash);

Security Benefits:
- Original sensitive data never stored in database
- Fast searches without exposing data
- Password hashes use industry best practices
- Automatic salt/pepper handling
- Protection against rainbow table attacks
*/