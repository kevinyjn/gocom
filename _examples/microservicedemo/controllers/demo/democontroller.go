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
	microsvc.AbstractController
	SerializationStrategy interface{} `serialization:"json"`
	FiltersStrategy       string      `filters:"BlackUserFilter,AppFilter"`
	Name                  string
	observers             []string
}

// HandleHello : hello event handle
func (c *demoController) HandleHello(msg mqenv.MQConsumerMessage, param params.HelloParam) (params.HelloResponse, microsvc.HandlerError) {
	response := params.HelloResponse{Name: param.Name, Title: "HandleHello"}
	fmt.Printf(" ==> handler process HandleHello %+v\n", param)
	return response, nil
}

// HandleHello2 : hello event handle
func (c *demoController) HandleHello2(msg mqenv.MQConsumerMessage, param *params.HelloParam) (*params.HelloResponse, error) {
	response := params.HelloResponse{Name: param.Name, Title: "HandleHello2"}
	fmt.Printf(" ==> handler process HandleHello2 %+v\n", param)
	return &response, fmt.Errorf("Demo of error response")
}

// HandleHello3 : hello event handle
func (c *demoController) HandleHello3(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "HandleHello3"}
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
	response := params.HelloResponse{Name: param.Name, Title: "HandleHello5"}
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
	response := params.HelloResponse{Name: param.Name, Title: "HandleHello6"}
	respObj := results.NewResultObjectWithRequestSn(msg.MessageID)
	respObj.Code = results.OK
	respObj.Message = "Success"
	respObj.Data = response
	body, _ := json.Marshal(&respObj)
	pubMsg := mqenv.NewMQResponseMessage(body, &msg)
	return pubMsg, nil
}

// GetHello : hello event handle
func (c *demoController) GetHello(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "GetHello"}
	fmt.Printf(" ==> handler process GetHello %+v\n", param)
	return response
}

// PutHello : hello event handle
func (c *demoController) PutHello(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "PutHello"}
	fmt.Printf(" ==> handler process PutHello %+v\n", param)
	return response
}

// PatchHello : hello event handle
func (c *demoController) PatchHello(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "PatchHello"}
	fmt.Printf(" ==> handler process PatchHello %+v\n", param)
	return response
}

// DeleteHello : hello event handle
func (c *demoController) DeleteHello(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "DeleteHello"}
	fmt.Printf(" ==> handler process DeleteHello %+v\n", param)
	return response
}

// HeadHello : hello event handle
func (c *demoController) HeadHello(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "HeadHello"}
	fmt.Printf(" ==> handler process HeadHello %+v\n", param)
	return response
}

// OptionHello : hello event handle
func (c *demoController) OptionHello(param params.HelloParam) params.HelloResponse {
	response := params.HelloResponse{Name: param.Name, Title: "OptionHello"}
	fmt.Printf(" ==> handler process OptionHello %+v\n", param)
	return response
}
