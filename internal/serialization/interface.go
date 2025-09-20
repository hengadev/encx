package serialization

// Serializer defines an interface for converting Go data types to and from byte arrays.
// Implementations of this interface handle the encoding of struct fields before
// encryption or hashing, and potentially the decoding after decryption.
type Serializer interface {
	// Serialize takes any value and returns its byte representation and an error
	// if serialization fails. Different implementations offer varying trade-offs
	// in terms of performance, size, and interoperability.
	Serialize(v any) ([]byte, error)

	// Deserialize takes a byte array and a pointer to the target value
	// and populates it with the deserialized data.
	Deserialize(data []byte, v any) error
}
