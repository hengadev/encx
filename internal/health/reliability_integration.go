package health

import (
	"context"
	"fmt"
	"time"

	"github.com/hengadev/encx/internal/reliability"
)

// ReliabilityHealthChecker provides health checks for reliability services
type ReliabilityHealthChecker struct {
	reliabilityManager *reliability.ReliabilityManager
}

// NewReliabilityHealthChecker creates a new reliability health checker
func NewReliabilityHealthChecker(reliabilityManager *reliability.ReliabilityManager) *ReliabilityHealthChecker {
	return &ReliabilityHealthChecker{
		reliabilityManager: reliabilityManager,
	}
}

// CreateCircuitBreakerHealthCheck creates a health check for a circuit breaker service
func (rhc *ReliabilityHealthChecker) CreateCircuitBreakerHealthCheck(serviceName string) *HealthCheck {
	return &HealthCheck{
		Name:        fmt.Sprintf("circuit_breaker_%s", serviceName),
		Description: fmt.Sprintf("Circuit breaker health for %s service", serviceName),
		Critical:    false,
		Timeout:     time.Second * 2,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			service, exists := rhc.reliabilityManager.Get(serviceName)
			if !exists {
				return StatusUnknown, fmt.Errorf("reliability service %s not found", serviceName)
			}

			if service.IsHealthy() {
				return StatusHealthy, nil
			}

			// Get detailed stats for more information
			stats := service.GetStats()
			if stats.CircuitBreakerStats.State.String() == "OPEN" {
				return StatusDegraded, fmt.Errorf("circuit breaker is open (failures: %d)",
					stats.CircuitBreakerStats.FailureCount)
			}

			return StatusDegraded, fmt.Errorf("service is not healthy")
		},
	}
}

// CreateReliabilityOverviewHealthCheck creates a health check that provides an overview of all reliability services
func (rhc *ReliabilityHealthChecker) CreateReliabilityOverviewHealthCheck() *HealthCheck {
	return &HealthCheck{
		Name:        "reliability_overview",
		Description: "Overall reliability services health status",
		Critical:    false,
		Timeout:     time.Second * 5,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			healthyServices := rhc.reliabilityManager.GetHealthyServices()
			unhealthyServices := rhc.reliabilityManager.GetUnhealthyServices()

			total := len(healthyServices) + len(unhealthyServices)
			if total == 0 {
				return StatusHealthy, nil // No services registered yet
			}

			details := map[string]interface{}{
				"healthy_services":   healthyServices,
				"unhealthy_services": unhealthyServices,
				"total_services":     total,
				"healthy_count":      len(healthyServices),
				"unhealthy_count":    len(unhealthyServices),
			}

			if len(unhealthyServices) == 0 {
				return StatusHealthy, nil
			}

			// If more than half are unhealthy, consider it unhealthy
			if len(unhealthyServices) > len(healthyServices) {
				return StatusUnhealthy, fmt.Errorf("majority of reliability services are unhealthy: %v", details)
			}

			// Some services are unhealthy but not majority
			return StatusDegraded, fmt.Errorf("some reliability services are unhealthy: %v", details)
		},
	}
}

// CreateRetryStatsHealthCheck creates a health check based on retry statistics
func (rhc *ReliabilityHealthChecker) CreateRetryStatsHealthCheck(serviceName string, maxFailureRate float64) *HealthCheck {
	return &HealthCheck{
		Name:        fmt.Sprintf("retry_stats_%s", serviceName),
		Description: fmt.Sprintf("Retry statistics health for %s service", serviceName),
		Critical:    false,
		Timeout:     time.Second * 2,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			service, exists := rhc.reliabilityManager.Get(serviceName)
			if !exists {
				return StatusUnknown, fmt.Errorf("reliability service %s not found", serviceName)
			}

			stats := service.GetStats()
			retryStats := stats.RetryStats

			// If no attempts have been made, consider it healthy
			totalAttempts := retryStats.SuccessfulRetries + retryStats.FailedRetries
			if totalAttempts == 0 {
				return StatusHealthy, nil
			}

			failureRate := float64(retryStats.FailedRetries) / float64(totalAttempts)

			details := map[string]interface{}{
				"successful_retries": retryStats.SuccessfulRetries,
				"failed_retries":    retryStats.FailedRetries,
				"failure_rate":      failureRate,
				"max_failure_rate":  maxFailureRate,
				"last_error":        retryStats.LastError,
			}

			if failureRate > maxFailureRate {
				return StatusDegraded, fmt.Errorf("high retry failure rate: %.2f%% > %.2f%%: %v",
					failureRate*100, maxFailureRate*100, details)
			}

			return StatusHealthy, nil
		},
	}
}

