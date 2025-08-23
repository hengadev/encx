package encx

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type CryptoService interface {
	GetPepper() []byte
	GetArgon2Params() *Argon2Params
	GetAlias() string
	GenerateDEK() ([]byte, error)
	EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error)
	DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error)
	ProcessStruct(ctx context.Context, object any) error
	DecryptStruct(ctx context.Context, object any) error
	EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error)
	DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error)
	RotateKEK(ctx context.Context) error
	HashBasic(ctx context.Context, value []byte) string
	HashSecure(ctx context.Context, value []byte) (string, error)
	CompareSecureHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
	CompareBasicHashAndValue(ctx context.Context, value any, hashValue string) (bool, error)
	EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
	DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error
}

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
	pepper, err := kmsService.GetSecret(ctx, pepperSecretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pepper from KMS: %w", err)
	}
	expectedPepperLength := 32
	if len(pepper) != expectedPepperLength {
		return nil, fmt.Errorf("invalid pepper length retrieved from KMS: expected %d, got %d", expectedPepperLength, len(pepper))
	}

	var dbPath string

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
			dbPath, err = cryptoInstance.getDatabasePathFromDB()
			if err != nil {
				return nil, err
			}
		}
	}

	if cryptoInstance.keyMetadataDB == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory for default DB path: %w", err)
		}
		defaultDataDir := filepath.Join(cwd, defaultDBDirName)
		if err := os.MkdirAll(defaultDataDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create default '%s' directory: %w", defaultDBDirName, err)
		}
		dbPath = filepath.Join(defaultDataDir, defaultDBFileName)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open default key metadata database at '%s': %v", dbPath, err)
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
	kmsKeyID, err := c.kmsService.GetKeyID(ctx, c.kekAlias)
	if err != nil {
		// Assume an error here likely means the key doesn't exist in KMS
		// (depending on the KMS implementation's error handling).
		// We'll proceed to create a new key.
		log.Printf("No KEK found in KMS for alias '%s', creating a new one.", c.kekAlias)
		kmsKeyID, err = c.kmsService.CreateKey(ctx, c.kekAlias) // Use the alias as a description
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
		return nil
	}
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return err
	}
	if currentVersion == 0 {
		// Key exists in KMS but not in our DB (this could happen if the DB was wiped or is new)
		// We should record it, assuming it's the first version.
		_, err = c.keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, c.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", c.kekAlias, kmsKeyID)
	}
	// Key exists in KMS and is recorded in our DB. We don't need to do anything.
	log.Printf("KEK found in KMS for alias '%s' with KMS ID '%s', current version is %d.", c.kekAlias, kmsKeyID, currentVersion)
	return nil
}

func (c *Crypto) getDatabasePathFromDB() (string, error) {
	var path string
	err := c.keyMetadataDB.QueryRow("PRAGMA database_list;").Scan(nil, &path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get database path from connection: %w", err)
	}
	return path, nil
}
