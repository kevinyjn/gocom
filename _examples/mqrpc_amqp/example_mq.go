package main

import (
	"fmt"
	"time"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/healthz"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
)

// InitServiceHandler initialize
func InitServiceHandler(app *iris.Application) error {
	healthz.InitHealthz(app)
	healthz.SetTraceLog(true)
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

	err = mq.InitMQWithRPC("demo", mq.MQTypePublisher, &mqCfg, getDemoMQConfig())
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", "demo", err)
		return err
	}

	err = mq.InitMQWithRPC("rpc-consumer", mq.MQTypeConsumer, &mqCfg, mq.GetMQConfig("rpc-consumer"))
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", "rpc-consumer", err)
		return err
	}

	for mqInstance := range mq.GetAllMQDriverConfigs() {
		for _, name := range mq.GetAllCategoryNamesByInstance(mqInstance) {
			logger.Debug.Printf("category:%s on instance:%s", name, mqInstance)
		}
	}

	// httpclient.HTTPGet("http://127.0.0.1:8060/healthz", nil, httpclient.WithRetry(10))

	go func() {
		ticker1 := time.NewTimer(time.Second * 3)
		ticker2 := time.NewTicker(time.Second * 13)
		for {
			select {
			case <-ticker1.C:
				o := getDemoMQConfig()
				consumerProxy2 := mqenv.MQConsumerProxy{
					Queue:       o.Queue,
					Callback:    handleMQServiceMessage,
					ConsumerTag: "demo",
					AutoAck:     false,
				}
				mq.ConsumeMQ("demo", &consumerProxy2)
				break
			case <-ticker2.C:
				testPublishRPCMessage()
				break
			}
		}
		ticker1.Stop()
		ticker2.Stop()
	}()

	return nil
}

func getDemoMQConfig() *mq.Config {
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
	return o
}

func handleMQServiceMessage(mqMsg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
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
	return nil
}

func testPublishRPCMessage() {
	pm := &mqenv.MQPublishMessage{
		RoutingKey: "demo.bindings.amqprpc",
		Body:       []byte("Testing data"),
		Response:   make(chan mqenv.MQConsumerMessage),
		ReplyTo:    mq.GetMQConfig("rpc-consumer").Queue,
	}
	err := mq.PublishMQ("demo", pm)
	if nil != err {
		fmt.Printf("WARNING: could not get valid mq instance by demo\n")
	}

	ticker := time.NewTicker(time.Duration(3000) * time.Millisecond)
	select {
	case res := <-pm.Response:
		fmt.Printf("Got response: %s\n", string(res.Body))
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
		Response: make(chan mqenv.MQConsumerMessage),
		ReplyTo:  mq.GetMQConfig("rpc-consumer").Queue,
	}
	err := mq.PublishMQ("demo", pm)
	if nil != err {
		ctx.WriteString(err.Error())
		return
	}

	ticker := time.NewTicker(time.Duration(3000) * time.Millisecond)
	select {
	case res := <-pm.Response:
		ctx.WriteString(fmt.Sprintf("Got response: %s\n", string(res.Body)))
		break
	case <-ticker.C:
		pm.OnClosed()
		ctx.WriteString("Waiting response timeout\n")
		break
	}
}
