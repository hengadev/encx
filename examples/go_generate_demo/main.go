// Package go_generate_demo demonstrates encx-gen integration
//
// Alternative generation methods:
//   1. Direct commands (recommended): encx-gen validate -v . && encx-gen generate -v .
//   2. Go generate (optional): go generate . (requires //go:generate directives below)
//
//go:generate encx-gen validate -v .
//go:generate encx-gen generate -v .
package go_generate_demo

// ExampleUser demonstrates a user struct with encrypted fields
// This struct will be automatically discovered by encx-gen regardless of go:generate directives
type ExampleUser struct {
	ID    int    `json:"id"`
	Email string `json:"email" encx:"encrypt,hash_basic"`
	Phone string `json:"phone" encx:"encrypt"`
	SSN   string `json:"ssn" encx:"hash_secure"`

	// Companion fields for encryption/hashing
	EmailEncrypted []byte `json:"email_encrypted" db:"email_encrypted"`
	EmailHash      string `json:"email_hash" db:"email_hash"`
	PhoneEncrypted []byte `json:"phone_encrypted" db:"phone_encrypted"`
	SSNHashSecure  string `json:"ssn_hash_secure" db:"ssn_hash_secure"`

	// Essential encryption fields
	DEKEncrypted []byte `json:"dek_encrypted" db:"dek_encrypted"`
	KeyVersion   int    `json:"key_version" db:"key_version"`
	Metadata     string `json:"metadata" db:"metadata"`

	// Standard fields
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Code generation options:
//   1. Direct commands (recommended): encx-gen generate ./examples
//   2. Go generate: go generate ./examples
//   3. Entire project: go generate ./...
//
// The generated file will be: examples/go_generate_example_encx.go