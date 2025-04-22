package encx

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Crypto struct {
	kmsService    KeyManagementService
	kekAlias      string
	pepper        []byte
	argon2Params  *Argon2Params
	serializer    Serializer
	keyMetadataDB *sql.DB
}

// NewCrypto creates a new Crypto instance, initializing the KMS service and retrieving necessary secrets and KEK ID.
func New(
	ctx context.Context,
	kmsService KeyManagementService,
	kekAlias string,
	pepperSecretPath string,
	options ...CryptoOption,
) (*Crypto, error) {
	pepperBytes, err := kmsService.GetSecret(ctx, pepperSecretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pepper from KMS: %w", err)
	}
	if len(pepperBytes) != 16 {
		return nil, fmt.Errorf("invalid pepper length retrieved from KMS: expected 16, got %d", len(pepperBytes))
	}
	var pepper []byte
	copy(pepper[:], pepperBytes)

	var dbPath string
	foundDBPathOption := false

	cryptoInstance := &Crypto{
		kmsService:    kmsService,
		kekAlias:      kekAlias,
		pepper:        pepper,
		argon2Params:  DefaultArgon2Params,
		serializer:    JSONSerializer{}, // default serializer
		keyMetadataDB: nil,
	}

	for _, opt := range options {
		if err := opt(cryptoInstance); err != nil {
			return nil, fmt.Errorf("setting option: %w", err)
		}
		if cryptoInstance.keyMetadataDB != nil {
			foundDBPathOption = true
		}
	}

	if foundDBPathOption {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory for default DB path: %w", err)
		}
		defaultDataDir := filepath.Join(cwd, ".encx")
		if err := os.MkdirAll(defaultDataDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create default '.encx' directory to store key tracking database: %w", err)
		}
		dbPath = filepath.Join(defaultDataDir, generateDBname())
	} else if cryptoInstance.keyMetadataDB != nil {
		dbPath = ""
	} else {
		// This case should ideally not happen if WithKeyMetadataDBPath works correctly
		return nil, fmt.Errorf("keyMetadataDB path was provided but database connection was not established")
	}

	if dbPath != "" {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open key metadata database: %v", err)
		}
		cryptoInstance.keyMetadataDB = db
	}

	_, err = cryptoInstance.keyMetadataDB.Exec(`
		CREATE TABLE IF NOT EXISTS kek_versions (
			alias TEXT NOT NULL,
			version INTEGER NOT NULL,
			creation_time DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_deprecated BOOLEAN DEFAULT FALSE,
			kms_key_id TEXT NOT NULL,
			PRIMARY KEY (alias, version)
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create kek_versions table in database at '%s': %w", dbPath, err)
	}

	// Ensure an initial KEK exists for this alias
	err = cryptoInstance.ensureInitialKEK(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure initial KEK: %w", err)
	}

	return cryptoInstance, nil
}

// ensureInitialKEK checks if a KEK exists for the given alias and creates one if not.
func (c *Crypto) ensureInitialKEK(ctx context.Context) error {
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return err
	}
	if currentVersion == 0 {
		// No key exists, create the first one
		kmsKeyID, err := c.kmsService.CreateKey(ctx, c.kekAlias) // Use the alias as a description
		if err != nil {
			return fmt.Errorf("failed to create initial KEK in KMS: %w", err)
		}
		_, err = c.keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, c.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", c.kekAlias, kmsKeyID)
	}
	return nil
}

// getCurrentKEKVersion retrieves the current active KEK version for a given alias.
func (c *Crypto) getCurrentKEKVersion(ctx context.Context, alias string) (int, error) {
	row := c.keyMetadataDB.QueryRowContext(ctx, `
		SELECT version FROM kek_versions
		WHERE alias = ? AND is_deprecated = FALSE
		ORDER BY version DESC
		LIMIT 1
	`, alias)
	var version int
	err := row.Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil // No active key found
	} else if err != nil {
		return 0, fmt.Errorf("failed to get current KEK version for alias '%s': %w", alias, err)
	}
	return version, nil
}
