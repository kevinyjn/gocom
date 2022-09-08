package rabbitmq

import (
	"fmt"
	"sync"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"

	"github.com/streadway/amqp"
)

var (
	amqpRpcs      = map[string]*RabbitRPC{}
	_cbs          = make(map[string]*mqenv.MQPublishMessage)
	amqpRpcsMutex = sync.RWMutex{}
	_cbsMutex     = sync.RWMutex{}
)

// InitRPCRabbitMQ init
func InitRPCRabbitMQ(key string, rpcType int, connCfg *mqenv.MQConnectorConfig, amqpCfg *AMQPConfig) *RabbitRPC {
	if amqpCfg == nil || connCfg == nil {
		return nil
	}

	amqpRpcsMutex.RLock()
	r, exists := amqpRpcs[key]
	amqpRpcsMutex.RUnlock()
	if !exists {
		r = generateRPCRabbitMQInstance(key, rpcType, connCfg, amqpCfg)
		amqpRpcsMutex.Lock()
		amqpRpcs[key] = r
		amqpRpcsMutex.Unlock()
		return r
	} else if !r.Config.Equals(amqpCfg) {
		r.close()
		close(r.Close)

		r = generateRPCRabbitMQInstance(key, rpcType, connCfg, amqpCfg)
		amqpRpcsMutex.Lock()
		amqpRpcs[key] = r
		amqpRpcsMutex.Unlock()
		return r
	}
	return r
}

// GetRPCRabbitMQWithConsumers get instance
func GetRPCRabbitMQWithConsumers(key string) *RabbitRPC {
	amqpRpcsMutex.RLock()
	r, exists := amqpRpcs[key]
	amqpRpcsMutex.RUnlock()
	if !exists {
		return nil
	}
	if r.Channel == nil {
		logger.Warning.Printf("Get rabbit mq rpc publisher by key:%s while the publisher channel were not connected.", key)
		return nil
	}
	r.queuesStatusMutex.RLock()
	queueStatus, ok := r.queuesStatus[r.queueInfo.initialName]
	r.queuesStatusMutex.RUnlock()
	if false == ok || nil == queueStatus || 0 == queueStatus.RefreshingTime {
		logger.Warning.Printf("Get rabbit mq rpc publisher by key:%s while the publisher channel queue were not initialized.", key)
		return nil
	}
	if r.CheckQueueConsumers(r.queueInfo.initialName) < 1 {
		// there is no consumres
		return nil
	}
	return r
}

// GetRPCRabbitMQWithoutConnectedChecking get instance
func GetRPCRabbitMQWithoutConnectedChecking(key string) *RabbitRPC {
	amqpRpcsMutex.RLock()
	r, exists := amqpRpcs[key]
	amqpRpcsMutex.RUnlock()
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

func (r *RabbitRPC) ensureRPCCorrelationID(pm *mqenv.MQPublishMessage) (string, string) {
	if "" == pm.CorrelationID {
		pm.CorrelationID = utils.GenLoweruuid()
	}
	_cbsMutex.Lock()
	originPm, _ := _cbs[pm.CorrelationID]
	if nil == originPm || originPm.Response == nil {
		_cbs[pm.CorrelationID] = pm
	}
	_cbsMutex.Unlock()
	// exchangeName, routingKey := pm.Exchange, pm.RoutingKey
	// if "" != routingKey {
	// 	return exchangeName, routingKey
	// }
	return "", r.queueInfo.name
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
		_cbsMutex.RLock()
		pm, pmExists := _cbs[correlationID]
		_cbsMutex.RUnlock()
		if pmExists {
			if pm.CallbackEnabled() {
				if logger.IsDebugEnabled() {
					logger.Trace.Printf("====> push back response data %s %+v\n", correlationID, pm)
				}
				resp := generateMQResponseMessage(&d, d.Exchange)
				pm.Response <- *resp
			}
			_cbsMutex.Lock()
			delete(_cbs, correlationID)
			_cbsMutex.Unlock()
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
