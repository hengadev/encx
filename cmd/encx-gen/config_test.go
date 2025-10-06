package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigValidFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "encx.yaml")

	configContent := `
generation:
  output_suffix: "_encrypted"
  function_prefix: "Transform"
  package_name: "mypackage"

packages:
  "./internal":
    skip: false
  "./test":
    skip: true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configFile)
	require.NoError(t, err)

	assert.Equal(t, "_encrypted", config.Generation.OutputSuffix)
	assert.Equal(t, "Transform", config.Generation.FunctionPrefix)
	assert.Equal(t, "mypackage", config.Generation.PackageName)

	assert.Len(t, config.Packages, 2)
	assert.False(t, config.Packages["./internal"].Skip)
	assert.True(t, config.Packages["./test"].Skip)
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.yaml")

	err := os.WriteFile(configFile, []byte(`
invalid: yaml: content:
  - missing
    proper: indentation
`), 0644)
	require.NoError(t, err)

	_, err = LoadConfig(configFile)
	assert.Error(t, err)
}

func TestLoadConfigEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "empty.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configFile)
	require.NoError(t, err)

	assert.NotNil(t, config)
	assert.Empty(t, config.Generation.OutputSuffix)
	assert.Empty(t, config.Generation.FunctionPrefix)
	assert.Empty(t, config.Generation.PackageName)
	assert.Empty(t, config.Packages)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "_encx", config.Generation.OutputSuffix)
	assert.Equal(t, "Process", config.Generation.FunctionPrefix)
	assert.Equal(t, "encx", config.Generation.PackageName)
	assert.Empty(t, config.Packages)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "_encx",
					FunctionPrefix: "Process",
					PackageName:    "encx",
				},
			},
			expectError: false,
		},
		{
			name: "Empty output suffix",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "",
					FunctionPrefix: "Process",
					PackageName:    "encx",
				},
			},
			expectError: true,
			errorMsg:    "output_suffix cannot be empty",
		},
		{
			name: "Empty function prefix",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "_encx",
					FunctionPrefix: "",
					PackageName:    "encx",
				},
			},
			expectError: true,
			errorMsg:    "function_prefix cannot be empty",
		},
		{
			name: "Empty package name",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "_encx",
					FunctionPrefix: "Process",
					PackageName:    "",
				},
			},
			expectError: true,
			errorMsg:    "package_name cannot be empty",
		},
		{
			name: "Invalid function prefix with special characters",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "_encx",
					FunctionPrefix: "Process-Func",
					PackageName:    "encx",
				},
			},
			expectError: true,
			errorMsg:    "function_prefix must be a valid Go identifier",
		},
		{
			name: "Invalid package name with special characters",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "_encx",
					FunctionPrefix: "Process",
					PackageName:    "my-package",
				},
			},
			expectError: true,
			errorMsg:    "package_name must be a valid Go identifier",
		},
		{
			name: "Invalid output suffix starting with number",
			config: Config{
				Generation: GenerationConfig{
					OutputSuffix:   "1encx",
					FunctionPrefix: "Process",
					PackageName:    "encx",
				},
			},
			expectError: true,
			errorMsg:    "output_suffix must start with underscore or letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

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

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	config := Config{
		Generation: GenerationConfig{
			OutputSuffix:   "_test",
			FunctionPrefix: "TestProcess",
			PackageName:    "testpkg",
		},
		Packages: map[string]PackageConfig{
			"./internal": {
				Skip: false,
			},
		},
	}

	err := SaveConfig(&config, configFile)
	require.NoError(t, err)

	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	loadedConfig, err := LoadConfig(configFile)
	require.NoError(t, err)

	assert.Equal(t, config.Generation.OutputSuffix, loadedConfig.Generation.OutputSuffix)
	assert.Equal(t, config.Generation.FunctionPrefix, loadedConfig.Generation.FunctionPrefix)
	assert.Equal(t, config.Generation.PackageName, loadedConfig.Generation.PackageName)

	assert.Len(t, loadedConfig.Packages, 1)
	assert.Equal(t, config.Packages["./internal"].Skip, loadedConfig.Packages["./internal"].Skip)
}

func TestConfigWithComplexPackageStructure(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "complex.yaml")

	configContent := `
generation:
  output_suffix: "_secure"
  function_prefix: "Secure"
  package_name: "security"

packages:
  "./cmd":
    skip: true
  "./internal/api":
    skip: false
  "./internal/models":
    skip: false
  "./test":
    skip: true
  "./pkg/utils":
    skip: false
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configFile)
	require.NoError(t, err)

	assert.Len(t, config.Packages, 5)

	assert.True(t, config.Packages["./cmd"].Skip)
	assert.False(t, config.Packages["./internal/api"].Skip)
	assert.False(t, config.Packages["./internal/models"].Skip)
	assert.True(t, config.Packages["./test"].Skip)
	assert.False(t, config.Packages["./pkg/utils"].Skip)
}