package rabbitmq

import (
	"errors"
	"fmt"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/netutils/pinger"
	"github.com/kevinyjn/gocom/netutils/sshtunnel"

	"github.com/streadway/amqp"
)

var amqpInsts = map[string]*RabbitMQ{}

// InitRabbitMQ init
func InitRabbitMQ(mqConnName string, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) (*RabbitMQ, error) {
	amqpInst, ok := amqpInsts[mqConnName]
	if ok && !amqpInst.Config.Equals(amqpCfg) {
		amqpInst.close()
		close(amqpInst.Close)
		ok = false
	}
	if !ok {
		amqpInst = NewRabbitMQ(mqConnName, connCfg, amqpCfg)
		amqpInsts[mqConnName] = amqpInst
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
	amqpInst, ok := amqpInsts[name]
	if ok {
		return amqpInst, nil
	}
	return nil, fmt.Errorf("RabbitMQ instance by %s not found", name)
}

func dial(amqpURI string) (*amqp.Connection, error) {
	logger.Info.Printf("rabbit dialing ...")
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

func inspectQueue(channel *amqp.Channel, amqpCfg *AMQPConfig) (*amqp.Queue, error) {
	durable := amqpCfg.QueueDurable
	autoDelete := false
	queueName := amqpCfg.Queue
	if amqpCfg.IsBroadcastExange() {
		autoDelete = true
		durable = false
		queueName = ""
	}
	queue, err := channel.QueueDeclare(
		queueName,  // name of the queue
		durable,    // durable
		autoDelete, // delete when usused
		false,      // exclusive
		false,      // noWait
		nil,        // arguments
	)
	return &queue, err
}

func createQueue(channel *amqp.Channel, amqpCfg *AMQPConfig) (*amqp.Queue, error) {
	if "" == amqpCfg.ExchangeName {
		logger.Info.Printf("declaring Queue %q", amqpCfg.Queue)
	} else {
		logger.Info.Printf("declared Exchange %s(%s), declaring Queue %q", amqpCfg.ExchangeName, amqpCfg.ExchangeType, amqpCfg.Queue)
	}
	queue, err := inspectQueue(channel, amqpCfg)
	if err != nil {
		logger.Error.Printf("Queue declare: %v", err)
		return nil, fmt.Errorf("Queue declare: %v", err)
	}

	if amqpCfg.BindingExchange {
		logger.Info.Printf("declared Queue (%q %d messages, %d consumers), binding to Exchange %s (key %q)",
			queue.Name, queue.Messages, queue.Consumers, amqpCfg.ExchangeName, amqpCfg.BindingKey)
		if err = channel.QueueBind(
			queue.Name,           // name of the queue
			amqpCfg.BindingKey,   // bindingKey
			amqpCfg.ExchangeName, // sourceExchange
			false,                // noWait
			nil,                  // arguments
		); err != nil {
			logger.Error.Printf("Queue bind failed with error:%v", err)
			return nil, fmt.Errorf("Queue bind: %v", err)
		}
	} else {
		logger.Info.Printf("declared Queue (%q %d messages, %d consumers)",
			queue.Name, queue.Messages, queue.Consumers)
	}
	return queue, nil
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
	r.Publish = make(chan *mqenv.MQPublishMessage)
	r.Consume = make(chan *RabbitConsumerProxy)
	r.Done = make(chan error)
	r.Close = make(chan interface{})
	r.QueueStatus = &RabbitQueueStatus{
		QueueName:      amqpCfg.Queue,
		RefreshingTime: 0,
	}
	r.consumers = make([]*RabbitConsumerProxy, 0)
	r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
	r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
	r.connecting = false
	r.queue = nil
	r.afterEnsureQueue = r.ensurePendings
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
			if len(r.consumers) > 0 {
				if nil == r.pendingConsumers {
					r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
				}
				for _, cm := range r.consumers {
					r.pendingConsumers = append(r.pendingConsumers, cm)
				}
				r.consumers = make([]*RabbitConsumerProxy, 0)
			}
			r.initConn()
			logger.Trace.Printf("rabbitmq %s pre running...", r.Name)
			// backstop
			if r.Conn != nil && !r.Conn.IsClosed() {
				if err := r.Channel.Cancel("", true); err != nil {
					logger.Error.Printf("RabbitMQ %s cancel channel failed with error:%v", r.Name, err)
				}
				if err := r.Conn.Close(); err != nil {
					logger.Error.Printf("RabbitMQ %s close connection failed with error:%v", r.Name, err)
				}
			}

			// IMPORTANT: 必须清空 Notify，否则死连接不会释放
			r.clearNotifyChan()
			logger.Trace.Printf("rabbitmq %s do running...", r.Name)
		}

		select {
		case pm := <-r.Publish:
			r.publish(pm)
		case cm := <-r.Consume:
			if "" != r.queueName {
				cm.Queue = r.queueName
			}
			logger.Info.Printf("consuming queue: %s\n", cm.Queue)
			r.consume(cm)
		case err := <-r.Done:
			logger.Error.Printf("RabbitMQ connection:%s done with error:%v", r.Name, err)
			if r.connecting == false {
				r.queueName = ""
				r.queue = nil
				r.close()
				break
			}
		case err := <-r.connClosed:
			logger.Error.Printf("RabbitMQ connection:%s closed with error:%v", r.Name, err)
			r.queueName = ""
			r.queue = nil
			r.clearNotifyChan()
			r.Conn = nil
			r.Channel = nil
			break
		case err := <-r.channelClosed:
			logger.Error.Printf("RabbitMQ channel:%s closed with error:%v", r.Name, err)
			r.queueName = ""
			r.queue = nil
			r.Channel = nil
			r.close()
			break
		case <-r.Close:
			r.queueName = ""
			r.queue = nil
			r.Channel.Close()
			tick.Stop()
			return
		case <-tick.C:
			// fmt.Println("amqp tick...")
			if nil == r.Conn {
				break
			}
			if r.Conn.IsClosed() {
				r.Conn = nil
				r.queue = nil
				r.queueName = ""
				logger.Error.Printf("RabbitMQ connection:%s were closed on ticker checking", r.Name)
				break
			} else if nil == r.Channel {
				logger.Warning.Printf("RabbitMQ channel:%s were nil", r.Name)
			} else {
				//
			}
		}
	}
}

