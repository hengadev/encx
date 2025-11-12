package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot searches for the project root directory by looking for go.mod
// starting from the given directory and traversing up the directory tree.
//
// It returns the absolute path to the directory containing go.mod, or an error
// if go.mod is not found in any parent directory.
//
// Example:
//
//	projectRoot, err := FindProjectRoot("/home/user/project/internal/config")
//	if err != nil {
//	    // go.mod not found, use fallback behavior
//	}
//	// projectRoot might be "/home/user/project" if go.mod exists there
func FindProjectRoot(startDir string) (string, error) {
	// Ensure we have an absolute path
	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	currentDir := absPath

	// Traverse up the directory tree
	for {
		// Check if go.mod exists in current directory
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod
			return currentDir, nil
		}

		// Get parent directory
		parentDir := filepath.Dir(currentDir)

		// Check if we've reached the root directory
		if parentDir == currentDir {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}

		currentDir = parentDir
	}
}
