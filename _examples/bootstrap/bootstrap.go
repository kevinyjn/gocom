package main

import (
	"fmt"

	"github.com/kevinyjn/gocom/application"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"

	"github.com/kataras/iris"
)

func main() {
	env, err := config.Init("../etc/config.yaml")
	if err != nil {
		return
	}

	logger.Init(&env.Logger)

	listenAddr := fmt.Sprintf("%s:%d", env.Server.Host, env.Server.Port)
	logger.Info.Printf("Starting server on %s...", listenAddr)

	app := application.GetApplication(config.ServerMain)
	accessLogger, accessLoggerClose := logger.NewAccessLogger()
	defer accessLoggerClose()
	app.Use(accessLogger)
	app.RegisterView(iris.HTML(config.TemplateViewPath, config.TemplateViewEndfix))
	app.Favicon(config.Favicon)
	app.StaticWeb(config.StaticRoute, config.StaticAssets)
	app.Run(iris.Addr(listenAddr), iris.WithoutServerError(iris.ErrServerClosed))
}
