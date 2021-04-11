package serializers

import (
	"bytes"

	"github.com/lenaten/hl7"
)

// HL7Param serializable paremeter wrapper
type HL7Param struct {
	DataPtr interface{}
}

// NewHL7Param serializable parameter wrapper
func NewHL7Param(data interface{}) *HL7Param {
	return &HL7Param{DataPtr: data}
}

// GetHL7Serializer instance
func GetHL7Serializer() Serializable {
	return _hl7Serializer
}

// hl7Serializer hl7 serializing instance
type hl7Serializer struct{}

var _hl7Serializer = &hl7Serializer{}

// ContentType serialization content type name
func (p *HL7Param) ContentType() string {
	return "application/hl7"
}

// Serialize serialize the payload data as hl7 content
func (p *HL7Param) Serialize() ([]byte, error) {
	// todo
	mi := hl7.MsgInfo{
		SendingApp:        "MicroSvc",
		SendingFacility:   "MicroSvcPlace",
		ReceivingApp:      "EMR",
		ReceivingFacility: "MedicalPlace",
		MessageType:       "ORM^001",
	}
	msg, err := hl7.StartMessage(mi)
	if nil != err {
		return nil, err
	}
	return hl7.Marshal(msg, p.DataPtr)
}

// ParseFrom parse payload data from hl7 content
func (p *HL7Param) ParseFrom(buffer []byte) error {
	msgs, err := hl7.NewDecoder(bytes.NewReader(buffer)).Messages()
	if nil != err {
		return err
	}
	return msgs[0].Unmarshal(p.DataPtr)
}

// ContentType serialization content type name
func (s *hl7Serializer) ContentType() string {
	return "application/hl7"
}

// Serialize serialize the payload data as hl7 content
func (s *hl7Serializer) Serialize(payloadObject interface{}) ([]byte, error) {
	// todo
	mi := hl7.MsgInfo{
		SendingApp:        "MicroSvc",
		SendingFacility:   "MicroSvcPlace",
		ReceivingApp:      "EMR",
		ReceivingFacility: "MedicalPlace",
		MessageType:       "ORM^001",
	}
	msg, err := hl7.StartMessage(mi)
	if nil != err {
		return nil, err
	}
	return hl7.Marshal(msg, payloadObject)
}

// ParseFrom parse payload data from hl7 content
func (s *hl7Serializer) ParseFrom(buffer []byte, payloadObject interface{}) error {
	msgs, err := hl7.NewDecoder(bytes.NewReader(buffer)).Messages()
	if nil != err {
		return err
	}
	return msgs[0].Unmarshal(payloadObject)
}
