package encx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hengadev/encx/internal/config"
)

// Config holds the configuration for creating a Crypto instance.
//
// This struct contains only data, no behavior. Configuration can be loaded from
// any source (environment variables, files, code, etc.) and passed explicitly to NewCrypto.
//
// Required fields:
//   - KEKAlias: The KMS key identifier for encrypting/decrypting DEKs
//   - PepperAlias: The service identifier for pepper storage
//
// Optional fields (defaults are applied if empty):
//   - DBPath: Database directory (default: .encx)
//   - DBFilename: Database filename (default: keys.db)
//
// Example usage:
//
//	// Explicit configuration
//	cfg := encx.Config{
//	    KEKAlias:    "user-service-kek",
//	    PepperAlias: "user-service",
//	    DBPath:      "/var/lib/encx",
//	    DBFilename:  "production.db",
//	}
//
//	// Validate and apply defaults
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
type Config struct {
	// KEKAlias is the Key Encryption Key identifier in the KMS.
	//
	// This identifies which key to use for encrypting/decrypting DEKs.
	// The format depends on the KMS provider:
	//   - AWS KMS: "alias/my-key" or full ARN
	//   - HashiCorp Vault: key name (e.g., "my-service-kek")
	//
	// Required field. Maximum length: 256 characters.
	KEKAlias string

	// PepperAlias is the service identifier for pepper storage.
	//
	// This is used to construct the storage path in the SecretManagementService.
	// Each service should use a unique alias to isolate peppers:
	//   - Microservice: use service name (e.g., "user-service", "payment-service")
	//   - Monolith: use application name (e.g., "myapp")
	//
	// The SecretManagementService implementation uses this to create the full path:
	//   - AWS: "encx/{PepperAlias}/pepper"
	//   - Vault: "secret/data/encx/{PepperAlias}/pepper"
	//
	// Required field.
	PepperAlias string

	// DBPath is the directory where the key metadata database is stored.
	//
	// This SQLite database stores KEK version information for key rotation.
	// If empty, the default ".encx" is used.
	//
	// Optional field. Default: .encx
	DBPath string

	// DBFilename is the filename of the key metadata database.
	//
	// If empty, the default "keys.db" is used.
	//
	// Optional field. Default: keys.db
	DBFilename string
}

// Validate checks that the configuration is valid and applies defaults to optional fields.
//
// This method:
//   - Ensures required fields (KEKAlias, PepperAlias) are not empty
//   - Validates KEKAlias length (must be <= MaxKEKAliasLength)
//   - Applies defaults to optional fields (DBPath, DBFilename) if empty
//
// Returns an error if validation fails.
//
// Example:
//
//	cfg := encx.Config{
//	    KEKAlias:    "my-service-kek",
//	    PepperAlias: "my-service",
//	    // DBPath and DBFilename will be set to defaults
//	}
//
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//	// cfg.DBPath is now ".encx"
//	// cfg.DBFilename is now "keys.db"
func (c *Config) Validate() error {
	// Validate required fields
	if c.KEKAlias == "" {
		return fmt.Errorf("KEKAlias is required")
	}

	if len(c.KEKAlias) > MaxKEKAliasLength {
		return fmt.Errorf("KEKAlias must be %d characters or less, got %d", MaxKEKAliasLength, len(c.KEKAlias))
	}

	if c.PepperAlias == "" {
		return fmt.Errorf("PepperAlias is required")
	}

	// Apply defaults to optional fields
	if c.DBPath == "" {
		// Try to find project root (directory with go.mod), fall back to relative path if not found
		cwd, err := os.Getwd()
		if err == nil {
			if projectRoot, err := config.FindProjectRoot(cwd); err == nil {
				c.DBPath = filepath.Join(projectRoot, DefaultDBPath)
			} else {
				// If go.mod not found, use relative path (for non-Go-module projects)
				c.DBPath = DefaultDBPath
			}
		} else {
			// If we can't get cwd, fall back to relative path
			c.DBPath = DefaultDBPath
		}
	}

	if c.DBFilename == "" {
		c.DBFilename = DefaultDBFilename
	}

	return nil
}
