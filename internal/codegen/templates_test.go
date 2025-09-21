package codegen

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateEngine(t *testing.T) {
	engine, err := NewTemplateEngine()
	require.NoError(t, err)
	assert.NotNil(t, engine)
}

func TestGenerateCodeSimpleStruct(t *testing.T) {
	engine, err := NewTemplateEngine()
	require.NoError(t, err)

	// Create template data for a simple struct with one encrypted field
	data := TemplateData{
		PackageName:            "test",
		StructName:             "User",
		SourceFile:             "user.go",
		GeneratedTime:          time.Now().Format(time.RFC3339),
		SerializerType:         "json",
		SerializerFromMetadata: "&serialization.JSONSerializer{}",
		GeneratorVersion:       "1.0.0",
		EncryptedFields: []TemplateField{
			{
				Name:      "EmailEncrypted",
				Type:      "[]byte",
				DBColumn:  "email_encrypted",
				JSONField: "email_encrypted",
			},
		},
		ProcessingSteps: []string{
			"// Process Email (encrypt)",
			"if source.Email != \"\" {",
			"\tEmailBytes, err := serializer.Serialize(source.Email)",
			"\tif err != nil {",
			"\t\terrs.Set(\"Email serialization\", err)",
			"\t} else {",
			"\t\tresult.EmailEncrypted, err = crypto.EncryptData(ctx, EmailBytes, dek)",
			"\t\tif err != nil {",
			"\t\t\terrs.Set(\"Email encryption\", err)",
			"\t\t}",
			"\t}",
			"}",
		},
		DecryptionSteps: []string{
			"// Decrypt Email",
			"if len(source.EmailEncrypted) > 0 {",
			"\tEmailBytes, err := crypto.DecryptData(ctx, source.EmailEncrypted, dek)",
			"\tif err != nil {",
			"\t\terrs.Set(\"Email decryption\", err)",
			"\t} else {",
			"\t\terr = serializer.Deserialize(EmailBytes, &result.Email)",
			"\t\tif err != nil {",
			"\t\t\terrs.Set(\"Email deserialization\", err)",
			"\t\t}",
			"\t}",
			"}",
		},
	}

	code, err := engine.GenerateCode(data)
	require.NoError(t, err)

	codeStr := string(code)

	// Verify generated code contains expected elements
	assert.Contains(t, codeStr, "package test")
	assert.Contains(t, codeStr, "type UserEncx struct")
	assert.Contains(t, codeStr, "EmailEncrypted []byte")
	assert.Contains(t, codeStr, "func ProcessUserEncx(ctx context.Context, crypto *encx.Crypto, source *User) (*UserEncx, error)")
	assert.Contains(t, codeStr, "func DecryptUserEncx(ctx context.Context, crypto *encx.Crypto, source *UserEncx) (*User, error)")

	// Verify encryption logic
	assert.Contains(t, codeStr, "crypto.EncryptData(ctx, EmailBytes, dek)")

	// Verify decryption logic
	assert.Contains(t, codeStr, "crypto.DecryptData(ctx, source.EmailEncrypted, dek)")

	// Verify error handling
	assert.Contains(t, codeStr, "errsx.Map")
	assert.Contains(t, codeStr, "errs.AsError()")

	// Verify serialization
	assert.Contains(t, codeStr, "serializer.Serialize(source.Email)")
	assert.Contains(t, codeStr, "serializer.Deserialize(EmailBytes, &result.Email)")
}

func TestGenerateCodeWithMultipleFields(t *testing.T) {
	engine, err := NewTemplateEngine()
	require.NoError(t, err)

	// Create template data with multiple encrypted fields
	data := TemplateData{
		PackageName:            "test",
		StructName:             "User",
		SourceFile:             "user.go",
		GeneratedTime:          time.Now().Format(time.RFC3339),
		SerializerType:         "json",
		SerializerFromMetadata: "&serialization.JSONSerializer{}",
		GeneratorVersion:       "1.0.0",
		EncryptedFields: []TemplateField{
			{
				Name:      "EmailEncrypted",
				Type:      "[]byte",
				DBColumn:  "email_encrypted",
				JSONField: "email_encrypted",
			},
			{
				Name:      "PhoneEncrypted",
				Type:      "[]byte",
				DBColumn:  "phone_encrypted",
				JSONField: "phone_encrypted",
			},
		},
		ProcessingSteps: []string{
			"// Processing steps would be here",
		},
		DecryptionSteps: []string{
			"// Decryption steps would be here",
		},
	}

	code, err := engine.GenerateCode(data)
	require.NoError(t, err)

	codeStr := string(code)

	// Verify multiple fields in struct
	assert.Contains(t, codeStr, "EmailEncrypted []byte")
	assert.Contains(t, codeStr, "PhoneEncrypted []byte")
	assert.Contains(t, codeStr, "package test")
	assert.Contains(t, codeStr, "type UserEncx struct")
}

func TestGenerateCodeEmpty(t *testing.T) {
	engine, err := NewTemplateEngine()
	require.NoError(t, err)

	// Create template data with no fields
	data := TemplateData{
		PackageName:            "test",
		StructName:             "Empty",
		SourceFile:             "empty.go",
		GeneratedTime:          time.Now().Format(time.RFC3339),
		SerializerType:         "json",
		SerializerFromMetadata: "&serialization.JSONSerializer{}",
		GeneratorVersion:       "1.0.0",
		EncryptedFields:        []TemplateField{},
		ProcessingSteps:        []string{},
		DecryptionSteps:        []string{},
	}

	code, err := engine.GenerateCode(data)
	require.NoError(t, err)

	codeStr := string(code)

	// Verify basic structure
	assert.Contains(t, codeStr, "package test")
	assert.Contains(t, codeStr, "type EmptyEncx struct")
	assert.Contains(t, codeStr, "func ProcessEmptyEncx")
	assert.Contains(t, codeStr, "func DecryptEmptyEncx")
}

