package awskms

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hengadev/encx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock KMS client for testing
type mockKMSClient struct {
	describeKeyFunc func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
	createKeyFunc   func(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error)
	encryptFunc     func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
	decryptFunc     func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

func (m *mockKMSClient) DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
	if m.describeKeyFunc != nil {
		return m.describeKeyFunc(ctx, params, optFns...)
	}
	return &kms.DescribeKeyOutput{}, nil
}

func (m *mockKMSClient) CreateKey(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error) {
	if m.createKeyFunc != nil {
		return m.createKeyFunc(ctx, params, optFns...)
	}
	return &kms.CreateKeyOutput{}, nil
}

func (m *mockKMSClient) Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
	if m.encryptFunc != nil {
		return m.encryptFunc(ctx, params, optFns...)
	}
	return &kms.EncryptOutput{}, nil
}

func (m *mockKMSClient) Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
	if m.decryptFunc != nil {
		return m.decryptFunc(ctx, params, optFns...)
	}
	return &kms.DecryptOutput{}, nil
}

func TestNew(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		cfg       Config
		wantErr   bool
		errMsg    string
		checkFunc func(t *testing.T, svc *KMSService)
	}{
		{
			name: "with region specified",
			cfg: Config{
				Region: "us-east-1",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, svc *KMSService) {
				assert.Equal(t, "us-east-1", svc.region)
				assert.NotNil(t, svc.client)
			},
		},
		{
			name: "with custom AWS config",
			cfg: Config{
				AWSConfig: &aws.Config{
					Region: "eu-west-1",
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, svc *KMSService) {
				assert.Equal(t, "eu-west-1", svc.region)
				assert.NotNil(t, svc.client)
			},
		},
		{
			name:    "with empty config uses defaults",
			cfg:     Config{},
			wantErr: false,
			checkFunc: func(t *testing.T, svc *KMSService) {
				assert.NotNil(t, svc.client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(ctx, tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
				if tt.checkFunc != nil {
					tt.checkFunc(t, svc)
				}
			}
		})
	}
}

func TestGetKeyID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		alias         string
		mockFunc      func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
		wantKeyID     string
		wantErr       bool
		errMsg        string
		checkErrType  error
	}{
		{
			name:  "successful with alias prefix",
			alias: "alias/my-key",
			mockFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
				assert.Equal(t, "alias/my-key", *params.KeyId)
				return &kms.DescribeKeyOutput{
					KeyMetadata: &types.KeyMetadata{
						KeyId: aws.String("1234abcd-12ab-34cd-56ef-1234567890ab"),
					},
				}, nil
			},
			wantKeyID: "1234abcd-12ab-34cd-56ef-1234567890ab",
			wantErr:   false,
		},
		{
			name:  "successful without alias prefix",
			alias: "my-key",
			mockFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
				// Should automatically add "alias/" prefix
				assert.Equal(t, "alias/my-key", *params.KeyId)
				return &kms.DescribeKeyOutput{
					KeyMetadata: &types.KeyMetadata{
						KeyId: aws.String("5678efgh-56ef-78gh-90ij-5678901234ef"),
					},
				}, nil
			},
			wantKeyID: "5678efgh-56ef-78gh-90ij-5678901234ef",
			wantErr:   false,
		},
		{
			name:  "empty alias",
			alias: "",
			mockFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
				return nil, nil
			},
			wantErr:      true,
			errMsg:       "alias cannot be empty",
			checkErrType: encx.ErrInvalidConfiguration,
		},
		{
			name:  "key not found",
			alias: "alias/nonexistent",
			mockFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
				return nil, errors.New("NotFoundException: Key 'alias/nonexistent' does not exist")
			},
			wantErr:      true,
			errMsg:       "failed to describe KMS key",
			checkErrType: encx.ErrKMSUnavailable,
		},
		{
			name:  "nil key metadata",
			alias: "alias/my-key",
			mockFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
				return &kms.DescribeKeyOutput{
					KeyMetadata: nil,
				}, nil
			},
			wantErr:      true,
			errMsg:       "no key metadata returned",
			checkErrType: encx.ErrKMSUnavailable,
		},
		{
			name:  "nil key ID in metadata",
			alias: "alias/my-key",
			mockFunc: func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
				return &kms.DescribeKeyOutput{
					KeyMetadata: &types.KeyMetadata{
						KeyId: nil,
					},
				}, nil
			},
			wantErr:      true,
			errMsg:       "no key metadata returned",
			checkErrType: encx.ErrKMSUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKMSClient{
				describeKeyFunc: tt.mockFunc,
			}

			svc := &KMSService{
				client: mock,
				region: "us-east-1",
			}

			keyID, err := svc.GetKeyID(ctx, tt.alias)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				if tt.checkErrType != nil {
					assert.ErrorIs(t, err, tt.checkErrType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantKeyID, keyID)
			}
		})
	}
}

