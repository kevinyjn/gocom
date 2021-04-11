package serializers

import (
	"encoding/xml"
)

// XMLParam serializable paremeter wrapper
type XMLParam struct {
	DataPtr interface{}
}

// NewXMLParam serializable parameter wrapper
func NewXMLParam(data interface{}) *XMLParam {
	return &XMLParam{DataPtr: data}
}

// GetXMLSerializer instance
func GetXMLSerializer() Serializable {
	return _xmlSerializer
}

// xmlSerializer xml serializing instance
type xmlSerializer struct{}

var _xmlSerializer = &xmlSerializer{}

// ContentType serialization content type name
func (p *XMLParam) ContentType() string {
	return "application/xml"
}

// Serialize serialize the payload data as xml content
func (p *XMLParam) Serialize() ([]byte, error) {
	return xml.Marshal(p.DataPtr)
}

// ParseFrom parse payload data from xml content
func (p *XMLParam) ParseFrom(buffer []byte) error {
	return xml.Unmarshal(buffer, p.DataPtr)
}

// ContentType serialization content type name
func (s *xmlSerializer) ContentType() string {
	return "application/xml"
}

// Serialize serialize the payload data as xml content
func (s *xmlSerializer) Serialize(payloadObject interface{}) ([]byte, error) {
	return xml.Marshal(payloadObject)
}

// ParseFrom parse payload data from xml content
func (s *xmlSerializer) ParseFrom(buffer []byte, payloadObject interface{}) error {
	return xml.Unmarshal(buffer, payloadObject)
}
