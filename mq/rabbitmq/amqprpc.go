package rabbitmq

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"

	"github.com/streadway/amqp"
)

var amqpRpcs = map[string]*RabbitRPCMQ{}

var _cbs = make(map[string]*mqenv.MQPublishMessage)

// InitRPCRabbitMQ init
func InitRPCRabbitMQ(key string, rpcType int, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitRPCMQ {
	if amqpCfg == nil || connCfg == nil {
		return nil
	}

	r, exists := amqpRpcs[key]
	if !exists {
		r = generateRPCRabbitMQInstance(key, rpcType, connCfg, amqpCfg)
		amqpRpcs[key] = r
		return r
	} else if !r.Config.Equals(amqpCfg) {
		r.close()
		close(r.Close)

		r = generateRPCRabbitMQInstance(key, rpcType, connCfg, amqpCfg)
		amqpRpcs[key] = r
		return r
	}
	return r
}

// GetRPCRabbitMQ get instance
func GetRPCRabbitMQ(key string) *RabbitRPCMQ {
	r, exists := amqpRpcs[key]
	if !exists {
		return nil
	}
	if r.Channel == nil || r.QueueStatus.RefreshingTime == 0 {
		logger.Warning.Printf("Get rabbit mq rpc publisher by key:%s while the publisher channel were not connected.", key)
		return nil
	}
	now := time.Now().Unix()
	if (r.QueueStatus.Consumers < 1 && now-r.QueueStatus.RefreshingTime > 1) || now-r.QueueStatus.RefreshingTime > AMQPQueueStatusFreshDuration {
		queue, err := r.Channel.QueueDeclare(
			r.QueueStatus.QueueName,
			r.Config.QueueDurable,
			false,
			false,
			false,
			nil,
		)
		// qs, err := r.Channel.QueueInspect(r.QueueStatus.QueueName)
		if err != nil {
			logger.Warning.Printf("Could not get rabbit mq queue status by queue name:%s", r.QueueStatus.QueueName)
			return nil
		}
		r.QueueStatus.Consumers = queue.Consumers
		r.QueueStatus.Messages = queue.Messages
		r.QueueStatus.RefreshingTime = now
	}
	if r.QueueStatus.Consumers < 1 {
		logger.Warning.Printf("Get rabbit mq rpc publisher by key:%s while there is no consumers on publisher queue.", key)
		return nil
	}
	return r
}

// GetRPCRabbitMQWithoutConnectedChecking get instance
func GetRPCRabbitMQWithoutConnectedChecking(key string) *RabbitRPCMQ {
	r, exists := amqpRpcs[key]
	if !exists {
		return nil
	}
	return r
}

func generateRPCRabbitMQInstance(name string, rpcType int, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitRPCMQ {
	r := &RabbitRPCMQ{
		Name:       name,
		Publish:    make(chan *mqenv.MQPublishMessage),
		Consume:    make(chan *RabbitConsumerProxy),
		Done:       make(chan error),
		Close:      make(chan interface{}),
		Config:     amqpCfg,
		ConnConfig: connCfg,
		RPCType:    rpcType,
		QueueStatus: &RabbitQueueStatus{
			QueueName:      amqpCfg.Queue,
			RefreshingTime: 0,
		},
		consumers:        make([]*RabbitConsumerProxy, 0),
		pendingConsumers: make([]*RabbitConsumerProxy, 0),
		pendingPublishes: make([]*mqenv.MQPublishMessage, 0),
		connecting:       false,
	}
	go r.Run()
	return r
}

func getDeliveriesChannel(channel *amqp.Channel, queueName string) (<-chan amqp.Delivery, error) {
	return channel.Consume(
		queueName, // name
		"",        // consumerTag,
		false,     // noAck
		false,     // exclusive
		false,     // noLocal
		false,     // noWait
		nil,       // arguments
	)
}

// Run start
// 1. init the rabbitmq conneciton
// 2. expect messages from the message hub on the Publish channel
// 3. if the connection is closed, try to restart it
func (r *RabbitRPCMQ) Run() {
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
			logger.Trace.Printf("amqprpc %s pre running...", r.Name)
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
			if r.channelClosed != nil {
				for err := range r.channelClosed {
					println(err)
				}
			}
			if r.connClosed != nil {
				for err := range r.connClosed {
					println(err)
				}
			}
			logger.Trace.Printf("amqprpc %s do running...", r.Name)
		}

		select {
		case pm := <-r.Publish:
			if logger.IsDebugEnabled() {
				logger.Trace.Printf("publish message: %s", string(pm.Body))
			}
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
		case err := <-r.channelClosed:
			logger.Error.Printf("RabbitMQ channel:%s closed with error:%v", r.Name, err)
			r.queueName = ""
			r.queue = nil
			r.Channel = nil
			r.close()
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

func (r *RabbitRPCMQ) clearNotifyChan() {
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

func (r *RabbitRPCMQ) close() {
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
func (r *RabbitRPCMQ) initConn() {
	cnf := r.ConnConfig
	if cnf.Driver != mqenv.DriverTypeAMQP {
		logger.Error.Printf("Initialize rabbitmq connection failed, the configure driver:%s does not fit.", cnf.Driver)
		return
	}

	r.connecting = true
	amqpAddr := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", cnf.User, cnf.Password, cnf.Host, cnf.Port, cnf.Path)

	go func() {
		ticker := time.NewTicker(AMQPReconnectDuration * time.Second)
		for {
			select {
			case <-ticker.C:
				if conn, err := dial(amqpAddr); err != nil {
					r.connecting = false
					logger.Error.Println(err)
					logger.Error.Println("node will only be able to respond to local connections")
					logger.Error.Println("trying to reconnect in 5 seconds...")
				} else {
					logger.Info.Printf("Connecting amqp:%s:%d/%s by user:%s succeed", cnf.Host, cnf.Port, cnf.Path, cnf.User)
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
}

func (r *RabbitRPCMQ) ensureQueue() error {
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
	r.connClosed = r.Conn.NotifyClose(make(chan *amqp.Error))
	r.channelClosed = r.Channel.NotifyClose(make(chan *amqp.Error))
	if mqenv.MQTypeConsumer == r.RPCType {
		logger.Info.Printf("consuming RPC messages...")
		r.Deliveries, _ = getDeliveriesChannel(r.Channel, r.Config.Queue)
		go handleConsume(r.Deliveries, r.Done)
	} else {
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
			r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
			for _, pm := range publishes {
				r.publish(pm)
			}
		}
	}

	return nil
}

func (r *RabbitRPCMQ) consume(cm *RabbitConsumerProxy) error {
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
		logger.Error.Printf("consuming queue:%s failed with error:%v", cm.Queue, err)
		return err
	}
	logger.Info.Printf("Now consuming mq with queue:%s ...", cm.Queue)
	go r.handleConsumes(cm.Callback, cm.AutoAck, deliveries)
	return nil
}

func (r *RabbitRPCMQ) handleConsumes(cb AMQPConsumerCallback, autoAck bool, deliveries <-chan amqp.Delivery) {
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

func genCorrelationID() string {
	unix32bits := uint32(time.Now().UTC().Unix())
	buff := make([]byte, 12)
	numRead, err := rand.Read(buff)
	if numRead != len(buff) || err != nil {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x-%x", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:8], buff[8:])
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func (r *RabbitRPCMQ) publish(pm *mqenv.MQPublishMessage) error {
	// body, _ := json.Marshal(pm)
	if r.Channel == nil {
		logger.Warning.Printf("publishing to rabbitmq while connection not ready yet, with %dB body (%s)", len(pm.Body), pm.Body)
		return fmt.Errorf("connection to rabbitmq might not be ready yet")
	}
	if nil == r.queue && !r.Config.IsBroadcastExange() {
		err := r.ensureQueue()
		if nil != err {
			return err
		}
	}

	if "" == pm.CorrelationID {
		pm.CorrelationID = genCorrelationID()
	}
	originPm, _ := _cbs[pm.CorrelationID]
	if nil == originPm || originPm.Response == nil {
		_cbs[pm.CorrelationID] = pm
	}

	if logger.IsDebugEnabled() {
		fmt.Printf("====> caching publish message inst %s %+v\n", pm.Body, pm)
		logger.Trace.Printf("publishing %s with %dB body (%s)", pm.CorrelationID, len(pm.Body), pm.Body)
	}

	headers := amqp.Table{}
	for k, v := range pm.Headers {
		headers[k] = v
	}
	err := r.Channel.Publish(
		"",             // publish to an exchange
		r.Config.Queue, // routing to 0 or more queues
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			Headers:         headers,
			ContentType:     "application/json",
			ContentEncoding: "",
			CorrelationId:   pm.CorrelationID,
			ReplyTo:         pm.ReplyTo,
			Body:            pm.Body,
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

func handleConsume(deliveries <-chan amqp.Delivery, done chan error) {
	logger.Info.Println("waiting for rabbitmq deliveries...")
	for d := range deliveries {
		body := d.Body
		if logger.IsDebugEnabled() {
			logger.Trace.Printf(
				"got %dB delivery: [%v] [%s] %s",
				len(body),
				d.DeliveryTag,
				d.CorrelationId,
				string(body),
			)
		}
		// fmt.Println("---- got delivery message d:", d)
		correlationID := d.CorrelationId
		pm, pmExists := _cbs[correlationID]
		if pmExists {
			if pm.CallbackEnabled() {
				fmt.Printf("====> push back response data %s %+v\n", correlationID, pm)
				pm.Response <- body
			}
			delete(_cbs, correlationID)
			d.Ack(false)
		} else {
			// detect if the message is old data, if the message is old data and belongs to myself, acknowledge the message
			// d.Ack(false)
			// d.Reject(true)
			d.Nack(false, false)
		}
	}
	done <- fmt.Errorf("error: deliveries channel closed")
}
