package encx

import (
	"context"
	"database/sql"
	"fmt"
	"io"

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
	kmsService        KeyManagementService
	kekAlias          string
	pepper            []byte
	argon2Params      *Argon2Params
	serializer        Serializer
	keyMetadataDB     *sql.DB
	metricsCollector  MetricsCollector
	observabilityHook ObservabilityHook
}

// New creates a new Crypto instance using the legacy constructor signature.
// Deprecated: Use NewCrypto with options instead for better validation and flexibility.
//
// Example migration:
//
//	// Old way:
//	crypto, err := encx.New(ctx, kmsService, "my-kek", "secret/pepper")
//
//	// New way:
//	crypto, err := encx.NewCrypto(ctx,
//	    encx.WithKMSService(kmsService),
//	    encx.WithKEKAlias("my-kek"),
//	    encx.WithPepperSecretPath("secret/pepper"),
//	)
func New(
	ctx context.Context,
	kmsService KeyManagementService,
	kekAlias string,
	pepperSecretPath string,
	options ...CryptoOption,
) (*Crypto, error) {
	return NewCryptoLegacy(ctx, kmsService, kekAlias, pepperSecretPath, options...)
}

func (c *Crypto) getDatabasePathFromDB() (string, error) {
	var path string
	err := c.keyMetadataDB.QueryRow("PRAGMA database_list;").Scan(nil, &path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get database path from connection: %w", err)
	}
	return path, nil
}
