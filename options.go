package encx

// Re-export the configuration options for public use
import "github.com/hengadev/encx/internal/config"

// Configuration option functions
var (
	WithKMSService              = config.WithKMSService
	WithKEKAlias                = config.WithKEKAlias
	WithPepper                  = config.WithPepper
	WithPepperSecretPath        = config.WithPepperSecretPath
	WithArgon2Params            = config.WithArgon2Params
	WithSerializer              = config.WithSerializer
	WithKeyMetadataDB           = config.WithKeyMetadataDB
	WithDBPath                  = config.WithDBPath
	WithDBFilename              = config.WithDBFilename
	WithKeyMetadataDBPath       = config.WithKeyMetadataDBPath
	WithKeyMetadataDBFilename   = config.WithKeyMetadataDBFilename
	WithMetricsCollector        = config.WithMetricsCollector
	WithObservabilityHook       = config.WithObservabilityHook
)

// Helper functions
var (
	DefaultConfig = config.DefaultConfig
	ApplyOptions  = config.ApplyOptions
)
