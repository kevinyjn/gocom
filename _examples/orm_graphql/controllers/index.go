package controllers

import (
	"github.com/kataras/iris"

	"github.com/kevinyjn/gocom/_examples/orm_graphql/controllers/api"
)

// InitAll initialize all handlers
func InitAll(app *iris.Application) {
	api.Init(app)
}
