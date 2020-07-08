package rabbitmq

import (
	"errors"
	"fmt"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"

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
		amqpInst = &RabbitMQ{
			Name:             mqConnName,
			Config:           amqpCfg,
			ConnConfig:       connCfg,
			Publish:          make(chan *RabbitPublishingMsg),
			Consume:          make(chan *RabbitConsumerProxy),
			Done:             make(chan error),
			Close:            make(chan interface{}),
			consumers:        make([]*RabbitConsumerProxy, 0),
			pendingConsumers: make([]*RabbitConsumerProxy, 0),
			pendingPublishes: make([]*RabbitPublishingMsg, 0),
			connecting:       false,
			queue:            nil,
		}
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
		logger.Error.Printf("Channel: %s", err.Error())
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
			logger.Error.Printf("Exchange Declare: %s", err.Error())
			return nil, err
		}
	}
	return channel, nil
}

func createQueue(channel *amqp.Channel, amqpCfg *AMQPConfig) (*amqp.Queue, error) {
	logger.Info.Printf("declared Exchange, declaring Queue %q", amqpCfg.Queue)
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
	if err != nil {
		logger.Error.Printf("Queue declare: %s", err.Error())
		return nil, fmt.Errorf("Queue Declare: %s", err.Error())
	}

	logger.Info.Printf("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)",
		queue.Name, queue.Messages, queue.Consumers, amqpCfg.BindingKey)

	if amqpCfg.BindingExchange {
		if err = channel.QueueBind(
			queue.Name,           // name of the queue
			amqpCfg.BindingKey,   // bindingKey
			amqpCfg.ExchangeName, // sourceExchange
			false,                // noWait
			nil,                  // arguments
		); err != nil {
			logger.Error.Printf("Queue bind: %s", err.Error())
			return nil, fmt.Errorf("Queue Bind: %s", err.Error())
		}
	}
	return &queue, nil
}

func (r *RabbitMQ) init() error {
	logger.Info.Printf("Initializing amqp instance:%s", r.Name)
	return r.initConn()
}

// Run start
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
					logger.Error.Printf("RabbitMQ %s cancel channel failed with error:%s", r.Name, err.Error())
				}
				if err := r.Conn.Close(); err != nil {
					logger.Error.Printf("RabbitMQ %s close connection failed with error:%s", r.Name, err.Error())
				}
			}

			// IMPORTANT: 必须清空 Notify，否则死连接不会释放
			r.clearNotifyChan()
			logger.Trace.Printf("rabbitmq %s do running...", r.Name)
		}

		select {
		case pm := <-r.Publish:
			logger.Trace.Printf("publish message: %s\n", pm.Body)
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
	r.Conn = nil
}

// try to start a new connection, channel and deliveries channel. if failed, try again in 5 sec.
func (r *RabbitMQ) initConn() error {
	cnf := r.ConnConfig
	if cnf.Driver != mqenv.DriverTypeAMQP {
		logger.Error.Printf("Initialize rabbitmq connection by configure:%s failed, the configure driver:%s does not fit.", r.Name, cnf.Driver)
		return errors.New("Invalid driver for rabbitmq")
	}

	r.connecting = true
	amqpAddr := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", cnf.User, cnf.Password, cnf.Host, cnf.Port, cnf.Path)

	go func() {
		ticker := time.NewTicker(AMQPReconnectDuration * time.Second)
		quitTiker := make(chan struct{})
		for {
			select {
			case <-ticker.C:
				if conn, err := dial(amqpAddr); err != nil {
					// r.connecting = false
					logger.Error.Println(err)
					logger.Error.Println("node will only be able to respond to local connections")
					logger.Error.Printf("trying to reconnect in %d seconds...", AMQPReconnectDuration)
				} else {
					logger.Info.Printf("Connecting amqp:%s:%d/%s by user:%s succeed", cnf.Host, cnf.Port, cnf.Path, cnf.User)
					r.connecting = false
					r.Conn = conn
					if nil != quitTiker {
						close(quitTiker)
					}
					r.Channel, err = createChannel(conn, r.Config)
					if err != nil {
						conn.Close()
						r.Conn = nil
						logger.Fatal.Printf("create channel failed with error:%s", err.Error())
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
			case <-quitTiker:
				ticker.Stop()
				return
			}
		}
	}()
	return nil
}

func (r *RabbitMQ) ensureQueue() error {
	if nil == r.Conn {
		return fmt.Errorf("RabbitMQ connection were not connected when ensuring queue:%s", r.Config.Queue)
	}
	queue, err := createQueue(r.Channel, r.Config)
	if err != nil {
		r.Conn.Close()
		r.Conn = nil
		logger.Fatal.Printf("create queue:%s failed with error:%s", r.Config.Queue, err.Error())
		return err
	}
	r.queue = queue
	r.queueName = queue.Name
	r.connClosed = make(chan *amqp.Error)
	r.channelClosed = make(chan *amqp.Error)
	r.Conn.NotifyClose(r.connClosed)
	r.Channel.NotifyClose(r.channelClosed)
	if r.pendingConsumers != nil && len(r.pendingConsumers) > 0 {
		consumers := r.pendingConsumers
		r.pendingConsumers = make([]*RabbitConsumerProxy, 0)
		for _, cm := range consumers {
			cm.Queue = queue.Name
			r.consume(cm)
		}
	}
	if r.pendingPublishes != nil && len(r.pendingPublishes) > 0 {
		publishes := r.pendingPublishes
		r.pendingPublishes = make([]*RabbitPublishingMsg, 0)
		for _, pm := range publishes {
			r.publish(pm)
		}
	}
	return nil
}

func (r *RabbitMQ) publish(pm *RabbitPublishingMsg) error {
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
	// logger.Trace.Printf("publishing %dB body (%s)", len(pm.Body), pm.Body)

	headers := amqp.Table{}
	for k, v := range pm.Headers {
		headers[k] = v
	}
	err := r.Channel.Publish(
		r.Config.ExchangeName, // publish to an exchange
		pm.RoutingKey,         // routing to 0 or more queues
		false,                 // mandatory
		false,                 // immediate
		amqp.Publishing{
			Headers:         headers,
			ContentType:     "application/json",
			ContentEncoding: "",
			Body:            pm.Body,
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// CorrelationId:   pm.CorrelationID,
			// ReplyTo:         pm.ReplyTo,
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
		logger.Error.Printf("consuming queue:%s failed with error:%s", cm.Queue, err.Error())
		return err
	}
	logger.Info.Printf("Consuming mq with queue:%s ...", cm.Queue)
	go r.handleConsumes(cm.Callback, cm.AutoAck, deliveries)
	return nil
}

func (r *RabbitMQ) handleConsumes(cb AMQPConsumerCallback, autoAck bool, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		logger.Trace.Printf(
			"got %dB delivery: [%v] %s",
			len(d.Body),
			d.DeliveryTag,
			d.Body,
		)
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

// GenerateRabbitMQPublishMessage generate publish message
func GenerateRabbitMQPublishMessage(publishMsg *mqenv.MQPublishMessage) *RabbitPublishingMsg {
	msg := &RabbitPublishingMsg{
		Body:          publishMsg.Body,
		RoutingKey:    publishMsg.RoutingKey,
		CorrelationID: publishMsg.CorrelationID,
		ReplyTo:       publishMsg.ReplyTo,
		PublishStatus: publishMsg.PublishStatus,
		EventLabel:    publishMsg.EventLabel,
		Headers:       publishMsg.Headers,
	}
	return msg
}
