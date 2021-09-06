package controllers

import (
	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/_examples/microservicedemo/controllers/demo"
	"github.com/kevinyjn/gocom/_examples/microservicedemo/middlewares"
	"github.com/kevinyjn/gocom/healthz"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/microsvc/acl"
	"github.com/kevinyjn/gocom/microsvc/acl/builtincontrollers"
)

// Constants
const (
	BizConsumerCategory = "biz-consumer"
)

// Init all controllers
func Init(app *iris.Application) bool {
	middlewares.Init()
	healthz.SetDelayCheckNetworking(60)
	controllers := demo.GetControllers()
	microsvc.LoadControllers(BizConsumerCategory, builtincontrollers.GetControllers(), app)
	microsvc.LoadControllers(BizConsumerCategory, controllers, app)
	acl.InitBuiltinRBACModels()
	return true
}
