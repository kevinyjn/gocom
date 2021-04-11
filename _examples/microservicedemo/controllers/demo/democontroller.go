package demo

import (
	"encoding/json"
	"fmt"

	"github.com/kevinyjn/gocom/_examples/microservicedemo/models/params"
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/mq/mqenv"
)

// demoController processes with demo business
type demoController struct {
	microsvc.AbstractController `serialize:""`
	SerializationStrategy       interface{} `serialization:"json"`
	FiltersStrategy             string      `filters:"BlackUserFilter,AppFilter"`
	Name                        string
	observers                   []string
}

// HandleHello : hello event handle
func (c *demoController) HandleHello(msg mqenv.MQConsumerMessage, param params.HelloParam) (params.HelloResponse, microsvc.HandlerError) {
	response := params.HelloResponse{Name: param.Name, Title: "hello"}
	fmt.Printf(" ==> handler process HandleHello %+v\n", param)
	return response, nil
}

// HandleHello2 : hello event handle
func (c *demoController) HandleHello2(msg mqenv.MQConsumerMessage, param *params.HelloParam) (*params.HelloResponse, error) {
	response := params.HelloResponse{Name: param.Name, Title: "hello"}
	fmt.Printf(" ==> handler process HandleHello2 %+v\n", param)
	return &response, fmt.Errorf("Demo of error response")
}

// HandleHello3 : hello event handle
func (c *demoController) HandleHello3(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "hello"}
	fmt.Printf(" ==> handler process HandleHello3 %+v\n", param)
	return response
}

// HandleHello4 : hello event handle
func (c *demoController) HandleHello4(msg mqenv.MQConsumerMessage, param *params.HelloParam) (*params.HelloResponse, microsvc.HandlerError) {
	fmt.Printf(" ==> handler process HandleHello4 %+v\n", param)
	return nil, microsvc.NewHandlerError(results.NotImplemented, "Testing not implemented handler")
}

// HandleHello5 : hello event handle
func (c *demoController) HandleHello5(msg mqenv.MQConsumerMessage, param params.HelloParam) mqenv.MQPublishMessage {
	fmt.Printf(" ==> handler process HandleHello5 %+v\n", param)
	response := params.HelloResponse{Name: param.Name, Title: "hello"}
	respObj := results.NewResultObjectWithRequestSn(msg.MessageID)
	respObj.Code = results.OK
	respObj.Message = "Success"
	respObj.Data = response
	body, _ := json.Marshal(&respObj)
	pubMsg := mqenv.NewMQResponseMessage(body, &msg)
	return *pubMsg
}

// HandleHello6 : hello event handle
func (c *demoController) HandleHello6(msg mqenv.MQConsumerMessage, param params.HelloParam) (*mqenv.MQPublishMessage, microsvc.HandlerError) {
	fmt.Printf(" ==> handler process HandleHello6 %+v\n", param)
	response := params.HelloResponse{Name: param.Name, Title: "hello"}
	respObj := results.NewResultObjectWithRequestSn(msg.MessageID)
	respObj.Code = results.OK
	respObj.Message = "Success"
	respObj.Data = response
	body, _ := json.Marshal(&respObj)
	pubMsg := mqenv.NewMQResponseMessage(body, &msg)
	return pubMsg, nil
}