func TestCreateKey(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		description  string
		mockFunc     func(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error)
		wantKeyID    string
		wantErr      bool
		errMsg       string
		checkErrType error
	}{
		{
			name:        "successful key creation",
			description: "My encryption key",
			mockFunc: func(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error) {
				assert.Equal(t, "My encryption key", *params.Description)
				assert.Equal(t, types.KeyUsageTypeEncryptDecrypt, params.KeyUsage)
				assert.Equal(t, types.KeySpecSymmetricDefault, params.KeySpec)
				assert.False(t, *params.MultiRegion)
				return &kms.CreateKeyOutput{
					KeyMetadata: &types.KeyMetadata{
						KeyId: aws.String("new-key-1234"),
					},
				}, nil
			},
			wantKeyID: "new-key-1234",
			wantErr:   false,
		},
		{
			name:        "KMS service error",
			description: "Test key",
			mockFunc: func(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error) {
				return nil, errors.New("AccessDeniedException: User not authorized")
			},
			wantErr:      true,
			errMsg:       "failed to create KMS key",
			checkErrType: encx.ErrKMSUnavailable,
		},
		{
			name:        "nil key metadata",
			description: "Test key",
			mockFunc: func(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error) {
				return &kms.CreateKeyOutput{
					KeyMetadata: nil,
				}, nil
			},
			wantErr:      true,
			errMsg:       "no key metadata returned after creation",
			checkErrType: encx.ErrKMSUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKMSClient{
				createKeyFunc: tt.mockFunc,
			}

			svc := &KMSService{
				client: mock,
				region: "us-east-1",
			}

			keyID, err := svc.CreateKey(ctx, tt.description)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				if tt.checkErrType != nil {
					assert.ErrorIs(t, err, tt.checkErrType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantKeyID, keyID)
			}
		})
	}
}

func TestEncryptDEK(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		keyID          string
		plaintext      []byte
		mockFunc       func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
		wantCiphertext []byte
		wantErr        bool
		errMsg         string
		checkErrType   error
	}{
		{
			name:      "successful encryption",
			keyID:     "1234abcd-12ab-34cd-56ef-1234567890ab",
			plaintext: []byte("my-secret-dek-32-bytes-length!"),
			mockFunc: func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
				assert.Equal(t, "1234abcd-12ab-34cd-56ef-1234567890ab", *params.KeyId)
				assert.Equal(t, []byte("my-secret-dek-32-bytes-length!"), params.Plaintext)
				// Simulate AWS KMS encryption
				return &kms.EncryptOutput{
					CiphertextBlob: []byte("encrypted-dek-blob"),
				}, nil
			},
			wantCiphertext: []byte(base64.StdEncoding.EncodeToString([]byte("encrypted-dek-blob"))),
			wantErr:        false,
		},
		{
			name:      "empty plaintext",
			keyID:     "test-key",
			plaintext: []byte{},
			mockFunc: func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
				return nil, nil
			},
			wantErr:      true,
			errMsg:       "plaintext cannot be empty",
			checkErrType: encx.ErrEncryptionFailed,
		},
		{
			name:      "KMS encryption error",
			keyID:     "invalid-key",
			plaintext: []byte("test-dek"),
			mockFunc: func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
				return nil, errors.New("InvalidKeyUsageException: Key not valid for encryption")
			},
			wantErr:      true,
			errMsg:       "failed to encrypt DEK with KMS key",
			checkErrType: encx.ErrEncryptionFailed,
		},
		{
			name:      "nil ciphertext blob",
			keyID:     "test-key",
			plaintext: []byte("test-dek"),
			mockFunc: func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
				return &kms.EncryptOutput{
					CiphertextBlob: nil,
				}, nil
			},
			wantErr:      true,
			errMsg:       "no ciphertext returned from KMS",
			checkErrType: encx.ErrEncryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKMSClient{
				encryptFunc: tt.mockFunc,
			}

			svc := &KMSService{
				client: mock,
				region: "us-east-1",
			}

			ciphertext, err := svc.EncryptDEK(ctx, tt.keyID, tt.plaintext)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				if tt.checkErrType != nil {
					assert.ErrorIs(t, err, tt.checkErrType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCiphertext, ciphertext)
			}
		})
	}
}

