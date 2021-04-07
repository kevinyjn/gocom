package mq

import (
	"errors"
	"fmt"
	"sync"

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

var (
	mqCategoryDrivers           = map[string]string{}
	mqCategoriesByInstance      = map[string]map[string]bool{}
	mqConnConfigs               = map[string]mqenv.MQConnectorConfig{}
	mqCategoryDriversMutex      = sync.RWMutex{}
	mqCategoriesByInstanceMutex = sync.RWMutex{}
	mqConnConfigsMutex          = sync.RWMutex{}
)

// Init initializer
func Init(mqConfigFile string, mqDriverConfigs map[string]mqenv.MQConnectorConfig) error {
	mqRoutesEnv, err := InitMQRoutesEnv(mqConfigFile)
	if err != nil {
		return err
	}
	if nil != mqDriverConfigs {
		for connName, cfg := range mqDriverConfigs {
			mqConnConfigsMutex.Lock()
			_, ok := mqConnConfigs[connName]
			if false == ok {
				mqConnConfigs[connName] = cfg
			}
			mqConnConfigsMutex.Unlock()
		}
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
	if false == ok {
		mqConnConfigsMutex.RLock()
		instCnf, ok = mqConnConfigs[topicConfig.Instance]
		mqConnConfigsMutex.RUnlock()
		if false == ok {
			logger.Error.Printf("Initialize mq:%s with connection instance:%s failed, the instance not configured.", topicCategory, topicConfig.Instance)
			return fmt.Errorf("Initialize mq:%s with connection instance:%s failed, the instance not configured", topicCategory, topicConfig.Instance)
		}
	} else {
		mqConnConfigsMutex.Lock()
		_, ok = mqConnConfigs[topicConfig.Instance]
		if false == ok {
			mqConnConfigs[topicConfig.Instance] = instCnf
		}
		mqConnConfigsMutex.Unlock()
	}
	if nil == GetMQConfig(topicCategory) {
		GetMQRoutes()[topicCategory] = *topicConfig
	}
	var initErr error
	mqCategoryDriversMutex.Lock()
	mqCategoryDrivers[topicCategory] = instCnf.Driver
	mqCategoryDriversMutex.Unlock()
	mqCategoriesByInstanceMutex.Lock()
	if mqCategoriesByInstance[topicConfig.Instance] == nil {
		mqCategoriesByInstance[topicConfig.Instance] = map[string]bool{}
	}
	mqCategoriesByInstance[topicConfig.Instance][topicCategory] = true
	mqCategoriesByInstanceMutex.Unlock()
	if instCnf.Driver == mqenv.DriverTypeAMQP {
		amqpCfg := &rabbitmq.AMQPConfig{
			ConnConfigName:  topicConfig.Instance,
			Queue:           topicConfig.Queue,
			QueueDurable:    topicConfig.Durable,
			BindingExchange: topicConfig.Exchange.Name != "",
			ExchangeName:    topicConfig.Exchange.Name,
			ExchangeType:    topicConfig.Exchange.Type,
			BindingKey:      topicConfig.BindingKey,
			QueueAutoDelete: topicConfig.AutoDelete,
		}
		if topicConfig.RPCEnabled {
			return nil
		}
		_, initErr = rabbitmq.InitRabbitMQ(topicCategory, &instCnf, amqpCfg)
	} else if mqenv.DriverTypeKafka == instCnf.Driver {
		kafakCfg := kafka.Config{
			Hosts:              instCnf.Host,
			Partition:          topicConfig.Partition,
			GroupID:            topicConfig.GroupID,
			MaxPollIntervalMS:  topicConfig.MaxPollIntervalMS,
			SaslUsername:       instCnf.User,
			SaslPassword:       instCnf.Password,
			MessageType:        topicConfig.MessageType,
			UseOriginalContent: topicConfig.UseOriginalContent,
		}
		_, initErr = kafka.InitKafka(topicCategory, kafakCfg)
	}

	if initErr != nil {
		logger.Error.Printf("Initialize mq:%s failed with error:%s", topicCategory, initErr.Error())
		return initErr
	}
	return nil
}

// GetAllMQDriverConfigs configs
func GetAllMQDriverConfigs() map[string]mqenv.MQConnectorConfig {
	result := map[string]mqenv.MQConnectorConfig{}
	mqConnConfigsMutex.RLock()
	if nil != mqConnConfigs {
		for connName, cfg := range mqConnConfigs {
			result[connName] = cfg
		}
	}
	mqConnConfigsMutex.RUnlock()
	return result
}

// GetAllCategoryNamesByInstance by instancename
func GetAllCategoryNamesByInstance(instanceName string) []string {
	result := []string{}
	mqCategoriesByInstanceMutex.RLock()
	categories, ok := mqCategoriesByInstance[instanceName]
	if ok {
		for key := range categories {
			result = append(result, key)
		}
	}
	mqCategoriesByInstanceMutex.RUnlock()
	return result
}

// FindOneCategoryNameByInstance first hit category
func FindOneCategoryNameByInstance(instanceName string) string {
	category := ""
	mqCategoriesByInstanceMutex.RLock()
	categories, ok := mqCategoriesByInstance[instanceName]
	mqCategoriesByInstanceMutex.RUnlock()
	if ok {
		for key := range categories {
			category = key
			break
		}
	}
	return category
}

// InitMQWithRPC init mq with RPC
func InitMQWithRPC(topicCategory string, rpcType int, connCfg *mqenv.MQConnectorConfig, mqCfg *Config) error {
	if mqCfg == nil || connCfg == nil {
		return fmt.Errorf("Initialize mq rpc with key:%s rpc_type:%d failed, invalid conn_cofig or invalid mq_config", topicCategory, rpcType)
	}
	if connCfg.Driver == mqenv.DriverTypeAMQP {
		mqCategoryDriversMutex.Lock()
		if "" == mqCategoryDrivers[topicCategory] {
			mqCategoryDrivers[topicCategory] = connCfg.Driver
		}
		mqCategoryDriversMutex.Unlock()
		mqCategoriesByInstanceMutex.Lock()
		if mqCategoriesByInstance[mqCfg.Instance] == nil {
			mqCategoriesByInstance[mqCfg.Instance] = map[string]bool{}
		}
		mqCategoriesByInstance[mqCfg.Instance][topicCategory] = true
		mqCategoriesByInstanceMutex.Unlock()
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
			QueueAutoDelete: mqCfg.AutoDelete,
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

// GetKafka get kafka.
func GetKafka(name string) (*kafka.KafkaWorker, error) {
	return kafka.GetKafka(name)
}

// ConsumeMQ consume
func ConsumeMQ(mqCategory string, consumeProxy *mqenv.MQConsumerProxy) error {
	var err error
	mqConfig := GetMQConfig(mqCategory)
	if nil == mqConfig {
		return fmt.Errorf("Consume MQ with invalid category:%s", mqCategory)
	}
	mqCategoryDriversMutex.RLock()
	mqDriver := mqCategoryDrivers[mqCategory]
	mqCategoryDriversMutex.RUnlock()
	if mqenv.DriverTypeAMQP == mqDriver {
		if mqConfig.RPCEnabled {
			rpcInst := rabbitmq.GetRPCRabbitMQWithoutConnectedChecking(mqCategory)
			if nil == rpcInst {
				return fmt.Errorf("No RPC rabbitmq instance by %s found", mqCategory)
			}
			pxy := rabbitmq.GenerateRabbitMQConsumerProxy(consumeProxy, mqConfig.Exchange.Name)
			rpcInst.Consume <- pxy
		} else {
			inst, err := rabbitmq.GetRabbitMQ(mqCategory)
			if nil != err {
				return err
			}
			pxy := rabbitmq.GenerateRabbitMQConsumerProxy(consumeProxy, mqConfig.Exchange.Name)
			inst.Consume <- pxy
		}
	} else if mqenv.DriverTypeKafka == mqDriver {
		inst, err := kafka.GetKafka(mqCategory)
		if nil != err {
			return err
		}
		inst.Subscribe(consumeProxy.Queue, consumeProxy)

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
	mqCategoryDriversMutex.RLock()
	mqDriver := mqCategoryDrivers[mqCategory]
	mqCategoryDriversMutex.RUnlock()
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
		inst.Send(publishMsg.Exchange, publishMsg, false)

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
	mqCategoryDriversMutex.RLock()
	mqDriver := mqCategoryDrivers[mqCategory]
	mqCategoryDriversMutex.RUnlock()
	if mqenv.DriverTypeAMQP == mqDriver {
		inst, err := rabbitmq.GetRabbitMQ(mqCategory)
		if nil != err {
			return nil, err
		}
		return inst.QueryRPC(pm)
	} else if mqenv.DriverTypeKafka == mqDriver {
		inst, err := kafka.GetKafka(mqCategory)
		if nil != err {
			return nil, err
		}
		return inst.Send(pm.Exchange, pm, true)
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
