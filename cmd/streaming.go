package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/hengadev/encx"
)

// StreamingWorkflow demonstrates file encryption and decryption using streaming operations
func StreamingWorkflow() {
	ctx := context.Background()

	// Initialize crypto instance (this would typically be done with proper configuration)
	crypto, err := encx.NewCrypto(ctx)
	if err != nil {
		fmt.Printf("Failed to initialize crypto: %v\n", err)
		return
	}

	// Generate a DEK for this operation
	dek, err := crypto.GenerateDEK()
	if err != nil {
		fmt.Printf("Failed to generate DEK: %v\n", err)
		return
	}

	inputFile := ASSETS_PATH + "large_video.mp4"
	encryptedFile := ASSETS_PATH + "encrypted_video.bin"
	decryptedFile := ASSETS_PATH + "decrypted_video.mp4"

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Printf("Input file %s does not exist. Skipping streaming demo.\n", inputFile)
		return
	}

	// Open input file for reading
	inputReader, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		return
	}
	defer inputReader.Close()

	// Create output file for encrypted data
	encryptedWriter, err := os.Create(encryptedFile)
	if err != nil {
		fmt.Printf("Error creating encrypted file: %v\n", err)
		return
	}
	defer encryptedWriter.Close()

	// Encrypt the file using streaming
	fmt.Println("Encrypting file using streaming...")
	if err := crypto.EncryptStream(ctx, inputReader, encryptedWriter, dek); err != nil {
		fmt.Printf("Error encrypting file: %v\n", err)
		return
	}
	fmt.Println("File encrypted successfully")

	// Open encrypted file for reading
	encryptedReader, err := os.Open(encryptedFile)
	if err != nil {
		fmt.Printf("Error opening encrypted file: %v\n", err)
		return
	}
	defer encryptedReader.Close()

	// Create output file for decrypted data
	decryptedWriter, err := os.Create(decryptedFile)
	if err != nil {
		fmt.Printf("Error creating decrypted file: %v\n", err)
		return
	}
	defer decryptedWriter.Close()

	// Decrypt the file using streaming
	fmt.Println("Decrypting file using streaming...")
	if err := crypto.DecryptStream(ctx, encryptedReader, decryptedWriter, dek); err != nil {
		fmt.Printf("Error decrypting file: %v\n", err)
		return
	}
	fmt.Println("File decrypted successfully")

	// Clean up temporary files
	os.Remove(encryptedFile)
	fmt.Printf("Streaming workflow completed. Check %s for the result.\n", decryptedFile)
}
