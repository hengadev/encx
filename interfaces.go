package encx

import "context"

// KeyManagementService defines the contract for cryptographic key operations.
//
// This interface is implemented by KMS providers (AWS KMS, HashiCorp Vault Transit Engine, etc.)
// and handles cryptographic operations with Key Encryption Keys (KEKs). It is responsible for:
//   - Managing KEK lifecycle (creation, retrieval)
//   - Encrypting Data Encryption Keys (DEKs) with KEKs
//   - Decrypting DEKs that were encrypted with KEKs
//
// This interface is separate from SecretManagementService, which handles secret storage.
// KeyManagementService performs cryptographic operations, while SecretManagementService
// stores and retrieves secret values.
//
// Implementations:
//   - AWS KMS: github.com/hengadev/encx/providers/aws.KMSService
//   - HashiCorp Vault Transit: github.com/hengadev/encx/providers/hashicorp.TransitService
//
// Example usage:
//
//	import "github.com/hengadev/encx/providers/aws"
//
//	kms, err := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use with Crypto
//	crypto, err := encx.NewCrypto(ctx, kms, secretStore, cfg)
type KeyManagementService interface {
	// GetKeyID resolves a key alias to a key ID.
	//
	// For AWS KMS, this resolves an alias like "alias/my-key" to the underlying key ID.
	// For HashiCorp Vault, this returns the key name directly.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - alias: The key alias to resolve (e.g., "alias/myapp-kek", "transit/keys/myapp")
	//
	// Returns:
	//   - The resolved key ID
	//   - Error if the key cannot be found or accessed
	GetKeyID(ctx context.Context, alias string) (string, error)

	// CreateKey creates a new encryption key in the KMS.
	//
	// This creates a symmetric encryption key suitable for encrypting DEKs.
	// The key remains in the KMS and is never exposed.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - description: Human-readable description for the key
	//
	// Returns:
	//   - The created key ID
	//   - Error if key creation fails
	CreateKey(ctx context.Context, description string) (string, error)

	// EncryptDEK encrypts a Data Encryption Key (DEK) using the specified KMS key.
	//
	// The DEK is encrypted with the KEK identified by keyID. The encrypted DEK can
	// be safely stored and later decrypted using DecryptDEK.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - keyID: The KMS key ID to use for encryption
	//   - plaintext: The plaintext DEK to encrypt (typically 32 bytes)
	//
	// Returns:
	//   - The encrypted DEK (ciphertext)
	//   - Error if encryption fails
	EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)

	// DecryptDEK decrypts a Data Encryption Key (DEK) that was encrypted by EncryptDEK.
	//
	// The ciphertext DEK is decrypted using the KEK identified by keyID. The KMS
	// performs the decryption operation without exposing the KEK.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - keyID: The KMS key ID to use for decryption
	//   - ciphertext: The encrypted DEK to decrypt
	//
	// Returns:
	//   - The plaintext DEK
	//   - Error if decryption fails
	DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}

// SecretManagementService defines the contract for secret storage and retrieval operations.
//
// This interface is implemented by secret storage providers (AWS Secrets Manager,
// HashiCorp Vault KV Engine, in-memory store for testing, etc.) and handles the storage
// and retrieval of sensitive configuration values such as peppers.
//
// This interface is separate from KeyManagementService, which handles cryptographic operations.
// SecretManagementService stores secret values, while KeyManagementService performs
// cryptographic operations on keys.
//
// Implementations:
//   - AWS Secrets Manager: github.com/hengadev/encx/providers/aws.SecretsManagerStore
//   - HashiCorp Vault KV v2: github.com/hengadev/encx/providers/hashicorp.KVStore
//   - In-Memory (testing): encx.InMemorySecretStore
//
// Example usage:
//
//	import "github.com/hengadev/encx/providers/aws"
//
//	secrets, err := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use with Crypto
//	crypto, err := encx.NewCrypto(ctx, kmsService, secrets, cfg)
//
// Storage Path Convention:
//
// Each implementation determines its own storage path based on the alias.
// For example:
//   - AWS Secrets Manager: "encx/{alias}/pepper"
//   - Vault KV v2: "secret/data/encx/{alias}/pepper"
//   - In-Memory: "memory://{alias}/pepper"
//
// This allows for service isolation in microservices architectures where each
// service uses a unique alias (e.g., "user-service", "payment-service").
type SecretManagementService interface {
	// StorePepper stores a pepper secret for the specified alias.
	//
	// The pepper must be exactly 32 bytes (PepperLength constant). If a pepper
	// already exists for this alias, it will be updated.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - alias: The service identifier (e.g., "user-service", "myapp")
	//   - pepper: The pepper bytes to store (must be 32 bytes)
	//
	// Returns:
	//   - Error if storage fails or pepper length is invalid
	//
	// Example:
	//
	//	pepper := make([]byte, encx.PepperLength)
	//	rand.Read(pepper)
	//	err := secretStore.StorePepper(ctx, "user-service", pepper)
	StorePepper(ctx context.Context, alias string, pepper []byte) error

	// GetPepper retrieves the pepper secret for the specified alias.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - alias: The service identifier (e.g., "user-service", "myapp")
	//
	// Returns:
	//   - The pepper bytes (always 32 bytes)
	//   - Error if the pepper doesn't exist or retrieval fails
	//
	// Example:
	//
	//	pepper, err := secretStore.GetPepper(ctx, "user-service")
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	GetPepper(ctx context.Context, alias string) ([]byte, error)

	// PepperExists checks if a pepper exists for the specified alias.
	//
	// This is useful for determining whether to generate a new pepper or load
	// an existing one during initialization.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - alias: The service identifier (e.g., "user-service", "myapp")
	//
	// Returns:
	//   - true if the pepper exists, false otherwise
	//   - Error only if the check itself fails (not if pepper doesn't exist)
	//
	// Example:
	//
	//	exists, err := secretStore.PepperExists(ctx, "user-service")
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	if !exists {
	//	    // Generate and store new pepper
	//	}
	PepperExists(ctx context.Context, alias string) (bool, error)

	// GetStoragePath returns the full storage path for a given alias.
	//
	// This method is primarily for debugging and logging purposes, allowing
	// users to see exactly where their secrets are stored in the underlying
	// secret management system.
	//
	// The returned path format is implementation-specific:
	//   - AWS Secrets Manager: "encx/{alias}/pepper"
	//   - Vault KV v2: "secret/data/encx/{alias}/pepper"
	//   - In-Memory: "memory://{alias}/pepper"
	//
	// Parameters:
	//   - alias: The service identifier (e.g., "user-service", "myapp")
	//
	// Returns:
	//   - The full storage path as a string
	//
	// Example:
	//
	//	path := secretStore.GetStoragePath("user-service")
	//	log.Printf("Pepper will be stored at: %s", path)
	GetStoragePath(alias string) string
}
