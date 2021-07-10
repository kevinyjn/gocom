package main

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/_examples/microservicedemo/controllers"
	"github.com/kevinyjn/gocom/application"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/healthz"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"
)

func main() {
	env, err := config.Init("../etc/config.yaml")
	if err != nil {
		return
	}

	logger.Init(&env.Logger)

	err = mq.Init("../etc/mq.yaml", env.MQs)
	if err != nil {
		logger.Error.Printf("Please check the mq configuration and restart. error:%s", err.Error())
	}

	app := application.GetApplication(config.ServerMain)
	controllers.Init(app)

	listenAddr := fmt.Sprintf("%s:%d", env.Server.Host, env.Server.Port)
	logger.Info.Printf("Starting server on %s...", listenAddr)

	accessLogger, accessLoggerClose := logger.NewAccessLogger()
	defer accessLoggerClose()
	app.Use(accessLogger)
	healthz.InitHealthz(app)
	app.Run(iris.Addr(listenAddr), iris.WithoutServerError(iris.ErrServerClosed))
}
