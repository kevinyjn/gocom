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

var cbs = make(map[string]*RabbitRPCMsgProxy)

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
		logger.Warning.Printf("Get rabbit mq rpc publisher by appid:%s while the publisher channel were not connected.", key)
		return nil
	}
	now := time.Now().Unix()
	if now-r.QueueStatus.RefreshingTime > AMQPQueueStatusFreshDuration {
		qs, err := r.Channel.QueueInspect(r.QueueStatus.QueueName)
		if err != nil {
			logger.Warning.Printf("Could not get rabbit mq queue status by queue name:%s", r.QueueStatus.QueueName)
			return nil
		}
		r.QueueStatus.Consumers = qs.Consumers
		r.QueueStatus.Messages = qs.Messages
		r.QueueStatus.RefreshingTime = now
	}
	if r.QueueStatus.Consumers < 1 {
		logger.Warning.Printf("Get rabbit mq rpc publisher by appid:%s while there is no consumers on publisher queue.", key)
		return nil
	}
	return r
}

func generateRPCRabbitMQInstance(name string, rpcType int, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitRPCMQ {
	r := &RabbitRPCMQ{
		Name:       name,
		Publish:    make(chan *RabbitRPCMsgProxy),
		Done:       make(chan error),
		Close:      make(chan interface{}),
		Config:     amqpCfg,
		ConnConfig: connCfg,
		RPCType:    rpcType,
		QueueStatus: &RabbitQueueStatus{
			QueueName:      amqpCfg.Queue,
			RefreshingTime: 0,
		},
		connecting: false,
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
	for {
		if r.connecting == false && r.Conn == nil {
			r.initConn()
			logger.Trace.Printf("amqprpc %s pre running...", r.Name)
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
			logger.Trace.Printf("publish message: %s\n", pm.Request)
			r.publish(pm)
		case err := <-r.Done:
			logger.Error.Printf("RabbitMQ connection:%s done with error:%s", r.Name, err.Error())
			r.close()
		case err := <-r.connClosed:
			logger.Error.Printf("RabbitMQ connection:%s closed with error:%s", r.Name, err.Error())
			r.close()
		case err := <-r.channelClosed:
			logger.Error.Printf("RabbitMQ channel:%s closed with error:%s", r.Name, err.Error())
			r.close()
		case <-r.Close:
			r.Channel.Close()
			return
		}
	}
}

func (r *RabbitRPCMQ) close() {
	r.connecting = false
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}

// try to start a new connection, channel and deliveries channel. if failed, try again in 5 sec.
func (r *RabbitRPCMQ) initConn() {
	cnf := r.ConnConfig
	if cnf.Driver != mqenv.DriverTypeAMQP {
		logger.Error.Printf("Initialize rabbitmq connection failed, the configure driver:%s does not fit.", cnf.Driver)
		return
	}

	r.connecting = true
	ticker := time.NewTicker(AMQPReconnectDuration * time.Second)
	quitTiker := make(chan struct{})
	amqpAddr := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", cnf.User, cnf.Password, cnf.Host, cnf.Port, cnf.Path)

	go func() {
		for {
			select {
			case <-ticker.C:
				if conn, err := dial(amqpAddr); err != nil {
					r.connecting = false
					logger.Error.Println(err)
					logger.Error.Println("node will only be able to respond to local connections")
					logger.Error.Println("trying to reconnect in 5 seconds...")
				} else {
					r.connecting = false
					r.Conn = conn
					close(quitTiker)
					r.Channel, err = createChannel(conn, r.Config)
					if err != nil {
						conn.Close()
						r.Conn = nil
						logger.Fatal.Printf("create channel failed with error:%s", err.Error())
						return
					}
					qs, err := createQueue(r.Channel, r.Config)
					if err != nil {
						conn.Close()
						r.Conn = nil
						logger.Fatal.Printf("create queue:%s failed with error:%s", r.Config.Queue, err.Error())
						return
					}
					r.QueueStatus.QueueName = qs.Name
					r.QueueStatus.Consumers = qs.Consumers
					r.QueueStatus.Messages = qs.Messages
					r.QueueStatus.RefreshingTime = time.Now().Unix()
					r.connClosed = conn.NotifyClose(make(chan *amqp.Error))
					r.channelClosed = r.Channel.NotifyClose(make(chan *amqp.Error))
					if mqenv.MQTypeConsumer == r.RPCType {
						r.Deliveries, _ = getDeliveriesChannel(r.Channel, r.Config.Queue)
						go handleConsume(r.Deliveries, r.Done)
					}
				}
			case <-quitTiker:
				ticker.Stop()
				return
			}
		}
	}()
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

func (r *RabbitRPCMQ) publish(pm *RabbitRPCMsgProxy) error {
	// body, _ := json.Marshal(pm)
	c := r.Channel
	body := pm.Request
	logger.Trace.Printf("publishing %dB body (%s)", len(body), body)
	if c == nil {
		return fmt.Errorf("connection to rabbitmq might not be ready yet")
	}
	correlationID := genCorrelationID()

	cbs[correlationID] = pm

	if err := c.Publish(
		"",             // publish to an exchange
		r.Config.Queue, // routing to 0 or more queues
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "application/json",
			ContentEncoding: "",
			CorrelationId:   correlationID,
			ReplyTo:         pm.ReplyToQueue,
			Body:            []byte(body),
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}
	return nil
}

func handleConsume(deliveries <-chan amqp.Delivery, done chan error) {
	logger.Info.Println("waiting for rabbitmq deliveries...")
	for d := range deliveries {
		s := string(d.Body)
		logger.Trace.Printf(
			"got %dB delivery: [%v] %s",
			len(s),
			d.DeliveryTag,
			s,
		)
		// fmt.Println("---- got delivery message d:", d)
		correlationID := d.CorrelationId
		pm, pmExists := cbs[correlationID]
		if pmExists {
			// fmt.Println("==== push back response data ", s)
			if !pm.callbackDisabled {
				pm.Response <- s
			}
			delete(cbs, correlationID)
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
