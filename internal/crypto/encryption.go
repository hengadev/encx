package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// DataEncryption handles data encryption and decryption operations
type DataEncryption struct{}

// NewDataEncryption creates a new DataEncryption instance
func NewDataEncryption() *DataEncryption {
	return &DataEncryption{}
}

// EncryptData encrypts the provided data using the provided DEK.
func (e *DataEncryption) EncryptData(ctx context.Context, plaintext []byte, dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptData decrypts the provided ciphertext using the provided DEK.
func (e *DataEncryption) DecryptData(ctx context.Context, ciphertext []byte, dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("invalid ciphertext size")
	}
	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// EncryptStream encrypts data from an io.Reader to an io.Writer using the provided DEK.
func (e *DataEncryption) EncryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error {
	buffer := make([]byte, 4096) // Choose an appropriate buffer size
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read from input stream: %w", err)
		}
		ciphertext, err := e.EncryptData(ctx, buffer[:n], dek)
		if err != nil {
			return fmt.Errorf("failed to encrypt chunk: %w", err)
		}
		_, err = writer.Write(ciphertext)
		if err != nil {
			return fmt.Errorf("failed to write to output stream: %w", err)
		}
	}
	return nil
}

// DecryptStream decrypts data from an io.Reader to an io.Writer using the provided DEK.
func (e *DataEncryption) DecryptStream(ctx context.Context, reader io.Reader, writer io.Writer, dek []byte) error {
	buffer := make([]byte, 4096) // Choose an appropriate buffer size
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read from input stream: %w", err)
		}
		plaintext, err := e.DecryptData(ctx, buffer[:n], dek)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk: %w", err)
		}
		_, err = writer.Write(plaintext)
		if err != nil {
			return fmt.Errorf("failed to write to output stream: %w", err)
		}
	}
	return nil
}

