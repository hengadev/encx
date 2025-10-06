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

	// Manual encryption approach (without ProcessStruct)
	// Generate a DEK for this document
	dek, err := crypto.GenerateDEK()
	if err != nil {
		log.Fatal("Failed to generate DEK:", err)
	}

	// Encrypt the sensitive content
	contentBytes := []byte(doc.Content)
	encryptedContent, err := crypto.EncryptData(ctx, contentBytes, dek)
	if err != nil {
		log.Fatal("Failed to encrypt content:", err)
	}

	// Encrypt the DEK with the KEK
	encryptedDEK, err := crypto.EncryptDEK(ctx, dek)
	if err != nil {
		log.Fatal("Failed to encrypt DEK:", err)
	}

	// Store encrypted data and clear original
	doc.ContentEncrypted = encryptedContent
	doc.DEKEncrypted = encryptedDEK
	doc.KeyVersion = 1 // In practice, get this from key metadata
	doc.Content = "" // Clear for security

	fmt.Printf("Content after encryption: '%s' (cleared)\n", doc.Content)
	fmt.Printf("Encrypted content size: %d bytes\n", len(doc.ContentEncrypted))
	fmt.Printf("Key version: %d\n", doc.KeyVersion)

	// Manual decryption approach
	// First decrypt the DEK
	decryptedDEK, err := crypto.DecryptDEKWithVersion(ctx, doc.DEKEncrypted, doc.KeyVersion)
	if err != nil {
		log.Fatal("Failed to decrypt DEK:", err)
	}

	// Then decrypt the content
	decryptedContent, err := crypto.DecryptData(ctx, doc.ContentEncrypted, decryptedDEK)
	if err != nil {
		log.Fatal("Failed to decrypt content:", err)
	}

	// Restore the original content
	doc.Content = string(decryptedContent)

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

	// Manual encryption for multiple fields
	// Generate a DEK for this note
	noteDEK, err := crypto.GenerateDEK()
	if err != nil {
		log.Fatal("Failed to generate DEK for note:", err)
	}

	// Encrypt both fields with the same DEK
	noteBytes := []byte(note.Note)
	encryptedNote, err := crypto.EncryptData(ctx, noteBytes, noteDEK)
	if err != nil {
		log.Fatal("Failed to encrypt note:", err)
	}

	locationBytes := []byte(note.Location)
	encryptedLocation, err := crypto.EncryptData(ctx, locationBytes, noteDEK)
	if err != nil {
		log.Fatal("Failed to encrypt location:", err)
	}

	// Encrypt the DEK
	encryptedNoteDEK, err := crypto.EncryptDEK(ctx, noteDEK)
	if err != nil {
		log.Fatal("Failed to encrypt DEK:", err)
	}

	// Store encrypted data and clear originals
	note.NoteEncrypted = encryptedNote
	note.LocationEncrypted = encryptedLocation
	note.DEKEncrypted = encryptedNoteDEK
	note.KeyVersion = 1
	note.Note = ""        // Clear for security
	note.Location = ""    // Clear for security

	fmt.Printf("Note after encryption: '%s' (cleared)\n", note.Note)
	fmt.Printf("Location after encryption: '%s' (cleared)\n", note.Location)
	fmt.Printf("Note encrypted size: %d bytes\n", len(note.NoteEncrypted))
	fmt.Printf("Location encrypted size: %d bytes\n", len(note.LocationEncrypted))

	// Manual decryption for multiple fields
	// Decrypt the DEK first
	decryptedNoteDEK, err := crypto.DecryptDEKWithVersion(ctx, note.DEKEncrypted, note.KeyVersion)
	if err != nil {
		log.Fatal("Failed to decrypt DEK:", err)
	}

	// Decrypt both fields
	decryptedNoteBytes, err := crypto.DecryptData(ctx, note.NoteEncrypted, decryptedNoteDEK)
	if err != nil {
		log.Fatal("Failed to decrypt note:", err)
	}

	decryptedLocationBytes, err := crypto.DecryptData(ctx, note.LocationEncrypted, decryptedNoteDEK)
	if err != nil {
		log.Fatal("Failed to decrypt location:", err)
	}

	// Restore original values
	note.Note = string(decryptedNoteBytes)
	note.Location = string(decryptedLocationBytes)

	fmt.Printf("Decrypted note: %s\n", note.Note)
	fmt.Printf("Decrypted location: %s\n", note.Location)
}

// Example usage patterns:

// Pattern 1: Manual encryption for sensitive text
func EncryptSensitiveDocument(content string) (*Document, error) {
	ctx := context.Background()
	crypto, _ := encx.NewTestCrypto(nil)

	doc := &Document{
		Title:   "Sensitive Document",
		Content: content,
	}

	// Manual encryption process
	dek, err := crypto.GenerateDEK()
	if err != nil {
		return nil, fmt.Errorf("DEK generation failed: %w", err)
	}

	contentBytes := []byte(doc.Content)
	encryptedContent, err := crypto.EncryptData(ctx, contentBytes, dek)
	if err != nil {
		return nil, fmt.Errorf("content encryption failed: %w", err)
	}

	encryptedDEK, err := crypto.EncryptDEK(ctx, dek)
	if err != nil {
		return nil, fmt.Errorf("DEK encryption failed: %w", err)
	}

	doc.ContentEncrypted = encryptedContent
	doc.DEKEncrypted = encryptedDEK
	doc.KeyVersion = 1
	doc.Content = "" // Clear for security

	return doc, nil
}

// Pattern 2: Manual decryption for viewing
func ViewDocument(doc *Document) (string, error) {
	ctx := context.Background()
	crypto, _ := encx.NewTestCrypto(nil)

	// Manual decryption process
	decryptedDEK, err := crypto.DecryptDEKWithVersion(ctx, doc.DEKEncrypted, doc.KeyVersion)
	if err != nil {
		return "", fmt.Errorf("DEK decryption failed: %w", err)
	}

	decryptedContent, err := crypto.DecryptData(ctx, doc.ContentEncrypted, decryptedDEK)
	if err != nil {
		return "", fmt.Errorf("content decryption failed: %w", err)
	}

	return string(decryptedContent), nil
}

// Pattern 3: Check if data is encrypted
func IsDocumentEncrypted(doc *Document) bool {
	// If content is empty but encrypted data exists, it's encrypted
	return doc.Content == "" && len(doc.ContentEncrypted) > 0
}

/*
Key Concepts Demonstrated:

1. **Manual Encryption**: Direct use of EncryptData/DecryptData for fine control
2. **DEK Management**: Generate, encrypt, and decrypt Data Encryption Keys
3. **Multiple Fields**: Use same DEK for multiple fields in same record
4. **Data Clearing**: Clear original data after encryption for security
5. **Version Tracking**: Track key versions for proper decryption

When to Use This Pattern:
- When you need fine control over the encryption process
- For educational purposes to understand the underlying mechanics
- When you can't use generated code (encx-gen)
- For simple scenarios with few encrypted fields

Modern Alternative:
For production code, consider using encx-gen for generated type-safe functions:
- Run: encx-gen on your struct files
- Use generated functions like ProcessDocumentEncx(ctx, crypto, doc)
- Provides better performance and compile-time safety

Security Notes:
- Use proper KMS service in production (not NewTestCrypto)
- Always clear original sensitive data after encryption
- Store DEK encrypted, never in plaintext
- Use same DEK for related fields in one record to minimize key operations
*/