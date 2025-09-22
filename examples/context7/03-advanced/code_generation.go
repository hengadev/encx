// Package advanced demonstrates high-performance code generation with encx
//
// Context7 Tags: code-generation, performance-optimization, type-safety, production-ready
// Complexity: Advanced
// Use Case: High-performance encryption with compile-time type safety

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hengadev/encx"
)

//go:generate encx-gen validate -v .
//go:generate encx-gen generate -v .

// User demonstrates the new code generation approach
// This replaces the reflection-based ProcessStruct with generated functions
// Notice: NO companion fields needed - they're generated automatically
type User struct {
	// Basic information
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Searchable encrypted fields
	Email     string `json:"email" encx:"encrypt,hash_basic" db:"email"`
	Phone     string `json:"phone" encx:"encrypt,hash_basic" db:"phone"`

	// Personal information (encrypted only)
	FirstName string `json:"first_name" encx:"encrypt" db:"first_name"`
	LastName  string `json:"last_name" encx:"encrypt" db:"last_name"`
	Address   string `json:"address" encx:"encrypt" db:"address"`

	// Secure data (hashed only - no decryption needed)
	SSN string `json:"ssn" encx:"hash_secure" db:"ssn"`

	// No companion fields needed! Generated automatically in UserEncx struct
}

// Product demonstrates different encryption patterns
type Product struct {
	ID          int     `json:"id" db:"id"`
	PublicName  string  `json:"name" db:"name"`  // Not encrypted
	Price       float64 `json:"price" db:"price"` // Not encrypted

	// Encrypted fields
	InternalNotes string `json:"internal_notes" encx:"encrypt" db:"internal_notes"`
	SupplierInfo  string `json:"supplier_info" encx:"encrypt" db:"supplier_info"`

	// Searchable product code
	ProductCode string `json:"product_code" encx:"hash_basic" db:"product_code"`
}

func main() {
	ctx := context.Background()
	crypto, err := encx.NewTestCrypto(nil)
	if err != nil {
		log.Fatal("Failed to create crypto service:", err)
	}

	// Example 1: Using generated functions for User
	fmt.Println("=== Code Generation Example: User Processing ===")

	user := &User{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     "john.doe@example.com",
		Phone:     "+1-555-123-4567",
		FirstName: "John",
		LastName:  "Doe",
		Address:   "123 Main St, Springfield, IL",
		SSN:       "123-45-6789",
	}

	fmt.Printf("Original user: %+v\n", user)

	// Use generated function instead of crypto.ProcessStruct()
	// This is 10x faster and provides compile-time type safety
	userEncx, err := ProcessUserEncx(ctx, crypto, user)
	if err != nil {
		log.Fatal("Failed to process user with code generation:", err)
	}

	fmt.Printf("Generated UserEncx struct has encrypted data:\n")
	fmt.Printf("  EmailEncrypted: %d bytes\n", len(userEncx.EmailEncrypted))
	fmt.Printf("  EmailHash: %s...\n", userEncx.EmailHash[:16])
	fmt.Printf("  FirstNameEncrypted: %d bytes\n", len(userEncx.FirstNameEncrypted))
	fmt.Printf("  SSNHashSecure: %s...\n", userEncx.SSNHashSecure[:20])

	// Decrypt using generated function
	decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
	if err != nil {
		log.Fatal("Failed to decrypt user:", err)
	}

	fmt.Printf("Decrypted user: %+v\n", decryptedUser)

	fmt.Println()

	// Example 2: Product processing
	fmt.Println("=== Code Generation Example: Product Processing ===")

	product := &Product{
		ID:            100,
		PublicName:    "Widget Pro",
		Price:         29.99,
		InternalNotes: "High margin item, promote heavily",
		SupplierInfo:  "Acme Corp - contact: supplier@acme.com",
		ProductCode:   "WIDGET-PRO-2024",
	}

	// Use generated functions
	productEncx, err := ProcessProductEncx(ctx, crypto, product)
	if err != nil {
		log.Fatal("Failed to process product:", err)
	}

	fmt.Printf("Product encrypted successfully\n")
	fmt.Printf("  Public fields remain: Name=%s, Price=%.2f\n",
		productEncx.PublicName, productEncx.Price)
	fmt.Printf("  InternalNotesEncrypted: %d bytes\n", len(productEncx.InternalNotesEncrypted))
	fmt.Printf("  ProductCodeHash: %s...\n", productEncx.ProductCodeHash[:16])

	// Performance comparison demonstration
	fmt.Println("\n=== Performance Comparison ===")
	performanceComparison(ctx, crypto)
}

// Performance comparison between code generation and reflection
func performanceComparison(ctx context.Context, crypto *encx.Crypto) {
	user := &User{
		Email:     "perf.test@example.com",
		FirstName: "Performance",
		LastName:  "Test",
	}

	iterations := 1000

	// Benchmark code generation approach
	start := time.Now()
	for i := 0; i < iterations; i++ {
		userCopy := *user // Copy to avoid mutation
		_, err := ProcessUserEncx(ctx, crypto, &userCopy)
		if err != nil {
			log.Fatal("Code generation failed:", err)
		}
	}
	codeGenDuration := time.Since(start)

	// Benchmark reflection approach (deprecated)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		userCopy := *user // Copy to avoid mutation
		err := crypto.ProcessStruct(ctx, &userCopy)
		if err != nil {
			log.Fatal("Reflection failed:", err)
		}
	}
	reflectionDuration := time.Since(start)

	fmt.Printf("Performance results (%d iterations):\n", iterations)
	fmt.Printf("  Code Generation: %v (%.2f ns/op)\n",
		codeGenDuration, float64(codeGenDuration.Nanoseconds())/float64(iterations))
	fmt.Printf("  Reflection:      %v (%.2f ns/op)\n",
		reflectionDuration, float64(reflectionDuration.Nanoseconds())/float64(iterations))
	fmt.Printf("  Speedup:         %.1fx faster\n",
		float64(reflectionDuration)/float64(codeGenDuration))
}