// RegisterAllReliabilityHealthChecks registers health checks for all known reliability services
func (rhc *ReliabilityHealthChecker) RegisterAllReliabilityHealthChecks(healthChecker *HealthChecker) error {
	// Register the overview health check
	overviewCheck := rhc.CreateReliabilityOverviewHealthCheck()
	if err := healthChecker.RegisterCheck(overviewCheck); err != nil {
		return fmt.Errorf("failed to register reliability overview health check: %w", err)
	}

	// Get all reliability service stats to register individual checks
	allStats := rhc.reliabilityManager.GetAllStats()
	for serviceName := range allStats {
		// Register circuit breaker health check for each service
		cbCheck := rhc.CreateCircuitBreakerHealthCheck(serviceName)
		if err := healthChecker.RegisterCheck(cbCheck); err != nil {
			return fmt.Errorf("failed to register circuit breaker health check for %s: %w", serviceName, err)
		}

		// Register retry stats health check with 50% max failure rate
		retryCheck := rhc.CreateRetryStatsHealthCheck(serviceName, 0.5)
		if err := healthChecker.RegisterCheck(retryCheck); err != nil {
			return fmt.Errorf("failed to register retry stats health check for %s: %w", serviceName, err)
		}
	}

	return nil
}

// CryptoReliabilityHealthChecker provides health checks specifically for crypto reliability services
type CryptoReliabilityHealthChecker struct {
	cryptoReliabilityManager *reliability.CryptoReliabilityManager
}

// NewCryptoReliabilityHealthChecker creates a new crypto reliability health checker
func NewCryptoReliabilityHealthChecker(cryptoManager *reliability.CryptoReliabilityManager) *CryptoReliabilityHealthChecker {
	return &CryptoReliabilityHealthChecker{
		cryptoReliabilityManager: cryptoManager,
	}
}

// CreateKMSOperationsHealthCheck creates a health check for KMS operations
func (crhc *CryptoReliabilityHealthChecker) CreateKMSOperationsHealthCheck() *HealthCheck {
	return &HealthCheck{
		Name:        "kms_operations_health",
		Description: "Health of KMS operations reliability",
		Critical:    true, // KMS is critical for encryption operations
		Timeout:     time.Second * 5,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			allStats := crhc.cryptoReliabilityManager.GetAllStats()

			kmsServices := make([]string, 0)
			unhealthyKMS := make([]string, 0)

			for serviceName := range allStats {
				if len(serviceName) > 4 && serviceName[:4] == "kms_" {
					kmsServices = append(kmsServices, serviceName)
					if !crhc.cryptoReliabilityManager.IsOperationHealthy("kms", serviceName[4:]) {
						unhealthyKMS = append(unhealthyKMS, serviceName)
					}
				}
			}

			if len(kmsServices) == 0 {
				return StatusHealthy, nil // No KMS services registered yet
			}

			details := map[string]interface{}{
				"total_kms_services":     len(kmsServices),
				"healthy_kms_services":   len(kmsServices) - len(unhealthyKMS),
				"unhealthy_kms_services": unhealthyKMS,
			}

			if len(unhealthyKMS) == 0 {
				return StatusHealthy, nil
			}

			// If any KMS service is unhealthy, it's critical
			if len(unhealthyKMS) > 0 {
				return StatusUnhealthy, fmt.Errorf("critical KMS operations are unhealthy: %v", details)
			}

			return StatusHealthy, nil
		},
	}
}

// CreateDatabaseOperationsHealthCheck creates a health check for database operations
func (crhc *CryptoReliabilityHealthChecker) CreateDatabaseOperationsHealthCheck() *HealthCheck {
	return &HealthCheck{
		Name:        "database_operations_health",
		Description: "Health of database operations reliability",
		Critical:    true, // Database operations are critical
		Timeout:     time.Second * 5,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			allStats := crhc.cryptoReliabilityManager.GetAllStats()

			dbServices := make([]string, 0)
			unhealthyDB := make([]string, 0)

			for serviceName := range allStats {
				if len(serviceName) > 3 && serviceName[:3] == "db_" {
					dbServices = append(dbServices, serviceName)
					if !crhc.cryptoReliabilityManager.IsOperationHealthy("db", serviceName[3:]) {
						unhealthyDB = append(unhealthyDB, serviceName)
					}
				}
			}

			if len(dbServices) == 0 {
				return StatusHealthy, nil // No database services registered yet
			}

			details := map[string]interface{}{
				"total_db_services":     len(dbServices),
				"healthy_db_services":   len(dbServices) - len(unhealthyDB),
				"unhealthy_db_services": unhealthyDB,
			}

			if len(unhealthyDB) == 0 {
				return StatusHealthy, nil
			}

			// If more than half are unhealthy, consider it unhealthy
			if len(unhealthyDB) > len(dbServices)/2 {
				return StatusUnhealthy, fmt.Errorf("majority of database operations are unhealthy: %v", details)
			}

			return StatusDegraded, fmt.Errorf("some database operations are unhealthy: %v", details)
		},
	}
}

