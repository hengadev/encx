package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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

	// Load generation cache
	if err := g.loadCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load cache: %v\n", err)
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

			// Check for validation errors
			hasErrors := false
			for _, field := range structInfo.Fields {
				if !field.IsValid {
					hasErrors = true
					for _, errMsg := range field.ValidationErrors {
						fmt.Fprintf(os.Stderr, "Validation error in %s.%s: %s\n", structInfo.StructName, field.Name, errMsg)
					}
				}
			}

			if hasErrors {
				fmt.Fprintf(os.Stderr, "Skipping code generation for struct %s due to validation errors\n", structInfo.StructName)
				continue
			}

			// Build template data
			codegenConfig, err := g.config.Generation.ToCodegenConfig()
			if err != nil {
				return fmt.Errorf("failed to convert config for struct %s: %w", structInfo.StructName, err)
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

			// Check if regeneration is needed (incremental generation)
			needsRegen, err := g.needsRegeneration(structInfo, outputFileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to check regeneration need for %s: %v\n", outputFileName, err)
				needsRegen = true // Default to regeneration on error
			}

			if !needsRegen && !dryRun {
				if g.verbose {
					fmt.Printf("Skipping %s (up to date)\n", outputFileName)
				}
				continue
			}

			if dryRun {
				if needsRegen {
					fmt.Printf("Would generate: %s\n", outputFileName)
				} else {
					fmt.Printf("Would skip: %s (up to date)\n", outputFileName)
				}
				if g.verbose {
					fmt.Printf("Generated code:\n%s\n", string(code))
				}
			} else {
				// Write file
				if err := os.WriteFile(outputFileName, code, 0644); err != nil {
					return fmt.Errorf("failed to write generated file %s: %w", outputFileName, err)
				}

				// Update cache
				if err := g.updateCache(structInfo, outputFileName); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to update cache for %s: %v\n", outputFileName, err)
				}

				if g.verbose {
					fmt.Printf("Generated: %s\n", outputFileName)
				}
			}
		}
	}

	// Save cache after successful generation
	if !dryRun {
		if err := g.saveCache(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
		}
	}

	fmt.Println("Code generation complete!")
	return nil
}

// loadCache loads the generation cache from disk
func (g *Generator) loadCache() error {
	cacheFile := ".encx-gen-cache.json"
	data, err := os.ReadFile(cacheFile)
	if os.IsNotExist(err) {
		// No cache file exists, start fresh
		g.cache = &GenerationCache{
			SourceHashes:   make(map[string]string),
			GeneratedFiles: make(map[string]GeneratedFileInfo),
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	return json.Unmarshal(data, g.cache)
}

// saveCache saves the generation cache to disk
func (g *Generator) saveCache() error {
	cacheFile := ".encx-gen-cache.json"
	data, err := json.MarshalIndent(g.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// calculateFileHash calculates SHA256 hash of a file
func (g *Generator) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// needsRegeneration checks if a struct needs to be regenerated
func (g *Generator) needsRegeneration(structInfo codegen.StructInfo, outputPath string) (bool, error) {
	// Check if source file hash changed
	sourceHash, err := g.calculateFileHash(structInfo.SourceFile)
	if err != nil {
		return true, err // If we can't read the file, regenerate
	}

	cachedHash, exists := g.cache.SourceHashes[structInfo.SourceFile]
	if !exists || cachedHash != sourceHash {
		return true, nil // Source file changed or not in cache
	}

	// Check if output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return true, nil // Output file doesn't exist
	}

	// Check if generated file info exists in cache
	genInfo, exists := g.cache.GeneratedFiles[outputPath]
	if !exists {
		return true, nil // Not in cache
	}

	// Check if the source file is newer than generated file
	sourceInfo, err := os.Stat(structInfo.SourceFile)
	if err != nil {
		return true, err
	}

	if sourceInfo.ModTime().After(genInfo.GeneratedTime) {
		return true, nil // Source file is newer
	}

	return false, nil // No regeneration needed
}

// updateCache updates the cache with new file information
func (g *Generator) updateCache(structInfo codegen.StructInfo, outputPath string) error {
	sourceHash, err := g.calculateFileHash(structInfo.SourceFile)
	if err != nil {
		return err
	}

	g.cache.SourceHashes[structInfo.SourceFile] = sourceHash
	g.cache.GeneratedFiles[outputPath] = GeneratedFileInfo{
		SourceFile:    structInfo.SourceFile,
		SourceHash:    sourceHash,
		GeneratedTime: time.Now(),
	}
	g.cache.LastGenerated = time.Now()

	return nil
}
