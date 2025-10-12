package encx

import (
	"fmt"
	"os"
)

// LoadConfigFromEnvironment loads configuration from environment variables.
//
// This function reads configuration from standard environment variables and returns
// a validated Config struct. It follows the 12-factor app methodology where configuration
// is read from the environment.
//
// Required environment variables:
//   - ENCX_KEK_ALIAS: KMS key identifier for encrypting/decrypting DEKs
//   - ENCX_PEPPER_ALIAS: Service identifier for pepper storage
//
// Optional environment variables (defaults are applied if not set):
//   - ENCX_DB_PATH: Database directory (default: .encx)
//   - ENCX_DB_FILENAME: Database filename (default: keys.db)
//
// Returns an error if required variables are missing or validation fails.
//
// Example usage (12-factor app):
//
//	// Set environment variables (typically in deployment config):
//	// export ENCX_KEK_ALIAS="user-service-kek"
//	// export ENCX_PEPPER_ALIAS="user-service"
//	// export ENCX_DB_PATH="/var/lib/encx"  # optional
//
//	cfg, err := encx.LoadConfigFromEnvironment()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
//
// Example usage (convenience with NewCryptoFromEnv):
//
//	// NewCryptoFromEnv calls this function internally
//	crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
func LoadConfigFromEnvironment() (Config, error) {
	// Read required environment variables
	kekAlias := os.Getenv(EnvKEKAlias)
	if kekAlias == "" {
		return Config{}, fmt.Errorf("%s environment variable is required", EnvKEKAlias)
	}

	pepperAlias := os.Getenv(EnvPepperAlias)
	if pepperAlias == "" {
		return Config{}, fmt.Errorf("%s environment variable is required", EnvPepperAlias)
	}

	// Read optional environment variables with defaults
	dbPath := getEnvOrDefault(EnvDBPath, DefaultDBPath)
	dbFilename := getEnvOrDefault(EnvDBFilename, DefaultDBFilename)

	// Create config
	cfg := Config{
		KEKAlias:    kekAlias,
		PepperAlias: pepperAlias,
		DBPath:      dbPath,
		DBFilename:  dbFilename,
	}

	// Validate config (this also applies defaults if needed)
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// getEnvOrDefault returns the value of an environment variable, or a default value if not set.
//
// Parameters:
//   - key: The environment variable name
//   - defaultValue: The default value to return if the variable is not set
//
// Returns:
//   - The environment variable value if set and non-empty
//   - The default value if the variable is not set or empty
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
