package rabbitmq

import (
	"github.com/kevinyjn/gocom/mq/mqenv"

	"github.com/streadway/amqp"
)

const (
	// AMQPReconnectDuration reconnect duration
	AMQPReconnectDuration = 1
	// AMQPQueueStatusFreshDuration queue status refresh duration
	AMQPQueueStatusFreshDuration = 30
)

// AMQPConfig queue config
type AMQPConfig struct {
	Queue           string
	QueueDurable    bool
	BindingExchange bool
	ExchangeName    string
	ExchangeType    string
	BindingKey      string
}

// RabbitConsumerProxy consumer proxy
type RabbitConsumerProxy struct {
	Queue       string
	Callback    AMQPConsumerCallback
	ConsumerTag string
	AutoAck     bool
	Exclusive   bool
	NoLocal     bool
	NoWait      bool
	Arguments   amqp.Table
}

// RabbitPublishingMsg publishing message
type RabbitPublishingMsg struct {
	Body          []byte
	RoutingKey    string
	CorrelationID string             `json:"correlationId"`
	ReplyTo       string             `json:"replyTo"`
	PublishStatus chan mqenv.MQEvent `json:"-"`
	EventLabel    string             `json:"eventLabel"`
	Headers       map[string]string  `json:"headers"`
}

// RabbitQueueStatus queue status
type RabbitQueueStatus struct {
	RefreshingTime int64
	QueueName      string
	Consumers      int
	Messages       int
}

// AMQPConsumerCallback callback
type AMQPConsumerCallback func(amqp.Delivery)

// RabbitMQ instance
type RabbitMQ struct {
	Name       string
	Publish    chan *RabbitPublishingMsg
	Consume    chan *RabbitConsumerProxy
	Done       chan error
	Channel    *amqp.Channel
	Conn       *amqp.Connection
	Config     *AMQPConfig
	ConnConfig *mqenv.MQConnectorConfig
	Close      chan interface{}

	connClosed       chan *amqp.Error
	channelClosed    chan *amqp.Error
	consumers        []*RabbitConsumerProxy
	pendingConsumers []*RabbitConsumerProxy
	pendingPublishes []*RabbitPublishingMsg
	connecting       bool
	queueName        string
	queue            *amqp.Queue
}

// RabbitRPCMQ rpc instance
type RabbitRPCMQ struct {
	Name        string
	Publish     chan *mqenv.MQPublishMessage
	Consume     chan *RabbitConsumerProxy
	Deliveries  <-chan amqp.Delivery
	Done        chan error
	Channel     *amqp.Channel
	Conn        *amqp.Connection
	QueueStatus *RabbitQueueStatus
	Config      *AMQPConfig
	ConnConfig  *mqenv.MQConnectorConfig
	Close       chan interface{}
	RPCType     int

	connClosed       chan *amqp.Error
	channelClosed    chan *amqp.Error
	consumers        []*RabbitConsumerProxy
	pendingConsumers []*RabbitConsumerProxy
	pendingPublishes []*mqenv.MQPublishMessage
	connecting       bool
	queueName        string
	queue            *amqp.Queue
}

// Equals check if equals
func (me *AMQPConfig) Equals(to *AMQPConfig) bool {
	return (me.Queue == to.Queue &&
		me.QueueDurable == to.QueueDurable &&
		me.BindingExchange == to.BindingExchange &&
		me.ExchangeName == to.ExchangeName &&
		me.ExchangeType == to.ExchangeType &&
		me.BindingKey == to.BindingKey)
}

// IsBroadcastExange check if the configure is fanout
func (me *AMQPConfig) IsBroadcastExange() bool {
	return "fanout" == me.ExchangeType
}
