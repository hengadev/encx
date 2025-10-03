// Package intermediate demonstrates searchable encryption with encx
//
// Context7 Tags: searchable-encryption, encrypt-and-hash, database-search, user-lookup
// Complexity: Intermediate
// Use Case: Encrypt sensitive data while maintaining search capability

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

// Customer demonstrates the searchable encryption pattern
// This allows you to encrypt customer data for privacy while
// still being able to search by email or phone number
type Customer struct {
	// Basic information
	ID        int    `json:"id" db:"id"`
	FirstName string `json:"first_name" db:"first_name"`
	LastName  string `json:"last_name" db:"last_name"`

	// Searchable encrypted fields - both encrypted AND hashed
	Email           string `encx:"encrypt,hash_basic" json:"email" db:"email"`
	EmailEncrypted  []byte `json:"-" db:"email_encrypted"`  // For secure storage
	EmailHash       string `json:"-" db:"email_hash"`       // For fast searches

	Phone           string `encx:"encrypt,hash_basic" json:"phone" db:"phone"`
	PhoneEncrypted  []byte `json:"-" db:"phone_encrypted"`
	PhoneHash       string `json:"-" db:"phone_hash"`

	// Encrypted-only fields (no search needed)
	Address         string `encx:"encrypt" json:"address" db:"address"`
	AddressEncrypted []byte `json:"-" db:"address_encrypted"`

	// Hash-only field for customer lookup (no encryption needed)
	CustomerNumber  string `encx:"hash_basic" json:"customer_number" db:"customer_number"`
	CustomerNumberHash string `json:"-" db:"customer_number_hash"`

	// Required encryption fields
	DEK          []byte `json:"-" db:"dek"`
	DEKEncrypted []byte `json:"-" db:"dek_encrypted"`
	KeyVersion   int    `json:"-" db:"key_version"`
}

// UserAccount demonstrates password handling with recovery capability
// Helper functions for manual encryption

// processCustomer manually encrypts and hashes customer fields
func processCustomer(ctx context.Context, crypto *encx.Crypto, customer *Customer) error {
	// Generate a DEK for this customer
	dek, err := crypto.GenerateDEK()
	if err != nil {
		return fmt.Errorf("failed to generate DEK: %w", err)
	}

	// Encrypt and hash email (searchable)
	if customer.Email != "" {
		emailBytes := []byte(customer.Email)
		customer.EmailEncrypted, err = crypto.EncryptData(ctx, emailBytes, dek)
		if err != nil {
			return fmt.Errorf("failed to encrypt email: %w", err)
		}
		customer.EmailHash = crypto.HashBasic(ctx, emailBytes)
		customer.Email = ""
	}

	// Encrypt and hash phone (searchable)
	if customer.Phone != "" {
		phoneBytes := []byte(customer.Phone)
		customer.PhoneEncrypted, err = crypto.EncryptData(ctx, phoneBytes, dek)
		if err != nil {
			return fmt.Errorf("failed to encrypt phone: %w", err)
		}
		customer.PhoneHash = crypto.HashBasic(ctx, phoneBytes)
		customer.Phone = ""
	}

	// Encrypt address (no hash needed)
	if customer.Address != "" {
		addrBytes := []byte(customer.Address)
		customer.AddressEncrypted, err = crypto.EncryptData(ctx, addrBytes, dek)
		if err != nil {
			return fmt.Errorf("failed to encrypt address: %w", err)
		}
		customer.Address = ""
	}

	// Hash customer number (searchable, no encryption)
	if customer.CustomerNumber != "" {
		customer.CustomerNumberHash = crypto.HashBasic(ctx, []byte(customer.CustomerNumber))
		customer.CustomerNumber = ""
	}

	// Encrypt and store the DEK
	customer.DEKEncrypted, err = crypto.EncryptDEK(ctx, dek)
	if err != nil {
		return fmt.Errorf("failed to encrypt DEK: %w", err)
	}
	customer.KeyVersion = 1
	customer.DEK = nil // Clear DEK from memory

	return nil
}

