package mqenv

// Constants
const (
	DriverTypeAMQP  = "rabbitmq"
	DriverTypeKafka = "kafka"

	MQTypeConsumer  = 1
	MQTypePublisher = 2

	MQEventCodeOk     = 0
	MQEventCodeFailed = -1
	MQEventCodeClosed = -9
)

// MQEvent event
type MQEvent struct {
	Code    int    `json:"code"`
	Label   string `json:"label"`
	Message string `json:"message"`
}

// MQConnectorConfig connector config
type MQConnectorConfig struct {
	Driver    string `yaml:"driver"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Path      string `yaml:"virtualHost"`
	User      string `yaml:"username"`
	Password  string `yaml:"password"`
	Timeout   int    `yaml:"timeout"`
	Heartbeat int    `yaml:"heartbeat"`
}

// MQConsumerMessage consumer message
type MQConsumerMessage struct {
	Driver        string      `json:"driver"`
	Queue         string      `json:"queue"`
	CorrelationID string      `json:"correlationId"`
	ConsumerTag   string      `json:"consumerTag"`
	ReplyTo       string      `json:"replyTo"`
	RoutingKey    string      `json:"routingKey"`
	Body          []byte      `json:"body"`
	BindData      interface{} `json:"-"`
}

// MQPublishMessage publish message
type MQPublishMessage struct {
	Body             []byte            `json:"body"`
	RoutingKey       string            `json:"routingKey"`
	CorrelationID    string            `json:"correlationId"`
	ReplyTo          string            `json:"replyTo"`
	PublishStatus    chan MQEvent      `json:"-"`
	EventLabel       string            `json:"eventLabel"`
	Headers          map[string]string `json:"headers"`
	Response         chan []byte
	callbackDisabled bool
}

// MQConsumerCallback callback
type MQConsumerCallback func(MQConsumerMessage)

// MQConsumerProxy consumer proxy
type MQConsumerProxy struct {
	Queue       string
	Callback    MQConsumerCallback
	ConsumerTag string
	AutoAck     bool
	Exclusive   bool
	NoLocal     bool
	NoWait      bool
}

// OnClosed on close event
func (m *MQPublishMessage) OnClosed() {
	m.callbackDisabled = true
}

// CallbackEnabled is callback enabled
func (m *MQPublishMessage) CallbackEnabled() bool {
	return false == m.callbackDisabled
}
