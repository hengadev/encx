package encx

// Field name constants - exported for public use
const (
	FieldKeyVersion = "KeyVersion"
	FieldDEK        = "DEK"
	FieldDEKEncrypted = FieldDEK + SuffixEncrypted
)

// Suffix constants for generated fields
const (
	SuffixEncrypted = "Encrypted"
	SuffixHashed    = "Hash"
)

// Tag constants for struct field annotations
const (
	StructTag     = "encx"
	TagEncrypt    = "encrypt"
	TagHashSecure = "hash_secure"
	TagHashBasic  = "hash_basic"
)

// Internal fields that should be skipped during processing
var (
	fieldsToSkip = [3]string{FieldDEK, FieldDEKEncrypted, FieldKeyVersion}
)
