package main

import (
	"fmt"

	"github.com/kevinyjn/gocom/_examples/orm_graphql/controllers"

	"github.com/kevinyjn/gocom/application"
	"github.com/kevinyjn/gocom/caching"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/utils"

	"github.com/kataras/iris"
)

func main() {
	env, err := config.Init(ConfigFile)
	if nil != err {
		return
	}

	logger.Init(&env.Logger)

	if len(env.MQs) > 0 {
		err = mq.Init(MQConfigFile, env.MQs)
		if nil != err {
			logger.Error.Printf("Please check the mq configuration and restart. error:%v", err)
		}
	}
	if len(env.Caches) > 0 {
		if !caching.InitCacheProxy(env.Caches) {
			logger.Warning.Println("Please check the caching configuration and restart.")
		}
	}
	if len(env.DBs) > 0 {
		for category, dbConfig := range env.DBs {
			_, err = rdbms.GetInstance().Init(category, &dbConfig)
			if nil != err {
				logger.Error.Printf("Please check the database:%s configuration and restart. error:%v", category, err)
			}
		}
	}

	listenAddr := fmt.Sprintf("%s:%d", env.Server.Host, env.Server.Port)
	logger.Info.Printf("Starting server on %s...", listenAddr)

	app := application.GetApplication(config.ServerMain)
	accessLogger, accessLoggerClose := logger.NewAccessLogger()
	defer accessLoggerClose()
	app.Use(accessLogger)
	controllers.InitAll(app)
	app.RegisterView(iris.HTML(config.TemplateViewPath, config.TemplateViewEndfix))
	if utils.IsPathExists(config.Favicon) {
		app.Favicon(config.Favicon)
	}
	app.StaticWeb(config.StaticRoute, config.StaticAssets)
	app.Run(iris.Addr(listenAddr), iris.WithoutServerError(iris.ErrServerClosed))
}
