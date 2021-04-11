package widgets

// // MQHandler interface of mq handler
// type MQHandler interface {
// 	GetGroup() string
// 	GetName() string
// 	Execute(mqenv.MQConsumerMessage) *mqenv.MQPublishMessage
// 	SetMQTopic(string)
// 	GetMQTopic() string
// }

// // MQHandlers interface of mq handler manager
// type MQHandlers interface {
// 	Execute(mqenv.MQConsumerMessage) *mqenv.MQPublishMessage
// 	RegisterHandlers(topicCategory string, hs map[string]MQHandler) error
// 	GetHandler(routingKey string) MQHandler
// 	AllDocHandlers() []MQHandler
// }
