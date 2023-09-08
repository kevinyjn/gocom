package rabbitmq

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/netutils/pinger"
	"github.com/kevinyjn/gocom/netutils/sshtunnel"
	"github.com/kevinyjn/gocom/utils"

	"github.com/streadway/amqp"
)

// Constants
const (
	ChannelConsumerDeliveryCheckIntervalSeconds = int64(9)
	JinDie                                      = "_jindie_mq" // 获取金蝶环境变量的名字

)

// Variables
var (
	amqpInsts            = map[string]*RabbitMQ{}
	trackerQueue  string = ""
	amqpInstMutex        = sync.RWMutex{}
)

// InitRabbitMQ init
func InitRabbitMQ(mqConnName string, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) (*RabbitMQ, error) {
	amqpInstMutex.RLock()
	amqpInst, ok := amqpInsts[mqConnName]
	amqpInstMutex.RUnlock()
	if ok && !amqpInst.Config.Equals(amqpCfg) {
		amqpInst.close()
		close(amqpInst.Close)
		ok = false
	}
	if !ok {
		amqpInst = NewRabbitMQ(mqConnName, connCfg, amqpCfg)
		amqpInstMutex.Lock()
		amqpInsts[mqConnName] = amqpInst
		amqpInstMutex.Unlock()
		err := amqpInst.init()
		if err == nil {
			go amqpInst.Run()
		} else {
			return nil, err
		}
	}
	return amqpInst, nil
}

// GetRabbitMQ get
func GetRabbitMQ(name string) (*RabbitMQ, error) {
	amqpInstMutex.RLock()
	amqpInst, ok := amqpInsts[name]
	amqpInstMutex.RUnlock()
	if ok {
		return amqpInst, nil
	}
	return nil, fmt.Errorf("RabbitMQ instance by %s not found", name)
}

// SetupTrackerQueue name
func SetupTrackerQueue(queueName string) {
	trackerQueue = queueName
}

func dial(amqpURI string, connDescription string) (*amqp.Connection, error) {
	logger.Info.Printf("rabbit dialing %s ...", connDescription)
	connection, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}
	return connection, nil
}

func createChannel(c *amqp.Connection, amqpCfg *AMQPConfig) (*amqp.Channel, error) {
	logger.Info.Printf("got Connection, getting Channel")
	channel, err := c.Channel()
	if err != nil {
		logger.Error.Printf("Channel create failed with error: %v", err)
		return nil, err
	}

	if amqpCfg.BindingExchange {
		logger.Info.Printf("got Channel, declaring %q Exchange (%q)", amqpCfg.Queue, amqpCfg.ExchangeName)
		if err := channel.ExchangeDeclare(
			amqpCfg.ExchangeName, // name
			amqpCfg.ExchangeType, // type
			amqpCfg.QueueDurable, // durable
			false,                // auto-deleted
			false,                // internal
			false,                // noWait
			nil,                  // arguments
		); err != nil {
			channel.Close()
			logger.Error.Printf("Exchange Declare: %v", err)
			return nil, err
		}
	}
	return channel, nil
}

func inspectQueue(channel *amqp.Channel, amqpCfg *AMQPConfig) (queueDescribes, error) {
	queueInfo := queueDescribes{
		name:        amqpCfg.Queue,
		durable:     amqpCfg.QueueDurable,
		autoDelete:  amqpCfg.QueueAutoDelete,
		exclusive:   false,
		noWait:      false,
		isBroadcast: false,
		initialName: amqpCfg.Queue,
	}
	if amqpCfg.IsBroadcastExange() {
		queueInfo.autoDelete = true
		queueInfo.durable = false
		queueInfo.name = ""
		queueInfo.isBroadcast = true
		// exclusive = true
		if os.Getenv(JinDie) == "1" {
			queueInfo.durable = true
		}
	} else if "" == queueInfo.name {
		queueInfo.autoDelete = true
		queueInfo.durable = false
	}
	queue, err := channel.QueueDeclare(
		queueInfo.name,       // name of the queue
		queueInfo.durable,    // durable
		queueInfo.autoDelete, // delete when usused
		queueInfo.exclusive,  // exclusive
		queueInfo.noWait,     // noWait
		nil,                  // arguments
	)
	if nil == err {
		queueInfo.name = queue.Name
		queueInfo.messages = queue.Messages
		queueInfo.consumers = queue.Consumers
	}
	return queueInfo, err
}

func createQueue(channel *amqp.Channel, amqpCfg *AMQPConfig) (queueDescribes, error) {
	if "" == amqpCfg.ExchangeName {
		logger.Info.Printf("declaring Queue %q", amqpCfg.Queue)
	} else {
		logger.Info.Printf("declared Exchange %s(%s), declaring Queue %q", amqpCfg.ExchangeName, amqpCfg.ExchangeType, amqpCfg.Queue)
	}
	queueInfo, err := inspectQueue(channel, amqpCfg)
	if err != nil {
		logger.Error.Printf("Queue declare: %v", err)
		return queueInfo, fmt.Errorf("Queue declare: %v", err)
	}

	if amqpCfg.BindingExchange {
		logger.Info.Printf("declared Queue (%q %d messages, %d consumers), binding to Exchange %s (key %q)",
			queueInfo.name, queueInfo.messages, queueInfo.consumers, amqpCfg.ExchangeName, amqpCfg.BindingKey)
		if err = channel.QueueBind(
			queueInfo.name,       // name of the queue
			amqpCfg.BindingKey,   // bindingKey
			amqpCfg.ExchangeName, // sourceExchange
			false,                // noWait
			nil,                  // arguments
		); err != nil {
			logger.Error.Printf("Queue bind failed with error:%v", err)
			return queueInfo, fmt.Errorf("Queue bind: %v", err)
		}
	} else {
		logger.Info.Printf("declared Queue (%q %d messages, %d consumers)",
			queueInfo.name, queueInfo.messages, queueInfo.consumers)
	}
	return queueInfo, nil
}

