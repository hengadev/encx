// Package basic demonstrates simple field-level encryption with encx
//
// Context7 Tags: basic-encryption, field-encryption, data-protection, golang-crypto
// Complexity: Beginner
// Use Case: Protecting sensitive data fields in structs

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hengadev/encx"
)

// Document demonstrates basic field encryption
// Use this pattern when you need to protect sensitive data
// but don't need to search by the encrypted field
type Document struct {
	// Public fields - not encrypted
	ID          int    `json:"id"`
	Title       string `json:"title"`
	AuthorName  string `json:"author"`

	// Sensitive field - will be encrypted
	Content         string `encx:"encrypt" json:"content"`
	ContentEncrypted []byte `json:"content_encrypted"` // Companion field

	// Required encryption fields - populated automatically
	DEK          []byte `json:"-"`          // Data Encryption Key
	DEKEncrypted []byte `json:"dek_encrypted"` // Encrypted DEK
	KeyVersion   int    `json:"key_version"`   // Key version for rotation
}

// PersonalNote demonstrates encrypting personal information
type PersonalNote struct {
	// Basic information
	Date    string `json:"date"`
	Subject string `json:"subject"`

	// Encrypted fields
	Note            string `encx:"encrypt" json:"note"`
	NoteEncrypted   []byte `json:"note_encrypted"`

	Location         string `encx:"encrypt" json:"location"`
	LocationEncrypted []byte `json:"location_encrypted"`

	// Required fields
	DEK          []byte `json:"-"`
	DEKEncrypted []byte `json:"dek_encrypted"`
	KeyVersion   int    `json:"key_version"`
}

func main() {
	ctx := context.Background()

	// Create a test crypto instance (use proper KMS in production)
	crypto, err := encx.NewTestCrypto(nil)
	if err != nil {
		log.Fatal("Failed to create crypto service:", err)
	}

	// Example 1: Document encryption
	fmt.Println("=== Document Encryption Example ===")

	doc := &Document{
		ID:         1,
		Title:      "Meeting Notes",
		AuthorName: "John Doe",
		Content:    "Confidential meeting discussion about Q4 strategy...",
	}

	fmt.Printf("Original content: %s\n", doc.Content)

	// Encrypt the document
	if err := crypto.ProcessStruct(ctx, doc); err != nil {
		log.Fatal("Failed to encrypt document:", err)
	}

	fmt.Printf("Content after encryption: '%s' (cleared)\n", doc.Content)
	fmt.Printf("Encrypted content size: %d bytes\n", len(doc.ContentEncrypted))
	fmt.Printf("Key version: %d\n", doc.KeyVersion)

	// Decrypt the document
	if err := crypto.DecryptStruct(ctx, doc); err != nil {
		log.Fatal("Failed to decrypt document:", err)
	}

	fmt.Printf("Decrypted content: %s\n", doc.Content)

	fmt.Println()

	// Example 2: Personal note with multiple encrypted fields
	fmt.Println("=== Personal Note Example ===")

	note := &PersonalNote{
		Date:     "2024-01-15",
		Subject:  "Doctor Appointment",
		Note:     "Discussed treatment options and medication changes",
		Location: "Main Street Medical Center, Room 201",
	}

	fmt.Printf("Original note: %s\n", note.Note)
	fmt.Printf("Original location: %s\n", note.Location)

	// Encrypt the note
	if err := crypto.ProcessStruct(ctx, note); err != nil {
		log.Fatal("Failed to encrypt note:", err)
	}

	fmt.Printf("Note after encryption: '%s' (cleared)\n", note.Note)
	fmt.Printf("Location after encryption: '%s' (cleared)\n", note.Location)
	fmt.Printf("Note encrypted size: %d bytes\n", len(note.NoteEncrypted))
	fmt.Printf("Location encrypted size: %d bytes\n", len(note.LocationEncrypted))

	// Decrypt the note
	if err := crypto.DecryptStruct(ctx, note); err != nil {
		log.Fatal("Failed to decrypt note:", err)
	}

	fmt.Printf("Decrypted note: %s\n", note.Note)
	fmt.Printf("Decrypted location: %s\n", note.Location)
}

// Example usage patterns:

// Pattern 1: Basic encryption for sensitive text
func EncryptSensitiveDocument(content string) (*Document, error) {
	ctx := context.Background()
	crypto, _ := encx.NewTestCrypto(nil)

	doc := &Document{
		Title:   "Sensitive Document",
		Content: content,
	}

	if err := crypto.ProcessStruct(ctx, doc); err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return doc, nil
}

// Pattern 2: Decrypt for viewing
func ViewDocument(doc *Document) (string, error) {
	ctx := context.Background()
	crypto, _ := encx.NewTestCrypto(nil)

	if err := crypto.DecryptStruct(ctx, doc); err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return doc.Content, nil
}

// Pattern 3: Check if data is encrypted
func IsDocumentEncrypted(doc *Document) bool {
	// If content is empty but encrypted data exists, it's encrypted
	return doc.Content == "" && len(doc.ContentEncrypted) > 0
}

/*
Key Concepts Demonstrated:

1. **Basic Encryption**: Use `encx:"encrypt"` tag for sensitive fields
2. **Companion Fields**: Every encrypted field needs a `*Encrypted []byte` companion
3. **Required Fields**: All structs need DEK, DEKEncrypted, KeyVersion
4. **Process/Decrypt Cycle**: ProcessStruct encrypts, DecryptStruct decrypts
5. **Data Clearing**: Original data is cleared after encryption for security

When to Use This Pattern:
- Protecting sensitive content that doesn't need to be searchable
- Personal information, medical records, financial details
- Any data that should be encrypted at rest in database

Security Notes:
- Use proper KMS service in production (not NewTestCrypto)
- Original sensitive data is automatically cleared after encryption
- Decryption should only be done when data needs to be displayed/used
- Store only the encrypted version in your database
*/