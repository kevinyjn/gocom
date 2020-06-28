package kafka

import (
	"time"

	"github.com/kevinyjn/gocom/mq/mqenv"

	kafka "github.com/segmentio/kafka-go"
)

// Constants
const (
	KafkaReconnectDuration        = 1
	KafkaQueStatusRefreshDuration = 60
)

// Config kafka config
type Config struct {
	Topic   string
	GroupID string
}

// ConsumerProxy kafka
type ConsumerProxy struct {
	Topic       string
	Callback    ConsumerCallback
	ConsumerTag string
	AutoAck     bool
	Exclusive   bool
	NoLocal     bool
	NoWait      bool
}

// ConsumerMessage struct
type ConsumerMessage struct {
	Topic     string
	Partition int
	Key       []byte
	Value     []byte
	Offset    int64
	Headers   map[string][]byte
	Time      time.Time
}

// ConsumerCallback callback
type ConsumerCallback func(ConsumerMessage)

// PublishingMsg publishing msg
type PublishingMsg struct {
	Body          []byte
	Key           []byte
	Topic         string
	Partition     int
	Offset        int64
	Headers       map[string][]byte
	PublishStatus chan mqenv.MQEvent `json:"-"`
	EventLabel    string             `json:"eventLabel"`
}

// Kafka instance
type Kafka struct {
	Name       string
	Publish    chan *PublishingMsg
	Consume    chan *ConsumerProxy
	Done       chan error
	Reader     *kafka.Reader
	Writer     *kafka.Writer
	Config     *Config
	ConnConfig *mqenv.MQConnectorConfig
	Close      chan interface{}

	connClosed       chan error
	channelClosed    chan error
	pendingConsumers []*ConsumerProxy
	pendingPublishes []*PublishingMsg
	connecting       bool
}

// Equals check
func (me *Config) Equals(to *Config) bool {
	return (me.Topic == to.Topic &&
		me.GroupID == to.GroupID)
}

// InstStats stats
type InstStats struct {
	Bytes         int64  `json:"bytes"`
	Dials         int64  `json:"connections"`
	Topic         string `json:"topic"`
	Messages      int64  `json:"messages"`
	Rebalances    int64  `json:"rebalances"`
	Errors        int64  `json:"errors"`
	Timeouts      int64  `json:"timeouts"`
	ClientID      string `json:"clientID"`
	QueueLength   int64  `json:"queueLength"`
	QueueCapacity int64  `json:"queueCapacity"`
}

// Stats struct
type Stats struct {
	Consumer InstStats `json:"consumer"`
	Producer InstStats `json:"producer"`
}
