package encx

// Pepper constants
const (
	// PepperLength defines the required length for pepper values in bytes.
	// Peppers must be exactly 32 bytes for cryptographic operations.
	PepperLength = 32
)

// Environment variable names
const (
	// EnvKEKAlias is the environment variable name for the KEK (Key Encryption Key) alias.
	// This identifies which key to use in the KMS for encrypting/decrypting DEKs.
	// Example: "user-service-kek" or "alias/myapp-kek"
	EnvKEKAlias = "ENCX_KEK_ALIAS"

	// EnvPepperAlias is the environment variable name for the pepper storage alias.
	// This identifies which pepper to use for secure hashing operations.
	// The alias is used to construct the storage path in the SecretManagementService.
	// Example: "user-service" or "myapp"
	EnvPepperAlias = "ENCX_PEPPER_ALIAS"

	// EnvDBPath is the environment variable name for the database directory path.
	// This specifies where the SQLite database for key metadata is stored.
	// Default: .encx
	EnvDBPath = "ENCX_DB_PATH"

	// EnvDBFilename is the environment variable name for the database filename.
	// This specifies the filename of the SQLite database for key metadata.
	// Default: keys.db
	EnvDBFilename = "ENCX_DB_FILENAME"
)

// Default values
const (
	// DefaultDBPath is the default directory for the key metadata database.
	DefaultDBPath = ".encx"

	// DefaultDBFilename is the default filename for the key metadata database.
	DefaultDBFilename = "keys.db"
)

// Storage path templates for different secret management providers
const (
	// AWSPepperPathTemplate is the path template for storing peppers in AWS Secrets Manager.
	// The %s placeholder is replaced with the pepper alias (service identifier).
	// Example: "encx/user-service/pepper"
	AWSPepperPathTemplate = "encx/%s/pepper"

	// VaultPepperPathTemplate is the path template for storing peppers in HashiCorp Vault KV v2.
	// The %s placeholder is replaced with the pepper alias (service identifier).
	// Example: "secret/data/encx/user-service/pepper"
	// Note: This follows the KV v2 API path convention where "secret/data/" is the mount point.
	VaultPepperPathTemplate = "secret/data/encx/%s/pepper"
)

// KEK constraints
const (
	// MaxKEKAliasLength is the maximum allowed length for a KEK alias.
	// This prevents excessively long identifiers that could cause issues with storage or KMS APIs.
	MaxKEKAliasLength = 256
)
