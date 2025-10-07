package benchmarks

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hengadev/encx/internal/codegen"
)

// BenchmarkCodeGeneration benchmarks the code generation process
func BenchmarkCodeGeneration(b *testing.B) {
	// Create a temporary directory for test files
	tempDir := b.TempDir()

	// Create test files with different complexity levels
	testFiles := map[string]string{
		"simple.go": `package test

type Simple struct {
	Name string ` + "`encx:\"encrypt\"`" + `
	NameEncrypted []byte
}`,
		"medium.go": `package test

type Medium struct {
	Email string ` + "`encx:\"encrypt,hash_basic\"`" + `
	EmailEncrypted []byte
	EmailHash string
	Phone string ` + "`encx:\"encrypt\"`" + `
	PhoneEncrypted []byte
	SSN string ` + "`encx:\"hash_secure\"`" + `
	SSNHashSecure string
}`,
		"complex.go": `package test

type Complex struct {
	ID             int
	Email          string ` + "`encx:\"encrypt,hash_basic\"`" + `
	EmailEncrypted []byte
	EmailHash      string
	Phone          string ` + "`encx:\"encrypt\"`" + `
	PhoneEncrypted []byte
	SSN            string ` + "`encx:\"hash_secure\"`" + `
	SSNHashSecure  string
	CreditCard     string ` + "`encx:\"encrypt,hash_secure\"`" + `
	CreditCardEncrypted []byte
	CreditCardHashSecure string
	Username       string ` + "`encx:\"hash_basic\"`" + `
	UsernameHash   string
	Password       string ` + "`encx:\"hash_secure\"`" + `
	PasswordHashSecure string
}`,
	}

	// Write test files
	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	b.Run("StructDiscovery", func(b *testing.B) {
		config := &codegen.DiscoveryConfig{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := codegen.DiscoverStructs(tempDir, config)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("TemplateEngineCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := codegen.NewTemplateEngine()
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("CodeGeneration", func(b *testing.B) {
		// Discover structs once
		config := &codegen.DiscoveryConfig{}
		structs, err := codegen.DiscoverStructs(tempDir, config)
		if err != nil {
			b.Fatalf("Failed to discover structs: %v", err)
		}

		templateEngine, err := codegen.NewTemplateEngine()
		if err != nil {
			b.Fatalf("Failed to create template engine: %v", err)
		}

		codegenConfig := codegen.GenerationConfig{
			OutputSuffix: "_encx",
			PackageName:  "test",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, structInfo := range structs {
				// Skip structs with validation errors
				hasErrors := false
				for _, field := range structInfo.Fields {
					if !field.IsValid {
						hasErrors = true
						break
					}
				}
				if hasErrors {
					continue
				}

				templateData := codegen.BuildTemplateData(structInfo, codegenConfig)
				_, err := templateEngine.GenerateCode(templateData)
				if err != nil {
					b.Error(err)
				}
			}
		}
	})

	b.Run("EndToEndGeneration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			config := &codegen.DiscoveryConfig{}
			structs, err := codegen.DiscoverStructs(tempDir, config)
			if err != nil {
				b.Error(err)
				continue
			}

			templateEngine, err := codegen.NewTemplateEngine()
			if err != nil {
				b.Error(err)
				continue
			}

			codegenConfig := codegen.GenerationConfig{
				OutputSuffix: "_encx",
				PackageName:  "test",
			}

			for _, structInfo := range structs {
				// Skip structs with validation errors
				hasErrors := false
				for _, field := range structInfo.Fields {
					if !field.IsValid {
						hasErrors = true
						break
					}
				}
				if hasErrors {
					continue
				}

				templateData := codegen.BuildTemplateData(structInfo, codegenConfig)
				code, err := templateEngine.GenerateCode(templateData)
				if err != nil {
					b.Error(err)
					continue
				}

				// Write to temporary file to simulate real usage
				outputFile := filepath.Join(tempDir, "bench_"+structInfo.StructName+"_encx.go")
				err = os.WriteFile(outputFile, code, 0644)
				if err != nil {
					b.Error(err)
				}
			}
		}
	})
}

// BenchmarkTemplateData benchmarks template data building
func BenchmarkTemplateData(b *testing.B) {
	tempDir := b.TempDir()

	// Create a complex test file
	complexFile := `package test

type BenchmarkStruct struct {
	Field1  string ` + "`encx:\"encrypt\"`" + `
	Field1Encrypted []byte
	Field2  string ` + "`encx:\"hash_basic\"`" + `
	Field2Hash string
	Field3  string ` + "`encx:\"hash_secure\"`" + `
	Field3HashSecure string
	Field4  string ` + "`encx:\"encrypt,hash_basic\"`" + `
	Field4Encrypted []byte
	Field4Hash string
	Field5  string ` + "`encx:\"encrypt,hash_secure\"`" + `
	Field5Encrypted []byte
	Field5HashSecure string
}`

	err := os.WriteFile(filepath.Join(tempDir, "benchmark.go"), []byte(complexFile), 0644)
	if err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	// Discover the struct
	config := &codegen.DiscoveryConfig{}
	structs, err := codegen.DiscoverStructs(tempDir, config)
	if err != nil {
		b.Fatalf("Failed to discover structs: %v", err)
	}

	if len(structs) == 0 {
		b.Fatalf("No structs discovered")
	}

	structInfo := structs[0]
	codegenConfig := codegen.GenerationConfig{
		OutputSuffix: "_encx",
		PackageName:  "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codegen.BuildTemplateData(structInfo, codegenConfig)
	}
}

// BenchmarkValidation benchmarks field validation
func BenchmarkValidation(b *testing.B) {
	tempDir := b.TempDir()

	// Create files with validation errors
	invalidFile := `package test

type Invalid struct {
	BadField string ` + "`encx:\"hash_basic,hash_secure\"`" + ` // Invalid combination
	MissingField string ` + "`encx:\"encrypt\"`" + ` // Missing companion
	WrongType string ` + "`encx:\"encrypt\"`" + `
	WrongTypeEncrypted string // Should be []byte
}`

	err := os.WriteFile(filepath.Join(tempDir, "invalid.go"), []byte(invalidFile), 0644)
	if err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	config := &codegen.DiscoveryConfig{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		structs, err := codegen.DiscoverStructs(tempDir, config)
		if err != nil {
			b.Error(err)
			continue
		}

		// Validation is performed during discovery
		for _, structInfo := range structs {
			for _, field := range structInfo.Fields {
				_ = field.IsValid
				_ = field.ValidationErrors
			}
		}
	}
}

// BenchmarkLargeProject simulates code generation for a large project
func BenchmarkLargeProject(b *testing.B) {
	tempDir := b.TempDir()

	// Create multiple files simulating a large project
	for i := 0; i < 20; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("model_%d.go", i))
		content := fmt.Sprintf(`package test

type Model%d struct {
	ID       int
	Name     string `+"`encx:\"encrypt\"`"+`
	NameEncrypted []byte
	Email    string `+"`encx:\"encrypt,hash_basic\"`"+`
	EmailEncrypted []byte
	EmailHash string
	Phone    string `+"`encx:\"encrypt\"`"+`
	PhoneEncrypted []byte
	Username string `+"`encx:\"hash_basic\"`"+`
	UsernameHash string
}

type Related%d struct {
	Token string `+"`encx:\"hash_secure\"`"+`
	TokenHashSecure string
	Data  string `+"`encx:\"encrypt\"`"+`
	DataEncrypted []byte
}`, i, i)

		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := &codegen.DiscoveryConfig{}
		structs, err := codegen.DiscoverStructs(tempDir, config)
		if err != nil {
			b.Error(err)
			continue
		}

		templateEngine, err := codegen.NewTemplateEngine()
		if err != nil {
			b.Error(err)
			continue
		}

		codegenConfig := codegen.GenerationConfig{
			OutputSuffix: "_encx",
			PackageName:  "test",
		}

		for _, structInfo := range structs {
			templateData := codegen.BuildTemplateData(structInfo, codegenConfig)
			_, err := templateEngine.GenerateCode(templateData)
			if err != nil {
				b.Error(err)
			}
		}
	}
}