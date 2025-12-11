package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverStructs(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test files
	testFile1 := filepath.Join(tempDir, "user.go")
	err := os.WriteFile(testFile1, []byte(`package test

// User represents a user with encrypted fields
type User struct {
	ID    int    ` + "`json:\"id\"`" + `
	Email string ` + "`json:\"email\" encx:\"encrypt,hash_basic\"`" + `
	Phone string ` + "`json:\"phone\" encx:\"encrypt\"`" + `
	SSN   string ` + "`json:\"ssn\" encx:\"hash_secure\"`" + `
	Name  string ` + "`json:\"name\"`" + `

	// Companion fields
	EmailEncrypted []byte ` + "`json:\"email_encrypted\" db:\"email_encrypted\"`" + `
	EmailHash      string ` + "`json:\"email_hash\" db:\"email_hash\"`" + `
	PhoneEncrypted []byte ` + "`json:\"phone_encrypted\" db:\"phone_encrypted\"`" + `
	SSNHashSecure  string ` + "`json:\"ssn_hash_secure\" db:\"ssn_hash_secure\"`" + `
}
`), 0644)
	require.NoError(t, err)

	testFile2 := filepath.Join(tempDir, "product.go")
	err = os.WriteFile(testFile2, []byte(`package test

// Product with no encx tags
type Product struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// SecureProduct with encx tags
type SecureProduct struct {
	ID          int    ` + "`json:\"id\"`" + `
	Description string ` + "`json:\"description\" encx:\"encrypt\"`" + `

	// Companion field
	DescriptionEncrypted []byte ` + "`json:\"description_encrypted\"`" + `
}
`), 0644)
	require.NoError(t, err)

	config := &DiscoveryConfig{}
	structs, err := DiscoverStructs(tempDir, config)

	require.NoError(t, err)
	assert.Len(t, structs, 2) // User and SecureProduct

	// Find User struct
	var userStruct *StructInfo
	var productStruct *StructInfo
	for i := range structs {
		if structs[i].StructName == "User" {
			userStruct = &structs[i]
		} else if structs[i].StructName == "SecureProduct" {
			productStruct = &structs[i]
		}
	}

	require.NotNil(t, userStruct)
	require.NotNil(t, productStruct)

	// Test User struct
	assert.Equal(t, "User", userStruct.StructName)
	assert.Equal(t, "test", userStruct.PackageName)
	assert.Equal(t, "user.go", filepath.Base(userStruct.SourceFile))
	assert.True(t, userStruct.HasEncxTags)

	// Find fields with encx tags
	var fieldsWithEncxTags []FieldInfo
	for _, field := range userStruct.Fields {
		if len(field.EncxTags) > 0 {
			fieldsWithEncxTags = append(fieldsWithEncxTags, field)
		}
	}
	assert.Len(t, fieldsWithEncxTags, 3) // Email, Phone, SSN

	// Test Email field with multiple tags
	emailField := findField(userStruct.Fields, "Email")
	require.NotNil(t, emailField)
	assert.Equal(t, "Email", emailField.Name)
	assert.Equal(t, "string", emailField.Type)
	assert.True(t, emailField.IsValid)
	assert.Len(t, emailField.EncxTags, 2)
	assert.Contains(t, emailField.EncxTags, "encrypt")
	assert.Contains(t, emailField.EncxTags, "hash_basic")

	// Test Phone field with single tag
	phoneField := findField(userStruct.Fields, "Phone")
	require.NotNil(t, phoneField)
	assert.Equal(t, "Phone", phoneField.Name)
	assert.Equal(t, "string", phoneField.Type)
	assert.True(t, phoneField.IsValid)
	assert.Len(t, phoneField.EncxTags, 1)
	assert.Contains(t, phoneField.EncxTags, "encrypt")

	// Test SSN field
	ssnField := findField(userStruct.Fields, "SSN")
	require.NotNil(t, ssnField)
	assert.Equal(t, "SSN", ssnField.Name)
	assert.Equal(t, "string", ssnField.Type)
	assert.True(t, ssnField.IsValid)
	assert.Len(t, ssnField.EncxTags, 1)
	assert.Contains(t, ssnField.EncxTags, "hash_secure")

	// Test SecureProduct struct
	assert.Equal(t, "SecureProduct", productStruct.StructName)
	assert.Equal(t, "test", productStruct.PackageName)
	assert.Equal(t, "product.go", filepath.Base(productStruct.SourceFile))
	assert.True(t, productStruct.HasEncxTags)

	// Find fields with encx tags
	var productFieldsWithEncxTags []FieldInfo
	for _, field := range productStruct.Fields {
		if len(field.EncxTags) > 0 {
			productFieldsWithEncxTags = append(productFieldsWithEncxTags, field)
		}
	}
	assert.Len(t, productFieldsWithEncxTags, 1) // Description

	descField := findField(productStruct.Fields, "Description")
	require.NotNil(t, descField)
	assert.Equal(t, "Description", descField.Name)
	assert.Equal(t, "string", descField.Type)
	assert.True(t, descField.IsValid)
	assert.Len(t, descField.EncxTags, 1)
	assert.Contains(t, descField.EncxTags, "encrypt")
}

