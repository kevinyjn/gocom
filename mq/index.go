package mq

import (
	"errors"
	"fmt"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/kafka"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/mq/rabbitmq"
)

// Constants
const (
	DriverTypeAMQP  = mqenv.DriverTypeAMQP
	DriverTypeKafka = mqenv.DriverTypeKafka

	MQTypeConsumer  = mqenv.MQTypeConsumer
	MQTypePublisher = mqenv.MQTypePublisher

	MQEventCodeOk     = mqenv.MQEventCodeOk
	MQEventCodeFailed = mqenv.MQEventCodeFailed
)

var mqCategoryDrivers = map[string]string{}

// Init initializer
func Init(mqConfigFile string, mqDriverConfigs map[string]mqenv.MQConnectorConfig) error {
	mqRoutesEnv, err := InitMQRoutesEnv(mqConfigFile)
	if err != nil {
		return err
	}
	return InitWithMQRoutes(mqRoutesEnv, mqDriverConfigs)
}

// InitWithMQRoutes patitionally init with RoutesEnv
func InitWithMQRoutes(mqRoutesEnv *RoutesEnv, mqDriverConfigs map[string]mqenv.MQConnectorConfig) error {
	var lastErr error
	for category, cnf := range mqRoutesEnv.MQs {
		err := InitMQTopic(category, &cnf, mqDriverConfigs)
		if nil != err {
			lastErr = err
		}
	}
	return lastErr
}

// InitMQTopic initialize sigle mq topic with drivers
func InitMQTopic(topicCategory string, topicConfig *Config, mqDriverConfigs map[string]mqenv.MQConnectorConfig) error {
	if nil == topicConfig {
		return fmt.Errorf("Initialize MQ topic for category:%s while topic config nil", topicCategory)
	}
	instCnf, ok := mqDriverConfigs[topicConfig.Instance]
	if ok == false {
		logger.Error.Printf("Initialize mq:%s with connection instance:%s failed, the instance not configured.", topicCategory, topicConfig.Instance)
		return fmt.Errorf("Initialize mq:%s with connection instance:%s failed, the instance not configured", topicCategory, topicConfig.Instance)
	}
	var initErr error
	mqCategoryDrivers[topicCategory] = instCnf.Driver
	if instCnf.Driver == mqenv.DriverTypeAMQP {
		amqpCfg := &rabbitmq.AMQPConfig{
			ConnConfigName:  topicConfig.Instance,
			Queue:           topicConfig.Queue,
			QueueDurable:    topicConfig.Durable,
			BindingExchange: topicConfig.Exchange.Name != "",
			ExchangeName:    topicConfig.Exchange.Name,
			ExchangeType:    topicConfig.Exchange.Type,
			BindingKey:      topicConfig.BindingKey,
		}
		if topicConfig.RPCEnabled {
			return nil
		}
		_, initErr = rabbitmq.InitRabbitMQ(topicCategory, &instCnf, amqpCfg)
	} else if mqenv.DriverTypeKafka == instCnf.Driver {
		kafkaCfg := &kafka.Config{
			Topic:   topicConfig.Topic,
			GroupID: topicConfig.GroupID,
		}
		_, initErr = kafka.InitKafka(topicCategory, &instCnf, kafkaCfg)
	}

	if initErr != nil {
		logger.Error.Printf("Initialize mq:%s failed with error:%s", topicCategory, initErr.Error())
		return initErr
	}
	return nil
}

// InitMQWithRPC init mq with RPC
func InitMQWithRPC(topicCategory string, rpcType int, connCfg *mqenv.MQConnectorConfig, mqCfg *Config) error {
	if mqCfg == nil || connCfg == nil {
		return fmt.Errorf("Initialize mq rpc with key:%s rpc_type:%d failed, invalid conn_cofig or invalid mq_config", topicCategory, rpcType)
	}
	if connCfg.Driver == mqenv.DriverTypeAMQP {
		if "" == mqCategoryDrivers[topicCategory] {
			mqCategoryDrivers[topicCategory] = connCfg.Driver
		}
		if nil == GetMQConfig(topicCategory) {
			GetMQRoutes()[topicCategory] = *mqCfg
		}
		amqpCfg := &rabbitmq.AMQPConfig{
			Queue:           mqCfg.Queue,
			QueueDurable:    mqCfg.Durable,
			BindingExchange: mqCfg.Exchange.Name != "",
			ExchangeName:    mqCfg.Exchange.Name,
			ExchangeType:    mqCfg.Exchange.Type,
			BindingKey:      mqCfg.BindingKey,
		}
		if rabbitmq.InitRPCRabbitMQ(topicCategory, rpcType, connCfg, amqpCfg) == nil {
			return errors.New("Initialize rabbitmq mq rpc failed")
		}
	} else {
		logger.Error.Printf("Initialize mq rpc with key:%s rpc_type:%d and driver:%s failed, driver not supported", topicCategory, rpcType, connCfg.Driver)
		return errors.New("Invalid mq rpc driver")
	}
	return nil
}

