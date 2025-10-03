package s3bucket

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/hengadev/encx"
)

// AWSS3Uploader defines the method used to upload to S3
type AWSS3Uploader interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type s3Writer struct {
	writer   *io.PipeWriter
	reader   *io.PipeReader
	s3Client AWSS3Uploader // Use the interface
	bucket   string
	key      string
	cancel   context.CancelFunc // Add a cancel func
}

func (w *s3Writer) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

func (w *s3Writer) Close() error {
	// Close both ends.  Crucially, close the writer *before* cancelling
	// the context.  Closing the writer signals EOF to the reader.
	err := w.writer.Close()
	w.cancel() // Cancel the context to stop the upload goroutine
	if err != nil {
		return err
	}
	return w.reader.Close() // close reader as well
}

func createS3FileWriter(ctx context.Context, s3Client AWSS3Uploader, bucket, key string) (io.WriteCloser, error) {
	reader, writer := io.Pipe()

	// Create a context with a cancel function.  This context will be used
	// for the S3 upload, and we'll use the cancel function to stop the
	// upload goroutine when the caller closes the WriteCloser.
	uploadCtx, cancel := context.WithCancel(ctx)

	s3Writer := &s3Writer{
		writer:   writer,
		reader:   reader,
		s3Client: s3Client, // Use the interface
		bucket:   bucket,
		key:      key,
		cancel:   cancel,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in S3 upload: %v", r)
				// important: close the reader on error, to unblock the writer
				reader.CloseWithError(fmt.Errorf("panic during upload: %v", r))
			}
		}()
		// Upload to S3 from the reader
		_, err := s3Client.PutObject(uploadCtx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        reader,                   // Use the reader here
			ContentType: aws.String("image/jpeg"), // set content type
		})
		if err != nil {
			log.Printf("Failed to upload to S3: %v", err)
			// important: close the reader on error, to unblock the writer
			reader.CloseWithError(err)
			return
		}
		log.Printf("Uploaded to S3: %s/%s", bucket, key)
	}()

	return s3Writer, nil
}

// uploadImageToS3 handles the HTTP upload, encryption, and S3 upload.
func uploadImageToS3(w http.ResponseWriter, r *http.Request, cryptoService *encx.Crypto, s3Client AWSS3Uploader, bucket string) {
	// 1. Get the image from the HTTP request.
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file from request: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Generate a unique key for the S3 object.
	key := uuid.New().String() + ".jpg" // Or .png, etc.

	// 3. Create an S3 writer using the createS3FileWriter function.
	s3Writer, err := createS3FileWriter(r.Context(), s3Client, bucket, key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create S3 writer: %v", err), http.StatusInternalServerError)
		return
	}
	defer s3Writer.Close() // Ensure the writer is closed.

	// 4. Generate a Data Encryption Key (DEK).
	dek, err := cryptoService.GenerateDEK()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate DEK: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. Encrypt the image data and write it to the S3 writer.
	ctx := context.Background()
	err = cryptoService.EncryptStream(ctx, file, s3Writer, dek)
	if err != nil {
		http.Error(w, fmt.Sprintf("Encryption or upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 6. (Optional) Encrypt the DEK with the KEK.
	encryptedDEK, err := cryptoService.EncryptData(dek, cryptoService.kek)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encrypt DEK: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Encrypted DEK: %s", bytes.NewBuffer(encryptedDEK).String())

	// 7. Respond to the client.
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Image uploaded and encrypted to S3://%s/%s\n", bucket, key)
	fmt.Fprintf(w, "Encrypted DEK (for your storage): %x\n", encryptedDEK) // hex
}

func main() {
	// 1. Load AWS configuration.
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	// 2. Create an S3 client.
	s3Client := s3.NewFromConfig(cfg)

	// 3.  Initialize Crypto (you'll need your KMS setup)
	kmsService := &YourKmsService{} // Replace with your actual KMS implementation
	cryptoService, err := New(ctx, kmsService, "your-kek-alias", "your-pepper-secret-path")
	if err != nil {
		log.Fatalf("failed to create Crypto service: %v", err)
	}

	// 4.  Define S3 bucket
	bucketName := "your-s3-bucket-name"

	// 5. Set up the HTTP handler.
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadImageToS3(w, r, cryptoService, s3Client, bucketName)
	})

	// 6. Start the server.
	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// Dummy KMS Service
type YourKmsService struct{}

func (k *YourKmsService) GetKeyID(ctx context.Context, alias string) (string, error) {
	return "dummy-kms-key-id", nil
}
func (k *YourKmsService) CreateKey(ctx context.Context, alias string) (string, error) {
	return "dummy-kms-key-id", nil
}
func (k *YourKmsService) GetSecret(ctx context.Context, path string) ([]byte, error) {
	return []byte("thisisatotallysecurepepper12345678"), nil
}

// Crypto represents a local crypto service for the s3 provider example
type Crypto struct{}

func (c *Crypto) GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	return dek, nil
}
