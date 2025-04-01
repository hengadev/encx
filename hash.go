package encx

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

func (s Encryptor) Hash(value string) (string, error) {
	// Generate a random salt
	salt := make([]byte, s.Argon2Params.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Combine value with pepper
	peppered := append([]byte(value), s.Pepper...)

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		peppered,
		salt,
		s.Argon2Params.Iterations,
		s.Argon2Params.Memory,
		s.Argon2Params.Parallelism,
		s.Argon2Params.KeyLength,
	)

	// Encode params, salt, and hash into a string
	params := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.Argon2Params.Memory,
		s.Argon2Params.Iterations,
		s.Argon2Params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return params, nil
}
