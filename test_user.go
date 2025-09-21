package encx

// User represents a user with encrypted fields
type User struct {
	ID    int    `json:"id"`
	Email string `json:"email" encx:"encrypt,hash_basic"`
	Phone string `json:"phone" encx:"encrypt"`
	SSN   string `json:"ssn" encx:"hash_secure"`
	// Added a comment to change the file hash
	Name  string `json:"name"`

	// Companion fields for encryption/hashing
	EmailEncrypted []byte `json:"email_encrypted" db:"email_encrypted"`
	EmailHash      string `json:"email_hash" db:"email_hash"`
	PhoneEncrypted []byte `json:"phone_encrypted" db:"phone_encrypted"`
	SSNHashSecure  string `json:"ssn_hash_secure" db:"ssn_hash_secure"`
}