package rabbitmq

import (
	"sync"

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
	Driver          string
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
	_consumed   int
	_lastTimer  int64
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
	Name       string
	Publish    chan *mqenv.MQPublishMessage
	Consume    chan *RabbitConsumerProxy
	Done       chan error
	Channel    *amqp.Channel
	Conn       *amqp.Connection
	Config     *AMQPConfig
	ConnConfig *mqenv.MQConnectorConfig
	Close      chan interface{}

	eventConnClosed     chan *amqp.Error
	eventChannelClosed  chan *amqp.Error
	eventConnBlocked    chan amqp.Blocking
	eventChannelReturn  chan amqp.Return
	eventChannelCancel  chan string
	consumers           map[string]*RabbitConsumerProxy
	pendingConsumers    []*RabbitConsumerProxy
	pendingPublishes    []*mqenv.MQPublishMessage
	deliveryQueue       chan deliveryData
	rpcResponseChannel  chan mqenv.MQConsumerMessage
	connecting          bool
	queueInfo           queueDescribes
	sshTunnel           *sshtunnel.TunnelForwarder
	afterEnsureQueue    func()
	beforePublish       func(*mqenv.MQPublishMessage) (string, string)
	hostName            string
	rpcInstanceName     string
	rpcCallbacks        map[string]*mqenv.MQPublishMessage
	pendingReplies      map[string]amqp.Delivery
	queuesStatus        map[string]*RabbitQueueStatus
	rpcCallbacksMutex   sync.RWMutex
	pendingRepliesMutex sync.RWMutex
	consumersMutex      sync.RWMutex
	queuesStatusMutex   sync.RWMutex
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

// Clone a AMQPConfig object
func (me *AMQPConfig) Clone() AMQPConfig {
	return AMQPConfig{
		ConnConfigName:  me.ConnConfigName,
		Queue:           me.Queue,
		QueueDurable:    me.QueueDurable,
		BindingExchange: me.BindingExchange,
		ExchangeName:    me.ExchangeName,
		ExchangeType:    me.ExchangeType,
		BindingKey:      me.BindingKey,
		QueueAutoDelete: me.QueueAutoDelete,
	}
}

// queueDescribes queue that declared
type queueDescribes struct {
	name        string
	messages    int
	consumers   int
	durable     bool
	autoDelete  bool
	exclusive   bool
	noWait      bool
	isBroadcast bool
	lastName    string
	initialName string
}

func (d *queueDescribes) clear() {
	d.lastName = d.name
	d.name = ""
	d.messages = 0
	d.consumers = 0
	d.durable = false
	d.autoDelete = false
	d.exclusive = false
	d.noWait = false
	d.isBroadcast = false
}

func (d *queueDescribes) copy(s queueDescribes) {
	d.lastName = ""
	d.name = s.name
	d.messages = s.messages
	d.consumers = s.consumers
	d.durable = s.durable
	d.autoDelete = s.autoDelete
	d.exclusive = s.exclusive
	d.noWait = s.noWait
	d.isBroadcast = s.isBroadcast
}

// NotInitialized if the queue describes information were not initialized
func (d *queueDescribes) NotInitialized() bool {
	return d.name == ""
}

type deliveryData struct {
	delivery amqp.Delivery
	callback AMQPConsumerCallback
	autoAck  bool
}
