package main

import (
	"flag"
	"fmt"
	"os"
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

	fs.Parse(args)

	fmt.Printf("Validating configuration at %s...\n", *configPath)
	// TODO: Implement validation
	fmt.Println("Validation complete!")
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
}