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
	app.Get("/publish", testPublishWebMessage)

	mq.SetupTrackerQueue("tracker.example")
	env := config.GetEnv()
	mqCfg := env.MQs["default"]
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
	mqTopic = "stage2"
	configs := map[string]mqenv.MQConnectorConfig{
		"stage2": mqCfg,
	}
	mqConfig = getDemoMQConfig()
	mq.GetMQRoutes()[mqTopic] = *mqConfig
	err = mq.InitMQTopic(mqTopic, mqConfig, configs)
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", mqTopic, err)
		return err
	}
	consumerProxy2 := mqenv.MQConsumerProxy{
		Queue:       mqConfig.Queue,
		Callback:    handleMQServiceMessage2,
		ConsumerTag: mqTopic,
		AutoAck:     false,
	}
	err = mq.ConsumeMQ(mqTopic, &consumerProxy2)
	if nil != err {
		logger.Error.Printf("Initialize consumer %s failed with error:%v", mqTopic, err)
		return err
	}

	go func() {
		ticker1 := time.NewTimer(time.Second * 1)
		for {
			select {
			case <-ticker1.C:
				testPublishRPCMessage()
				break
			}
		}
		ticker1.Stop()
	}()

	return nil
}

func getDemoMQConfig() *mq.Config {
	o := &mq.Config{
		Instance: "stage2",
		Queue:    "demo.queue.amqprpc.testing",
		Exchange: mq.Exchange{
			Type:    "topic",
			Name:    "demo.ex.amqprpc.testing",
			Durable: true,
		},
		BindingKey:  "demo.bindings.amqprpc",
		RoutingKeys: map[string]string{},
		Durable:     true,
	}
	return o
}

func handleMQServiceMessage(cm mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	fmt.Printf("handling mq service response message ... %+v\n", cm)
	body := string(cm.Body)
	pm := &mqenv.MQPublishMessage{
		Body: []byte("2. " + body),
	}
	resp, err := mq.QueryMQ("stage2", pm)
	if nil != err {
		fmt.Printf("WARNING: query biz producer failed with error:%v\n", err)
		return mq.NewMQResponseMessage([]byte(err.Error()), &cm)
	}
	fmt.Printf("Got stage1 response %s %s : %s\n", resp.CorrelationID, resp.ReplyTo, resp.Body)
	response := mq.NewMQResponseMessage(resp.Body, &cm)
	return response
}

func handleMQServiceMessage2(cm mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	fmt.Printf("handling mq service response message stage 2 ... %+v\n", cm)
	response := mq.NewMQResponseMessage([]byte(fmt.Sprintf("Response: %s", string(cm.Body))), &cm)
	return response
}

func testPublishRPCMessage() {
	pm := &mqenv.MQPublishMessage{
		Body: []byte("1. Testing data"),
	}
	resp, err := mq.QueryMQ("biz-consumer", pm)
	if nil != err {
		fmt.Printf("WARNING: query biz consumer rpc failed with error:%v\n", err)
	} else {
		fmt.Printf("Got response %s %s : %s\n", resp.CorrelationID, resp.ReplyTo, resp.Body)
	}
}

func testPublishWebMessage(ctx iris.Context) {
	pm := &mqenv.MQPublishMessage{
		Body: []byte("1. Testing data"),
	}
	resp, err := mq.QueryMQ("biz-consumer", pm)
	if nil != err {
		ctx.WriteString(fmt.Sprintf("query failed with error:%v", err))
	} else {
		ctx.WriteString(fmt.Sprintf("Got response %s %s : %s", resp.CorrelationID, resp.ReplyTo, resp.Body))
	}
}