// NewRabbitMQ with parameters
func NewRabbitMQ(mqConnName string, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitMQ {
	r := &RabbitMQ{}
	r.initWithParameters(mqConnName, connCfg, amqpCfg)
	return r
}

func (r *RabbitMQ) initWithParameters(mqConnName string, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) {
	r.Name = mqConnName
	r.Config = amqpCfg
	r.ConnConfig = connCfg
	r.Publish = make(chan *mqenv.MQPublishMessage, mqenv.GetPublishMessageChannelSize())
	// r.Publish = make(chan *mqenv.MQPublishMessage)
	r.Consume = make(chan *RabbitConsumerProxy)
	r.Done = make(chan error)
	r.Close = make(chan interface{})
	r.queuesStatus = map[string]*RabbitQueueStatus{
		amqpCfg.Queue: &RabbitQueueStatus{
			QueueName:      amqpCfg.Queue,
			RefreshingTime: 0,
		},
	}
	r.consumers = map[string]*RabbitConsumerProxy{}
	r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
	r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
	r.rpcResponseChannel = make(chan mqenv.MQConsumerMessage)
	r.connecting = false
	r.afterEnsureQueue = r.ensurePendings
	r.rpcInstanceName = fmt.Sprintf("@rpc-%s", r.Config.ConnConfigName)
	r.rpcCallbacks = make(map[string]*mqenv.MQPublishMessage)
	r.pendingReplies = make(map[string]amqp.Delivery)
	r.rpcCallbacksMutex = sync.RWMutex{}
	r.pendingRepliesMutex = sync.RWMutex{}
	r.consumersMutex = sync.RWMutex{}
	hostName, err := os.Hostname()
	if nil != err {
		logger.Error.Printf("RabbitMQ %s initialize while get hostname failed with error:%v", r.Name, err)
	} else {
		r.hostName = hostName
	}
}

func (r *RabbitMQ) init() error {
	logger.Info.Printf("Initializing amqp instance:%s", r.Name)
	return r.initConn()
}

// Run start
// 1. init the rabbitmq conneciton
// 2. expect messages from the message hub on the Publish channel
// 3. if the connection is closed, try to restart it
func (r *RabbitMQ) Run() {
	tick := time.NewTicker(time.Second * 2)
	for {
		if r.connecting == false && r.Conn == nil {
			r.consumersMutex.RLock()
			if len(r.consumers) > 0 {
				r.consumersMutex.RUnlock()
				r.consumersMutex.Lock()
				if nil == r.pendingConsumers {
					r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
				}
				for _, cm := range r.consumers {
					r.pendingConsumers = append(r.pendingConsumers, cm)
				}
				r.consumers = map[string]*RabbitConsumerProxy{}
				r.consumersMutex.Unlock()
			} else {
				r.consumersMutex.RUnlock()
			}
			logger.Trace.Printf("rabbitmq %s pre running...", r.Name)
			r.initConn()
			// // backstop
			// if r.Conn != nil && !r.Conn.IsClosed() {
			// 	if err := r.Channel.Cancel("", true); err != nil {
			// 		logger.Error.Printf("RabbitMQ %s cancel channel failed with error:%v", r.Name, err)
			// 	}
			// 	if err := r.Conn.Close(); err != nil {
			// 		logger.Error.Printf("RabbitMQ %s close connection failed with error:%v", r.Name, err)
			// 	}
			// }

			// // IMPORTANT: 必须清空 Notify，否则死连接不会释放
			// r.clearNotifyChan()
			logger.Trace.Printf("rabbitmq %s do running...", r.Name)
		}

		select {
		case pm := <-r.Publish:
			r.publish(pm)
		case cm := <-r.Consume:
			if "" != r.queueInfo.name {
				cm.Queue = r.queueInfo.name
			}
			logger.Info.Printf("consuming queue: %s\n", cm.Queue)
			r.consume(cm)
		case err := <-r.Done:
			logger.Error.Printf("RabbitMQ connection:%s done with error:%v", r.Name, err)
			if r.connecting == false {
				r.queueInfo.clear()
				r.close()
				break
			}
			break
		case sts := <-r.eventConnBlocked:
			if sts.Active {
				logger.Info.Printf("RabbitMQ connection:%s blocking actived with reason:%s", r.Name, sts.Reason)
			} else {
				logger.Warning.Printf("RabbitMQ connection:%s blocked with reason:%s", r.Name, sts.Reason)
			}
			break
		case sts := <-r.eventChannelReturn:
			logger.Warning.Printf("RabbitMQ connection:%s, a message with routingKey:%s were returned with reason:%s", r.Name, sts.RoutingKey, sts.ReplyText)
			break
		case reason := <-r.eventChannelCancel:
			logger.Error.Printf("RabbitMQ connection:%s channel were canceled with reason:%s", r.Name, reason)
			r.queueInfo.clear()
			r.close()
			break
		case err := <-r.eventConnClosed:
			logger.Error.Printf("RabbitMQ connection:%s closed with error:%v", r.Name, err)
			r.queueInfo.clear()
			r.Channel = nil
			r.close()
			break
		case err := <-r.eventChannelClosed:
			logger.Error.Printf("RabbitMQ channel:%s closed with error:%v", r.Name, err)
			r.queueInfo.clear()
			r.Channel = nil
			r.close()
			break
		case <-r.Close:
			logger.Warning.Printf("RabbitMQ %s got an event that closing the connection", r.Name)
			r.queueInfo.clear()
			r.close()
			tick.Stop()
			return
		case <-tick.C:
			// fmt.Println("amqp tick...")
			if nil == r.Conn {
				break
			}
			if r.Conn.IsClosed() {
				r.Conn = nil
				r.queueInfo.clear()
				r.connecting = false
				logger.Error.Printf("RabbitMQ connection:%s were closed on ticker checking", r.Name)
				break
			} else if nil == r.Channel {
				logger.Warning.Printf("RabbitMQ channel:%s were nil", r.Name)
			} else {
				// check consumer were abnormal consuming message
				shouldReconnect := false
				r.consumersMutex.RLock()
				if nil != r.consumers {
					for qname, cm := range r.consumers {
						// v := os.Getenv(JinDie)
						// fmt.Println(v)
						if os.Getenv(JinDie) == "1" {
							// 因为金蝶的mq 在调用 QueueInspect 后并不会返回队列订阅者的数量，所以暂时屏蔽掉这个。等对方加好
							continue
						}
						q, err := r.Channel.QueueInspect(qname)
						now := time.Now().Unix()
						interval := now - cm._lastTimer
						if nil == err {
							// logger.Debug.Printf("RabbitMQ connection:%s queue:%s consumers:%d and queued messages:%d", r.Name, qname, q.Consumers, q.Messages)
							if 0 >= q.Consumers || (0 < q.Messages && 0 == cm._consumed) {
								if ChannelConsumerDeliveryCheckIntervalSeconds < interval {
									logger.Warning.Printf("RabbitMQ connection:%s with queue:%s detected that has %d consumers and queued messages:%d without delivering consuming messages in %d seconds, would trying close connection and reconnect.", r.Name, qname, q.Consumers, q.Messages, interval)
									shouldReconnect = true
								}
							}
						} else {
							logger.Error.Printf("RabbitMQ connection:%s inspect queue:%s info failed with error:%v, would closing the connection and reconnect", r.Name, qname, err)
							shouldReconnect = true
						}
						if ChannelConsumerDeliveryCheckIntervalSeconds < interval {
							if 30 < cm._consumed {
								logger.Info.Printf("RabbitMQ connection:%s queue:%s consumed %d messages in past %d seconds", r.Name, qname, cm._consumed, interval)
							}
							cm._consumed = 0
							cm._lastTimer = now
						}
					}
				}
				if shouldReconnect {
					for _, cm := range r.consumers {
						logger.Warning.Printf("RabbitMQ connection:%s detected that should reconnect, now cancel consumer:%s", r.Name, cm.ConsumerTag)
						r.Channel.Cancel(cm.ConsumerTag, true)
					}
				}
				r.consumersMutex.RUnlock()
				if shouldReconnect {
					logger.Warning.Printf("RabbitMQ connection:%s detected that should reconnect the connection", r.Name)
					r.close()
				}
			}
			break
		}
	}
}

func (r *RabbitMQ) clearNotifyChan() {
	if r.eventChannelClosed != nil {
		logger.Info.Printf("Clearing RabbitMQ:%s channel closed event", r.Name)
		// for err := range r.eventChannelClosed {
		// 	logger.Warning.Printf("Clearing RabbitMQ:%s channel closed event:%v", r.Name, err)
		// }
		// close(r.eventChannelClosed)
		r.eventChannelClosed = nil
	}
	if r.eventConnClosed != nil {
		logger.Info.Printf("Clearing RabbitMQ:%s connection closed event", r.Name)
		// for err := range r.eventConnClosed {
		// 	logger.Warning.Printf("Clearing RabbitMQ:%s connection closed event:%v", r.Name, err)
		// }
		// close(r.eventConnClosed)
		r.eventConnClosed = nil
	}
	if r.eventConnBlocked != nil {
		// for _ := range r.eventConnBlocked {
		// 	//
		// }
		// close(r.eventConnBlocked)
		r.eventConnBlocked = nil
	}
	r.eventChannelReturn = nil
	r.eventChannelCancel = nil
}

func (r *RabbitMQ) close() {
	r.connecting = false
	logger.Info.Printf("RabbitMQ connection:%s closing", r.Name)
	r.clearNotifyChan()
	if r.Channel != nil {
		logger.Info.Printf("RabbitMQ connection:%s closing channel", r.Name)
		r.Channel.Close()
		r.Channel = nil
		logger.Info.Printf("RabbitMQ connection:%s closing channel done", r.Name)
	}
	if r.Conn != nil && !r.Conn.IsClosed() {
		logger.Info.Printf("RabbitMQ connection:%s closing connection", r.Name)
		err := r.Conn.Close()
		if nil != err {
			logger.Warning.Printf("RabbitMQ connection:%s closing connection got error:%v", r.Name, err)
		}
		logger.Info.Printf("RabbitMQ connection:%s closing connection done", r.Name)
	}
	if nil != r.sshTunnel {
		logger.Info.Printf("RabbitMQ connection:%s closing ssh tunnel", r.Name)
		r.sshTunnel.Stop()
		r.sshTunnel = nil
		logger.Info.Printf("RabbitMQ connection:%s closing ssh tunnel done", r.Name)
	}
	r.Conn = nil
	r.eventConnClosed = nil
	r.eventChannelClosed = nil
	r.eventConnBlocked = nil
	r.eventChannelReturn = nil
	r.eventChannelCancel = nil
	logger.Info.Printf("RabbitMQ connection:%s closing finished", r.Name)
}

// try to start a new connection, channel and deliveries channel. if failed, try again in 5 sec.
func (r *RabbitMQ) initConn() error {
	if r.ConnConfig.Driver != mqenv.DriverTypeAMQP {
		logger.Error.Printf("Initialize rabbitmq connection by configure:%s failed, the configure driver:%s does not fit.", r.Name, r.ConnConfig.Driver)
		return errors.New("Invalid driver for rabbitmq")
	}

	r.connecting = true
	connDSN, connDescription, err := r.formatConnectionDSN()
	logger.Info.Printf("RabbitMQ %s formatted connection dsn:%s", r.Name, connDescription)
	if nil != err {
		r.connecting = false
		logger.Error.Printf("Initialize rabbitmq connection by configure:%s while format amqp conneciton DSN failed with error:%v", r.Name, err)
		return err
	}

	func() {
		ticker := time.NewTicker(AMQPReconnectDuration * time.Second)
		for nil != ticker {
			select {
			case <-ticker.C:
				if conn, err := dial(connDSN, connDescription); err != nil {
					// r.connecting = false
					// logger.Error.Println(err)
					// logger.Error.Println("node will only be able to respond to local connections")
					logger.Error.Printf("RabbitMQ %s connecting %s failed with error:%v trying to reconnect in %d seconds...", r.Name, connDescription, err, AMQPReconnectDuration)
				} else {
					logger.Info.Printf("RabbitMQ %s connecting %s succeed", r.Name, connDescription)
					r.connecting = false
					r.Conn = conn
					r.eventConnClosed = make(chan *amqp.Error)
					r.Conn.NotifyClose(r.eventConnClosed)
					r.eventConnBlocked = make(chan amqp.Blocking)
					r.Conn.NotifyBlocked(r.eventConnBlocked)
					ticker.Stop()
					ticker = nil
					r.Channel, err = createChannel(conn, r.Config)
					if err != nil {
						conn.Close()
						r.Conn = nil
						logger.Fatal.Printf("RabbitMQ %s create channel failed with error:%v", r.Name, err)
						return
					}
					// r.Channel.Qos(128, 2048000, false)
					r.eventChannelClosed = make(chan *amqp.Error)
					r.Channel.NotifyClose(r.eventChannelClosed)
					r.eventChannelReturn = make(chan amqp.Return)
					r.Channel.NotifyReturn(r.eventChannelReturn)
					r.eventChannelCancel = make(chan string)
					r.Channel.NotifyCancel(r.eventChannelCancel)

					r.consumersMutex.RLock()
					if r.Config.IsBroadcastExange() && len(r.consumers) <= 0 && len(r.pendingConsumers) <= 0 {
						r.consumersMutex.RUnlock()
						break
					} else {
						r.consumersMutex.RUnlock()
					}
					err = r.ensureQueue()
					if nil != err {
						return
					}
				}
			}
		}
	}()
	return nil
}

func (r *RabbitMQ) formatConnectionDSN() (string, string, error) {
	cnf := r.ConnConfig
	host := cnf.Host
	port := cnf.Port
	var err error
	if "" != cnf.SSHTunnelDSN && !pinger.Connectable(host, port) {
		if nil != r.sshTunnel {
			r.sshTunnel.Stop()
			r.sshTunnel = nil
		}
		for {
			var sshTunnel *sshtunnel.TunnelForwarder
			sshTunnel, err = sshtunnel.NewSSHTunnel(cnf.SSHTunnelDSN, host, port)
			err = sshTunnel.ParseFromDSN(cnf.SSHTunnelDSN)
			if nil != err {
				logger.Error.Printf("format rabbitmq address while parse SSH Tunnel DSN:%s failed with error:%v", cnf.SSHTunnelDSN, err)
				break
			}

			err = sshTunnel.Start()
			if nil != err {
				logger.Error.Printf("format rabbitmq address while start SSH Tunnel failed with error:%v", err)
				break
			}
			r.sshTunnel = sshTunnel
			host = sshTunnel.LocalHost()
			port = sshTunnel.LocalPort()
			break
		}
	}

	connDSN := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", cnf.User, cnf.Password, host, port, cnf.Path)
	connDescription := fmt.Sprintf("amqp://%s:***@%s:%d/%s", cnf.User, cnf.Host, cnf.Port, cnf.Path)
	return connDSN, connDescription, err
}

func (r *RabbitMQ) ensureQueue() error {
	if nil == r.Conn {
		return fmt.Errorf("RabbitMQ connection were not connected when ensuring queue:%s", r.Config.Queue)
	}
	queueInfo, err := createQueue(r.Channel, r.Config)
	if err != nil {
		r.close()
		logger.Fatal.Printf("RabbitMQ %s create queue:%s failed with error:%v", r.Name, r.Config.Queue, err)
		return err
	}
	if "" != queueInfo.initialName {
		r.queuesStatusMutex.Lock()
		queueStatus, ok := r.queuesStatus[queueInfo.initialName]
		if false == ok || nil == queueStatus {
			queueStatus = &RabbitQueueStatus{
				QueueName:      queueInfo.name,
				RefreshingTime: 0,
			}
			r.queuesStatus[queueInfo.initialName] = queueStatus
		}
		r.queuesStatusMutex.Unlock()
		queueStatus.QueueName = queueInfo.name
		queueStatus.Consumers = queueInfo.consumers
		queueStatus.Messages = queueInfo.messages
		queueStatus.RefreshingTime = time.Now().Unix()
	}
	r.queueInfo.initialName = r.Config.Queue
	r.queueInfo.copy(queueInfo)
	if nil != r.afterEnsureQueue {
		r.afterEnsureQueue()
	}
	if "" != r.queueInfo.lastName && r.queueInfo.name != r.queueInfo.lastName && r.queueInfo.autoDelete {
		var purged int
		if r.queueInfo.isBroadcast {
			purged, err = r.Channel.QueueDelete(r.queueInfo.lastName, true, false, false)
		} else {
			purged, err = r.Channel.QueueDelete(r.queueInfo.lastName, true, true, false)
		}
		if nil == err {
			logger.Info.Printf("RabbitMQ %s resumed queue:%s and remove unused queue:%s succeed with %d messages purged", r.Name, r.queueInfo.name, r.queueInfo.lastName, purged)
		} else {
			logger.Info.Printf("RabbitMQ %s resumed queue:%s and remove unused queue:%s failed with error:%v", r.Name, r.queueInfo.name, r.queueInfo.lastName, err)
		}
		r.queueInfo.lastName = ""
	}
	return nil
}

func (r *RabbitMQ) ensurePendings() {
	if r.pendingConsumers != nil && len(r.pendingConsumers) > 0 {
		consumers := r.pendingConsumers
		r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
		for _, cm := range consumers {
			cm.Queue = r.queueInfo.name
			r.consume(cm)
		}
	}
	if r.pendingPublishes != nil && len(r.pendingPublishes) > 0 {
		publishes := r.pendingPublishes
		r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
		for _, pm := range publishes {
			r.publish(pm)
		}
	}
}

func (r *RabbitMQ) publish(pm *mqenv.MQPublishMessage) error {
	if r.Channel == nil {
		logger.Warning.Printf("pending publishing %dB body (%s)", len(pm.Body), utils.HumanByteText(pm.Body))
		r.pendingPublishes = append(r.pendingPublishes, pm)
		return nil
	}
	if r.queueInfo.NotInitialized() && !r.Config.IsBroadcastExange() {
		err := r.ensureQueue()
		if nil != err {
			return err
		}
	}

	exchangeName, routingKey := pm.Exchange, pm.RoutingKey
	if nil != r.beforePublish { // specially rpc instance for old version
		exchangeName, routingKey = r.beforePublish(pm)
	} else if nil != pm.Response {
		rpc, err := r.getRPCInstance()
		if nil != err {
			logger.Error.Printf("RabbitMQ %s publish message while get rpc callback instance failed with error:%v", r.Name, err)
		} else {
			rpc.ensureRPCMessage(pm)
		}
	}
	if pm.SkipExchange || ("" != pm.CorrelationID && false == r.isReplyNeededMessageAnswered(pm.CorrelationID)) {
		exchangeName = ""
	} else if "" == exchangeName {
		exchangeName = r.Config.ExchangeName
	}
	if "" == routingKey {
		routingKey = r.queueInfo.name
		exchangeName = ""
	}
	if logger.IsDebugEnabled() {
		if false == strings.HasPrefix(routingKey, "healthz") {
			logger.Trace.Printf("publishing message(%s) to %s(%s) with %dB body (%s)", pm.CorrelationID, routingKey, exchangeName, len(pm.Body), utils.HumanByteText(pm.Body))
		}
	}

	headers := amqp.Table{}
	for k, v := range pm.Headers {
		headers[k] = v
	}
	err := r.Channel.Publish(
		exchangeName, // publish to an exchange
		routingKey,   // routing to 0 or more queues
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			Headers:         headers,
			ContentType:     pm.ContentType,
			ContentEncoding: "",
			Body:            pm.Body,
			CorrelationId:   pm.CorrelationID,
			ReplyTo:         pm.ReplyTo,
			MessageId:       pm.MessageID,
			AppId:           pm.AppID,
			UserId:          pm.UserID,
			Timestamp:       time.Now(),
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	)
	if "" != pm.CorrelationID {
		r.answerReplyNeededMessage(pm.CorrelationID)
		if "" != trackerQueue {
			r.publishTrackerMQMessage(pm, strconv.Itoa(int(mqenv.MQTypePublisher)))
		}
	}
	if nil != pm.PublishStatus {
		status := mqenv.MQEvent{
			Code:    mqenv.MQEventCodeOk,
			Label:   pm.EventLabel,
			Message: "Publish success",
		}
		if nil != err {
			status.Code = mqenv.MQEventCodeFailed
			status.Message = err.Error()
		}
		pm.PublishStatus <- status
	}
	if err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}
	return nil
}

