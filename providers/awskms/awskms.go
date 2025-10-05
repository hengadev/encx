// Package awskms provides AWS Key Management Service (KMS) integration for encx.
//
// This provider implements the KeyManagementService interface using AWS KMS
// for secure key encryption operations (KEK management).
package awskms

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hengadev/encx"
)

// kmsClient interface for AWS KMS operations (allows mocking)
type kmsClient interface {
	DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
	CreateKey(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error)
	Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

// KMSService implements encx.KeyManagementService using AWS KMS.
type KMSService struct {
	client kmsClient
	region string
}

// Config holds configuration for AWS KMS service.
type Config struct {
	// Region is the AWS region (e.g., "us-east-1")
	// If empty, uses AWS_REGION environment variable or AWS config file
	Region string

	// AWSConfig is an optional pre-configured AWS config
	// If provided, Region is ignored
	AWSConfig *aws.Config
}

// New creates a new AWS KMS service instance.
//
// Usage:
//
//	// Using default AWS configuration
//	kmsService, err := awskms.New(ctx, awskms.Config{})
//
//	// With specific region
//	kmsService, err := awskms.New(ctx, awskms.Config{Region: "us-east-1"})
//
//	// With custom AWS config
//	awsCfg, _ := config.LoadDefaultConfig(ctx)
//	kmsService, err := awskms.New(ctx, awskms.Config{AWSConfig: &awsCfg})
func New(ctx context.Context, cfg Config) (*KMSService, error) {
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
			return nil, fmt.Errorf("%w: failed to load AWS config: %w", encx.ErrKMSUnavailable, err)
		}
	}

	return &KMSService{
		client: kms.NewFromConfig(awsConfig),
		region: awsConfig.Region,
	}, nil
}

// GetKeyID returns the KMS key ID for a given alias.
//
// In AWS KMS, aliases are in the format "alias/your-key-name".
// If the alias doesn't include the "alias/" prefix, it's automatically added.
//
// Returns the key ARN or key ID that the alias points to.
func (k *KMSService) GetKeyID(ctx context.Context, alias string) (string, error) {
	// Ensure alias has proper format
	if len(alias) == 0 {
		return "", fmt.Errorf("%w: alias cannot be empty", encx.ErrInvalidConfiguration)
	}

	// Add "alias/" prefix if not present
	aliasName := alias
	if len(alias) < 6 || alias[:6] != "alias/" {
		aliasName = "alias/" + alias
	}

	// Describe the key to get its ID
	input := &kms.DescribeKeyInput{
		KeyId: aws.String(aliasName),
	}

	result, err := k.client.DescribeKey(ctx, input)
	if err != nil {
		return "", fmt.Errorf("%w: failed to describe KMS key %s: %w", encx.ErrKMSUnavailable, aliasName, err)
	}

	if result.KeyMetadata == nil || result.KeyMetadata.KeyId == nil {
		return "", fmt.Errorf("%w: no key metadata returned for alias %s", encx.ErrKMSUnavailable, aliasName)
	}

	return *result.KeyMetadata.KeyId, nil
}

// CreateKey creates a new KMS key with the given description.
//
// The description is used as the key's description in AWS KMS.
// Returns the key ID of the newly created key.
//
// Note: This creates a symmetric encryption key suitable for data encryption.
// You'll typically want to create an alias for the key separately using AWS CLI or Console.
func (k *KMSService) CreateKey(ctx context.Context, description string) (string, error) {
	input := &kms.CreateKeyInput{
		Description: aws.String(description),
		KeyUsage:    types.KeyUsageTypeEncryptDecrypt,
		KeySpec:     types.KeySpecSymmetricDefault,
		MultiRegion: aws.Bool(false),
	}

	result, err := k.client.CreateKey(ctx, input)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create KMS key: %w", encx.ErrKMSUnavailable, err)
	}

	if result.KeyMetadata == nil || result.KeyMetadata.KeyId == nil {
		return "", fmt.Errorf("%w: no key metadata returned after creation", encx.ErrKMSUnavailable)
	}

	return *result.KeyMetadata.KeyId, nil
}

// EncryptDEK encrypts a Data Encryption Key (DEK) using the specified KMS key.
//
// The keyID can be:
//   - Key ID: "1234abcd-12ab-34cd-56ef-1234567890ab"
//   - Key ARN: "arn:aws:kms:us-east-1:123456789012:key/1234abcd-12ab-34cd-56ef-1234567890ab"
//   - Alias name: "alias/my-key"
//   - Alias ARN: "arn:aws:kms:us-east-1:123456789012:alias/my-key"
//
// Returns the encrypted DEK as a base64-encoded ciphertext blob.
func (k *KMSService) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("%w: plaintext cannot be empty", encx.ErrEncryptionFailed)
	}

	input := &kms.EncryptInput{
		KeyId:     aws.String(keyID),
		Plaintext: plaintext,
	}

	result, err := k.client.Encrypt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encrypt DEK with KMS key %s: %w", encx.ErrEncryptionFailed, keyID, err)
	}

	if result.CiphertextBlob == nil {
		return nil, fmt.Errorf("%w: no ciphertext returned from KMS", encx.ErrEncryptionFailed)
	}

	// AWS KMS returns raw bytes, but we encode to base64 for storage compatibility
	encoded := base64.StdEncoding.EncodeToString(result.CiphertextBlob)
	return []byte(encoded), nil
}

// DecryptDEK decrypts a Data Encryption Key (DEK) that was encrypted by AWS KMS.
//
// The keyID parameter is optional and can be empty - AWS KMS will automatically
// use the correct key based on the ciphertext metadata.
//
// The ciphertext should be base64-encoded (as returned by EncryptDEK).
// Returns the decrypted DEK in plaintext.
func (k *KMSService) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("%w: ciphertext cannot be empty", encx.ErrDecryptionFailed)
	}

	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode ciphertext: %w", encx.ErrDecryptionFailed, err)
	}

	input := &kms.DecryptInput{
		CiphertextBlob: decoded,
	}

	// If keyID is provided, include it (though AWS KMS doesn't require it)
	if keyID != "" {
		input.KeyId = aws.String(keyID)
	}

	result, err := k.client.Decrypt(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decrypt DEK: %w", encx.ErrDecryptionFailed, err)
	}

	if result.Plaintext == nil {
		return nil, fmt.Errorf("%w: no plaintext returned from KMS", encx.ErrDecryptionFailed)
	}

	return result.Plaintext, nil
}

// Region returns the AWS region this KMS service is configured for.
func (k *KMSService) Region() string {
	return k.region
}
