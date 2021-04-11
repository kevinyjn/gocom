package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/_examples/microservicedemo/controllers/demo"
	"github.com/kevinyjn/gocom/_examples/microservicedemo/models/params"
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/microsvc/serializers"
	"github.com/kevinyjn/gocom/testingutil"
)

func TestHelloHandler(t *testing.T) {
	testHelloDemoHandler(t, "hello", results.OK)
}

func TestHello2Handler(t *testing.T) {
	testHelloDemoHandler(t, "hello2", results.InnerError)
}

func TestHello3Handler(t *testing.T) {
	testHelloDemoHandler(t, "hello3", results.OK)
}

func TestHello4Handler(t *testing.T) {
	testHelloDemoHandler(t, "hello4", results.NotImplemented)
}

func TestHello5Handler(t *testing.T) {
	testHelloDemoHandler(t, "hello5", results.OK)
}

func TestHello6Handler(t *testing.T) {
	testHelloDemoHandler(t, "hello6", results.OK)
}

func testHelloDemoHandler(t *testing.T, handlerName string, expectedStatusCode int) {
	mqCategory := loadTestingControllers(t, demo.GetControllers())
	ctx := microsvc.SessionContext{AppID: "demoApp", UserID: "demoUser01"}
	param := params.HelloParam{Name: handlerName}
	response := results.NewResultObject()
	err := microsvc.QueryMQ(mqCategory, "demo."+handlerName, ctx, &serializers.JSONParam{DataPtr: &param}, &serializers.JSONParam{DataPtr: &response})
	testingutil.AssertNil(t, err, "microsvc.QueryMQ")
	testingutil.AssertEquals(t, expectedStatusCode, response.Code, "response.Status")
	fmt.Printf("Testing hello handler response:%+v\n", response)
}

func TestHelloHandlerWithBlackListDenied(t *testing.T) {
	mqCategory := loadTestingControllers(t, demo.GetControllers())
	ctx := microsvc.SessionContext{AppID: "demoApp", UserID: ""}
	param := params.HelloParam{Name: "hello"}
	response := results.NewResultObject()
	err := microsvc.QueryMQ(mqCategory, "demo.hello", ctx, &serializers.JSONParam{DataPtr: &param}, &serializers.JSONParam{DataPtr: &response})
	testingutil.AssertNil(t, err, "microsvc.QueryMQ")
	testingutil.AssertEquals(t, results.Forbidden, response.Code, "response.Status")
	fmt.Printf("Testing hello handler response:%+v\n", response)
}

func TestHelloHandlerWithValidateDenied(t *testing.T) {
	mqCategory := loadTestingControllers(t, demo.GetControllers())
	ctx := microsvc.SessionContext{AppID: "demoApp", UserID: "demoUser01"}
	param := params.HelloParam{Name: ""}
	response := results.NewResultObject()
	err := microsvc.QueryMQ(mqCategory, "demo.hello", ctx, &serializers.JSONParam{DataPtr: &param}, &serializers.JSONParam{DataPtr: &response})
	testingutil.AssertNil(t, err, "microsvc.QueryMQ")
	testingutil.AssertEquals(t, results.Forbidden, response.Code, "response.Status")
	fmt.Printf("Testing hello handler response:%+v\n", response)
}

func TestNonExistsHandler(t *testing.T) {
	mqCategory := loadTestingControllers(t, demo.GetControllers())
	ctx := microsvc.SessionContext{AppID: "demoApp", UserID: "demoUser01"}
	param := params.HelloParam{Name: "hello"}
	response := results.NewResultObject()
	err := microsvc.QueryMQ(mqCategory, "demo.non-exists", ctx, &serializers.JSONParam{DataPtr: &param}, &serializers.JSONParam{DataPtr: &response})
	testingutil.AssertNil(t, err, "microsvc.QueryMQ")
	testingutil.AssertEquals(t, results.NotFound, response.Code, "response.Status")
	fmt.Printf("Testing hello handler response:%+v\n", response)
}