func (r *RabbitMQ) consume(cm *RabbitConsumerProxy) error {
	if nil == r.deliveryQueue {
		r.ensureDeliveryQueue()
	}
	if r.Channel == nil {
		logger.Warning.Printf("Consuming queue:%s failed while the channel not ready, pending.", cm.Queue)
		r.pendingConsumers = append(r.pendingConsumers, cm)
		return nil
	}
	if r.queueInfo.NotInitialized() {
		err := r.ensureQueue()
		if nil != err {
			return err
		}
	}
	queueName := r.queueInfo.name
	if cm.Queue != r.queueInfo.name && cm.Queue != r.queueInfo.initialName {
		cfg := AMQPConfig{
			ConnConfigName:  r.Config.ConnConfigName,
			Queue:           cm.Queue,
			QueueDurable:    r.Config.QueueDurable,
			QueueAutoDelete: false,
		}
		queueInfo, err := createQueue(r.Channel, &cfg)
		if err != nil {
			logger.Error.Printf("RabbitMQ %s create queue:%s failed with error:%v", r.Name, cfg.Queue, err)
			return err
		}
		queueName = queueInfo.name
	}

	r.consumersMutex.Lock()
	if nil == r.consumers {
		r.consumers = map[string]*RabbitConsumerProxy{}
	}
	cm._consumed = 0
	cm._lastTimer = time.Now().Unix()
	r.consumers[queueName] = cm
	r.consumersMutex.Unlock()

	deliveries, err := r.Channel.Consume(
		queueName,      // name
		cm.ConsumerTag, // consumerTag,
		false,          // autoAck
		cm.Exclusive,   // exclusive
		cm.NoLocal,     // noLocal
		cm.NoWait,      // noWait
		cm.Arguments,   // arguments
	)
	if err != nil {
		logger.Error.Printf("consuming mq(%s) queue:%s and tag:%s with declared queue:%s failed with error:%v", r.Name, cm.Queue, cm.ConsumerTag, queueName, err)
		return err
	}
	logger.Info.Printf("Now consuming mq(%s) queue:%s and tag:%s with declared queue:%s ...", r.Name, cm.Queue, cm.ConsumerTag, queueName)
	go r.handleConsumes(cm.Callback, cm.AutoAck, deliveries, cm)
	return nil
}