func TestDiscoverStructsWithValidationErrors(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test file with validation errors
	testFile := filepath.Join(tempDir, "invalid.go")
	err := os.WriteFile(testFile, []byte(`package test

// InvalidUser demonstrates validation errors
type InvalidUser struct {
	// Invalid combination: hash_basic and hash_secure
	Email string ` + "`json:\"email\" encx:\"hash_basic,hash_secure\"`" + `

	// Valid fields (no companion fields needed with code generation)
	Phone string ` + "`json:\"phone\" encx:\"encrypt\"`" + `
	Name  string ` + "`json:\"name\" encx:\"encrypt\"`" + `

	// Unknown tag
	BadField string ` + "`json:\"bad_field\" encx:\"unknown_tag\"`" + `
}
`), 0644)
	require.NoError(t, err)

	config := &DiscoveryConfig{}
	structs, err := DiscoverStructs(tempDir, config)

	require.NoError(t, err)
	assert.Len(t, structs, 1)

	invalidStruct := structs[0]
	assert.Equal(t, "InvalidUser", invalidStruct.StructName)

	// Check that only invalid tag combinations and unknown tags have validation errors
	emailField := findField(invalidStruct.Fields, "Email")
	require.NotNil(t, emailField)
	assert.False(t, emailField.IsValid)
	assert.NotEmpty(t, emailField.ValidationErrors)
	// Check that the error contains the conflicting hash tags (order may vary)
	errorMsg := emailField.ValidationErrors[0]
	assert.True(t, strings.Contains(errorMsg, "hash_basic") && strings.Contains(errorMsg, "hash_secure"),
		"Expected error to contain both hash_basic and hash_secure, got: %s", errorMsg)

	// Phone and Name should be valid (no companion fields required)
	phoneField := findField(invalidStruct.Fields, "Phone")
	require.NotNil(t, phoneField)
	assert.True(t, phoneField.IsValid)
	assert.Empty(t, phoneField.ValidationErrors)

	nameField := findField(invalidStruct.Fields, "Name")
	require.NotNil(t, nameField)
	assert.True(t, nameField.IsValid)
	assert.Empty(t, nameField.ValidationErrors)

	// BadField should have unknown tag error
	badField := findField(invalidStruct.Fields, "BadField")
	require.NotNil(t, badField)
	assert.False(t, badField.IsValid)
	assert.NotEmpty(t, badField.ValidationErrors)
	assert.Contains(t, badField.ValidationErrors[0], "unknown tag 'unknown_tag'")
}

func TestDiscoverStructsEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	config := &DiscoveryConfig{}
	structs, err := DiscoverStructs(tempDir, config)

	require.NoError(t, err)
	assert.Empty(t, structs)
}

func TestDiscoverStructsNonExistentDirectory(t *testing.T) {
	config := &DiscoveryConfig{}
	_, err := DiscoverStructs("/nonexistent/path", config)

	assert.Error(t, err)
}

func TestDiscoverStructsWithBuildTags(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file with build tags
	testFile := filepath.Join(tempDir, "tagged.go")
	err := os.WriteFile(testFile, []byte(`//go:build testing
// +build testing

package test

// TaggedStruct only available with testing build tag
type TaggedStruct struct {
	Secret string ` + "`json:\"secret\" encx:\"encrypt\"`" + `

	SecretEncrypted []byte ` + "`json:\"secret_encrypted\"`" + `
}
`), 0644)
	require.NoError(t, err)

	config := &DiscoveryConfig{}
	structs, err := DiscoverStructs(tempDir, config)

	// Should still discover the struct even with build tags
	require.NoError(t, err)
	assert.Len(t, structs, 1)
	assert.Equal(t, "TaggedStruct", structs[0].StructName)
}

