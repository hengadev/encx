package serialization

import (
	"encoding/json"
)

// JSONSerializer implements the Serializer interface using the encoding/json package.
// It provides good compatibility with complex data structures and decent human readability
// (of the serialized form), but might have a slight performance overhead for basic types
// compared to more direct conversions. It is a good default choice for general use.
type JSONSerializer struct{}

func (j JSONSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j JSONSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
