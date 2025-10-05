package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/hengadev/encx/internal/codegen"
)

// Config represents the configuration for the code generator
type Config struct {
	Version     string                      `yaml:"version"`
	Generation  GenerationConfig            `yaml:"generation"`
	Packages    map[string]PackageConfig    `yaml:"packages"`
	Serializers map[string]SerializerConfig `yaml:"serializers"`
}

// GenerationConfig holds general generation settings
type GenerationConfig struct {
	OutputSuffix      string `yaml:"output_suffix"`
	FunctionPrefix    string `yaml:"function_prefix"`
	PackageName       string `yaml:"package_name"`
	DefaultSerializer string `yaml:"default_serializer"`
}

// PackageConfig holds per-package overrides
type PackageConfig struct {
	Serializer string `yaml:"serializer"`
	OutputDir  string `yaml:"output_dir"`
	Skip       bool   `yaml:"skip"`
}

// SerializerConfig defines available serializers
type SerializerConfig struct {
	Type    string `yaml:"type"`
	Import  string `yaml:"import"`
	Factory string `yaml:"factory"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with empty config, not defaults
	config := &Config{
		Packages:    make(map[string]PackageConfig),
		Serializers: make(map[string]SerializerConfig),
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1",
		Generation: GenerationConfig{
			OutputSuffix:      "_encx",
			FunctionPrefix:    "Process",
			PackageName:       "encx",
			DefaultSerializer: "json",
		},
		Packages: make(map[string]PackageConfig),
		Serializers: map[string]SerializerConfig{
			"json": {
				Type:   "json",
				Import: "encoding/json",
			},
			"gob": {
				Type:   "gob",
				Import: "encoding/gob",
			},
			"basic": {
				Type:   "basic",
				Import: "github.com/hengadev/encx/internal/serialization",
			},
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Version is optional - default to "1" if not set
	if c.Version == "" {
		c.Version = "1"
	}

	// Validate generation config
	if c.Generation.OutputSuffix == "" {
		return fmt.Errorf("output_suffix cannot be empty")
	}

	if c.Generation.FunctionPrefix == "" {
		return fmt.Errorf("function_prefix cannot be empty")
	}

	if c.Generation.PackageName == "" {
		return fmt.Errorf("package_name cannot be empty")
	}

	if c.Generation.DefaultSerializer == "" {
		return fmt.Errorf("default serializer is required")
	}

	// Validate identifiers
	if !isValidGoIdentifier(c.Generation.FunctionPrefix) {
		return fmt.Errorf("function_prefix must be a valid Go identifier")
	}

	if c.Generation.PackageName != "auto" && !isValidGoIdentifier(c.Generation.PackageName) {
		return fmt.Errorf("package_name must be a valid Go identifier")
	}

	// Validate output suffix format
	if !isValidOutputSuffix(c.Generation.OutputSuffix) {
		return fmt.Errorf("output_suffix must start with underscore or letter")
	}

	// Check if default serializer exists or is a known type
	validSerializers := map[string]bool{"json": true, "protobuf": true}
	if !validSerializers[c.Generation.DefaultSerializer] {
		return fmt.Errorf("default_serializer must be one of: json, protobuf")
	}

	// Validate each package config
	for pkg, pkgConfig := range c.Packages {
		if pkgConfig.Serializer != "" {
			// Check against known serializers or configured serializers
			if len(c.Serializers) > 0 {
				// If serializers are configured, check against those
				if _, exists := c.Serializers[pkgConfig.Serializer]; !exists {
					return fmt.Errorf("serializer '%s' for package '%s' not found in serializers config", pkgConfig.Serializer, pkg)
				}
			} else {
				// If no serializers configured, check against known valid ones
				if !validSerializers[pkgConfig.Serializer] {
					return fmt.Errorf("invalid serializer for package %s", pkg)
				}
			}
		}
	}

	return nil
}

// isValidGoIdentifier checks if a string is a valid Go identifier
func isValidGoIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// Must start with letter or underscore
	first := rune(s[0])
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Rest can be letters, digits, or underscores
	for _, r := range s[1:] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}

// isValidOutputSuffix checks if output suffix is valid
func isValidOutputSuffix(s string) bool {
	if s == "" {
		return false
	}

	// Must start with underscore or letter
	first := rune(s[0])
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_'
}

// ToCodegenConfig converts the YAML config to the codegen GenerationConfig
func (gc GenerationConfig) ToCodegenConfig() (codegen.GenerationConfig, error) {
	return codegen.GenerationConfig{
		OutputSuffix:   gc.OutputSuffix,
		FunctionPrefix: gc.FunctionPrefix,
		PackageName:    gc.PackageName,
	}, nil
}