func findField(fields []FieldInfo, name string) *FieldInfo {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func TestDiscoverStructsWithEmbeddedFields(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test file with embedded structs (similar to the user's DocumentBase and Estimate)
	testFile := filepath.Join(tempDir, "embedded.go")
	err := os.WriteFile(testFile, []byte(`package test

import "time"

// DocumentBase contains metadata shared by all document types
type DocumentBase struct {
	ID        int       `+"`json:\"id\"`"+`
	CaseID    int       `+"`json:\"case_id\" encx:\"encrypt\"`"+`
	ClientID  int       `+"`json:\"client_id\" encx:\"encrypt\"`"+`
	Status    string    `+"`json:\"status\"`"+`
	CreatedAt time.Time `+"`json:\"created_at\"`"+`
	UpdatedAt time.Time `+"`json:\"updated_at\"`"+`
}

// Estimate represents a price quotation with embedded DocumentBase
type Estimate struct {
	DocumentBase

	EstimateNumber string    `+"`encx:\"encrypt\" json:\"estimate_number\"`"+`
	IssueDate      time.Time `+"`json:\"issue_date\"`"+`
	ValidUntil     *time.Time `+"`json:\"valid_until,omitempty\"`"+`
	EstimatedTotal float64   `+"`encx:\"encrypt\" json:\"estimated_total\"`"+`
	Notes          *string   `+"`encx:\"encrypt\" json:\"notes,omitempty\"`"+`
	Accepted       bool      `+"`json:\"accepted\"`"+`
}
`), 0644)
	require.NoError(t, err)

	config := &DiscoveryConfig{}
	structs, err := DiscoverStructs(tempDir, config)

	require.NoError(t, err)

	// Find the Estimate struct (DocumentBase has encx tags but they're only in CaseID and ClientID)
	var estimateStruct *StructInfo
	var documentBaseStruct *StructInfo
	for i := range structs {
		if structs[i].StructName == "Estimate" {
			estimateStruct = &structs[i]
		} else if structs[i].StructName == "DocumentBase" {
			documentBaseStruct = &structs[i]
		}
	}

	// Both structs should be discovered since they have encx tags
	require.NotNil(t, estimateStruct, "Estimate struct should be discovered")
	require.NotNil(t, documentBaseStruct, "DocumentBase struct should be discovered")

	// Test that Estimate includes all fields from DocumentBase
	assert.Equal(t, "Estimate", estimateStruct.StructName)
	assert.True(t, estimateStruct.HasEncxTags)

	// Estimate should have fields from both DocumentBase and itself
	// DocumentBase: ID, CaseID, ClientID, Status, CreatedAt, UpdatedAt (6 fields)
	// Estimate: EstimateNumber, IssueDate, ValidUntil, EstimatedTotal, Notes, Accepted (6 fields)
	// Total: 12 fields
	assert.Len(t, estimateStruct.Fields, 12, "Estimate should have all fields from DocumentBase plus its own")

	// Verify DocumentBase fields are included
	idField := findField(estimateStruct.Fields, "ID")
	assert.NotNil(t, idField, "ID from DocumentBase should be included")
	assert.Equal(t, "int", idField.Type)

	caseIDField := findField(estimateStruct.Fields, "CaseID")
	assert.NotNil(t, caseIDField, "CaseID from DocumentBase should be included")
	assert.Equal(t, "int", caseIDField.Type)
	assert.Contains(t, caseIDField.EncxTags, "encrypt", "CaseID should have encrypt tag")

	clientIDField := findField(estimateStruct.Fields, "ClientID")
	assert.NotNil(t, clientIDField, "ClientID from DocumentBase should be included")
	assert.Equal(t, "int", clientIDField.Type)
	assert.Contains(t, clientIDField.EncxTags, "encrypt", "ClientID should have encrypt tag")

	statusField := findField(estimateStruct.Fields, "Status")
	assert.NotNil(t, statusField, "Status from DocumentBase should be included")
	assert.Equal(t, "string", statusField.Type)

	createdAtField := findField(estimateStruct.Fields, "CreatedAt")
	assert.NotNil(t, createdAtField, "CreatedAt from DocumentBase should be included")
	assert.Equal(t, "time.Time", createdAtField.Type)

	updatedAtField := findField(estimateStruct.Fields, "UpdatedAt")
	assert.NotNil(t, updatedAtField, "UpdatedAt from DocumentBase should be included")
	assert.Equal(t, "time.Time", updatedAtField.Type)

	// Verify Estimate's own fields
	estimateNumberField := findField(estimateStruct.Fields, "EstimateNumber")
	assert.NotNil(t, estimateNumberField, "EstimateNumber should be included")
	assert.Equal(t, "string", estimateNumberField.Type)
	assert.Contains(t, estimateNumberField.EncxTags, "encrypt")

	issueDateField := findField(estimateStruct.Fields, "IssueDate")
	assert.NotNil(t, issueDateField, "IssueDate should be included")
	assert.Equal(t, "time.Time", issueDateField.Type)

	validUntilField := findField(estimateStruct.Fields, "ValidUntil")
	assert.NotNil(t, validUntilField, "ValidUntil should be included")
	assert.Equal(t, "*time.Time", validUntilField.Type)

	estimatedTotalField := findField(estimateStruct.Fields, "EstimatedTotal")
	assert.NotNil(t, estimatedTotalField, "EstimatedTotal should be included")
	assert.Equal(t, "float64", estimatedTotalField.Type)
	assert.Contains(t, estimatedTotalField.EncxTags, "encrypt")

	notesField := findField(estimateStruct.Fields, "Notes")
	assert.NotNil(t, notesField, "Notes should be included")
	assert.Equal(t, "*string", notesField.Type)
	assert.Contains(t, notesField.EncxTags, "encrypt")

	acceptedField := findField(estimateStruct.Fields, "Accepted")
	assert.NotNil(t, acceptedField, "Accepted should be included")
	assert.Equal(t, "bool", acceptedField.Type)

	// Count fields with encx tags (should be 5: CaseID, ClientID, EstimateNumber, EstimatedTotal, Notes)
	var fieldsWithEncxTags []FieldInfo
	for _, field := range estimateStruct.Fields {
		if len(field.EncxTags) > 0 {
			fieldsWithEncxTags = append(fieldsWithEncxTags, field)
		}
	}
	assert.Len(t, fieldsWithEncxTags, 5, "Should have 5 fields with encx tags")
}