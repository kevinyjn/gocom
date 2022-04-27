package main

import (
	"fmt"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
)

// InitServiceHandler initialize
func InitServiceHandler() error {
	mqTopic := "biz-consumer"
	mqConfig := mq.GetMQConfig(mqTopic)
	if nil == mqConfig {
		return fmt.Errorf("could not get mq topic config of:%s", mqTopic)
	}
	consumerProxy := mqenv.MQConsumerProxy{
		Queue:       mqConfig.Queue,
		Callback:    handleMQServiceMessage,
		ConsumerTag: mqTopic,
		AutoAck:     false,
	}
	if consumerProxy.Queue == "" {
		consumerProxy.Queue = mqConfig.Topic
	}
	err := mq.ConsumeMQ(mqTopic, &consumerProxy)
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", mqTopic, err)
		return err
	}
	// testEnsureKafkaProducer()

	go func() {
		tiker := time.NewTicker(time.Second * 5)
		<-tiker.C
		testPublishFanoutMessage()
		tiker.Stop()
	}()

	return nil
}

func handleMQServiceMessage(mqMsg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	fmt.Println("handling mq service response message...")
	return nil
}

func testPublishFanoutMessage() {
	mqTopic := "biz-producer"
	mqConfig := mq.GetMQConfig(mqTopic)
	if nil == mqConfig {
		return
	}
	pubMsg := mqenv.MQPublishMessage{
		Body:       []byte("Testing data"),
		RoutingKey: mqConfig.RoutingKeys["biz"],
		Headers:    map[string]string{"User-Agent": "userAgent", "BizCode": "bizCode"},
	}

	err := mq.PublishMQ(mqTopic, &pubMsg)
	if nil != err {
		logger.Error.Printf("Publish mq request to service side failed with error:%v", err)
	}
}

func testEnsureKafkaProducer() {
	category := "kafka-demo"
	mqConnConfigs := map[string]mqenv.MQConnectorConfig{
		category: generateKafkaConnection(),
	}
	mqCfg := &mq.Config{
		Instance: category,
		Queue:    "",
		Exchange: mq.Exchange{
			Name:    "",
			Type:    "",
			Durable: true,
		},
		BindingKey:  "",
		RoutingKeys: map[string]string{},
		Durable:     true,
		AutoDelete:  true,
		Topic:       "mq.kafka.producer.demo01",
		GroupID:     "01",
	}
	err := mq.InitMQTopic("demo-kafka-producer", mqCfg, mqConnConfigs)
	if nil != err {
		logger.Error.Printf("Initialize mq topic for category:%s with config:%+v failed with error:%v", category, mqCfg, err)
		return
	}
}

func generateKafkaConnection() mqenv.MQConnectorConfig {
	connCfg := mqenv.MQConnectorConfig{
		Driver:    "kafka",
		Host:      "localhost",
		Port:      9092,
		Path:      "",
		User:      "",
		Password:  "",
		Timeout:   0,
		Heartbeat: 0,
	}
	return connCfg
}
