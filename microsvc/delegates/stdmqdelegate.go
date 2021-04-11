package delegates

import (
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
	"github.com/kevinyjn/gocom/mq"
)

// mqDelegate outside system mq delegate
type mqDelegate struct {
	widgets.AbstractStrategyWidget
	topicCategory      string
	delegateTopic      string
	delegateRoutingKey string
}

// NewStdMQDelegate new mq delegate
func NewStdMQDelegate(topicCategory string, delegateTopic string, delegateRoutingKey string, matchPattern string) Delegate {
	delegate := mqDelegate{topicCategory: topicCategory, delegateTopic: delegateTopic, delegateRoutingKey: delegateRoutingKey}
	delegate.Init("MQDelegate", matchPattern)
	return &delegate
}

// Visit records the access action
func (s *mqDelegate) Execute(event events.Event) events.Event {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on delegate with %s from %s.%s appId:%s userId:%s\n", s.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
	payload := event.GetOriginBody()
	pubMsg := events.CreatePublishMessageFromEventAndPayload(event, payload)
	pubMsg.ReplyTo = ""
	pubMsg.Exchange = s.delegateTopic
	pubMsg.RoutingKey = s.delegateRoutingKey
	resp, err := mq.QueryMQ(s.topicCategory, &pubMsg)
	var respEvent events.Event
	if nil != err {
		logger.Error.Printf("%s on delegate with %s from %s.%s while execute delegate to %s topic:%s routingKey:%s failed with error:%v", s.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), s.topicCategory, s.delegateTopic, s.delegateRoutingKey, err)
		resObj := results.NewResultObjectWithRequestSn(event.GetIdentifier())
		resObj.Code = results.InnerError
		resObj.Message = err.Error()
		respEvent = event.Clone()
		respEvent.SetOriginBody([]byte(resObj.Encode()))
	} else {
		respEvent = events.CreateEventFromConsumerMessage(event.GetCategory(), event.GetFrom(), *resp)
	}
	return respEvent
}
