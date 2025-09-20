package serialization

import "reflect"

// Serializer defines an interface for converting Go data types to and from byte arrays.
// Implementations of this interface handle the encoding of struct fields before
// encryption or hashing, and potentially the decoding after decryption.
type Serializer interface {
	// Serialize takes a reflect.Value representing a field and returns its byte
	// representation and an error if serialization fails. Different implementations
	// offer varying trade-offs in terms of performance, size, and interoperability.
	Serialize(v reflect.Value) ([]byte, error)

	// Deserialize takes a byte array and a reflect.Value (pointer to the field)
	// and populates the field with the deserialized data. This method is optional
	// if the package user handles deserialization outside of the core processing.
	Deserialize(data []byte, v reflect.Value) error
}