func (r *RabbitMQ) handleConsumes(cb AMQPConsumerCallback, autoAck bool, deliveries <-chan amqp.Delivery, cm *RabbitConsumerProxy) {
	for d := range deliveries {
		if logger.IsDebugEnabled() {
			// skip the healthz checking log
			if false == strings.HasPrefix(d.RoutingKey, "healthz") {
				msgLen := len(d.Body)
				if 4096 > msgLen {
					logger.Trace.Printf("got %dB delivery: [%v] from rk:%s(%s) %s",
						msgLen, d.DeliveryTag, d.RoutingKey, d.Exchange, utils.HumanByteText(d.Body))
				} else {
					logger.Trace.Printf("got large message %dB delivery: [%v] from rk:%s(%s)",
						msgLen, d.DeliveryTag, d.RoutingKey, d.Exchange)
				}
			}
		}
		// fmt.Println("---- got delivery message d:", d)
		dd := deliveryData{delivery: d, callback: cb, autoAck: autoAck}
		if nil == r.deliveryQueue {
			r.ensureDeliveryQueue()
		}
		r.deliveryQueue <- dd
		// go r.handleConsumeCallback(d, cb, autoAck)
		if nil != cm {
			cm._consumed++
		}
	}
	r.Done <- fmt.Errorf("error: deliveries channel closed")
}