func (r *RabbitMQ) clearNotifyChan() {
	if r.channelClosed != nil {
		for err := range r.channelClosed {
			println(err)
		}
		r.channelClosed = nil
	}
	if r.connClosed != nil {
		for err := range r.connClosed {
			println(err)
		}
		r.connClosed = nil
	}
}

func (r *RabbitMQ) close() {
	r.connecting = false
	r.clearNotifyChan()
	if r.Channel != nil {
		r.Channel.Close()
		r.Channel = nil
	}
	if r.Conn != nil && !r.Conn.IsClosed() {
		r.Conn.Close()
	}
	if nil != r.sshTunnel {
		r.sshTunnel.Stop()
		r.sshTunnel = nil
	}
	r.Conn = nil
}

// try to start a new connection, channel and deliveries channel. if failed, try again in 5 sec.
func (r *RabbitMQ) initConn() error {
	if r.ConnConfig.Driver != mqenv.DriverTypeAMQP {
		logger.Error.Printf("Initialize rabbitmq connection by configure:%s failed, the configure driver:%s does not fit.", r.Name, r.ConnConfig.Driver)
		return errors.New("Invalid driver for rabbitmq")
	}

	r.connecting = true
	connDSN, connDescription, err := r.formatConnectionDSN()
	if nil != err {
		logger.Error.Printf("Initialize rabbitmq connection by configure:%s while format amqp conneciton DSN failed with error:%v", r.Name, err)
		return err
	}

	go func() {
		ticker := time.NewTicker(AMQPReconnectDuration * time.Second)
		for {
			select {
			case <-ticker.C:
				if conn, err := dial(connDSN); err != nil {
					// r.connecting = false
					logger.Error.Println(err)
					logger.Error.Println("node will only be able to respond to local connections")
					logger.Error.Printf("trying to reconnect in %d seconds...", AMQPReconnectDuration)
				} else {
					logger.Info.Printf("Connecting rabbitmq %s succeed", connDescription)
					r.connecting = false
					r.Conn = conn
					ticker.Stop()
					r.Channel, err = createChannel(conn, r.Config)
					if err != nil {
						conn.Close()
						r.Conn = nil
						logger.Fatal.Printf("create channel failed with error:%v", err)
						return
					}
					if r.Config.IsBroadcastExange() && len(r.consumers) <= 0 && len(r.pendingConsumers) <= 0 {
						break
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
	connDescription := fmt.Sprintf("amqp://%s:<password>@%s:%d/%s", cnf.User, cnf.Host, cnf.Port, cnf.Path)
	return connDSN, connDescription, err
}

func (r *RabbitMQ) ensureQueue() error {
	if nil == r.Conn {
		return fmt.Errorf("RabbitMQ connection were not connected when ensuring queue:%s", r.Config.Queue)
	}
	queue, err := createQueue(r.Channel, r.Config)
	if err != nil {
		r.Conn.Close()
		r.Conn = nil
		logger.Fatal.Printf("create queue:%s failed with error:%v", r.Config.Queue, err)
		return err
	}
	r.QueueStatus.QueueName = queue.Name
	r.QueueStatus.Consumers = queue.Consumers
	r.QueueStatus.Messages = queue.Messages
	r.QueueStatus.RefreshingTime = time.Now().Unix()
	r.queue = queue
	r.queueName = queue.Name
	r.connClosed = make(chan *amqp.Error)
	r.channelClosed = make(chan *amqp.Error)
	r.Conn.NotifyClose(r.connClosed)
	r.Channel.NotifyClose(r.channelClosed)
	if nil != r.afterEnsureQueue {
		r.afterEnsureQueue()
	}
	return nil
}

func (r *RabbitMQ) ensurePendings() {
	if r.pendingConsumers != nil && len(r.pendingConsumers) > 0 {
		consumers := r.pendingConsumers
		r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
		for _, cm := range consumers {
			cm.Queue = r.queueName
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
		logger.Warning.Printf("pending publishing %dB body (%s)", len(pm.Body), pm.Body)
		r.pendingPublishes = append(r.pendingPublishes, pm)
		return nil
	}
	if nil == r.queue && !r.Config.IsBroadcastExange() {
		err := r.ensureQueue()
		if nil != err {
			return err
		}
	}

	exchangeName, routingKey := r.Config.ExchangeName, pm.RoutingKey
	if nil != r.beforePublish {
		exchangeName, routingKey = r.beforePublish(pm)
	}
	if logger.IsDebugEnabled() {
		logger.Trace.Printf("publishing message(%s) with %dB body (%s)", pm.CorrelationID, len(pm.Body), pm.Body)
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
			ContentType:     "application/json",
			ContentEncoding: "",
			Body:            pm.Body,
			CorrelationId:   pm.CorrelationID,
			ReplyTo:         pm.ReplyTo,
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	)
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
	if r.Channel == nil {
		logger.Warning.Printf("Consuming queue:%s failed while the channel not ready, pending.", cm.Queue)
		r.pendingConsumers = append(r.pendingConsumers, cm)
		return nil
	}
	if nil == r.queue {
		err := r.ensureQueue()
		if nil != err {
			return err
		}
	}
	if nil == r.consumers {
		r.consumers = make([]*RabbitConsumerProxy, 0)
	}
	r.consumers = append(r.consumers, cm)

	deliveries, err := r.Channel.Consume(
		cm.Queue,       // name
		cm.ConsumerTag, // consumerTag,
		false,          // noAck
		cm.Exclusive,   // exclusive
		cm.NoLocal,     // noLocal
		cm.NoWait,      // noWait
		cm.Arguments,   // arguments
	)
	if err != nil {
		logger.Error.Printf("consuming mq(%s) queue:%s failed with error:%v", r.Name, cm.Queue, err)
		return err
	}
	logger.Info.Printf("Now consuming mq(%s) with queue:%s ...", r.Name, cm.Queue)
	go r.handleConsumes(cm.Callback, cm.AutoAck, deliveries)
	return nil
}

func (r *RabbitMQ) handleConsumes(cb AMQPConsumerCallback, autoAck bool, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		if logger.IsDebugEnabled() {
			logger.Trace.Printf(
				"got %dB delivery: [%v] %s",
				len(d.Body),
				d.DeliveryTag,
				d.Body,
			)
		}
		// fmt.Println("---- got delivery message d:", d)
		go handleConsumeCallback(d, cb, autoAck)
	}
	r.Done <- fmt.Errorf("error: deliveries channel closed")
}

func handleConsumeCallback(d amqp.Delivery, cb AMQPConsumerCallback, autoAck bool) {
	if cb != nil {
		cb(d)
	}
	if autoAck == false {
		d.Ack(false)
	}
}

// GenerateRabbitMQConsumerProxy generate rabbitmq consumer proxy
func GenerateRabbitMQConsumerProxy(consumeProxy *mqenv.MQConsumerProxy) *RabbitConsumerProxy {
	cb := func(msg amqp.Delivery) {
		mqMsg := mqenv.MQConsumerMessage{
			Driver:        mqenv.DriverTypeAMQP,
			Queue:         consumeProxy.Queue,
			CorrelationID: msg.CorrelationId,
			ConsumerTag:   msg.ConsumerTag,
			ReplyTo:       msg.ReplyTo,
			RoutingKey:    msg.RoutingKey,
			Body:          msg.Body,
			BindData:      &msg,
		}

		if nil != consumeProxy.Callback {
			consumeProxy.Callback(mqMsg)
		}
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
