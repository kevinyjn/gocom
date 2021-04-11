package serializers

import (
	"fmt"
	"reflect"

	proto "github.com/golang/protobuf/proto"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
)

// ProtobufParam serializable paremeter wrapper
type ProtobufParam struct {
	Data protoiface.MessageV1
}

// NewProtobufParam serializable parameter wrapper
func NewProtobufParam(data protoiface.MessageV1) *ProtobufParam {
	return &ProtobufParam{Data: data}
}

// GetProtobufSerializer instance
func GetProtobufSerializer() Serializable {
	return _protobufSerializer
}

// protobufSerializer protobuf serializing instance
type protobufSerializer struct{}

var _protobufSerializer = &protobufSerializer{}

// ContentType serialization content type name
func (p *ProtobufParam) ContentType() string {
	return "application/protobuf"
}

// Serialize serialize the payload data as protobuf content
func (p *ProtobufParam) Serialize() ([]byte, error) {
	return proto.Marshal(p.Data)
}

// ParseFrom parse payload data from protobuf content
func (p *ProtobufParam) ParseFrom(buffer []byte) error {
	return proto.Unmarshal(buffer, p.Data)
}

// ContentType serialization content type name
func (s *protobufSerializer) ContentType() string {
	return "application/protobuf"
}

// Serialize serialize the payload data as protobuf content
func (s *protobufSerializer) Serialize(payloadObject interface{}) ([]byte, error) {
	pb, ok := payloadObject.(protoiface.MessageV1)
	if false == ok {
		return nil, fmt.Errorf("Serializing %s were not protobuf object", reflect.TypeOf(payloadObject).Name())
	}
	return proto.Marshal(pb)
}

// ParseFrom parse payload data from protobuf content
func (s *protobufSerializer) ParseFrom(buffer []byte, payloadObject interface{}) error {
	pb, ok := payloadObject.(protoiface.MessageV1)
	if false == ok {
		return fmt.Errorf("Parsing %s were not protobuf object", reflect.TypeOf(payloadObject).Name())
	}
	return proto.Unmarshal(buffer, pb)
}
