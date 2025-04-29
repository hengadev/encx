package encx

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

type CryptoOption func(e *Crypto) error

func WithArgon2Params(params *Argon2Params) CryptoOption {
	return func(c *Crypto) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("validate Argon2Params: %w", err)
		}
		c.argon2Params = params
		return nil
	}
}

// WithKeyMetadataDBPath sets the full path to the key metadata database.
func WithKeyMetadataDBPath(path string) CryptoOption {
	return func(c *Crypto) error {
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			return fmt.Errorf("failed to open key metadata database with optional path '%s': %v", path, err)
		}
		c.keyMetadataDB = db
		return nil
	}
}

// WithKeyMetadataDBFilename sets the filename for the key metadata database within the default directory.
func WithKeyMetadataDBFilename(filename string) CryptoOption {
	return func(c *Crypto) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory for default DB path: %w", err)
		}
		defaultDataDir := filepath.Join(cwd, defaultDBDirName)
		if err := os.MkdirAll(defaultDataDir, 0700); err != nil {
			return fmt.Errorf("failed to create default '%s' directory: %w", defaultDBDirName, err)
		}
		dbPath := filepath.Join(defaultDataDir, filename)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open key metadata database with filename '%s': %v", filename, err)
		}
		c.keyMetadataDB = db
		return nil
	}
}

func WithSerializer(serializer Serializer) CryptoOption {
	return func(c *Crypto) error {
		c.serializer = serializer
		return nil
	}
}
