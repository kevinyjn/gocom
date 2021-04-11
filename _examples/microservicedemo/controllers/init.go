package controllers

import (
	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/_examples/microservicedemo/controllers/demo"
	"github.com/kevinyjn/gocom/_examples/microservicedemo/middlewares"
	"github.com/kevinyjn/gocom/microsvc"
)

// Constants
const (
	BizConsumerCategory = "biz-consumer"
)

// Init all controllers
func Init(app *iris.Application) bool {
	middlewares.Init()
	controllers := demo.GetControllers()
	microsvc.LoadControllers(BizConsumerCategory, controllers, app)
	return true
}
