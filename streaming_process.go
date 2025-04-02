package encx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
)

// EncryptStream takes an input stream and returns an encrypted stream
func EncryptStream(reader io.Reader, key string) (io.Reader, []byte, error) {
	encryptionKey, err := hex.DecodeString(key)
	if err != nil {
		return nil, nil, err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, nil, err
	}

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	stream := cipher.NewCTR(block, iv)
	encryptedReader := &cipher.StreamReader{S: stream, R: reader}

	return encryptedReader, iv, nil
}

func EncryptStreamingFile(inputPath, outputPath, key string) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	encryptionKey, err := hex.DecodeString(key)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	// Write the IV to the output file first
	if _, err := outFile.Write(iv); err != nil {
		return err
	}

	stream := cipher.NewCTR(block, iv)
	writer := &cipher.StreamWriter{S: stream, W: outFile}

	// Copy the input file to the output file, encrypting as we go
	if _, err := io.Copy(writer, inFile); err != nil {
		return err
	}

	return nil
}

// DecryptStream takes an encrypted stream and returns a decrypted stream
func DecryptStream(reader io.Reader, key string, iv []byte) (io.Reader, error) {
	encryptionKey, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)
	decryptedReader := &cipher.StreamReader{S: stream, R: reader}

	return decryptedReader, nil
}

func DecryptStreamingFile(inputPath, outputPath, key string) error {
	// Open the encrypted input file
	inFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	encryptionKey, err := hex.DecodeString(key)
	if err != nil {
		return err
	}

	// Create the AES cipher block
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return err
	}

	// Read the IV from the beginning of the file
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(inFile, iv); err != nil {
		return err
	}

	// Create the CTR mode stream
	stream := cipher.NewCTR(block, iv)

	// Create a reader that decrypts as it reads
	reader := &cipher.StreamReader{S: stream, R: inFile}

	// Copy the decrypted content to the output file
	if _, err := io.Copy(outFile, reader); err != nil {
		return err
	}

	return nil
}
