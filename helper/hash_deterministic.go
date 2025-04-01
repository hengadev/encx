package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func hashDeterministic(value string) string {
	valueHash := sha256.Sum256([]byte(strings.ToLower(value)))
	return hex.EncodeToString(valueHash[:])
}
