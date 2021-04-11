package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/testingutil"
)

func TestMockMQWithReply(t *testing.T) {
	mqCategory := "testing"
	topic := "testing.rpc"
	mq.InitMockMQTopic(mqCategory, topic)

	consuemrProxy := mqenv.MQConsumerProxy{
		Queue:       topic,
		ConsumerTag: topic,
		Callback: func(msg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
			fmt.Printf(" => Got mock mq consumer message with body:%s", string(msg.Body))
			return &mqenv.MQPublishMessage{
				Body:          []byte(`{"name": "testing-rpc response"}`),
				RoutingKey:    msg.ReplyTo,
				CorrelationID: msg.CorrelationID,
			}
		},
	}
	mq.ConsumeMQ(mqCategory, &consuemrProxy)
	pm := mqenv.MQPublishMessage{
		RoutingKey: topic,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"request": "testing-rpc"}`),
	}
	resp, err := mq.QueryMQ(mqCategory, &pm)
	testingutil.AssertNil(t, err, "mq.QueryMQ error")
	testingutil.AssertNotNil(t, resp, "mq.QueryMQ")
	fmt.Printf("Testing query mock mq response: %+v\n", resp)
}
