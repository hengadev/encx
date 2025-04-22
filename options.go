package encx

import (
	"database/sql"
	"fmt"
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

func WithSerializer(serializer Serializer) CryptoOption {
	return func(c *Crypto) error {
		c.serializer = serializer
		return nil
	}
}
