package integration

// This file provides examples and utilities for integrating encx-gen with go generate

// Example go:generate directive for use in Go source files:
//
//   //go:generate encx-gen generate -config=encx.yaml .
//
// This will generate encx code for all structs in the current package
// using the configuration from encx.yaml

// For projects with multiple packages:
//
//   //go:generate encx-gen generate -config=encx.yaml ./internal/models ./pkg/api
//
// This will generate code for structs in both packages

// For CI/CD environments where you want to verify nothing changed:
//
//   //go:generate encx-gen generate -config=encx.yaml -dry-run .
//   //go:generate encx-gen validate -config=encx.yaml .
//
// The first command shows what would be generated
// The second validates configuration and struct tags

// Example package-level go:generate directive:
//
//   package models
//
//   //go:generate encx-gen generate -v .
//
//   // User contains encrypted personal information
//   type User struct {
//       ID    int    `json:"id"`
//       Email string `json:"email" encx:"encrypt,hash_basic"`
//       Phone string `json:"phone" encx:"encrypt"`
//
//       // Companion fields
//       EmailEncrypted []byte `json:"email_encrypted"`
//       EmailHash      string `json:"email_hash"`
//       PhoneEncrypted []byte `json:"phone_encrypted"`
//   }

// Integration with go mod and go build:
//
// 1. Add to your Makefile:
//    generate:
//        go generate ./...
//
// 2. Add to your CI pipeline:
//    - name: Generate code
//      run: |
//        go generate ./...
//        git diff --exit-code  # Fail if generated code changed
//
// 3. Add to your build process:
//    go generate ./... && go build ./...

const ExampleGoGenerateComment = `
// Package models contains data structures with encrypted fields
//
//go:generate encx-gen generate -config=../encx.yaml -v .
package models
`

const ExampleConfigurableGenerate = `
// Advanced go:generate setup with environment variables
//
//go:generate sh -c "encx-gen generate -config=${ENCX_CONFIG:-encx.yaml} -output=${ENCX_OUTPUT:-.} ${ENCX_PACKAGES:-.}"
package models
`

const ExampleValidationGenerate = `
// Example with validation before generation
//
//go:generate encx-gen validate -config=encx.yaml .
//go:generate encx-gen generate -config=encx.yaml .
package models
`

// GoGenerateHelper provides utilities for working with go generate
type GoGenerateHelper struct {
	ConfigPath   string
	OutputDir    string
	Verbose      bool
	DryRun       bool
	ValidateOnly bool
}

// NewGoGenerateHelper creates a new helper with default settings
func NewGoGenerateHelper() *GoGenerateHelper {
	return &GoGenerateHelper{
		ConfigPath: "encx.yaml",
		OutputDir:  ".",
		Verbose:    false,
		DryRun:     false,
	}
}

// GenerateCommand returns the encx-gen command string for go:generate
func (h *GoGenerateHelper) GenerateCommand(packages ...string) string {
	cmd := "encx-gen generate"

	if h.ConfigPath != "" && h.ConfigPath != "encx.yaml" {
		cmd += " -config=" + h.ConfigPath
	}

	if h.OutputDir != "" && h.OutputDir != "." {
		cmd += " -output=" + h.OutputDir
	}

	if h.Verbose {
		cmd += " -v"
	}

	if h.DryRun {
		cmd += " -dry-run"
	}

	if len(packages) == 0 {
		cmd += " ."
	} else {
		for _, pkg := range packages {
			cmd += " " + pkg
		}
	}

	return cmd
}

// ValidateCommand returns the encx-gen validate command string
func (h *GoGenerateHelper) ValidateCommand(packages ...string) string {
	cmd := "encx-gen validate"

	if h.ConfigPath != "" && h.ConfigPath != "encx.yaml" {
		cmd += " -config=" + h.ConfigPath
	}

	if h.Verbose {
		cmd += " -v"
	}

	if len(packages) == 0 {
		cmd += " ."
	} else {
		for _, pkg := range packages {
			cmd += " " + pkg
		}
	}

	return cmd
}