func (r *RabbitMQ) processConsumerDeliveries() {
	for dd := range r.deliveryQueue {
		r.handleConsumeCallback(dd.delivery, dd.callback, dd.autoAck)
	}
	r.deliveryQueue = nil
}

func (r *RabbitMQ) getRPCInstance() (*RabbitMQ, error) {
	if nil == r.ConnConfig || nil == r.Config {
		logger.Error.Printf("%s rabbitmq instance get RPC instance failed that connection configuration empty", r.Name)
		return nil, fmt.Errorf("connection configuration nil")
	}
	amqpInstMutex.RLock()
	rpcInst, ok := amqpInsts[r.rpcInstanceName]
	amqpInstMutex.RUnlock()
	if ok {
		return rpcInst, nil
	}
	durable := false
	if os.Getenv(JinDie) == "1" {
		durable = true
	}

	config := &AMQPConfig{
		ConnConfigName:  r.Config.ConnConfigName,
		Queue:           fmt.Sprintf("rpc-%s-%s", r.hostName, utils.RandomString(8)),
		ExchangeName:    "",
		ExchangeType:    "topic",
		QueueDurable:    durable,
		BindingExchange: false,
		BindingKey:      "",
		QueueAutoDelete: true,
	}
	rpcInst = NewRabbitMQ(r.rpcInstanceName, r.ConnConfig, config)
	err := rpcInst.init()
	if err == nil {
		queue, err := inspectQueue(r.Channel, config)
		if nil != err {
			logger.Error.Printf("try inspect queue before %s rpc initialized failed with error:%v", rpcInst.Name, err)
			rpcInst.queueInfo.name = config.Queue
		} else {
			rpcInst.queueInfo.copy(queue)
		}
		go rpcInst.Run()
	} else {
		return nil, err
	}
	pxy := &RabbitConsumerProxy{
		Queue:       rpcInst.queueInfo.name,
		Callback:    rpcInst.handleRPCCallback,
		ConsumerTag: rpcInst.queueInfo.name,
		AutoAck:     true,
		Exclusive:   false,
		NoLocal:     false,
		NoWait:      false,
		// Arguments:   consumeProxy.Arguments,
	}
	err = rpcInst.consume(pxy)
	if nil != err {
		logger.Error.Printf("%s consume rpc failed with error:%v", rpcInst.Name, err)
	}
	amqpInstMutex.Lock()
	amqpInsts[r.rpcInstanceName] = rpcInst
	amqpInstMutex.Unlock()
	return rpcInst, nil
}

