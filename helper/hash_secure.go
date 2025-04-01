package helper

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// hashPassword generates a hashed password using the Argon2id algorithm, with added salt and pepper for extra security.
//
// Parameters:
//   - password: The plaintext password to be hashed.
//
// Returns:
//   - string: The generated hash, including Argon2 parameters, salt, and the password hash in a string format.
//   - error: An error if the salt generation, hashing, or string encoding fails. Returns nil if successful.
func hashSecure(
	value string,
	saltLength,
	iterations,
	memory,
	keyLength uint32,
	parallelism uint8,
	pepper []byte,
) (string, error) {
	// Generate a random salt
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Combine password with pepper
	peppered := append([]byte(value), pepper...)

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		peppered,
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	// Encode params, salt, and hash into a string
	params := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return params, nil
}
