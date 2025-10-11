package encx

// Re-export the configuration options for public use
import "github.com/hengadev/encx/internal/config"

// Configuration option functions
// Note: KMS service, KEK alias, and pepper are now handled automatically
// via required parameters and environment variables for better security
var (
	// Removed options (now handled automatically):
	// WithKMSService       -> Required parameter in NewCrypto()
	// WithKEKAlias         -> Environment variable ENCX_KEK_ALIAS
	// WithPepper           -> Auto-generated and stored
	// WithPepperSecretPath -> Environment variable ENCX_PEPPER_SECRET_PATH
	// WithKeyMetadataDB    -> Auto-managed

	// Available optional options:
	WithArgon2Params          = config.WithArgon2Params
	WithDBPath                = config.WithDBPath
	WithDBFilename            = config.WithDBFilename
	WithKeyMetadataDBPath     = config.WithKeyMetadataDBPath
	WithKeyMetadataDBFilename = config.WithKeyMetadataDBFilename
	WithMetricsCollector      = config.WithMetricsCollector
	WithObservabilityHook     = config.WithObservabilityHook
)

// Helper functions
var (
	DefaultConfig = config.DefaultConfig
	ApplyOptions  = config.ApplyOptions
)