func (r *RabbitMQ) ensureRPCMessage(pm *mqenv.MQPublishMessage) {
	if "" == pm.CorrelationID {
		pm.CorrelationID = utils.GenLoweruuid()
	}
	pm.ReplyTo = r.queueInfo.name
	r.rpcCallbacksMutex.Lock()
	if nil == r.rpcCallbacks {
		r.rpcCallbacks = make(map[string]*mqenv.MQPublishMessage)
	}
	originPm, _ := r.rpcCallbacks[pm.CorrelationID]
	if nil == originPm || originPm.Response == nil {
		r.rpcCallbacks[pm.CorrelationID] = pm
	}
	r.rpcCallbacksMutex.Unlock()
}

func (r *RabbitMQ) ensureDeliveryQueue() {
	if nil == r.deliveryQueue {
		r.deliveryQueue = make(chan deliveryData, 1024)
		go r.processConsumerDeliveries()
	}
}

func (r *RabbitMQ) handleConsumeCallback(d amqp.Delivery, cb AMQPConsumerCallback, autoAck bool) {
	if cb != nil {
		if "" != d.CorrelationId && "" != d.ReplyTo {
			r.setReplyNeededMessage(d)
			resp := cb(d)
			if nil != resp {
				if false == r.isReplyNeededMessageAnswered(d.CorrelationId) {
					// publish response
					resp.ReplyTo = d.ReplyTo
					r.publish(resp)
				}
			}
			if "" != trackerQueue {
				cmsg := r.generatePublishMessageByDelivery(&d, trackerQueue, []byte{})
				r.publishTrackerMQMessage(cmsg, strconv.Itoa(int(mqenv.MQTypeConsumer)))
			}
		} else {
			cb(d)
		}
	}
	if autoAck == false {
		d.Ack(false)
	}
}

