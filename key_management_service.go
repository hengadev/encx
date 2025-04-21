package encx

import "context"

// KeyManagementService defines the low-level contract for interacting with a key management service.
type KeyManagementService interface {

	// GetKey retrieves the actual key material (if allowed by the KMS and your policy).
	// This might not be applicable or advisable for all KMS (e.g., AWS KMS).
	GetKey(ctx context.Context, keyID string) ([]byte, error)

	// GetKeyID retrieves the identifier of a managed key.
	GetKeyID(ctx context.Context, alias string) (string, error)

	// CreateKey creates a new managed key and returns its ID.
	CreateKey(ctx context.Context, description string) (string, error)

	// RotateKey triggers a key rotation for the managed key.
	RotateKey(ctx context.Context, keyID string) error

	// GetSecret retrieves a secret by its path.
	GetSecret(ctx context.Context, path string) ([]byte, error)

	// SetSecret stores a secret at a given path.
	SetSecret(ctx context.Context, path string, value []byte) error

	EncryptDEK(ctx context.Context, keyID string, plaintextDEK []byte) ([]byte, error)

	DecryptDEK(ctx context.Context, keyID string, ciphertextDEK []byte) ([]byte, error)

	// GetCurrentKEKVersion(ctx, alias string) (int, error)

	EncryptDEKWithVersion(ctx, plaintextDEK []byte, version int) ([]byte, error)

	DecryptDEKWithVersion(ctx, ciphertextDEK []byte, version int) ([]byte, error)
}
