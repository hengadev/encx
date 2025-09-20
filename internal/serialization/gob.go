package serialization

import (
	"bytes"
	"encoding/gob"
)

// GOBSerializer implements the Serializer interface using the encoding/gob package.
// It offers efficient binary encoding specifically for Go data types, often resulting
// in smaller sizes and faster performance than JSON. However, it has limited
// interoperability with non-Go systems. Choose this if performance and Go-specific
// handling are primary concerns and cross-language compatibility is not required.
type GOBSerializer struct{}

func (g GOBSerializer) Serialize(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GOBSerializer) Deserialize(data []byte, v any) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(v)
}
