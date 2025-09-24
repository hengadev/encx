package reliability

import (
	"context"
	"time"
)

// CryptoReliabilityConfig holds configuration for crypto operations reliability
type CryptoReliabilityConfig struct {
	// KMS operations configuration
	KMSOperations ReliabilityConfig
	// Database operations configuration
	DatabaseOperations ReliabilityConfig
	// Network operations configuration
	NetworkOperations ReliabilityConfig
}

// DefaultCryptoReliabilityConfig returns default reliability configuration for crypto operations
func DefaultCryptoReliabilityConfig() CryptoReliabilityConfig {
	// KMS operations - typically more sensitive to failures
	kmsConfig := DefaultReliabilityConfig()
	kmsConfig.CircuitBreaker.FailureThreshold = 3
	kmsConfig.CircuitBreaker.Timeout = time.Second * 30
	kmsConfig.Retry.MaxAttempts = 3
	kmsConfig.Retry.InitialDelay = time.Millisecond * 200
	kmsConfig.Retry.MaxDelay = time.Second * 10

	// Database operations - can handle more retries
	dbConfig := DefaultReliabilityConfig()
	dbConfig.CircuitBreaker.FailureThreshold = 5
	dbConfig.CircuitBreaker.Timeout = time.Second * 15
	dbConfig.Retry.MaxAttempts = 5
	dbConfig.Retry.InitialDelay = time.Millisecond * 100
	dbConfig.Retry.MaxDelay = time.Second * 5

	// Network operations - most tolerant
	networkConfig := DefaultReliabilityConfig()
	networkConfig.CircuitBreaker.FailureThreshold = 7
	networkConfig.CircuitBreaker.Timeout = time.Second * 45
	networkConfig.Retry.MaxAttempts = 4
	networkConfig.Retry.InitialDelay = time.Millisecond * 150
	networkConfig.Retry.MaxDelay = time.Second * 15

	return CryptoReliabilityConfig{
		KMSOperations:      kmsConfig,
		DatabaseOperations: dbConfig,
		NetworkOperations:  networkConfig,
	}
}

// CryptoReliabilityManager manages reliability services for crypto operations
type CryptoReliabilityManager struct {
	manager *ReliabilityManager
	config  CryptoReliabilityConfig
}

// NewCryptoReliabilityManager creates a new crypto reliability manager
func NewCryptoReliabilityManager(config CryptoReliabilityConfig) *CryptoReliabilityManager {
	return &CryptoReliabilityManager{
		manager: NewReliabilityManager(),
		config:  config,
	}
}

// ExecuteKMSOperation executes a KMS operation with reliability protection
func (crm *CryptoReliabilityManager) ExecuteKMSOperation(
	ctx context.Context,
	operationName string,
	operation func(context.Context) error,
) error {
	serviceName := "kms_" + operationName
	service := crm.manager.GetOrCreate(serviceName, crm.config.KMSOperations)
	return service.Execute(ctx, operation)
}

// ExecuteKMSOperationWithFallback executes a KMS operation with reliability protection and fallback
func (crm *CryptoReliabilityManager) ExecuteKMSOperationWithFallback(
	ctx context.Context,
	operationName string,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	serviceName := "kms_" + operationName
	service := crm.manager.GetOrCreate(serviceName, crm.config.KMSOperations)
	return service.ExecuteWithFallback(ctx, operation, fallback)
}

// ExecuteDatabaseOperation executes a database operation with reliability protection
func (crm *CryptoReliabilityManager) ExecuteDatabaseOperation(
	ctx context.Context,
	operationName string,
	operation func(context.Context) error,
) error {
	serviceName := "db_" + operationName
	service := crm.manager.GetOrCreate(serviceName, crm.config.DatabaseOperations)
	return service.Execute(ctx, operation)
}

// ExecuteDatabaseOperationWithFallback executes a database operation with reliability protection and fallback
func (crm *CryptoReliabilityManager) ExecuteDatabaseOperationWithFallback(
	ctx context.Context,
	operationName string,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	serviceName := "db_" + operationName
	service := crm.manager.GetOrCreate(serviceName, crm.config.DatabaseOperations)
	return service.ExecuteWithFallback(ctx, operation, fallback)
}

// ExecuteNetworkOperation executes a network operation with reliability protection
func (crm *CryptoReliabilityManager) ExecuteNetworkOperation(
	ctx context.Context,
	operationName string,
	operation func(context.Context) error,
) error {
	serviceName := "network_" + operationName
	service := crm.manager.GetOrCreate(serviceName, crm.config.NetworkOperations)
	return service.Execute(ctx, operation)
}

// ExecuteNetworkOperationWithFallback executes a network operation with reliability protection and fallback
func (crm *CryptoReliabilityManager) ExecuteNetworkOperationWithFallback(
	ctx context.Context,
	operationName string,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	serviceName := "network_" + operationName
	service := crm.manager.GetOrCreate(serviceName, crm.config.NetworkOperations)
	return service.ExecuteWithFallback(ctx, operation, fallback)
}

// GetAllStats returns statistics for all crypto reliability services
func (crm *CryptoReliabilityManager) GetAllStats() map[string]ReliabilityStats {
	return crm.manager.GetAllStats()
}

// GetHealthyServices returns a list of healthy crypto services
func (crm *CryptoReliabilityManager) GetHealthyServices() []string {
	return crm.manager.GetHealthyServices()
}

