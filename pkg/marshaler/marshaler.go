package marshaler

import (
	"bytes"
	"io"

	"github.com/goccy/go-json"
)

// LoadJSONFromReader loads JSON data from a reader into the given interface.
func LoadJSONFromReader(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// MarshalJSONToString marshals a value to a JSON string.
func MarshalJSONToString(v any) (string, error) {
	res, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// MarshalJSON marshals a value to a JSON byte array.
func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalJSONToStringWithoutEscape marshals a value to a JSON string without HTML escaping.
func MarshalJSONToStringWithoutEscape(v any) (string, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)

	return buffer.String(), err
}

// MarshalJSONWithoutEscape marshals a value to a JSON byte array without HTML escaping.
func MarshalJSONWithoutEscape(v any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)

	return buffer.Bytes(), err
}

// UnmarshalJSONFromString unmarshals JSON from a string into the given interface.
func UnmarshalJSONFromString(str string, v any) error {
	return json.Unmarshal([]byte(str), v)
}

// UnmarshalJSON unmarshals JSON from a byte array into the given interface.
func UnmarshalJSON(b []byte, v any) error {
	return json.Unmarshal(b, v)
}

// MarshalIndent marshals a value to a pretty-printed JSON byte array.
func MarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
