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
		return fmt.Errorf("Could not get mq topic config of:%s", mqTopic)
	}
	consumerProxy := mqenv.MQConsumerProxy{
		Queue:       mqConfig.Queue,
		Callback:    handleMQServiceMessage,
		ConsumerTag: mqTopic,
		AutoAck:     false,
	}
	err := mq.ConsumeMQ(mqTopic, &consumerProxy)
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", mqTopic, err)
		return err
	}

	go func() {
		tiker := time.NewTicker(time.Second * 5)
		select {
		case <-tiker.C:
			testPublishFanoutMessage()
			break
		}
		tiker.Stop()
	}()

	return nil
}

func handleMQServiceMessage(mqMsg mqenv.MQConsumerMessage) []byte {
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
