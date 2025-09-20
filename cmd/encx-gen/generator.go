package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hengadev/encx/internal/codegen"
)

// Generator handles the code generation process
type Generator struct {
	config    *Config
	outputDir string
	verbose   bool
	cache     *GenerationCache
}

// GenerationCache tracks what has been generated and when
type GenerationCache struct {
	SourceHashes    map[string]string
	ConfigHash      string
	GeneratedFiles  map[string]GeneratedFileInfo
	LastGenerated   time.Time
}

// GeneratedFileInfo tracks information about a generated file
type GeneratedFileInfo struct {
	SourceFile    string
	SourceHash    string
	GeneratedTime time.Time
}

// NewGenerator creates a new Generator instance
func NewGenerator(configPath, outputDir string, verbose bool) *Generator {
	config, err := LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation failed: %v\n", err)
		config = DefaultConfig()
	}

	return &Generator{
		config:    config,
		outputDir: outputDir,
		verbose:   verbose,
		cache:     &GenerationCache{
			SourceHashes:   make(map[string]string),
			GeneratedFiles: make(map[string]GeneratedFileInfo),
		},
	}
}

// Generate performs code generation for the specified packages
func (g *Generator) Generate(packages []string, dryRun bool) error {
	if g.verbose {
		fmt.Printf("Starting code generation for packages: %v\n", packages)
		if dryRun {
			fmt.Println("Running in dry-run mode")
		}
	}

	// Create template engine
	templateEngine, err := codegen.NewTemplateEngine()
	if err != nil {
		return fmt.Errorf("failed to create template engine: %w", err)
	}

	// Process each package
	for _, packagePath := range packages {
		if g.verbose {
			fmt.Printf("Processing package: %s\n", packagePath)
		}

		// Skip packages marked to skip
		if pkgConfig, exists := g.config.Packages[packagePath]; exists && pkgConfig.Skip {
			if g.verbose {
				fmt.Printf("Skipping package %s (marked as skip)\n", packagePath)
			}
			continue
		}

		// Discover structs with encx tags
		discoveryConfig := &codegen.DiscoveryConfig{}
		structs, err := codegen.DiscoverStructs(packagePath, discoveryConfig)
		if err != nil {
			return fmt.Errorf("failed to discover structs in package %s: %w", packagePath, err)
		}

		if g.verbose {
			fmt.Printf("Found %d structs with encx tags in %s\n", len(structs), packagePath)
		}

		// Generate code for each struct
		for _, structInfo := range structs {
			if g.verbose {
				fmt.Printf("Generating code for struct: %s\n", structInfo.StructName)
			}

			// Build template data
			codegenConfig := codegen.GenerationConfig{
				OutputSuffix:      g.config.Generation.OutputSuffix,
				FunctionPrefix:    g.config.Generation.FunctionPrefix,
				PackageName:       g.config.Generation.PackageName,
				DefaultSerializer: g.config.Generation.DefaultSerializer,
			}
			templateData := codegen.BuildTemplateData(structInfo, codegenConfig)

			// Generate code
			code, err := templateEngine.GenerateCode(templateData)
			if err != nil {
				return fmt.Errorf("failed to generate code for struct %s: %w", structInfo.StructName, err)
			}

			// Determine output file path
			outputFileName := structInfo.SourceFile
			if ext := ".go"; !strings.HasSuffix(outputFileName, ext) {
				outputFileName += ext
			}
			outputFileName = strings.TrimSuffix(outputFileName, ".go") + g.config.Generation.OutputSuffix + ".go"

			if dryRun {
				fmt.Printf("Would generate: %s\n", outputFileName)
				if g.verbose {
					fmt.Printf("Generated code:\n%s\n", string(code))
				}
			} else {
				// Write file
				if err := os.WriteFile(outputFileName, code, 0644); err != nil {
					return fmt.Errorf("failed to write generated file %s: %w", outputFileName, err)
				}
				if g.verbose {
					fmt.Printf("Generated: %s\n", outputFileName)
				}
			}
		}
	}

	fmt.Println("Code generation complete!")
	return nil
}

// generateStructCode generates code for a specific struct
func (g *Generator) generateStructCode(structInfo codegen.StructInfo) ([]byte, error) {
	// TODO: 1. Build template data
	// TODO: 2. Generate processing steps for each field
	// TODO: 3. Execute templates
	// TODO: 4. Format generated code

	return nil, fmt.Errorf("not implemented yet")
}