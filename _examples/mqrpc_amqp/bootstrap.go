package main

import (
	"fmt"

	"github.com/kevinyjn/gocom/application"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"

	"github.com/kataras/iris"
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

	listenAddr := fmt.Sprintf("%s:%d", env.Server.Host, env.Server.Port)
	logger.Info.Printf("Starting server on %s...", listenAddr)

	app := application.GetApplication(config.ServerMain)
	InitServiceHandler(app)

	accessLogger, accessLoggerClose := logger.NewAccessLogger()
	defer accessLoggerClose()
	app.Use(accessLogger)
	app.RegisterView(iris.HTML(config.TemplateViewPath, config.TemplateViewEndfix))
	app.Favicon(config.Favicon)
	app.StaticWeb(config.StaticRoute, config.StaticAssets)
	app.Run(iris.Addr(listenAddr), iris.WithoutServerError(iris.ErrServerClosed))
}
