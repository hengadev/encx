package encx

// Re-export monitoring types and constructors for public use
import "github.com/hengadev/encx/internal/monitoring"

// Constructor functions
var (
	NewInMemoryMetricsCollector    = monitoring.NewInMemoryMetricsCollector
	NewLoggingObservabilityHook    = monitoring.NewLoggingObservabilityHook
	NewMetricsObservabilityHook    = monitoring.NewMetricsObservabilityHook
	NewCompositeObservabilityHook  = monitoring.NewCompositeObservabilityHook
)

// Default implementations
var (
	NoOpMetricsCollector   = &monitoring.NoOpMetricsCollector{}
	NoOpObservabilityHook  = &monitoring.NoOpObservabilityHook{}
)