// GetUnhealthyServices returns a list of unhealthy crypto services
func (crm *CryptoReliabilityManager) GetUnhealthyServices() []string {
	return crm.manager.GetUnhealthyServices()
}

// IsOperationHealthy checks if a specific operation type is healthy
func (crm *CryptoReliabilityManager) IsOperationHealthy(operationType, operationName string) bool {
	serviceName := operationType + "_" + operationName
	if service, exists := crm.manager.Get(serviceName); exists {
		return service.IsHealthy()
	}
	return true // If service doesn't exist yet, consider it healthy
}

// CryptoOperationType represents the type of crypto operation
type CryptoOperationType string

const (
	KMSOperation      CryptoOperationType = "kms"
	DatabaseOperation CryptoOperationType = "db"
	NetworkOperation  CryptoOperationType = "network"
)

// ExecuteOperation executes a crypto operation with appropriate reliability configuration
func (crm *CryptoReliabilityManager) ExecuteOperation(
	ctx context.Context,
	operationType CryptoOperationType,
	operationName string,
	operation func(context.Context) error,
) error {
	switch operationType {
	case KMSOperation:
		return crm.ExecuteKMSOperation(ctx, operationName, operation)
	case DatabaseOperation:
		return crm.ExecuteDatabaseOperation(ctx, operationName, operation)
	case NetworkOperation:
		return crm.ExecuteNetworkOperation(ctx, operationName, operation)
	default:
		// Default to KMS configuration for unknown operation types
		return crm.ExecuteKMSOperation(ctx, operationName, operation)
	}
}

// ExecuteOperationWithFallback executes a crypto operation with appropriate reliability configuration and fallback
func (crm *CryptoReliabilityManager) ExecuteOperationWithFallback(
	ctx context.Context,
	operationType CryptoOperationType,
	operationName string,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	switch operationType {
	case KMSOperation:
		return crm.ExecuteKMSOperationWithFallback(ctx, operationName, operation, fallback)
	case DatabaseOperation:
		return crm.ExecuteDatabaseOperationWithFallback(ctx, operationName, operation, fallback)
	case NetworkOperation:
		return crm.ExecuteNetworkOperationWithFallback(ctx, operationName, operation, fallback)
	default:
		// Default to KMS configuration for unknown operation types
		return crm.ExecuteKMSOperationWithFallback(ctx, operationName, operation, fallback)
	}
}

// ReliabilityWrapper provides a simple interface for adding reliability to any operation
type ReliabilityWrapper struct {
	service *ReliabilityService
}

// NewReliabilityWrapper creates a new reliability wrapper with default configuration
func NewReliabilityWrapper(name string) *ReliabilityWrapper {
	return &ReliabilityWrapper{
		service: GetReliabilityService(name),
	}
}

// NewReliabilityWrapperWithConfig creates a new reliability wrapper with custom configuration
func NewReliabilityWrapperWithConfig(name string, config ReliabilityConfig) *ReliabilityWrapper {
	return &ReliabilityWrapper{
		service: GetReliabilityServiceWithConfig(name, config),
	}
}

// Wrap wraps an operation with reliability protection
func (rw *ReliabilityWrapper) Wrap(ctx context.Context, operation func(context.Context) error) error {
	return rw.service.Execute(ctx, operation)
}

// WrapWithFallback wraps an operation with reliability protection and fallback
func (rw *ReliabilityWrapper) WrapWithFallback(
	ctx context.Context,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	return rw.service.ExecuteWithFallback(ctx, operation, fallback)
}

// GetStats returns statistics for the wrapped service
func (rw *ReliabilityWrapper) GetStats() ReliabilityStats {
	return rw.service.GetStats()
}

// IsHealthy returns true if the wrapped service is healthy
func (rw *ReliabilityWrapper) IsHealthy() bool {
	return rw.service.IsHealthy()
}

// Global crypto reliability manager for convenience
var globalCryptoReliabilityManager = NewCryptoReliabilityManager(DefaultCryptoReliabilityConfig())

// ExecuteCryptoOperation executes a crypto operation with global reliability protection
func ExecuteCryptoOperation(
	ctx context.Context,
	operationType CryptoOperationType,
	operationName string,
	operation func(context.Context) error,
) error {
	return globalCryptoReliabilityManager.ExecuteOperation(ctx, operationType, operationName, operation)
}

// ExecuteCryptoOperationWithFallback executes a crypto operation with global reliability protection and fallback
func ExecuteCryptoOperationWithFallback(
	ctx context.Context,
	operationType CryptoOperationType,
	operationName string,
	operation func(context.Context) error,
	fallback func(context.Context) error,
) error {
	return globalCryptoReliabilityManager.ExecuteOperationWithFallback(
		ctx, operationType, operationName, operation, fallback,
	)
}

// GetCryptoReliabilityStats returns statistics for all crypto reliability services
func GetCryptoReliabilityStats() map[string]ReliabilityStats {
	return globalCryptoReliabilityManager.GetAllStats()
}

// GetHealthyCryptoServices returns a list of healthy crypto services
func GetHealthyCryptoServices() []string {
	return globalCryptoReliabilityManager.GetHealthyServices()
}

// GetUnhealthyCryptoServices returns a list of unhealthy crypto services
func GetUnhealthyCryptoServices() []string {
	return globalCryptoReliabilityManager.GetUnhealthyServices()
}