package rabbitmq

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"

	"github.com/streadway/amqp"
)

var amqpRpcs = map[string]*RabbitRPC{}

var _cbs = make(map[string]*mqenv.MQPublishMessage)

// InitRPCRabbitMQ init
func InitRPCRabbitMQ(key string, rpcType int, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitRPC {
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
func GetRPCRabbitMQ(key string) *RabbitRPC {
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
		queue, err := r.Channel.QueueInspect(r.QueueStatus.QueueName)
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
func GetRPCRabbitMQWithoutConnectedChecking(key string) *RabbitRPC {
	r, exists := amqpRpcs[key]
	if !exists {
		return nil
	}
	return r
}

func generateRPCRabbitMQInstance(name string, rpcType int, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitRPC {
	r := &RabbitRPC{
		RPCType: rpcType,
	}
	r.initWithParameters(name, connCfg, amqpCfg)
	r.afterEnsureQueue = r.ensureRPCConsumer
	r.beforePublish = r.ensureRPCCorrelationID
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

func (r *RabbitRPC) ensureRPCConsumer() {
	if mqenv.MQTypeConsumer == r.RPCType {
		logger.Info.Printf("consuming RPC messages...")
		r.Deliveries, _ = getDeliveriesChannel(r.Channel, r.Config.Queue)
		go handleRPCConsume(r.Deliveries, r.Done)
	} else {
		r.ensurePendings()
	}
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

func (r *RabbitRPC) ensureRPCCorrelationID(pm *mqenv.MQPublishMessage) (string, string) {
	if "" == pm.CorrelationID {
		pm.CorrelationID = genCorrelationID()
	}
	originPm, _ := _cbs[pm.CorrelationID]
	if nil == originPm || originPm.Response == nil {
		_cbs[pm.CorrelationID] = pm
	}
	// exchangeName, routingKey := pm.Exchange, pm.RoutingKey
	// if "" != routingKey {
	// 	return exchangeName, routingKey
	// }
	return "", r.queueName
}

func handleRPCConsume(deliveries <-chan amqp.Delivery, done chan error) {
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
				if logger.IsDebugEnabled() {
					logger.Trace.Printf("====> push back response data %s %+v\n", correlationID, pm)
				}
				resp := generateMQResponseMessage(&d, d.Exchange)
				pm.Response <- *resp
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
