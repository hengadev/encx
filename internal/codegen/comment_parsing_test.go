package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEncxOptions(t *testing.T) {
	tests := []struct {
		name           string
		commentText    string
		expectedOptions map[string]string
	}{
		{
			name:        "Single serializer option",
			commentText: "//encx:options serializer=gob",
			expectedOptions: map[string]string{
				"serializer": "gob",
			},
		},
		{
			name:        "Multiple options",
			commentText: "//encx:options serializer=basic,future_option=value",
			expectedOptions: map[string]string{
				"serializer":     "basic",
				"future_option": "value",
			},
		},
		{
			name:        "With whitespace",
			commentText: "// encx:options  serializer = json , option2 = value2 ",
			expectedOptions: map[string]string{
				"serializer": "json",
				"option2":    "value2",
			},
		},
		{
			name:        "Block comment",
			commentText: "/* encx:options serializer=gob */",
			expectedOptions: map[string]string{
				"serializer": "gob",
			},
		},
		{
			name:            "No encx:options",
			commentText:     "// This is just a regular comment",
			expectedOptions: map[string]string{},
		},
		{
			name:            "Empty options",
			commentText:     "//encx:options",
			expectedOptions: map[string]string{},
		},
		{
			name:        "Invalid format ignored",
			commentText: "//encx:options serializer=gob,invalid_format,key=value",
			expectedOptions: map[string]string{
				"serializer": "gob",
				"key":        "value",
			},
		},
		{
			name:            "Empty key or value ignored",
			commentText:     "//encx:options =value,key=,=,key=valid",
			expectedOptions: map[string]string{
				"key": "valid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a comment group with the test comment
			commentGroup := &ast.CommentGroup{
				List: []*ast.Comment{
					{Text: tt.commentText},
				},
			}

			options := make(map[string]string)
			parseEncxOptions(commentGroup, options)

			assert.Equal(t, tt.expectedOptions, options)
		})
	}
}

func TestParseOptionsPairs(t *testing.T) {
	tests := []struct {
		name           string
		optionsText    string
		expectedOptions map[string]string
	}{
		{
			name:        "Single pair",
			optionsText: "key=value",
			expectedOptions: map[string]string{
				"key": "value",
			},
		},
		{
			name:        "Multiple pairs",
			optionsText: "key1=value1,key2=value2,key3=value3",
			expectedOptions: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name:        "With whitespace",
			optionsText: " key1 = value1 , key2 = value2 ",
			expectedOptions: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:            "Empty string",
			optionsText:     "",
			expectedOptions: map[string]string{},
		},
		{
			name:            "Only commas",
			optionsText:     ",,,",
			expectedOptions: map[string]string{},
		},
		{
			name:        "Mixed valid and invalid",
			optionsText: "valid=value,invalid,another=good",
			expectedOptions: map[string]string{
				"valid":   "value",
				"another": "good",
			},
		},
		{
			name:            "Empty keys or values",
			optionsText:     "=value,key=,=",
			expectedOptions: map[string]string{},
		},
		{
			name:        "Value with equals sign",
			optionsText: "url=https://example.com,key=value",
			expectedOptions: map[string]string{
				"url": "https://example.com",
				"key": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := make(map[string]string)
			parseOptionsPairs(tt.optionsText, options)

			assert.Equal(t, tt.expectedOptions, options)
		})
	}
}

func TestValidateGenerationOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Serializer option no longer supported (json)",
			options: map[string]string{
				"serializer": "json",
			},
			expectError: true,
			errorMsg:    "serializer option is no longer supported",
		},
		{
			name: "Serializer option no longer supported (gob)",
			options: map[string]string{
				"serializer": "gob",
			},
			expectError: true,
			errorMsg:    "serializer option is no longer supported",
		},
		{
			name: "Serializer option no longer supported (basic)",
			options: map[string]string{
				"serializer": "basic",
			},
			expectError: true,
			errorMsg:    "serializer option is no longer supported",
		},
		{
			name: "Serializer option no longer supported (invalid)",
			options: map[string]string{
				"serializer": "invalid",
			},
			expectError: true,
			errorMsg:    "serializer option is no longer supported",
		},
		{
			name: "Unknown option ignored",
			options: map[string]string{
				"unknown_option": "value",
			},
			expectError: false,
		},
		{
			name: "Mixed serializer (unsupported) and unknown options",
			options: map[string]string{
				"serializer":     "json",
				"unknown_option": "value",
			},
			expectError: true,
			errorMsg:    "serializer option is no longer supported",
		},
		{
			name:        "Empty options",
			options:     map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGenerationOptions(tt.options)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAnalyzeStructWithComments(t *testing.T) {
	// Test complete struct analysis with comments
	source := `package test

//encx:options serializer=gob
type User struct {
	Email string ` + "`encx:\"encrypt\"`" + `
	Name  string ` + "`encx:\"hash_basic\"`" + `
}

// Regular comment
//encx:options serializer=basic,future_option=value
type Product struct {
	Name string ` + "`encx:\"encrypt\"`" + `
}

// No options comment
type NoOptions struct {
	Data string ` + "`encx:\"encrypt\"`" + `
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	var structs []StructInfo

	// We need to look at GenDecl -> TypeSpec to get comments properly associated
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						// The comments are on the GenDecl, not the TypeSpec
						originalDoc := typeSpec.Doc
						if originalDoc == nil && node.Doc != nil {
							typeSpec.Doc = node.Doc
						}

						structInfo := analyzeStruct(fset, "test.go", "test", typeSpec, structType)

						// Restore original doc
						typeSpec.Doc = originalDoc

						if structInfo.HasEncxTags {
							structs = append(structs, structInfo)
						}
					}
				}
			}
		}
		return true
	})

	require.Len(t, structs, 3)

	// Test User struct with gob serializer
	userStruct := findStructByName(structs, "User")
	require.NotNil(t, userStruct)
	assert.Equal(t, map[string]string{"serializer": "gob"}, userStruct.GenerationOptions)

	// Test Product struct with basic serializer and future option
	productStruct := findStructByName(structs, "Product")
	require.NotNil(t, productStruct)
	expectedOptions := map[string]string{
		"serializer":     "basic",
		"future_option": "value",
	}
	assert.Equal(t, expectedOptions, productStruct.GenerationOptions)

	// Test NoOptions struct
	noOptionsStruct := findStructByName(structs, "NoOptions")
	require.NotNil(t, noOptionsStruct)
	assert.Empty(t, noOptionsStruct.GenerationOptions)
}

func TestMultipleCommentLines(t *testing.T) {
	source := `package test

// This is a documentation comment
// explaining what the struct does
//encx:options serializer=gob
// More documentation
type User struct {
	Email string ` + "`encx:\"encrypt\"`" + `
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)

	var structInfo StructInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == "User" {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							// Comments are typically on the GenDecl, not the TypeSpec
							originalDoc := typeSpec.Doc
							if originalDoc == nil && node.Doc != nil {
								typeSpec.Doc = node.Doc
							}

							structInfo = analyzeStruct(fset, "test.go", "test", typeSpec, structType)

							// Restore original doc
							typeSpec.Doc = originalDoc
						}
					}
				}
			}
		}
		return true
	})

	assert.Equal(t, map[string]string{"serializer": "gob"}, structInfo.GenerationOptions)
}

// Helper function to find struct by name
func findStructByName(structs []StructInfo, name string) *StructInfo {
	for _, s := range structs {
		if s.StructName == name {
			return &s
		}
	}
	return nil
}