// processUserAccount manually encrypts and hashes user account fields
func processUserAccount(ctx context.Context, crypto *encx.Crypto, user *UserAccount) error {
	// Generate a DEK for this user account
	dek, err := crypto.GenerateDEK()
	if err != nil {
		return fmt.Errorf("failed to generate DEK: %w", err)
	}

	// Encrypt and hash email (searchable)
	if user.Email != "" {
		emailBytes := []byte(user.Email)
		user.EmailEncrypted, err = crypto.EncryptData(ctx, emailBytes, dek)
		if err != nil {
			return fmt.Errorf("failed to encrypt email: %w", err)
		}
		user.EmailHash = crypto.HashBasic(ctx, emailBytes)
		user.Email = ""
	}

	// Process password - both secure hash and encryption
	if user.Password != "" {
		passwordBytes := []byte(user.Password)
		// Secure hash for authentication
		user.PasswordHashSecure, err = crypto.HashSecure(ctx, passwordBytes)
		if err != nil {
			return fmt.Errorf("failed to hash password securely: %w", err)
		}
		// Encrypt for recovery scenarios
		user.PasswordEncrypted, err = crypto.EncryptData(ctx, passwordBytes, dek)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		user.Password = ""
	}

	// Encrypt and store the DEK
	user.DEKEncrypted, err = crypto.EncryptDEK(ctx, dek)
	if err != nil {
		return fmt.Errorf("failed to encrypt DEK: %w", err)
	}
	user.KeyVersion = 1
	user.DEK = nil // Clear DEK from memory

	return nil
}

// decryptCustomer manually decrypts customer fields
func decryptCustomer(ctx context.Context, crypto *encx.Crypto, customer *Customer) error {
	// Decrypt the DEK first
	dek, err := crypto.DecryptDEKWithVersion(ctx, customer.DEKEncrypted, customer.KeyVersion)
	if err != nil {
		return fmt.Errorf("failed to decrypt DEK: %w", err)
	}

	// Decrypt all encrypted fields
	if len(customer.EmailEncrypted) > 0 {
		emailBytes, err := crypto.DecryptData(ctx, customer.EmailEncrypted, dek)
		if err != nil {
			return fmt.Errorf("failed to decrypt email: %w", err)
		}
		customer.Email = string(emailBytes)
	}

	if len(customer.PhoneEncrypted) > 0 {
		phoneBytes, err := crypto.DecryptData(ctx, customer.PhoneEncrypted, dek)
		if err != nil {
			return fmt.Errorf("failed to decrypt phone: %w", err)
		}
		customer.Phone = string(phoneBytes)
	}

	if len(customer.AddressEncrypted) > 0 {
		addrBytes, err := crypto.DecryptData(ctx, customer.AddressEncrypted, dek)
		if err != nil {
			return fmt.Errorf("failed to decrypt address: %w", err)
		}
		customer.Address = string(addrBytes)
	}

	// Note: Hashed fields cannot be decrypted
	return nil
}

// decryptUserAccount manually decrypts user account fields
func decryptUserAccount(ctx context.Context, crypto *encx.Crypto, user *UserAccount) error {
	// Decrypt the DEK first
	dek, err := crypto.DecryptDEKWithVersion(ctx, user.DEKEncrypted, user.KeyVersion)
	if err != nil {
		return fmt.Errorf("failed to decrypt DEK: %w", err)
	}

	// Decrypt email
	if len(user.EmailEncrypted) > 0 {
		emailBytes, err := crypto.DecryptData(ctx, user.EmailEncrypted, dek)
		if err != nil {
			return fmt.Errorf("failed to decrypt email: %w", err)
		}
		user.Email = string(emailBytes)
	}

	// Decrypt password (for recovery scenarios)
	if len(user.PasswordEncrypted) > 0 {
		passwordBytes, err := crypto.DecryptData(ctx, user.PasswordEncrypted, dek)
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		user.Password = string(passwordBytes)
	}

	return nil
}

type UserAccount struct {
	// Basic information
	Username string `json:"username" db:"username"`

	// Email - searchable and recoverable
	Email           string `encx:"encrypt,hash_basic" json:"email"`
	EmailEncrypted  []byte `json:"-" db:"email_encrypted"`
	EmailHash       string `json:"-" db:"email_hash"`

	// Password - secure hash for auth + encrypted for recovery
	Password          string `encx:"hash_secure,encrypt" json:"-"`
	PasswordHashSecure string `json:"-" db:"password_hash"`      // For authentication
	PasswordEncrypted  []byte `json:"-" db:"password_encrypted"` // For recovery scenarios

	// Required fields
	DEK          []byte `json:"-" db:"dek"`
	DEKEncrypted []byte `json:"-" db:"dek_encrypted"`
	KeyVersion   int    `json:"-" db:"key_version"`
}

