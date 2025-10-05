// Package main demonstrates streaming encryption to AWS S3
//
// This example shows how to:
// 1. Receive an image upload via HTTP
// 2. Encrypt the image on-the-fly using encx
// 3. Stream the encrypted data directly to S3
// 4. Store the encrypted DEK for later retrieval
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/hengadev/encx"
	"github.com/hengadev/encx/providers/awskms"
)

// AWSS3Uploader defines the method used to upload to S3
type AWSS3Uploader interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type s3Writer struct {
	writer   *io.PipeWriter
	reader   *io.PipeReader
	s3Client AWSS3Uploader
	bucket   string
	key      string
	cancel   context.CancelFunc
	errChan  chan error
}

func (w *s3Writer) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

func (w *s3Writer) Close() error {
	// Close writer first to signal EOF to reader
	err := w.writer.Close()
	w.cancel() // Cancel the context to stop upload goroutine
	if err != nil {
		return err
	}

	// Wait for upload to complete and check for errors
	uploadErr := <-w.errChan
	if uploadErr != nil {
		return uploadErr
	}

	return w.reader.Close()
}

func createS3FileWriter(ctx context.Context, s3Client AWSS3Uploader, bucket, key, contentType string) (io.WriteCloser, error) {
	reader, writer := io.Pipe()

	// Create a context with cancel function for the S3 upload
	uploadCtx, cancel := context.WithCancel(ctx)

	errChan := make(chan error, 1)

	s3Writer := &s3Writer{
		writer:   writer,
		reader:   reader,
		s3Client: s3Client,
		bucket:   bucket,
		key:      key,
		cancel:   cancel,
		errChan:  errChan,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in S3 upload: %v", r)
				reader.CloseWithError(fmt.Errorf("panic during upload: %v", r))
				errChan <- fmt.Errorf("panic during upload: %v", r)
			}
		}()

		// Upload to S3 from the reader
		_, err := s3Client.PutObject(uploadCtx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        reader,
			ContentType: aws.String(contentType),
		})

		if err != nil {
			log.Printf("Failed to upload to S3: %v", err)
			reader.CloseWithError(err)
			errChan <- err
			return
		}

		log.Printf("Uploaded to S3: %s/%s", bucket, key)
		errChan <- nil
	}()

	return s3Writer, nil
}

// uploadImageToS3 handles the HTTP upload, encryption, and S3 upload.
func uploadImageToS3(w http.ResponseWriter, r *http.Request, cryptoService *encx.Crypto, s3Client AWSS3Uploader, bucket string) {
	// 1. Get the image from the HTTP request
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file from request: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Determine content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 3. Generate a unique key for the S3 object
	key := uuid.New().String() + "-encrypted"

	// 4. Create an S3 writer
	s3Writer, err := createS3FileWriter(r.Context(), s3Client, bucket, key, contentType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create S3 writer: %v", err), http.StatusInternalServerError)
		return
	}
	defer s3Writer.Close()

	// 5. Generate a Data Encryption Key (DEK)
	dek, err := cryptoService.GenerateDEK()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate DEK: %v", err), http.StatusInternalServerError)
		return
	}

	// 6. Encrypt the image data and write it to the S3 writer
	ctx := context.Background()
	err = cryptoService.EncryptStream(ctx, file, s3Writer, dek)
	if err != nil {
		http.Error(w, fmt.Sprintf("Encryption or upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 7. Encrypt the DEK with the KEK
	encryptedDEK, err := cryptoService.EncryptDEK(ctx, dek)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encrypt DEK: %v", err), http.StatusInternalServerError)
		return
	}

	// 8. Respond to the client
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Image uploaded and encrypted to S3://%s/%s\n", bucket, key)
	fmt.Fprintf(w, "Encrypted DEK (store this securely): %x\n", encryptedDEK)
	fmt.Fprintf(w, "Original filename: %s\n", header.Filename)
}

func main() {
	// 1. Load AWS configuration
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	// 2. Create an S3 client
	s3Client := s3.NewFromConfig(cfg)

	// 3. Initialize AWS KMS provider
	kmsService, err := awskms.New(ctx, awskms.Config{
		Region: os.Getenv("AWS_REGION"),
	})
	if err != nil {
		log.Fatalf("failed to create KMS service: %v", err)
	}

	// 4. Create crypto service with AWS KMS
	pepper := []byte(os.Getenv("ENCX_PEPPER"))
	if len(pepper) != 32 {
		log.Fatal("ENCX_PEPPER must be exactly 32 bytes")
	}

	cryptoService, err := encx.NewCrypto(ctx,
		encx.WithKMSService(kmsService),
		encx.WithKEKAlias(os.Getenv("KMS_KEY_ALIAS")),
		encx.WithPepper(pepper),
	)
	if err != nil {
		log.Fatalf("failed to create Crypto service: %v", err)
	}

	// 5. Define S3 bucket
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("S3_BUCKET_NAME environment variable is required")
	}

	// 6. Set up the HTTP handler
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadImageToS3(w, r, cryptoService, s3Client, bucketName)
	})

	// 7. Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on :%s", port)
	log.Printf("Using S3 bucket: %s", bucketName)
	log.Printf("Using KMS key: %s", os.Getenv("KMS_KEY_ALIAS"))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
