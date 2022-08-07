package mqenv

import (
	"time"
)

// Constants
const (
	DriverTypeAMQP   = "rabbitmq"
	DriverTypeKafka  = "kafka"
	DriverTypePulsar = "pulsar"
	DriverTypeMock   = "mock"

	MQTypeConsumer  = 1
	MQTypePublisher = 2

	MQEventCodeOk     = 0
	MQEventCodeFailed = -1
	MQEventCodeClosed = -9

	MQReconnectSeconds        = 1
	MQQueueStatusFreshSeconds = 30
)

// Parameter Variables
var (
	publishMessageChannelSize = 512
)

// MQEvent event
type MQEvent struct {
	Code    int    `json:"code"`
	Label   string `json:"label"`
	Message string `json:"message"`
}

// MQConnectorConfig connector config
type MQConnectorConfig struct {
	Driver       string `yaml:"driver" json:"driver"`
	Host         string `yaml:"host" json:"host"`
	Port         int    `yaml:"port" json:"port"`
	Path         string `yaml:"virtualHost" json:"virtualHost"`
	User         string `yaml:"username" json:"username"`
	Password     string `yaml:"password" json:"password"`
	Timeout      int    `yaml:"timeout" json:"timeout"`
	Heartbeat    int    `yaml:"heartbeat" json:"heartbeat"`
	SSHTunnelDSN string `yaml:"sshTunnel" json:"sshTunnel"`
}

// MQConsumerMessage consumer message
type MQConsumerMessage struct {
	Driver        string            `json:"driver"`
	Queue         string            `json:"queue"`
	CorrelationID string            `json:"correlationId"`
	ConsumerTag   string            `json:"consumerTag"`
	ReplyTo       string            `json:"replyTo"`
	MessageID     string            `json:"messageId"`
	AppID         string            `json:"appId"`
	UserID        string            `json:"userId"`
	ContentType   string            `json:"contentType"`
	Exchange      string            `json:"exchange"`
	RoutingKey    string            `json:"routingKey"`
	Timestamp     time.Time         `json:"-"`
	Body          []byte            `json:"body"`
	Headers       map[string]string `json:"headers"`
	BindData      interface{}       `json:"-"`
}

// MQPublishMessage publish message
type MQPublishMessage struct {
	Body             []byte                 `json:"body"`
	Exchange         string                 `json:"exchange"`
	RoutingKey       string                 `json:"routingKey"`
	CorrelationID    string                 `json:"correlationId"`
	ReplyTo          string                 `json:"replyTo"`
	MessageID        string                 `json:"messageId"`
	AppID            string                 `json:"appId"`
	UserID           string                 `json:"userId"`
	ContentType      string                 `json:"contentType"`
	PublishStatus    chan MQEvent           `json:"-"`
	EventLabel       string                 `json:"eventLabel"`
	Headers          map[string]string      `json:"headers"`
	Response         chan MQConsumerMessage `json:"-"`
	TimeoutSeconds   int
	callbackDisabled bool
	SkipExchange     bool // if publish a message only to a queue, not bind to exchange
}

// MQConsumerCallback callback
type MQConsumerCallback func(MQConsumerMessage) *MQPublishMessage

// MQConsumerProxy consumer proxy
type MQConsumerProxy struct {
	Queue       string
	Callback    MQConsumerCallback
	ConsumerTag string
	AutoAck     bool
	Exclusive   bool
	NoLocal     bool
	NoWait      bool
	Ready       chan bool // notifies if consumer subscribes ready
}

// GetHeader by key
func (m *MQConsumerMessage) GetHeader(name string) string {
	if nil == m.Headers {
		return ""
	}
	return m.Headers[name]
}

// SetHeader value by key
func (m *MQConsumerMessage) SetHeader(name string, value string) {
	if nil == m.Headers {
		m.Headers = map[string]string{}
	}
	m.Headers[name] = value
}

// OnClosed on close event
func (m *MQPublishMessage) OnClosed() {
	m.callbackDisabled = true
}

// CallbackEnabled is callback enabled
func (m *MQPublishMessage) CallbackEnabled() bool {
	return false == m.callbackDisabled
}

// NewMQResponseMessage new mq response publish messge depends on mq consumer message
func NewMQResponseMessage(body []byte, cm *MQConsumerMessage) *MQPublishMessage {
	pm := &MQPublishMessage{
		Body:    body,
		Headers: map[string]string{},
	}
	if nil != cm {
		pm.AppID = cm.AppID
		pm.RoutingKey = cm.ReplyTo
		pm.CorrelationID = cm.CorrelationID
		pm.ReplyTo = cm.ReplyTo
		pm.MessageID = cm.MessageID
		pm.AppID = cm.MessageID
		pm.UserID = cm.UserID
		pm.ContentType = cm.ContentType
		if nil != cm.Headers {
			for k, v := range cm.Headers {
				pm.Headers[k] = v
			}
		}
	}
	return pm
}

// NewConsumerMessageFromPublishMessage new consumer message from publish message
func NewConsumerMessageFromPublishMessage(pm *MQPublishMessage) MQConsumerMessage {
	msg := MQConsumerMessage{
		Driver:        DriverTypeMock,
		Queue:         "",
		CorrelationID: pm.CorrelationID,
		ConsumerTag:   "",
		ReplyTo:       pm.ReplyTo,
		MessageID:     pm.MessageID,
		AppID:         pm.AppID,
		UserID:        pm.UserID,
		ContentType:   pm.ContentType,
		Exchange:      pm.Exchange,
		RoutingKey:    pm.RoutingKey,
		Timestamp:     time.Now(),
		Body:          pm.Body,
		Headers:       pm.Headers,
		BindData:      nil,
	}
	return msg
}

// GetPublishMessageChannelSize get publishing message channel size for initializing mq publish channel
func GetPublishMessageChannelSize() int {
	return publishMessageChannelSize
}

// SetPublishMessageChannelSize set publishing message channel size for initializing mq publish channel
func SetPublishMessageChannelSize(value int) int {
	if value < 1 {
		value = 1
	}
	if value > 65536 {
		value = 65536
	}
	publishMessageChannelSize = value
	return publishMessageChannelSize
}