func main() {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(nil)
	if err != nil {
		log.Fatal("Failed to create crypto service:", err)
	}

	// Example 1: Customer with searchable encryption
	fmt.Println("=== Searchable Customer Encryption ===")

	customer := &Customer{
		ID:             1,
		FirstName:      "Alice",
		LastName:       "Johnson",
		Email:          "alice.johnson@example.com",
		Phone:          "+1-555-123-4567",
		Address:        "123 Main Street, Springfield, IL 62701",
		CustomerNumber: "CUST-2024-001",
	}

	fmt.Printf("Original email: %s\n", customer.Email)
	fmt.Printf("Original phone: %s\n", customer.Phone)

	// Process customer data
	if err := processCustomer(ctx, crypto, customer); err != nil {
		log.Fatal("Failed to process customer:", err)
	}

	fmt.Printf("Email after processing: '%s' (cleared for security)\n", customer.Email)
	fmt.Printf("Email hash: %s\n", customer.EmailHash[:16]+"...")
	fmt.Printf("Phone hash: %s\n", customer.PhoneHash[:16]+"...")
	fmt.Printf("Customer number hash: %s\n", customer.CustomerNumberHash[:16]+"...")

	// Demonstrate search capability
	fmt.Println("\n=== Search Simulation ===")

	// To search for a customer by email, create a temporary customer with just the email
	searchCustomer := &Customer{Email: "alice.johnson@example.com"}
	if err := processCustomer(ctx, crypto, searchCustomer); err != nil {
		log.Fatal("Failed to process search customer:", err)
	}

	// Now you can search in your database using the hash
	fmt.Printf("Search hash for 'alice.johnson@example.com': %s\n", searchCustomer.EmailHash[:16]+"...")
	fmt.Printf("Matches original hash: %t\n", searchCustomer.EmailHash == customer.EmailHash)

	// Decrypt for display
	if err := decryptCustomer(ctx, crypto, customer); err != nil {
		log.Fatal("Failed to decrypt customer:", err)
	}

	fmt.Printf("Decrypted email: %s\n", customer.Email)
	fmt.Printf("Decrypted address: %s\n", customer.Address)

	fmt.Println()

	// Example 2: User account with password security
	fmt.Println("=== User Account with Password Recovery ===")

	user := &UserAccount{
		Username: "alice_j",
		Email:    "alice.johnson@example.com",
		Password: "MySecurePassword123!",
	}

	fmt.Printf("Original password: %s\n", user.Password)

	// Process user account
	if err := processUserAccount(ctx, crypto, user); err != nil {
		log.Fatal("Failed to process user account:", err)
	}

	fmt.Printf("Password after processing: '%s' (cleared)\n", user.Password)
	fmt.Printf("Password hash: %s\n", user.PasswordHashSecure[:20]+"...")
	fmt.Printf("Has encrypted password backup: %t\n", len(user.PasswordEncrypted) > 0)

	// Demonstrate password verification
	fmt.Println("\n=== Password Verification ===")
	testPassword := "MySecurePassword123!"
	isValid, err := crypto.CompareSecureHashAndValue(ctx, testPassword, user.PasswordHashSecure)
	if err != nil {
		log.Fatal("Failed to verify password:", err)
	}
	fmt.Printf("Password '%s' is valid: %t\n", testPassword, isValid)

	// Demonstrate password recovery (admin function)
	fmt.Println("\n=== Password Recovery (Admin) ===")
	if err := decryptUserAccount(ctx, crypto, user); err != nil {
		log.Fatal("Failed to decrypt user for recovery:", err)
	}
	fmt.Printf("Recovered password: %s\n", user.Password)
}

// Practical usage patterns for databases

