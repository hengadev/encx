package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration for the code generator
type Config struct {
	Version    string                       `yaml:"version"`
	Generation GenerationConfig             `yaml:"generation"`
	Packages   map[string]PackageConfig     `yaml:"packages"`
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
	// If file doesn't exist, return default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
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
			FunctionPrefix:    "",
			PackageName:       "auto",
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
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	if c.Generation.DefaultSerializer == "" {
		return fmt.Errorf("default serializer is required")
	}

	// Check if default serializer exists
	if _, exists := c.Serializers[c.Generation.DefaultSerializer]; !exists {
		return fmt.Errorf("default serializer '%s' not found in serializers config", c.Generation.DefaultSerializer)
	}

	// Validate each package config
	for pkg, config := range c.Packages {
		if config.Serializer != "" {
			if _, exists := c.Serializers[config.Serializer]; !exists {
				return fmt.Errorf("serializer '%s' for package '%s' not found in serializers config", config.Serializer, pkg)
			}
		}
	}

	return nil
}