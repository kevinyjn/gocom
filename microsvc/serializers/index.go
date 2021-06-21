package serializers

import (
	"sync"

	"github.com/kevinyjn/gocom/logger"
)

// builtin serialization types
const (
	SerializationTypeJSON      = "json"
	SerializationTypeXML       = "xml"
	SerializationTypeProtobuf  = "protobuf"
	SerializationTypeHL7       = "hl7"
	SerializationTypeURLEncode = "x-www-form-urlencoded"
)

// SerializationStrategy interface
type SerializationStrategy interface {
	SerializationType() string
}

// Serializable handler parameter that were serializable
type Serializable interface {
	ContentType() string
	Serialize(payloadObject interface{}) ([]byte, error)
	ParseFrom(buffer []byte, payloadObject interface{}) error
}

// SerializableParam handler parameter that were serializable
type SerializableParam interface {
	ContentType() string
	Serialize() ([]byte, error)
	ParseFrom([]byte) error
}

// parameterData pointer of data
type parameterData interface{}

type serializerManager struct {
	serializers map[string]Serializable
	m           sync.RWMutex
}

var _serializers = serializerManager{
	serializers: map[string]Serializable{
		SerializationTypeJSON:      GetJSONSerializer(),
		SerializationTypeXML:       GetXMLSerializer(),
		SerializationTypeProtobuf:  GetProtobufSerializer(),
		SerializationTypeHL7:       GetHL7Serializer(),
		SerializationTypeURLEncode: GetURLEncodeSerializer(),
	},
	m: sync.RWMutex{},
}

// GetSerializer get serializable instance from factory
func GetSerializer(serializationType string) Serializable {
	return _serializers.get(serializationType)
}

// SetSerializer set serializable instance to factory
func SetSerializer(serializationType string, serializer Serializable) {
	_serializers.set(serializationType, serializer)
}

func (sm *serializerManager) get(serializationType string) Serializable {
	sm.m.RLock()
	serializer, _ := sm.serializers[serializationType]
	sm.m.RUnlock()
	if nil == serializer {
		logger.Error.Printf("get serializer instance by %s not found, consider as json serializer", serializationType)
		return GetJSONSerializer()
	}
	return serializer
}

func (sm *serializerManager) set(serializationType string, serializer Serializable) {
	if nil == serializer {
		logger.Error.Printf("set serializer of %s with nil", serializationType)
		return
	}
	sm.m.Lock()
	sm.serializers[serializationType] = serializer
	sm.m.RUnlock()
}