func TestDecryptDEK(t *testing.T) {
	ctx := context.Background()

	// Helper to create base64-encoded ciphertext
	createEncodedCiphertext := func(data string) []byte {
		return []byte(base64.StdEncoding.EncodeToString([]byte(data)))
	}

	tests := []struct {
		name          string
		keyID         string
		ciphertext    []byte
		mockFunc      func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
		wantPlaintext []byte
		wantErr       bool
		errMsg        string
		checkErrType  error
	}{
		{
			name:       "successful decryption with key ID",
			keyID:      "1234abcd-12ab-34cd-56ef-1234567890ab",
			ciphertext: createEncodedCiphertext("encrypted-dek-blob"),
			mockFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
				assert.Equal(t, []byte("encrypted-dek-blob"), params.CiphertextBlob)
				assert.Equal(t, "1234abcd-12ab-34cd-56ef-1234567890ab", *params.KeyId)
				return &kms.DecryptOutput{
					Plaintext: []byte("my-secret-dek-32-bytes-length!"),
				}, nil
			},
			wantPlaintext: []byte("my-secret-dek-32-bytes-length!"),
			wantErr:       false,
		},
		{
			name:       "successful decryption without key ID",
			keyID:      "",
			ciphertext: createEncodedCiphertext("encrypted-dek-blob"),
			mockFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
				assert.Nil(t, params.KeyId)
				return &kms.DecryptOutput{
					Plaintext: []byte("decrypted-dek"),
				}, nil
			},
			wantPlaintext: []byte("decrypted-dek"),
			wantErr:       false,
		},
		{
			name:       "empty ciphertext",
			keyID:      "test-key",
			ciphertext: []byte{},
			mockFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
				return nil, nil
			},
			wantErr:      true,
			errMsg:       "ciphertext cannot be empty",
			checkErrType: encx.ErrDecryptionFailed,
		},
		{
			name:       "invalid base64",
			keyID:      "test-key",
			ciphertext: []byte("not-valid-base64!@#$"),
			mockFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
				return nil, nil
			},
			wantErr:      true,
			errMsg:       "failed to decode ciphertext",
			checkErrType: encx.ErrDecryptionFailed,
		},
		{
			name:       "KMS decryption error",
			keyID:      "wrong-key",
			ciphertext: createEncodedCiphertext("encrypted-data"),
			mockFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
				return nil, errors.New("InvalidCiphertextException: Invalid ciphertext")
			},
			wantErr:      true,
			errMsg:       "failed to decrypt DEK",
			checkErrType: encx.ErrDecryptionFailed,
		},
		{
			name:       "nil plaintext",
			keyID:      "test-key",
			ciphertext: createEncodedCiphertext("encrypted-data"),
			mockFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
				return &kms.DecryptOutput{
					Plaintext: nil,
				}, nil
			},
			wantErr:      true,
			errMsg:       "no plaintext returned from KMS",
			checkErrType: encx.ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockKMSClient{
				decryptFunc: tt.mockFunc,
			}

			svc := &KMSService{
				client: mock,
				region: "us-east-1",
			}

			plaintext, err := svc.DecryptDEK(ctx, tt.keyID, tt.ciphertext)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				if tt.checkErrType != nil {
					assert.ErrorIs(t, err, tt.checkErrType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPlaintext, plaintext)
			}
		})
	}
}

func TestRegion(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		wantRegion string
	}{
		{
			name:       "us-east-1",
			region:     "us-east-1",
			wantRegion: "us-east-1",
		},
		{
			name:       "eu-west-1",
			region:     "eu-west-1",
			wantRegion: "eu-west-1",
		},
		{
			name:       "ap-southeast-1",
			region:     "ap-southeast-1",
			wantRegion: "ap-southeast-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &KMSService{
				region: tt.region,
			}

			assert.Equal(t, tt.wantRegion, svc.Region())
		})
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	ctx := context.Background()
	plainDEK := []byte("my-secret-dek-32-bytes-length!")

	// Create mock that simulates real encryption/decryption
	mock := &mockKMSClient{
		encryptFunc: func(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error) {
			// Simulate encryption by just prefixing
			blob := append([]byte("encrypted:"), params.Plaintext...)
			return &kms.EncryptOutput{
				CiphertextBlob: blob,
			}, nil
		},
		decryptFunc: func(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error) {
			// Simulate decryption by removing prefix
			if len(params.CiphertextBlob) > 10 && string(params.CiphertextBlob[:10]) == "encrypted:" {
				return &kms.DecryptOutput{
					Plaintext: params.CiphertextBlob[10:],
				}, nil
			}
			return nil, errors.New("invalid ciphertext")
		},
	}

	svc := &KMSService{
		client: mock,
		region: "us-east-1",
	}

	// Encrypt
	ciphertext, err := svc.EncryptDEK(ctx, "test-key", plainDEK)
	require.NoError(t, err)
	require.NotEmpty(t, ciphertext)

	// Verify it's base64 encoded
	_, err = base64.StdEncoding.DecodeString(string(ciphertext))
	require.NoError(t, err)

	// Decrypt
	decrypted, err := svc.DecryptDEK(ctx, "test-key", ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plainDEK, decrypted)
}
