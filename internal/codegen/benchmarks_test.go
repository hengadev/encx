package codegen

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkDiscoverStructs benchmarks the struct discovery performance
func BenchmarkDiscoverStructs(b *testing.B) {
	// Create a temporary directory with test files
	tempDir := b.TempDir()

	// Create multiple test files with varying complexity
	for i := 0; i < 10; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		content := generateTestStructFile(i)
		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	config := &DiscoveryConfig{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DiscoverStructs(tempDir, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTemplateGeneration benchmarks the template generation performance
func BenchmarkTemplateGeneration(b *testing.B) {
	engine, err := NewTemplateEngine()
	if err != nil {
		b.Fatal(err)
	}

	// Create template data for a complex struct
	data := TemplateData{
		PackageName:            "benchmark",
		StructName:             "ComplexUser",
		SourceFile:             "complex_user.go",
		GeneratedTime:          time.Now().Format(time.RFC3339),
		GeneratorVersion:       "1.0.0",
		EncryptedFields: []TemplateField{
			{Name: "EmailEncrypted", Type: "[]byte", DBColumn: "email_encrypted", JSONField: "email_encrypted"},
			{Name: "PhoneEncrypted", Type: "[]byte", DBColumn: "phone_encrypted", JSONField: "phone_encrypted"},
			{Name: "SSNHashSecure", Type: "string", DBColumn: "ssn_hash_secure", JSONField: "ssn_hash_secure"},
			{Name: "AddressEncrypted", Type: "[]byte", DBColumn: "address_encrypted", JSONField: "address_encrypted"},
			{Name: "CreditCardEncrypted", Type: "[]byte", DBColumn: "credit_card_encrypted", JSONField: "credit_card_encrypted"},
		},
		ProcessingSteps: generateProcessingSteps(),
		DecryptionSteps: generateDecryptionSteps(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.GenerateCode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTagValidation benchmarks the tag validation performance
func BenchmarkTagValidation(b *testing.B) {
	validator := NewTagValidator()
	testCases := []struct {
		fieldName string
		tags      []string
	}{
		{"Email", []string{"encrypt", "hash_basic"}},
		{"Phone", []string{"encrypt"}},
		{"SSN", []string{"hash_secure"}},
		{"Address", []string{"encrypt", "hash_secure"}},
		{"CreditCard", []string{"encrypt"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			validator.ValidateFieldTags(tc.fieldName, tc.tags)
		}
	}
}

// BenchmarkIncrementalGeneration benchmarks the incremental generation check performance
func BenchmarkIncrementalGeneration(b *testing.B) {
	tempDir := b.TempDir()

	// Create a test source file
	sourceFile := filepath.Join(tempDir, "user.go")
	err := os.WriteFile(sourceFile, []byte(generateTestStructFile(1)), 0644)
	if err != nil {
		b.Fatal(err)
	}

	// Create a generator (simplified version for benchmarking)
	type SimpleGenerator struct {
		cache map[string]string
	}

	generator := &SimpleGenerator{
		cache: make(map[string]string),
	}

	// Function to simulate file hash calculation
	calculateHash := func(filename string) (string, error) {
		data, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		hash := sha256.Sum256(data)
		return fmt.Sprintf("%x", hash), nil
	}

	// Function to simulate regeneration check
	needsRegeneration := func(sourceFile string) (bool, error) {
		currentHash, err := calculateHash(sourceFile)
		if err != nil {
			return true, err
		}

		cachedHash, exists := generator.cache[sourceFile]
		if !exists || cachedHash != currentHash {
			generator.cache[sourceFile] = currentHash
			return true, nil
		}
		return false, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := needsRegeneration(sourceFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStructDiscoveryScaling benchmarks struct discovery with different file counts
func BenchmarkStructDiscoveryScaling(b *testing.B) {
	fileCounts := []int{1, 5, 10, 25, 50, 100}

	for _, fileCount := range fileCounts {
		b.Run(fmt.Sprintf("Files_%d", fileCount), func(b *testing.B) {
			tempDir := b.TempDir()

			// Create multiple test files
			for i := 0; i < fileCount; i++ {
				filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
				content := generateTestStructFile(i)
				err := os.WriteFile(filename, []byte(content), 0644)
				if err != nil {
					b.Fatal(err)
				}
			}

			config := &DiscoveryConfig{}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := DiscoverStructs(tempDir, config)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage measures memory allocation during code generation
func BenchmarkMemoryUsage(b *testing.B) {
	engine, err := NewTemplateEngine()
	if err != nil {
		b.Fatal(err)
	}

	data := TemplateData{
		PackageName:            "benchmark",
		StructName:             "LargeStruct",
		SourceFile:             "large_struct.go",
		GeneratedTime:          time.Now().Format(time.RFC3339),
		GeneratorVersion:       "1.0.0",
		EncryptedFields:        generateLargeFieldList(50), // 50 fields
		ProcessingSteps:        generateLargeProcessingSteps(50),
		DecryptionSteps:        generateLargeDecryptionSteps(50),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.GenerateCode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for generating test data

func generateTestStructFile(index int) string {
	return fmt.Sprintf(`package test

// TestStruct%d is a test struct for benchmarking
type TestStruct%d struct {
	ID    int    ` + "`json:\"id\"`" + `
	Email string ` + "`json:\"email\" encx:\"encrypt,hash_basic\"`" + `
	Phone string ` + "`json:\"phone\" encx:\"encrypt\"`" + `
	SSN   string ` + "`json:\"ssn\" encx:\"hash_secure\"`" + `

	// Companion fields
	EmailEncrypted []byte ` + "`json:\"email_encrypted\"`" + `
	EmailHash      string ` + "`json:\"email_hash\"`" + `
	PhoneEncrypted []byte ` + "`json:\"phone_encrypted\"`" + `
	SSNHashSecure  string ` + "`json:\"ssn_hash_secure\"`" + `
}
`, index, index)
}

func generateProcessingSteps() []string {
	return []string{
		"// Process Email (encrypt + hash_basic)",
		"EmailBytes, err := serializer.Serialize(source.Email)",
		"result.EmailEncrypted, err = crypto.EncryptData(ctx, EmailBytes, dek)",
		"result.EmailHash, err = crypto.HashBasic(ctx, EmailBytes)",
		"// Process Phone (encrypt)",
		"PhoneBytes, err := serializer.Serialize(source.Phone)",
		"result.PhoneEncrypted, err = crypto.EncryptData(ctx, PhoneBytes, dek)",
		"// Process SSN (hash_secure)",
		"SSNBytes, err := serializer.Serialize(source.SSN)",
		"result.SSNHashSecure, err = crypto.HashSecure(ctx, SSNBytes)",
	}
}

func generateDecryptionSteps() []string {
	return []string{
		"// Decrypt Email",
		"EmailBytes, err := crypto.DecryptData(ctx, source.EmailEncrypted, dek)",
		"err = serializer.Deserialize(EmailBytes, &result.Email)",
		"// Decrypt Phone",
		"PhoneBytes, err := crypto.DecryptData(ctx, source.PhoneEncrypted, dek)",
		"err = serializer.Deserialize(PhoneBytes, &result.Phone)",
	}
}

func generateLargeFieldList(count int) []TemplateField {
	fields := make([]TemplateField, count)
	for i := 0; i < count; i++ {
		fields[i] = TemplateField{
			Name:      fmt.Sprintf("Field%dEncrypted", i),
			Type:      "[]byte",
			DBColumn:  fmt.Sprintf("field_%d_encrypted", i),
			JSONField: fmt.Sprintf("field_%d_encrypted", i),
		}
	}
	return fields
}

func generateLargeProcessingSteps(count int) []string {
	steps := make([]string, count*3) // 3 steps per field
	for i := 0; i < count; i++ {
		base := i * 3
		steps[base] = fmt.Sprintf("// Process Field%d", i)
		steps[base+1] = fmt.Sprintf("Field%dBytes, err := serializer.Serialize(source.Field%d)", i, i)
		steps[base+2] = fmt.Sprintf("result.Field%dEncrypted, err = crypto.EncryptData(ctx, Field%dBytes, dek)", i, i)
	}
	return steps
}

func generateLargeDecryptionSteps(count int) []string {
	steps := make([]string, count*3) // 3 steps per field
	for i := 0; i < count; i++ {
		base := i * 3
		steps[base] = fmt.Sprintf("// Decrypt Field%d", i)
		steps[base+1] = fmt.Sprintf("Field%dBytes, err := crypto.DecryptData(ctx, source.Field%dEncrypted, dek)", i, i)
		steps[base+2] = fmt.Sprintf("err = serializer.Deserialize(Field%dBytes, &result.Field%d)", i, i)
	}
	return steps
}