// Pattern 1: Search customers by email
func FindCustomerByEmail(crypto *encx.Crypto, email string) (string, error) {
	ctx := context.Background()

	// Create temporary customer to generate search hash
	searchCustomer := &Customer{Email: email}
	if err := processCustomer(ctx, crypto, searchCustomer); err != nil {
		return "", fmt.Errorf("failed to hash email: %w", err)
	}

	// In real application, you'd query your database:
	// SELECT * FROM customers WHERE email_hash = ?
	searchHash := searchCustomer.EmailHash

	// Simulate database query result
	fmt.Printf("Database query: SELECT * FROM customers WHERE email_hash = '%s'\n", searchHash[:16]+"...")

	return searchHash, nil
}

// Pattern 2: Customer registration with encryption
func RegisterCustomer(crypto *encx.Crypto, email, phone, address string) (*Customer, error) {
	ctx := context.Background()

	customer := &Customer{
		Email:   email,
		Phone:   phone,
		Address: address,
	}

	// Encrypt and hash the customer data
	if err := processCustomer(ctx, crypto, customer); err != nil {
		return nil, fmt.Errorf("failed to process customer: %w", err)
	}

	// At this point:
	// - customer.EmailEncrypted contains encrypted email
	// - customer.EmailHash contains searchable hash
	// - customer.Email is cleared for security
	// You would save customer to database here

	return customer, nil
}

// Pattern 3: Get customer profile for display
func GetCustomerProfile(crypto *encx.Crypto, encryptedCustomer *Customer) (*Customer, error) {
	ctx := context.Background()

	// Create a copy to avoid modifying the original
	profile := *encryptedCustomer

	// Decrypt sensitive data for display
	if err := decryptCustomer(ctx, crypto, &profile); err != nil {
		return nil, fmt.Errorf("failed to decrypt customer: %w", err)
	}

	return &profile, nil
}

// Pattern 4: Update customer email (maintains search capability)
func UpdateCustomerEmail(crypto *encx.Crypto, customer *Customer, newEmail string) error {
	ctx := context.Background()

	// Update the email field
	customer.Email = newEmail

	// Re-encrypt and hash with new email
	if err := processCustomer(ctx, crypto, customer); err != nil {
		return fmt.Errorf("failed to update customer email: %w", err)
	}

	// customer.EmailHash now contains hash of new email
	// customer.EmailEncrypted contains encrypted new email
	// You would update the database record here

	return nil
}

// Pattern 5: Batch customer search
func SearchCustomersByPhonePrefix(crypto *encx.Crypto, phoneNumbers []string) ([]string, error) {
	ctx := context.Background()
	var hashes []string

	for _, phone := range phoneNumbers {
		customer := &Customer{Phone: phone}
		if err := processCustomer(ctx, crypto, customer); err != nil {
			return nil, fmt.Errorf("failed to hash phone %s: %w", phone, err)
		}
		hashes = append(hashes, customer.PhoneHash)
	}

	// In real application:
	// SELECT * FROM customers WHERE phone_hash IN (?, ?, ?)
	fmt.Printf("Batch search for %d phone numbers\n", len(hashes))

	return hashes, nil
}

/*
Key Concepts Demonstrated:

1. **Combined Tags**: `encx:"encrypt,hash_basic"` provides both security and searchability
2. **Search Strategy**: Hash the search term to match stored hashes
3. **Security Balance**: Encrypt for protection, hash for performance
4. **Password Security**: Secure hashing for auth + encryption for recovery
5. **Data Lifecycle**: Register → Search → Display → Update

Database Schema for This Pattern:

CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100),
    last_name VARCHAR(100),

    -- Encrypted columns
    email_encrypted BYTEA,
    phone_encrypted BYTEA,
    address_encrypted BYTEA,

    -- Search hash columns
    email_hash VARCHAR(64) UNIQUE,
    phone_hash VARCHAR(64),
    customer_number_hash VARCHAR(64),

    -- Encryption metadata
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL
);

-- Indexes for fast searching
CREATE INDEX idx_customers_email_hash ON customers (email_hash);
CREATE INDEX idx_customers_phone_hash ON customers (phone_hash);

When to Use This Pattern:
- User management systems (email/phone lookup)
- Customer databases (search while protecting PII)
- E-commerce platforms (customer identification)
- Any system requiring both privacy and search capability

Security Benefits:
- Data encrypted at rest in database
- Search doesn't require decryption
- Original data never stored in plaintext
- Hash-based search prevents data leakage
*/