// GetRabbitMQ get rabbitmq
func GetRabbitMQ(name string) (*rabbitmq.RabbitMQ, error) {
	return rabbitmq.GetRabbitMQ(name)
}

// GetKafka get kafka
func GetKafka(name string) (*kafka.Kafka, error) {
	return kafka.GetKafka(name)
}

// ConsumeMQ consume
func ConsumeMQ(mqCategory string, consumeProxy *mqenv.MQConsumerProxy) error {
	var err error
	mqConfig := GetMQConfig(mqCategory)
	if nil == mqConfig {
		return fmt.Errorf("Consume MQ with invalid category:%s", mqCategory)
	}
	mqDriver := mqCategoryDrivers[mqCategory]
	if mqenv.DriverTypeAMQP == mqDriver {
		if mqConfig.RPCEnabled {
			rpcInst := rabbitmq.GetRPCRabbitMQWithoutConnectedChecking(mqCategory)
			if nil == rpcInst {
				return fmt.Errorf("No RPC rabbitmq instance by %s found", mqCategory)
			}
			pxy := rabbitmq.GenerateRabbitMQConsumerProxy(consumeProxy)
			rpcInst.Consume <- pxy
		} else {
			inst, err := rabbitmq.GetRabbitMQ(mqCategory)
			if nil != err {
				return err
			}
			pxy := rabbitmq.GenerateRabbitMQConsumerProxy(consumeProxy)
			inst.Consume <- pxy
		}
	} else if mqenv.DriverTypeKafka == mqDriver {
		inst, err := kafka.GetKafka(mqCategory)
		if nil != err {
			return err
		}
		pxy := kafka.GenerateKafkaConsumerProxy(consumeProxy)
		inst.Consume <- pxy
	} else {
		logger.Error.Printf("Consume MQ with category:%s failed, unknwon driver:%s", mqCategory, mqDriver)
		return fmt.Errorf("Invalid mq %s driver", mqCategory)
	}
	return err
}

// PublishMQ publish
func PublishMQ(mqCategory string, publishMsg *mqenv.MQPublishMessage) error {
	var err error
	mqConfig := GetMQConfig(mqCategory)
	if nil == mqConfig {
		return fmt.Errorf("Publish MQ with invalid category:%s", mqCategory)
	}
	mqDriver := mqCategoryDrivers[mqCategory]
	if mqenv.DriverTypeAMQP == mqDriver {
		if mqConfig.RPCEnabled {
			rpcInst := rabbitmq.GetRPCRabbitMQ(mqCategory)
			if nil == rpcInst {
				return fmt.Errorf("No RPC rabbitmq instance by %s found", mqCategory)
			}
			rpcInst.Publish <- publishMsg
		} else {
			inst, err := rabbitmq.GetRabbitMQ(mqCategory)
			if nil != err {
				return err
			}
			inst.Publish <- publishMsg
		}
	} else if mqenv.DriverTypeKafka == mqDriver {
		inst, err := kafka.GetKafka(mqCategory)
		if nil != err {
			return err
		}
		msg := kafka.GenerateKafkaPublishMessage(publishMsg, inst.Config.Topic)
		inst.Publish <- msg
	} else {
		logger.Error.Printf("Publish MQ with category:%s failed, unknwon driver:%s", mqCategory, mqDriver)
		return fmt.Errorf("Invalid mq %s driver", mqCategory)
	}
	return err
}

// QueryMQRPC publishes a message and waiting the response
func QueryMQRPC(mqCategory string, pm *mqenv.MQPublishMessage) (*mqenv.MQConsumerMessage, error) {
	mqConfig := GetMQConfig(mqCategory)
	if nil == mqConfig {
		return nil, fmt.Errorf("Query RPC MQ with invalid category:%s", mqCategory)
	}
	mqDriver := mqCategoryDrivers[mqCategory]
	if mqenv.DriverTypeAMQP == mqDriver {
		inst, err := rabbitmq.GetRabbitMQ(mqCategory)
		if nil != err {
			return nil, err
		}
		return inst.QueryRPC(pm)
	}
	return nil, fmt.Errorf("Query RPC MQ not supported driver:%s", mqDriver)
}

// SetupTrackerQueue name
func SetupTrackerQueue(queueName string) {
	// rabbitmq
	rabbitmq.SetupTrackerQueue(queueName)
	// kafka ...
}

// NewMQResponseMessage new mq response publish messge depends on mq consumer message
func NewMQResponseMessage(body []byte, cm *mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	return mqenv.NewMQResponseMessage(body, cm)
}
