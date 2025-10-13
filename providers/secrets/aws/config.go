package aws

import "github.com/aws/aws-sdk-go-v2/aws"

// Config holds configuration for AWS Secrets Manager service.
type Config struct {
	// Region is the AWS region (e.g., "us-east-1")
	// If empty, uses AWS_REGION environment variable or AWS config file
	Region string

	// AWSConfig is an optional pre-configured AWS config
	// If provided, Region is ignored
	AWSConfig *aws.Config
}