// CreateNetworkOperationsHealthCheck creates a health check for network operations
func (crhc *CryptoReliabilityHealthChecker) CreateNetworkOperationsHealthCheck() *HealthCheck {
	return &HealthCheck{
		Name:        "network_operations_health",
		Description: "Health of network operations reliability",
		Critical:    false, // Network operations are less critical
		Timeout:     time.Second * 5,
		CheckFunc: func(ctx context.Context) (HealthStatus, error) {
			allStats := crhc.cryptoReliabilityManager.GetAllStats()

			networkServices := make([]string, 0)
			unhealthyNetwork := make([]string, 0)

			for serviceName := range allStats {
				if len(serviceName) > 8 && serviceName[:8] == "network_" {
					networkServices = append(networkServices, serviceName)
					if !crhc.cryptoReliabilityManager.IsOperationHealthy("network", serviceName[8:]) {
						unhealthyNetwork = append(unhealthyNetwork, serviceName)
					}
				}
			}

			if len(networkServices) == 0 {
				return StatusHealthy, nil // No network services registered yet
			}

			details := map[string]interface{}{
				"total_network_services":     len(networkServices),
				"healthy_network_services":   len(networkServices) - len(unhealthyNetwork),
				"unhealthy_network_services": unhealthyNetwork,
			}

			if len(unhealthyNetwork) == 0 {
				return StatusHealthy, nil
			}

			// Network operations are less critical, so only degrade if many are unhealthy
			if len(unhealthyNetwork) > len(networkServices)*3/4 {
				return StatusDegraded, fmt.Errorf("most network operations are unhealthy: %v", details)
			}

			return StatusHealthy, nil // Some network issues are acceptable
		},
	}
}

// RegisterAllCryptoReliabilityHealthChecks registers health checks for crypto reliability services
func (crhc *CryptoReliabilityHealthChecker) RegisterAllCryptoReliabilityHealthChecks(healthChecker *HealthChecker) error {
	// Register KMS operations health check
	kmsCheck := crhc.CreateKMSOperationsHealthCheck()
	if err := healthChecker.RegisterCheck(kmsCheck); err != nil {
		return fmt.Errorf("failed to register KMS operations health check: %w", err)
	}

	// Register database operations health check
	dbCheck := crhc.CreateDatabaseOperationsHealthCheck()
	if err := healthChecker.RegisterCheck(dbCheck); err != nil {
		return fmt.Errorf("failed to register database operations health check: %w", err)
	}

	// Register network operations health check
	networkCheck := crhc.CreateNetworkOperationsHealthCheck()
	if err := healthChecker.RegisterCheck(networkCheck); err != nil {
		return fmt.Errorf("failed to register network operations health check: %w", err)
	}

	return nil
}

// Helper functions for integrating with global health checker

// RegisterReliabilityHealthChecks registers reliability health checks with the global health checker
func RegisterReliabilityHealthChecks(reliabilityManager *reliability.ReliabilityManager) error {
	healthChecker := GetGlobalHealthChecker()
	reliabilityHealthChecker := NewReliabilityHealthChecker(reliabilityManager)
	return reliabilityHealthChecker.RegisterAllReliabilityHealthChecks(healthChecker)
}

// RegisterCryptoReliabilityHealthChecks registers crypto reliability health checks with the global health checker
func RegisterCryptoReliabilityHealthChecks(cryptoManager *reliability.CryptoReliabilityManager) error {
	healthChecker := GetGlobalHealthChecker()
	cryptoReliabilityHealthChecker := NewCryptoReliabilityHealthChecker(cryptoManager)
	return cryptoReliabilityHealthChecker.RegisterAllCryptoReliabilityHealthChecks(healthChecker)
}