// Production patterns using code generation

// Pattern 1: Service layer with generated functions
type UserService struct {
	crypto *encx.Crypto
}

func NewUserService(crypto *encx.Crypto) *UserService {
	return &UserService{crypto: crypto}
}

func (s *UserService) CreateUser(ctx context.Context, user *User) (*UserEncx, error) {
	// Validate input
	if user.Email == "" || user.FirstName == "" {
		return nil, fmt.Errorf("email and first name are required")
	}

	// Use generated function for type-safe encryption
	userEncx, err := ProcessUserEncx(ctx, s.crypto, user)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt user data: %w", err)
	}

	// In real application, save userEncx to database here
	// db.Create(userEncx)

	return userEncx, nil
}

func (s *UserService) GetUserProfile(ctx context.Context, userEncx *UserEncx) (*User, error) {
	// Use generated function for type-safe decryption
	user, err := DecryptUserEncx(ctx, s.crypto, userEncx)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt user data: %w", err)
	}

	return user, nil
}

func (s *UserService) FindUserByEmail(ctx context.Context, email string) (string, error) {
	// Create temporary user to generate search hash
	tempUser := &User{Email: email}
	userEncx, err := ProcessUserEncx(ctx, s.crypto, tempUser)
	if err != nil {
		return "", fmt.Errorf("failed to hash email: %w", err)
	}

	// Return hash for database query
	// In real app: SELECT * FROM users WHERE email_hash = userEncx.EmailHash
	return userEncx.EmailHash, nil
}

// Pattern 2: Batch processing with generated functions
func (s *UserService) ProcessUsersBatch(ctx context.Context, users []*User) ([]*UserEncx, error) {
	results := make([]*UserEncx, 0, len(users))

	for i, user := range users {
		userEncx, err := ProcessUserEncx(ctx, s.crypto, user)
		if err != nil {
			return nil, fmt.Errorf("failed to process user %d: %w", i, err)
		}
		results = append(results, userEncx)
	}

	return results, nil
}

// Pattern 3: Database integration with generated types
type UserRepository struct {
	// db *sql.DB // Your database connection
}

func (r *UserRepository) SaveUser(ctx context.Context, userEncx *UserEncx) error {
	// Example SQL for PostgreSQL
	query := `
		INSERT INTO users (
			email_encrypted, email_hash,
			phone_encrypted, phone_hash,
			first_name_encrypted, last_name_encrypted,
			address_encrypted, ssn_hash_secure,
			dek_encrypted, key_version, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	// In real code, you'd execute this query with userEncx fields
	fmt.Printf("Would execute: %s\n", query)
	fmt.Printf("With encrypted data for user\n")

	return nil
}

func (r *UserRepository) FindByEmailHash(ctx context.Context, emailHash string) (*UserEncx, error) {
	query := `
		SELECT email_encrypted, email_hash, phone_encrypted, phone_hash,
			   first_name_encrypted, last_name_encrypted, address_encrypted,
			   ssn_hash_secure, dek_encrypted, key_version, metadata
		FROM users WHERE email_hash = $1
	`

	// In real code, scan results into UserEncx
	fmt.Printf("Would execute: %s\n", query)
	fmt.Printf("With emailHash: %s\n", emailHash[:16]+"...")

	// Return mock UserEncx
	return &UserEncx{
		EmailHash: emailHash,
		// ... other fields would be populated from database
	}, nil
}

/*
Code Generation Benefits:

1. **Performance**: 10x faster than reflection-based approach
2. **Type Safety**: Compile-time checking of all operations
3. **IDE Support**: Full autocompletion and refactoring support
4. **Maintainability**: Clear, readable generated code
5. **Debugging**: Easy to step through generated functions

Setup Instructions:

1. Install CLI:
   make build-cli && make install-cli

2. Add go:generate directives:
   //go:generate encx-gen validate -v .
   //go:generate encx-gen generate -v .

3. Generate code:
   go generate

4. Use generated functions:
   - ProcessUserEncx(ctx, crypto, user) instead of crypto.ProcessStruct()
   - DecryptUserEncx(ctx, crypto, userEncx) instead of crypto.DecryptStruct()

Generated Functions Pattern:
- Process{StructName}Encx(ctx, crypto, source) → (*{StructName}Encx, error)
- Decrypt{StructName}Encx(ctx, crypto, source) → (*{StructName}, error)

Migration from Reflection:
OLD: crypto.ProcessStruct(ctx, &user)
NEW: userEncx, err := ProcessUserEncx(ctx, crypto, &user)

Configuration (encx.yaml):
version: "1.0"
generation:
  output_suffix: "_encx"
  function_prefix: "Process"
  package_name: "main"

When to Use Code Generation:
- Production applications requiring high performance
- Large-scale data processing
- Systems with strict type safety requirements
- Applications with complex encryption patterns
- Any system where reflection overhead is a concern
*/