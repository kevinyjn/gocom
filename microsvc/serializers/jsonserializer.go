package serializers

import (
	"encoding/json"
)

// JSONParam serializable paremeter wrapper
type JSONParam struct {
	DataPtr interface{}
}

// NewJSONParam serializable parameter wrapper
func NewJSONParam(data interface{}) *JSONParam {
	return &JSONParam{DataPtr: data}
}

// GetJSONSerializer instance
func GetJSONSerializer() Serializable {
	return _jsonSerializer
}

// jsonSerializer json serializing instance
type jsonSerializer struct{}

var _jsonSerializer = &jsonSerializer{}

// ContentType serialization content type name
func (p *JSONParam) ContentType() string {
	return "application/json"
}

// Serialize serialize the payload data as json content
func (p *JSONParam) Serialize() ([]byte, error) {
	return json.Marshal(p.DataPtr)
}

// ParseFrom parse payload data from json content
func (p *JSONParam) ParseFrom(buffer []byte) error {
	return json.Unmarshal(buffer, p.DataPtr)
}

// ContentType serialization content type name
func (s *jsonSerializer) ContentType() string {
	return "application/json"
}

// Serialize serialize the payload data as json content
func (s *jsonSerializer) Serialize(payloadObject interface{}) ([]byte, error) {
	return json.Marshal(payloadObject)
}

// ParseFrom parse payload data from json content
func (s *jsonSerializer) ParseFrom(buffer []byte, payloadObject interface{}) error {
	return json.Unmarshal(buffer, payloadObject)
}
