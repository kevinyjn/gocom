package rabbitmq

import (
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/netutils/sshtunnel"

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
	ConnConfigName  string
	Queue           string
	QueueDurable    bool
	BindingExchange bool
	ExchangeName    string
	ExchangeType    string
	BindingKey      string
	QueueAutoDelete bool
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

// RabbitQueueStatus queue status
type RabbitQueueStatus struct {
	RefreshingTime int64
	QueueName      string
	Consumers      int
	Messages       int
}

// AMQPConsumerCallback callback
type AMQPConsumerCallback func(amqp.Delivery) *mqenv.MQPublishMessage

// RabbitMQ instance
type RabbitMQ struct {
	Name        string
	Publish     chan *mqenv.MQPublishMessage
	Consume     chan *RabbitConsumerProxy
	Done        chan error
	Channel     *amqp.Channel
	Conn        *amqp.Connection
	Config      *AMQPConfig
	ConnConfig  *mqenv.MQConnectorConfig
	Close       chan interface{}
	QueueStatus *RabbitQueueStatus

	connClosed       chan *amqp.Error
	channelClosed    chan *amqp.Error
	consumers        []*RabbitConsumerProxy
	pendingConsumers []*RabbitConsumerProxy
	pendingPublishes []*mqenv.MQPublishMessage
	connecting       bool
	queueName        string
	queue            *amqp.Queue
	sshTunnel        *sshtunnel.TunnelForwarder
	afterEnsureQueue func()
	beforePublish    func(*mqenv.MQPublishMessage) (string, string)
	hostName         string
	rpcInstanceName  string
	rpcCallbacks     map[string]*mqenv.MQPublishMessage
	pendingReplies   map[string]amqp.Delivery
}

// RabbitRPC rpc instance
type RabbitRPC struct {
	RabbitMQ
	Deliveries <-chan amqp.Delivery
	RPCType    int
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
