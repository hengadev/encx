package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hengadev/encx/internal/codegen"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "generate":
		generateCommand(os.Args[2:])
	case "validate":
		validateCommand(os.Args[2:])
	case "init":
		initCommand(os.Args[2:])
	case "version":
		versionCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  generate  Generate encx code for structs\n")
	fmt.Fprintf(os.Stderr, "  validate  Validate configuration and struct tags\n")
	fmt.Fprintf(os.Stderr, "  init      Initialize configuration file\n")
	fmt.Fprintf(os.Stderr, "  version   Show version information\n")
	fmt.Fprintf(os.Stderr, "\nRun '%s <command> -h' for help on a specific command.\n", os.Args[0])
}

func generateCommand(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	configPath := fs.String("config", "encx.yaml", "Path to configuration file")
	outputDir := fs.String("output", "", "Override output directory")
	verbose := fs.Bool("v", false, "Verbose output")
	dryRun := fs.Bool("dry-run", false, "Show what would be generated without writing files")

	fs.Parse(args)

	packages := fs.Args()
	if len(packages) == 0 {
		packages = []string{"."} // Current directory
	}

	generator := NewGenerator(*configPath, *outputDir, *verbose)
	err := generator.Generate(packages, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Generation failed: %v\n", err)
		os.Exit(1)
	}
}

func validateCommand(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	configPath := fs.String("config", "encx.yaml", "Path to configuration file")
	verbose := fs.Bool("v", false, "Verbose output")

	fs.Parse(args)

	packages := fs.Args()
	if len(packages) == 0 {
		packages = []string{"."} // Current directory
	}

	fmt.Printf("Validating configuration at %s...\n", *configPath)

	// Load and validate configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("✓ Configuration file is valid")
	}

	// Validate struct tags in packages
	hasErrors := false
	for _, pkg := range packages {
		if *verbose {
			fmt.Printf("Validating package: %s\n", pkg)
		}

		discoveryConfig := &codegen.DiscoveryConfig{}
		structs, err := codegen.DiscoverStructs(pkg, discoveryConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to discover structs in %s: %v\n", pkg, err)
			hasErrors = true
			continue
		}

		if len(structs) == 0 {
			if *verbose {
				fmt.Printf("  No structs with encx tags found in %s\n", pkg)
			}
			continue
		}

		fmt.Printf("Found %d structs with encx tags in %s:\n", len(structs), pkg)

		for _, structInfo := range structs {
			fmt.Printf("  %s (%s)\n", structInfo.StructName, structInfo.SourceFile)

			// Check for validation errors
			structHasErrors := false
			for _, field := range structInfo.Fields {
				if !field.IsValid {
					structHasErrors = true
					hasErrors = true
					for _, errMsg := range field.ValidationErrors {
						fmt.Printf("    ✗ %s.%s: %s\n", structInfo.StructName, field.Name, errMsg)
					}
				} else if len(field.EncxTags) > 0 {
					if *verbose {
						fmt.Printf("    ✓ %s.%s: %v\n", structInfo.StructName, field.Name, field.EncxTags)
					}
				}
			}

			if !structHasErrors {
				fmt.Printf("    ✓ All fields valid\n")
			}
		}
	}

	if hasErrors {
		fmt.Fprintf(os.Stderr, "\nValidation failed with errors.\n")
		os.Exit(1)
	}

	fmt.Println("\n✓ All validations passed!")
}

func initCommand(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing configuration file")

	fs.Parse(args)

	configPath := "encx.yaml"
	if !*force {
		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stderr, "Configuration file %s already exists. Use -force to overwrite.\n", configPath)
			os.Exit(1)
		}
	}

	fmt.Printf("Creating configuration file at %s...\n", configPath)

	config := DefaultConfig()
	if err := SaveConfig(config, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration file created!")
}

func versionCommand() {
	fmt.Println("encx-gen version 1.0.0")
	fmt.Println("Code generator for encx encryption library")
	fmt.Println("")
	fmt.Println("Features:")
	fmt.Println("  - AST-based struct discovery")
	fmt.Println("  - Incremental generation with caching")
	fmt.Println("  - Comprehensive tag validation")
	fmt.Println("  - Cross-database JSON metadata support")
	fmt.Println("  - Template-based code generation")
	fmt.Println("")
	fmt.Println("Supported tags: encrypt, hash_basic, hash_secure")
	fmt.Println("Supported databases: PostgreSQL, SQLite, MySQL")
	fmt.Println("Supported serializers: json")
}

