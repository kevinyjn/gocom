package visitors

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
	"github.com/kevinyjn/gocom/mq"
)

// mqVisitor outside system mq visitor
type mqVisitor struct {
	widgets.AbstractStrategyWidget
	topicCategory     string
	visitorTopic      string
	visitorRoutingKey string
}

// NewStdMQVisitor new visitor
func NewStdMQVisitor(topicCategory string, visitorTopic string, visitorRoutingKey string, matchPattern string) Visitor {
	visitor := mqVisitor{topicCategory: topicCategory, visitorTopic: visitorTopic, visitorRoutingKey: visitorRoutingKey}
	visitor.Init("MQVisitor", matchPattern)
	return &visitor
}

// Visit records the access action
func (s *mqVisitor) Visit(event events.Event) {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on observe with %s from %s.%s appId:%s userId:%s\n", s.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
	payload := event.GetOriginBody()
	pubMsg := events.CreatePublishMessageFromEventAndPayload(event, payload)
	pubMsg.ReplyTo = ""
	pubMsg.Exchange = s.visitorTopic
	pubMsg.RoutingKey = s.visitorRoutingKey
	mq.PublishMQ(s.topicCategory, &pubMsg)
}
