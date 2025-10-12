// Package awskms provides AWS Key Management Service (KMS) integration for the encx library.
//
// This package implements the encx KeyManagementService interface using AWS KMS,
// allowing you to use AWS KMS for Key Encryption Key (KEK) operations in your
// encx-based encryption workflows.
//
// # Overview
//
// AWS KMS is a managed service that makes it easy to create and control encryption keys.
// This provider allows encx to use AWS KMS for encrypting and decrypting Data Encryption
// Keys (DEKs), implementing the envelope encryption pattern.
//
// # Setup
//
// Before using this provider, you need:
//
//  1. An AWS account with KMS access
//  2. AWS credentials configured (via environment variables, AWS config file, or IAM role)
//  3. A KMS key created in your AWS account
//
// # IAM Permissions Required
//
// The AWS credentials must have the following KMS permissions:
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
//	    }
//	  ]
//	}
//
// # Usage Example
//
// Basic usage with default AWS configuration:
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
//	    // Create AWS KMS provider
//	    kmsService, err := aws.NewKMSService(ctx, aws.Config{
//	        Region: "us-east-1",  // Optional: uses AWS_REGION if not specified
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Create encx crypto service with AWS KMS
//	    crypto, err := encx.NewCrypto(ctx,
//	        encx.WithKMSService(kmsService),
//	        encx.WithKEKAlias("alias/my-encryption-key"),  // Your KMS key alias
//	        encx.WithPepper(pepper),
//	    )
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
// The provider uses the standard AWS SDK credential chain:
//
//  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//  2. Shared credentials file (~/.aws/credentials)
//  3. IAM role for Amazon EC2, ECS, or Lambda
//
// # Key Aliases
//
// It's recommended to use key aliases instead of key IDs for better maintainability:
//
//	// Create an alias in AWS CLI:
//	aws kms create-alias --alias-name alias/my-encryption-key --target-key-id 1234abcd-12ab-34cd-56ef-1234567890ab
//
//	// Use in encx:
//	crypto, err := encx.NewCrypto(ctx,
//	    encx.WithKMSService(kmsService),
//	    encx.WithKEKAlias("alias/my-encryption-key"),  // Use alias, not key ID
//	    ...
//	)
//
// # Performance Considerations
//
// - KMS operations involve network calls to AWS, adding latency
// - KMS has rate limits (varies by region, typically 1200-5500 requests/sec)
// - Consider caching DEKs when encrypting multiple records
// - Use connection pooling (handled automatically by AWS SDK)
//
// # Cost Considerations
//
// AWS KMS charges per:
//  - Key storage: ~$1/month per key
//  - API requests: $0.03 per 10,000 requests
//
// For high-volume applications, DEK caching can significantly reduce costs.
//
// # Error Handling
//
// All methods return encx-compatible errors:
//
//  - encx.ErrKMSUnavailable: AWS KMS service is unavailable or inaccessible
//  - encx.ErrEncryptionFailed: Encryption operation failed
//  - encx.ErrDecryptionFailed: Decryption operation failed
//  - encx.ErrInvalidConfiguration: Invalid configuration provided
//
// # Multi-Region Support
//
// For multi-region applications, create separate KMS service instances per region:
//
//	kmsUSEast, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
//	kmsEUWest, _ := aws.NewKMSService(ctx, aws.Config{Region: "eu-west-1"})
//
// Or use AWS KMS multi-region keys for automatic key replication.
package aws
