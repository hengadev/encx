// Package aws provides AWS Key Management Service (KMS) and Secrets Manager integration for the encx library.
//
// This package implements two encx interfaces:
//  - encx.KeyManagementService: Cryptographic operations using AWS KMS
//  - encx.SecretManagementService: Secret storage using AWS Secrets Manager
//
// # Overview
//
// AWS KMS is a managed service for cryptographic key operations, while AWS Secrets Manager
// provides secure secret storage with automatic rotation and audit logging. This provider
// allows encx to use both services for complete encryption key management:
//
//  - KMS encrypts and decrypts Data Encryption Keys (DEKs)
//  - Secrets Manager stores the pepper (secret value) securely
//
// # Setup
//
// Before using this provider, you need:
//
//  1. An AWS account with KMS and Secrets Manager access
//  2. AWS credentials configured (via environment variables, AWS config file, or IAM role)
//  3. A KMS key created in your AWS account
//  4. (Optional) Secrets Manager configured for pepper storage
//
// # IAM Permissions Required
//
// The AWS credentials must have the following permissions:
//
//	{
//	  "Version": "2012-10-17",
//	  "Statement": [
//	    {
//	      "Effect": "Allow",
//	      "Action": [
//	        "kms:Encrypt",
//	        "kms:Decrypt",
//	        "kms:DescribeKey",
//	        "kms:CreateKey",          // Optional: only if creating keys programmatically
//	        "kms:CreateAlias"         // Optional: only if creating aliases
//	      ],
//	      "Resource": "arn:aws:kms:REGION:ACCOUNT:key/KEY-ID"
//	    },
//	    {
//	      "Effect": "Allow",
//	      "Action": [
//	        "secretsmanager:CreateSecret",
//	        "secretsmanager:GetSecretValue",
//	        "secretsmanager:PutSecretValue",
//	        "secretsmanager:DescribeSecret"
//	      ],
//	      "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:encx/*"
//	    }
//	  ]
//	}
//
// # Usage Example
//
// Complete setup with both KMS and Secrets Manager:
//
//	import (
//	    "context"
//	    "github.com/hengadev/encx"
//	    "github.com/hengadev/encx/providers/aws"
//	)
//
//	func main() {
//	    ctx := context.Background()
//
//	    // Create AWS KMS service for cryptographic operations
//	    kms, err := aws.NewKMSService(ctx, aws.Config{
//	        Region: "us-east-1",
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Create AWS Secrets Manager service for pepper storage
//	    secrets, err := aws.NewSecretsManagerStore(ctx, aws.Config{
//	        Region: "us-east-1",
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Create encx crypto service with both AWS services
//	    crypto, err := encx.NewCrypto(ctx, kms, secrets, encx.Config{
//	        KEKAlias:    "my-encryption-key",  // Your KMS key alias
//	        PepperAlias: "my-service",         // Service identifier for pepper
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Use crypto service for encryption
//	    dek, _ := crypto.GenerateDEK()
//	    ciphertext, _ := crypto.EncryptData(ctx, plaintext, dek)
//	}
//
// # AWS Credentials
//
// Both services use the standard AWS SDK credential chain:
//
//  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//  2. Shared credentials file (~/.aws/credentials)
//  3. IAM role for Amazon EC2, ECS, or Lambda
//
// The same AWS credentials and region can be used for both services.
//
// # Key Aliases and Pepper Paths
//
// KMS Key Aliases (recommended over key IDs):
//
//	// Create an alias in AWS CLI:
//	aws kms create-alias --alias-name alias/my-encryption-key --target-key-id 1234abcd-12ab-34cd-56ef-1234567890ab
//
//	// Use in encx Config:
//	cfg := encx.Config{
//	    KEKAlias: "my-encryption-key",  // No "alias/" prefix needed in config
//	    ...
//	}
//
// Secrets Manager Pepper Paths:
//
// Peppers are automatically stored at: "encx/{pepperAlias}/pepper"
//
//	// If PepperAlias is "my-service", pepper is stored at:
//	// "encx/my-service/pepper"
//
// # Performance Considerations
//
// - KMS operations involve network calls to AWS, adding latency
// - KMS has rate limits (varies by region, typically 1200-5500 requests/sec)
// - Secrets Manager reads are cached by AWS SDK (5 minute default TTL)
// - Consider caching DEKs when encrypting multiple records
// - Use connection pooling (handled automatically by AWS SDK)
//
// # Cost Considerations
//
// AWS KMS charges:
//  - Key storage: ~$1/month per key
//  - API requests: $0.03 per 10,000 requests
//
// AWS Secrets Manager charges:
//  - Secret storage: ~$0.40/month per secret
//  - API requests: $0.05 per 10,000 requests
//
// For high-volume applications, DEK caching can significantly reduce KMS costs.
// Pepper retrieval is typically a one-time operation per application instance.
//
// # Error Handling
//
// All methods return encx-compatible errors:
//
//  - encx.ErrKMSUnavailable: AWS KMS service is unavailable or inaccessible
//  - encx.ErrSecretStorageUnavailable: AWS Secrets Manager is unavailable or inaccessible
//  - encx.ErrEncryptionFailed: Encryption operation failed
//  - encx.ErrDecryptionFailed: Decryption operation failed
//  - encx.ErrInvalidConfiguration: Invalid configuration provided
//
// # Multi-Region Support
//
// For multi-region applications, create separate service instances per region:
//
//	kmsUSEast, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
//	secretsUSEast, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})
//
//	kmsEUWest, _ := aws.NewKMSService(ctx, aws.Config{Region: "eu-west-1"})
//	secretsEUWest, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "eu-west-1"})
//
// Or use AWS KMS multi-region keys and Secrets Manager replication for automatic sync.
//
// # Separation of Concerns
//
// This package follows the Single Responsibility Principle by separating:
//
//  - Cryptographic operations (KMS) from secret storage (Secrets Manager)
//  - This matches AWS's service architecture and allows independent scaling/monitoring
//  - Both services can use the same AWS credentials and region
//
// For more information, see:
//  - AWS KMS Documentation: https://docs.aws.amazon.com/kms/
//  - AWS Secrets Manager Documentation: https://docs.aws.amazon.com/secretsmanager/
package aws
