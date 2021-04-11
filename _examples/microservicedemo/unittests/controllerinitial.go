package unittests

import (
	"testing"

	"github.com/kevinyjn/gocom/_examples/microservicedemo/middlewares"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/testingutil"
)

// loadTestingControllers load controllers for testing and returns testing mq topic category
func loadTestingControllers(t *testing.T, controllers []microsvc.MQController) string {
	mqCategory := "testing"
	topic := "testing.rpc"
	mq.InitMockMQTopic(mqCategory, topic)
	middlewares.Init()
	err := microsvc.LoadControllers(mqCategory, controllers)
	testingutil.AssertNil(t, err, "microsvc.LoadControllers")
	return mqCategory
}
