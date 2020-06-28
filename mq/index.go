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
func Init(mqConfigFile string, mqConfigs map[string]mqenv.MQConnectorConfig) error {
	mqEnv, err := InitMQRoutesEnv(mqConfigFile)
	if err != nil {
		return err
	}
	var lastErr error
	for category, cnf := range mqEnv.MQs {
		instCnf, ok := mqConfigs[cnf.Instance]
		if ok == false {
			logger.Error.Printf("Initialize mq:%s with connection instance:%s failed, the instance not configured.", category, cnf.Instance)
			lastErr = fmt.Errorf("Initialize mq:%s with connection instance:%s failed, the instance not configured", category, cnf.Instance)
		} else {
			var initErr error
			mqCategoryDrivers[category] = instCnf.Driver
			if instCnf.Driver == mqenv.DriverTypeAMQP {
				amqpCfg := &rabbitmq.AMQPConfig{
					Queue:           cnf.Queue,
					QueueDurable:    cnf.Durable,
					BindingExchange: cnf.Exchange.Name != "",
					ExchangeName:    cnf.Exchange.Name,
					ExchangeType:    cnf.Exchange.Type,
					BindingKey:      cnf.BindingKey,
				}
				if cnf.RPCEnabled {
					continue
				}
				_, initErr = rabbitmq.InitRabbitMQ(category, &instCnf, amqpCfg)
			} else if mqenv.DriverTypeKafka == instCnf.Driver {
				kafkaCfg := &kafka.Config{
					Topic:   cnf.Topic,
					GroupID: cnf.GroupID,
				}
				_, initErr = kafka.InitKafka(category, &instCnf, kafkaCfg)
			}

			if initErr != nil {
				logger.Error.Printf("Initialize mq:%s failed with error:%s", category, initErr.Error())
				lastErr = initErr
			}
		}
	}
	return lastErr
}

// InitMQWithRPC init mq with RPC
func InitMQWithRPC(key string, rpcType int, connCfg *mqenv.MQConnectorConfig, mqCfg *Config) error {
	if mqCfg == nil || connCfg == nil {
		return fmt.Errorf("Initialize mq rpc with key:%s rpc_type:%d failed, invalid conn_cofig or invalid mq_config", key, rpcType)
	}
	if connCfg.Driver == mqenv.DriverTypeAMQP {
		amqpCfg := &rabbitmq.AMQPConfig{
			Queue:           mqCfg.Queue,
			QueueDurable:    mqCfg.Durable,
			BindingExchange: mqCfg.Exchange.Name != "",
			ExchangeName:    mqCfg.Exchange.Name,
			ExchangeType:    mqCfg.Exchange.Type,
			BindingKey:      mqCfg.BindingKey,
		}
		if rabbitmq.InitRPCRabbitMQ(key, rpcType, connCfg, amqpCfg) == nil {
			return errors.New("Initialize rabbitmq mq rpc failed")
		}
	} else {
		logger.Error.Printf("Initialize mq rpc with key:%s rpc_type:%d and driver:%s failed, driver not supported", key, rpcType, connCfg.Driver)
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
		inst, err := rabbitmq.GetRabbitMQ(mqCategory)
		if nil != err {
			return err
		}
		pxy := rabbitmq.GenerateRabbitMQConsumerProxy(consumeProxy)
		inst.Consume <- pxy
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
		inst, err := rabbitmq.GetRabbitMQ(mqCategory)
		if nil != err {
			return err
		}
		msg := rabbitmq.GenerateRabbitMQPublishMessage(publishMsg)
		inst.Publish <- msg
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
