package pulsar

import (
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/netutils/sshtunnel"
)

// Constants
const (
	PropertyCorrelationID = "CorrelationId"
	PropertyReplyTo       = "ReplyTo"
	PropertyMessageID     = "MessageId"
	PropertyAppID         = "AppId"
	PropertyUserID        = "UserId"
	PropertyContentType   = "ContentType"
)

// Config Pulsar MQ configuration
type Config struct {
	Topic          string
	ConnConfigName string
	// 消息类型:
	//direct:组播,订阅同一个topic，消费者组会相同，一条消息只会被组内一个消费者接收
	//fanout:广播,订阅同一个topic，但是消费者组会使用uuid，所有组都会收到信息
	MessageType string `yaml:"messageType" json:"messageType"`
}

// PulsarMQ instance
type PulsarMQ struct {
	Name                  string
	Publish               chan *mqenv.MQPublishMessage
	Consume               chan *mqenv.MQConsumerProxy
	Done                  chan error
	Close                 chan interface{}
	namespace             string
	config                *Config
	connConfig            *mqenv.MQConnectorConfig
	topicName             string
	client                pulsar.Client
	connecting            bool
	sshTunnel             *sshtunnel.TunnelForwarder
	producers             map[string]pulsar.Producer
	consumers             map[string]consumerWrapper
	pendingConsumers      []*mqenv.MQConsumerProxy
	pendingPublishes      []*mqenv.MQPublishMessage
	waitingResponseTopic  string
	hostName              string
	rpcInstanceName       string
	healthzTopicPrefix    string
	isInstanceRPC         bool
	rpcCallbacks          map[string]*mqenv.MQPublishMessage
	pendingReplies        map[string]mqenv.MQConsumerMessage
	producersMutex        sync.RWMutex
	consumersMutex        sync.RWMutex
	rpcCallbacksMutex     sync.RWMutex
	pendingConsumersMutex sync.RWMutex
	pendingPublishesMutex sync.RWMutex
	pendingRepliesMutex   sync.RWMutex
}

// Equals check if equals
func (me *Config) Equals(to *Config) bool {
	return (me.Topic == to.Topic &&
		me.ConnConfigName == to.ConnConfigName &&
		me.MessageType == to.MessageType)
}

type consumerWrapper struct {
	consumer      pulsar.Consumer
	consumerProxy *mqenv.MQConsumerProxy
}
