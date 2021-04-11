package microsvc

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/autodocs"
	"github.com/kevinyjn/gocom/microsvc/serializers"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

// LoadControllers load micro service controllers as MQ consumer handlers and RESTful handlers
func LoadControllers(topicCategory string, controllers []MQController, app ...*iris.Application) error {
	var err error
	for _, controller := range controllers {
		err = LoadController(topicCategory, controller)
		if nil != err {
			break
		}
	}
	InitMQRegisterController(topicCategory)
	if nil != app && len(app) > 0 && false == autodocs.IsDocsHandlerLoaded() {
		// load docs http handler
		autodocs.LoadDocsHandler(app[0], GetHandlers())
	}
	return err
}

// LoadController load micro service controller as MQ consumer handlers and RESTful handlers
func LoadController(topicCategory string, controller MQController) error {
	controller.SetTopicCategory(topicCategory)
	handlers := analyzeControllerHandlers(controller)
	err := GetHandlers().RegisterHandlers(topicCategory, handlers)
	if nil != err {
		ct := reflect.TypeOf(controller)
		if ct.Kind() == reflect.Ptr {
			ct = ct.Elem()
		}
		logger.Error.Printf("load controller %s failed with error:%v", ct.Name(), err)
	}
	return err
}

// QueryMQ with serializable parameter
func QueryMQ(topicCategory string, routingKey string, ctx SessionContext, param serializers.SerializableParam, response serializers.SerializableParam) error {
	if nil == param {
		logger.Error.Printf("Query mq with category:%s routingKey:%s while giving nil parameter", topicCategory, routingKey)
		return fmt.Errorf("Query parameter should not be empty")
	}
	body, err := param.Serialize()
	if nil != err {
		logger.Error.Printf("Query mq with category:%s routingKey:%s while serializing %s data failed with error:%v", topicCategory, routingKey, reflect.TypeOf(param).Elem().Name(), err)
		return err
	}
	pubMsg := mqenv.MQPublishMessage{
		RoutingKey: routingKey,
		Headers: map[string]string{
			"Content-Type": param.ContentType(),
		},
		Body:      body,
		MessageID: utils.GenLoweruuid(),
		AppID:     ctx.AppID,
		UserID:    ctx.UserID,
	}
	if nil == response {
		err = mq.PublishMQ(topicCategory, &pubMsg)
	} else {
		var cm *mqenv.MQConsumerMessage
		cm, err = mq.QueryMQ(topicCategory, &pubMsg)
		if nil != cm {
			err = response.ParseFrom(cm.Body)
			if nil != err {
				logger.Error.Printf("Query mq with category:%s routingKey:%s while parsing response to %s failed with error:%v", topicCategory, routingKey, reflect.TypeOf(response).Elem().Name(), err)
			}
		}
	}
	if nil != err {
		logger.Error.Printf("Query mq with category:%s routingKey:%s failed with error:%v", topicCategory, routingKey, err)
	}
	return err
}
