package encx

const (
	VERSION_FIELD          = "KeyVersion"
	DEK_FIELD              = "DEK"
	ENCRYPTED_FIELD_SUFFIX = "Encrypted"
	DEK_ENCRYPTED_FIELD    = DEK_FIELD + ENCRYPTED_FIELD_SUFFIX
	HASHED_FIELD_SUFFIX    = "Hash"
	STRUCT_TAG             = "encx"

	// tags
	ENCRYPT = "encrypt"
	SECURE  = "hash_secure"
	BASIC   = "hash_basic"
)
