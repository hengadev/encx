// Package aws provides AWS Secrets Manager integration for encx.
//
// This package implements the encx.SecretManagementService interface using AWS Secrets Manager,
// enabling secure storage and retrieval of peppers (secret values) used in Argon2id password
// hashing operations.
//
// # Features
//
//   - Secure pepper storage in AWS Secrets Manager
//   - Automatic secret creation and updates
//   - Base64 encoding for storage compatibility
//   - IAM-based access control
//   - CloudTrail audit logging
//   - Automatic replication (multi-region secrets)
//   - Secret versioning and rotation support
//
// # Basic Usage
//
//	import (
//	    "context"
//	    "github.com/hengadev/encx"
//	    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
//	)
//
//	// Initialize AWS Secrets Manager store
//	secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{
//	    Region: "us-east-1",
//	})
//	if err != nil {
//	    // handle error
//	}
//
//	// Use with encx.NewCrypto() along with a KeyManagementService
//	crypto, err := encx.NewCrypto(ctx, kmsService, secrets, encx.Config{
//	    KEKAlias: "alias/my-app-kek",
//	    PepperAlias: "my-app-pepper",
//	})
//
// # Configuration
//
// The Config struct supports multiple configuration options:
//
//	// Option 1: Specify region explicitly
//	cfg := awssecrets.Config{Region: "us-east-1"}
//
//	// Option 2: Use default AWS configuration (from env vars or AWS config file)
//	cfg := awssecrets.Config{}
//
//	// Option 3: Provide custom AWS config
//	awsCfg, _ := config.LoadDefaultConfig(ctx)
//	cfg := awssecrets.Config{AWSConfig: &awsCfg}
//
// # Pepper Storage
//
// Peppers are stored in AWS Secrets Manager using the path format:
//
//	encx/{alias}/pepper
//
// For example, if your PepperAlias is "payment-api", the secret will be stored at:
//
//	encx/payment-api/pepper
//
// # IAM Permissions
//
// The IAM role or user needs the following Secrets Manager permissions:
//
//	{
//	    "Version": "2012-10-17",
//	    "Statement": [
//	        {
//	            "Effect": "Allow",
//	            "Action": [
//	                "secretsmanager:GetSecretValue",
//	                "secretsmanager:CreateSecret",
//	                "secretsmanager:PutSecretValue",
//	                "secretsmanager:DescribeSecret"
//	            ],
//	            "Resource": "arn:aws:secretsmanager:region:account-id:secret:encx/*"
//	        }
//	    ]
//	}
//
// # Error Handling
//
// Operations return wrapped errors from the encx package:
//
//   - encx.ErrSecretStorageUnavailable: Secrets Manager is unavailable or secret not found
//   - encx.ErrInvalidConfiguration: Invalid pepper length or configuration
//
// # Mix-and-Match Providers
//
// AWS Secrets Manager can be combined with different KeyManagementService implementations:
//
//	// AWS Secrets Manager + AWS KMS
//	import (
//	    awskms "github.com/hengadev/encx/providers/keys/aws"
//	    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
//	)
//
//	// AWS Secrets Manager + HashiCorp Vault Transit
//	import (
//	    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
//	    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
//	)
//
// # Automatic Pepper Management
//
// When using encx.NewCrypto(), peppers are automatically managed:
//
//   - If a pepper doesn't exist, it's automatically generated and stored
//   - If a pepper exists, it's retrieved and used
//   - Peppers are exactly 32 bytes (encx.PepperLength)
//   - Peppers are base64-encoded before storage
//
// # Testing
//
// For testing without AWS dependencies, use encx.NewTestCrypto():
//
//	func TestMyCode(t *testing.T) {
//	    crypto, _ := encx.NewTestCrypto(t)
//	    // Test code using crypto...
//	}
//
// # Secret Rotation
//
// AWS Secrets Manager supports automatic secret rotation. However, for encx peppers,
// rotation is NOT recommended as it would invalidate all existing password hashes.
//
// Peppers should be:
//   - Generated once during initial setup
//   - Stored securely and never rotated
//   - Backed up for disaster recovery
//
// # Multi-Region Secrets
//
// For high availability, consider using multi-region secrets:
//
//	aws secretsmanager replicate-secret-to-regions \
//	    --secret-id encx/my-app-pepper \
//	    --add-replica-regions Region=us-west-2
//
// # Cost Considerations
//
// AWS Secrets Manager charges:
//   - $0.40 per secret per month
//   - $0.05 per 10,000 API calls
//
// For a single pepper, monthly cost is approximately $0.40 plus negligible API costs.
//
// For more information, see https://github.com/hengadev/encx
package aws
