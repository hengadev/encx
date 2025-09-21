package codegen

import (
	"os"
	"path/filepath"
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

	// Missing companion field (should have PhoneEncrypted []byte)
	Phone string ` + "`json:\"phone\" encx:\"encrypt\"`" + `

	// Wrong type companion field (should be []byte, not string)
	Name          string ` + "`json:\"name\" encx:\"encrypt\"`" + `
	NameEncrypted string ` + "`json:\"name_encrypted\"`" + ` // Wrong type!
}
`), 0644)
	require.NoError(t, err)

	config := &DiscoveryConfig{}
	structs, err := DiscoverStructs(tempDir, config)

	require.NoError(t, err)
	assert.Len(t, structs, 1)

	invalidStruct := structs[0]
	assert.Equal(t, "InvalidUser", invalidStruct.StructName)

	// Check that fields have validation errors
	emailField := findField(invalidStruct.Fields, "Email")
	require.NotNil(t, emailField)
	assert.False(t, emailField.IsValid)
	assert.NotEmpty(t, emailField.ValidationErrors)

	phoneField := findField(invalidStruct.Fields, "Phone")
	require.NotNil(t, phoneField)
	assert.False(t, phoneField.IsValid)
	assert.NotEmpty(t, phoneField.ValidationErrors)

	nameField := findField(invalidStruct.Fields, "Name")
	require.NotNil(t, nameField)
	assert.False(t, nameField.IsValid)
	assert.NotEmpty(t, nameField.ValidationErrors)
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