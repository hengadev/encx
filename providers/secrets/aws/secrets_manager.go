// Package aws provides AWS Secrets Manager integration for encx.
//
// This provider implements the SecretManagementService interface using AWS Secrets Manager
// for secure pepper storage.
package aws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/hengadev/encx"
)

// secretsManagerClient interface for AWS Secrets Manager operations (allows mocking)
type secretsManagerClient interface {
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
}

// SecretsManagerStore implements encx.SecretManagementService using AWS Secrets Manager.
//
// This service stores peppers (secret values) in AWS Secrets Manager for secure,
// centralized secret storage with audit logging and automatic replication.
type SecretsManagerStore struct {
	client secretsManagerClient
	region string
}

// NewSecretsManagerStore creates a new AWS Secrets Manager store instance.
//
// Usage:
//
//	// Using default AWS configuration
//	store, err := aws.NewSecretsManagerStore(ctx, aws.Config{})
//
//	// With specific region
//	store, err := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})
//
//	// With custom AWS config
//	awsCfg, _ := config.LoadDefaultConfig(ctx)
//	store, err := aws.NewSecretsManagerStore(ctx, aws.Config{AWSConfig: &awsCfg})
func NewSecretsManagerStore(ctx context.Context, cfg Config) (*SecretsManagerStore, error) {
	var awsConfig aws.Config
	var err error

	if cfg.AWSConfig != nil {
		awsConfig = *cfg.AWSConfig
	} else {
		// Load default AWS configuration
		opts := []func(*config.LoadOptions) error{}
		if cfg.Region != "" {
			opts = append(opts, config.WithRegion(cfg.Region))
		}

		awsConfig, err = config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to load AWS config: %w", encx.ErrSecretStorageUnavailable, err)
		}
	}

	return &SecretsManagerStore{
		client: secretsmanager.NewFromConfig(awsConfig),
		region: awsConfig.Region,
	}, nil
}

// GetStoragePath returns the AWS Secrets Manager path for a given alias.
//
// Path format: "encx/{alias}/pepper"
//
// Examples:
//   - alias "my-service" → "encx/my-service/pepper"
//   - alias "payment-api" → "encx/payment-api/pepper"
func (s *SecretsManagerStore) GetStoragePath(alias string) string {
	return fmt.Sprintf(encx.AWSPepperPathTemplate, alias)
}

// StorePepper stores a pepper in AWS Secrets Manager.
//
// If a pepper already exists for this alias, it will be updated.
// The pepper must be exactly 32 bytes (encx.PepperLength).
//
// Example:
//
//	pepper := []byte("your-32-byte-pepper-secret-here!")
//	err := store.StorePepper(ctx, "my-service", pepper)
func (s *SecretsManagerStore) StorePepper(ctx context.Context, alias string, pepper []byte) error {
	if len(pepper) != encx.PepperLength {
		return fmt.Errorf("%w: pepper must be exactly %d bytes, got %d",
			encx.ErrInvalidConfiguration, encx.PepperLength, len(pepper))
	}

	secretName := s.GetStoragePath(alias)

	// Encode pepper to base64 for storage
	pepperBase64 := base64.StdEncoding.EncodeToString(pepper)

	// Check if secret already exists
	exists, err := s.PepperExists(ctx, alias)
	if err != nil {
		return fmt.Errorf("%w: failed to check if pepper exists: %w", encx.ErrSecretStorageUnavailable, err)
	}

	if exists {
		// Update existing secret
		_, err = s.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(secretName),
			SecretString: aws.String(pepperBase64),
		})
		if err != nil {
			return fmt.Errorf("%w: failed to update pepper in Secrets Manager: %w",
				encx.ErrSecretStorageUnavailable, err)
		}
	} else {
		// Create new secret
		_, err = s.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(secretName),
			Description:  aws.String(fmt.Sprintf("ENCX pepper for %s", alias)),
			SecretString: aws.String(pepperBase64),
		})
		if err != nil {
			return fmt.Errorf("%w: failed to create pepper in Secrets Manager: %w",
				encx.ErrSecretStorageUnavailable, err)
		}
	}

	return nil
}

// GetPepper retrieves a pepper from AWS Secrets Manager.
//
// Returns an error if the pepper doesn't exist or has invalid length.
//
// Example:
//
//	pepper, err := store.GetPepper(ctx, "my-service")
//	if err != nil {
//	    log.Fatalf("Failed to get pepper: %v", err)
//	}
func (s *SecretsManagerStore) GetPepper(ctx context.Context, alias string) ([]byte, error) {
	secretName := s.GetStoragePath(alias)

	result, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pepper from Secrets Manager: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	if result.SecretString == nil {
		return nil, fmt.Errorf("%w: pepper not found for alias: %s",
			encx.ErrSecretStorageUnavailable, alias)
	}

	// Decode from base64
	pepper, err := base64.StdEncoding.DecodeString(*result.SecretString)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode pepper: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	// Validate pepper length
	if len(pepper) != encx.PepperLength {
		return nil, fmt.Errorf("%w: invalid pepper length: expected %d bytes, got %d",
			encx.ErrSecretStorageUnavailable, encx.PepperLength, len(pepper))
	}

	return pepper, nil
}

// PepperExists checks if a pepper exists in AWS Secrets Manager.
//
// Returns true if the pepper exists, false if it doesn't.
// Returns an error only for actual failures (not for "secret not found").
//
// Example:
//
//	exists, err := store.PepperExists(ctx, "my-service")
//	if err != nil {
//	    log.Fatalf("Failed to check pepper: %v", err)
//	}
//	if !exists {
//	    // Generate and store new pepper
//	}
func (s *SecretsManagerStore) PepperExists(ctx context.Context, alias string) (bool, error) {
	secretName := s.GetStoragePath(alias)

	_, err := s.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})

	if err != nil {
		// Check if error is "ResourceNotFoundException" - secret doesn't exist
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			// Secret doesn't exist - this is not an error
			return false, nil
		}
		// Some other error occurred
		return false, fmt.Errorf("%w: failed to check if pepper exists: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	return true, nil
}

// Region returns the AWS region this Secrets Manager store is configured for.
func (s *SecretsManagerStore) Region() string {
	return s.region
}
