// Package go_generate_demo demonstrates encx-gen integration with go generate
//
//go:generate encx-gen validate -v .
//go:generate encx-gen generate -v .
package go_generate_demo

// ExampleUser demonstrates a user struct with encrypted fields
// The go:generate directives above will validate and generate code for this struct
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

// To run code generation manually:
//   go generate ./examples
//
// To run generation for entire project:
//   go generate ./...
//
// The generated file will be: examples/go_generate_example_encx.go