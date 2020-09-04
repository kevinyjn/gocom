package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
)

// InitServiceHandler initialize
func InitServiceHandler(app *iris.Application) error {
	app.Get("/test", testPublishWebMessage)

	mqTopic := "biz-consumer"
	env := config.GetEnv()
	mqCfg := env.MQs["default"]
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

	o := &mq.Config{
		Instance: "default",
		Queue:    "demo.queue.amqprpc.testing",
		Exchange: mq.Exchange{
			Type:    "topic",
			Name:    "demo.ex.amqprpc.testing",
			Durable: true,
		},
		BindingKey:  "demo.bindings.amqprpc",
		RoutingKeys: map[string]string{},
		Durable:     true,
		RPCEnabled:  true,
	}
	err = mq.InitMQWithRPC("demo", mq.MQTypePublisher, &mqCfg, o)
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", "demo", err)
		return err
	}

	consumerProxy2 := mqenv.MQConsumerProxy{
		Queue:       o.Queue,
		Callback:    handleMQServiceMessage,
		ConsumerTag: "demo",
		AutoAck:     false,
	}
	err = mq.ConsumeMQ("demo", &consumerProxy2)
	if nil != err {
		logger.Error.Printf("consumes %s failed with error:%v", "demo", err)
		return err
	}

	err = mq.InitMQWithRPC("rpc-consumer", mq.MQTypeConsumer, &mqCfg, mq.GetMQConfig("rpc-consumer"))
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", "rpc-consumer", err)
		return err
	}

	return err
}

func handleMQServiceMessage(mqMsg mqenv.MQConsumerMessage) {
	fmt.Println("handling mq service response message...", mqMsg)
	mqTopic := "rpc-consumer"
	pubMsg := mqenv.MQPublishMessage{
		Body:          []byte("Responsed testing data"),
		RoutingKey:    mqMsg.ReplyTo,
		CorrelationID: mqMsg.CorrelationID,
		Headers:       map[string]string{"User-Agent": "userAgent", "BizCode": "bizCode"},
	}

	err := mq.PublishMQ(mqTopic, &pubMsg)
	if nil != err {
		logger.Error.Printf("Publish mq request to service side failed with error:%v", err)
	}
}

func testPublishFanoutMessage() {
	pm := &mqenv.MQPublishMessage{
		Body:     []byte("Testing data"),
		Response: make(chan []byte),
		ReplyTo:  mq.GetMQConfig("rpc-consumer").Queue,
	}
	mq.PublishMQ("demo", pm)
	// rpc := rabbitmq.GetRPCRabbitMQ("biz-consumer")
	// rpc.Publish <- pm

	ticker := time.NewTicker(time.Duration(3000) * time.Millisecond)
	select {
	case res := <-pm.Response:
		fmt.Printf("Got response: %s\n", string(res))
		break
	case <-ticker.C:
		pm.OnClosed()
		fmt.Printf("Waiting response timeout\n")
		break
	}
}

func testPublishWebMessage(ctx iris.Context) {
	pm := &mqenv.MQPublishMessage{
		Body:     []byte("Testing data"),
		Response: make(chan []byte),
		ReplyTo:  mq.GetMQConfig("rpc-consumer").Queue,
	}
	mq.PublishMQ("demo", pm)
	// rpc := rabbitmq.GetRPCRabbitMQ("biz-consumer")
	// rpc.Publish <- pm

	ticker := time.NewTicker(time.Duration(3000) * time.Millisecond)
	select {
	case res := <-pm.Response:
		ctx.WriteString(fmt.Sprintf("Got response: %s\n", string(res)))
		break
	case <-ticker.C:
		pm.OnClosed()
		ctx.WriteString("Waiting response timeout\n")
		break
	}
}
