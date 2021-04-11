package observers

import (
	"encoding/json"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
	"github.com/kevinyjn/gocom/mq"
)

// mqObserver outside system mq observer
type mqObserver struct {
	widgets.AbstractStrategyWidget
	topicCategory      string
	observerTopic      string
	observerRoutingKey string
	payloadFrom        string
}

// NewStdMQObserver new observer
func NewStdMQObserver(payloadFrom string, topicCategory string, observerTopic string, observerRoutingKey string, matchPattern string) Observer {
	observer := mqObserver{topicCategory: topicCategory, observerTopic: observerTopic, observerRoutingKey: observerRoutingKey, payloadFrom: payloadFrom}
	observer.Init("MQObserver", matchPattern)
	return &observer
}

// Visit records the access action
func (s *mqObserver) Observe(event events.Event) {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on observe with %s from %s.%s appId:%s userId:%s\n", s.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
	payload := event.GetOriginBody()
	if "result" == s.payloadFrom {
		body, err := json.Marshal(event.GetProcessResult())
		if nil != err {
			logger.Error.Printf("%s on observe with %s from %s.%s while serialize %+v to json failed with error:%v", s.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetProcessResult(), err)
		} else {
			payload = body
		}
	}
	pubMsg := events.CreatePublishMessageFromEventAndPayload(event, payload)
	pubMsg.ReplyTo = ""
	pubMsg.Exchange = s.observerTopic
	pubMsg.RoutingKey = s.observerRoutingKey
	mq.PublishMQ(s.topicCategory, &pubMsg)
}
