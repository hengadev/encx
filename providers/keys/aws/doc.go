// Package aws provides AWS Key Management Service (KMS) integration for encx.
//
// This package implements the encx.KeyManagementService interface using AWS KMS,
// enabling secure encryption and decryption of Data Encryption Keys (DEKs) using
// AWS-managed Key Encryption Keys (KEKs).
//
// # Features
//
//   - KEK management via AWS KMS
//   - DEK encryption and decryption operations
//   - Support for key aliases and ARNs
//   - Automatic key creation and rotation support
//   - Base64 encoding for storage compatibility
//
// # Basic Usage
//
//	import (
//	    "context"
//	    "github.com/hengadev/encx"
//	    awskms "github.com/hengadev/encx/providers/keys/aws"
//	)
//
//	// Initialize AWS KMS service
//	kms, err := awskms.NewKMSService(ctx, awskms.Config{
//	    Region: "us-east-1",
//	})
//	if err != nil {
//	    // handle error
//	}
//
//	// Use with encx.NewCrypto() along with a SecretManagementService
//	crypto, err := encx.NewCrypto(ctx, kms, secretsStore, encx.Config{
//	    KEKAlias: "alias/my-app-kek",
//	    PepperAlias: "my-app-pepper",
//	})
//
// # Configuration
//
// The Config struct supports multiple configuration options:
//
//	// Option 1: Specify region explicitly
//	cfg := awskms.Config{Region: "us-east-1"}
//
//	// Option 2: Use default AWS configuration (from env vars or AWS config file)
//	cfg := awskms.Config{}
//
//	// Option 3: Provide custom AWS config
//	awsCfg, _ := config.LoadDefaultConfig(ctx)
//	cfg := awskms.Config{AWSConfig: &awsCfg}
//
// # IAM Permissions
//
// The IAM role or user needs the following KMS permissions:
//
//	{
//	    "Version": "2012-10-17",
//	    "Statement": [
//	        {
//	            "Effect": "Allow",
//	            "Action": [
//	                "kms:Encrypt",
//	                "kms:Decrypt",
//	                "kms:DescribeKey",
//	                "kms:CreateKey"
//	            ],
//	            "Resource": "arn:aws:kms:region:account-id:key/*"
//	        }
//	    ]
//	}
//
// # Error Handling
//
// Operations return wrapped errors from the encx package:
//
//   - encx.ErrKMSUnavailable: AWS KMS service is unavailable or key not found
//   - encx.ErrEncryptionFailed: Encryption operation failed
//   - encx.ErrDecryptionFailed: Decryption operation failed
//   - encx.ErrInvalidConfiguration: Invalid configuration (e.g., empty alias)
//
// # Mix-and-Match Providers
//
// AWS KMS can be combined with different SecretManagementService implementations:
//
//	// AWS KMS + AWS Secrets Manager
//	import (
//	    awskms "github.com/hengadev/encx/providers/keys/aws"
//	    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
//	)
//
//	// AWS KMS + HashiCorp Vault KV
//	import (
//	    awskms "github.com/hengadev/encx/providers/keys/aws"
//	    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
//	)
//
// # Key Naming Conventions
//
// AWS KMS uses aliases with the "alias/" prefix. The package automatically
// adds this prefix if not provided:
//
//	kms.GetKeyID(ctx, "my-key")        // Becomes "alias/my-key"
//	kms.GetKeyID(ctx, "alias/my-key")  // Used as-is
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
// For more information, see https://github.com/hengadev/encx
package aws