// QueryRPC publishes a message and waiting the response
func (r *RabbitMQ) QueryRPC(pm *mqenv.MQPublishMessage) (*mqenv.MQConsumerMessage, error) {
	// pm.Response = make(chan mqenv.MQConsumerMessage)
	pm.Response = r.rpcResponseChannel
	r.Publish <- pm

	var timer *time.Timer
	var resp *mqenv.MQConsumerMessage
	var err error
	if pm.TimeoutSeconds > 0 {
		timer = time.NewTimer(time.Duration(pm.TimeoutSeconds) * time.Second)
	} else {
		timer = time.NewTimer(30 * time.Second)
	}
	select {
	case res := <-pm.Response:
		if logger.IsDebugEnabled() {
			logger.Trace.Printf("RabbitMQ %s Got response %s", r.Name, utils.HumanByteText(res.Body))
		}
		resp = &res
		timer.Stop()
		break
	case <-timer.C:
		pm.OnClosed()
		err = fmt.Errorf("Query timeout")
		timer.Stop()
		break
	}
	return resp, err
}

func (r *RabbitMQ) handleRPCCallback(d amqp.Delivery) *mqenv.MQPublishMessage {
	if nil == r.rpcCallbacks {
		return nil
	}
	correlationID := d.CorrelationId
	r.rpcCallbacksMutex.RLock()
	pm, pmExists := r.rpcCallbacks[correlationID]
	r.rpcCallbacksMutex.RUnlock()
	if pmExists {
		if pm.CallbackEnabled() {
			// fmt.Printf("====> push back response data %s %+v\n", correlationID, pm)
			resp := generateMQResponseMessage(&d, r.Config.ExchangeName)
			resp.Queue = r.queueInfo.name
			resp.ReplyTo = pm.ReplyTo
			pm.Response <- *resp
		}
		r.rpcCallbacksMutex.Lock()
		delete(r.rpcCallbacks, correlationID)
		r.rpcCallbacksMutex.Unlock()
		d.Ack(false)
	} else {
		d.Ack(false)
	}
	return nil
}

func (r *RabbitMQ) setReplyNeededMessage(d amqp.Delivery) {
	if "" != d.CorrelationId && "" != d.ReplyTo {
		r.pendingRepliesMutex.Lock()
		if nil == r.pendingReplies {
			r.pendingReplies = make(map[string]amqp.Delivery)
		}
		r.pendingReplies[d.CorrelationId] = d
		r.pendingRepliesMutex.Unlock()
	}
}

func (r *RabbitMQ) answerReplyNeededMessage(correlationID string) {
	r.pendingRepliesMutex.Lock()
	if "" != correlationID && nil != r.pendingReplies {
		_, exists := r.pendingReplies[correlationID]
		if exists {
			delete(r.pendingReplies, correlationID)
		}
	}
	r.pendingRepliesMutex.Unlock()
}

func (r *RabbitMQ) isReplyNeededMessageAnswered(correlationID string) bool {
	if "" == correlationID || nil == r.pendingReplies {
		return true
	}
	r.pendingRepliesMutex.RLock()
	_, exists := r.pendingReplies[correlationID]
	r.pendingRepliesMutex.RUnlock()
	return false == exists
}

func (r *RabbitMQ) publishTrackerMQMessage(pm *mqenv.MQPublishMessage, typ string) {
	if "" == trackerQueue || nil == r.Channel {
		return
	}
	headers := amqp.Table{}
	for k, v := range pm.Headers {
		headers[k] = v
	}
	headers["ReplyTo"] = pm.ReplyTo
	headers["Type"] = typ
	err := r.Channel.Publish(
		"",           // publish to an exchange
		trackerQueue, // routing to 0 or more queues
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			Headers:         headers,
			ContentType:     pm.ContentType,
			ContentEncoding: "",
			Body:            []byte{},
			CorrelationId:   pm.CorrelationID,
			ReplyTo:         "",
			MessageId:       pm.MessageID,
			AppId:           pm.AppID,
			UserId:          pm.UserID,
			Timestamp:       time.Now(),
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	)
	if nil != err {
		logger.Error.Printf("publish tracker message(%s) to %s failed with error:%v", pm.CorrelationID, trackerQueue, err)
	}
}

func (r *RabbitMQ) generatePublishMessageByDelivery(d *amqp.Delivery, queueName string, body []byte) *mqenv.MQPublishMessage {
	headers := map[string]string{}
	if nil != d.Headers {
		for k, v := range d.Headers {
			headers[k] = utils.ToString(v)
		}
	}
	pub := &mqenv.MQPublishMessage{
		Body:          body,
		RoutingKey:    queueName,
		CorrelationID: d.CorrelationId,
		ReplyTo:       d.ReplyTo,
		MessageID:     d.MessageId,
		AppID:         d.AppId,
		UserID:        d.UserId,
		ContentType:   d.ContentType,
		Headers:       headers,
	}
	return pub
}

// CheckQueueConsumers check if queue has consumers listening
func (r *RabbitMQ) CheckQueueConsumers(queueName string) int {
	if "" == queueName {
		logger.Error.Printf("RabbitMQ %s check queue:%s consumers while giving empty queue name", r.Name, queueName)
		return 0
	}
	r.queuesStatusMutex.RLock()
	queueStatus, ok := r.queuesStatus[queueName]
	r.queuesStatusMutex.RUnlock()
	if false == ok || nil == queueStatus {
		queueStatus = &RabbitQueueStatus{
			QueueName:      queueName,
			RefreshingTime: 0,
		}
		r.queuesStatusMutex.Lock()
		r.queuesStatus[queueName] = queueStatus
		r.queuesStatusMutex.Unlock()
	}
	now := time.Now().Unix()
	if (queueStatus.Consumers < 1 && now-queueStatus.RefreshingTime > 1) || now-queueStatus.RefreshingTime > AMQPQueueStatusFreshDuration {
		queue, err := r.Channel.QueueInspect(queueName)
		if err != nil {
			logger.Warning.Printf("RabbitMQ %s check queue:%s consumers while could not get rabbit mq queue status", r.Name, queueName)
			return 0
		}
		queueStatus.Consumers = queue.Consumers
		queueStatus.Messages = queue.Messages
		queueStatus.RefreshingTime = now
	}
	if queueStatus.Consumers < 1 {
		logger.Warning.Printf("RabbitMQ %s check queue:%s consumers while there is no consumers on publisher queue.", r.Name, queueName)
	}
	return queueStatus.Consumers
}

// GenerateRabbitMQConsumerProxy generate rabbitmq consumer proxy
func GenerateRabbitMQConsumerProxy(consumeProxy *mqenv.MQConsumerProxy, exchangeName string) *RabbitConsumerProxy {
	cb := func(d amqp.Delivery) *mqenv.MQPublishMessage {
		msg := generateMQResponseMessage(&d, exchangeName)
		if nil != consumeProxy.Callback {
			return consumeProxy.Callback(*msg)
		}
		return nil
	}
	pxy := &RabbitConsumerProxy{
		Queue:       consumeProxy.Queue,
		Callback:    cb,
		ConsumerTag: consumeProxy.ConsumerTag,
		AutoAck:     consumeProxy.AutoAck,
		Exclusive:   consumeProxy.Exclusive,
		NoLocal:     consumeProxy.NoLocal,
		NoWait:      consumeProxy.NoWait,
		// Arguments:   consumeProxy.Arguments,
	}
	return pxy
}

func generateMQResponseMessage(d *amqp.Delivery, exchangeName string) *mqenv.MQConsumerMessage {
	msg := &mqenv.MQConsumerMessage{
		Driver:        mqenv.DriverTypeAMQP,
		Queue:         d.RoutingKey,
		CorrelationID: d.CorrelationId,
		ConsumerTag:   d.ConsumerTag,
		ReplyTo:       d.ReplyTo,
		MessageID:     d.MessageId,
		AppID:         d.AppId,
		UserID:        d.UserId,
		ContentType:   d.ContentType,
		Exchange:      d.Exchange,
		RoutingKey:    d.RoutingKey,
		Timestamp:     d.Timestamp,
		Body:          d.Body,
		Headers:       map[string]string{},
		BindData:      d,
	}
	if "" == msg.Exchange {
		msg.Exchange = exchangeName
	}
	for k, v := range d.Headers {
		msg.Headers[k] = utils.ToString(v)
	}
	return msg
}
