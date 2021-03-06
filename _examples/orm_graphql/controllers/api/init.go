package api

import (
	"github.com/kataras/iris"

	"github.com/kevinyjn/gocom/_examples/orm_graphql/models"
	"github.com/kevinyjn/gocom/orm/rdbms"
)

// Init initialize all api handlers
func Init(app *iris.Application) {
	apiApp := app.Party(GraphQLRoute, beforeAPIAuthMiddlewareHandler)
	beans := models.AllModelStructures()
	rdbms.RegisterGraphQLRoutes(apiApp, beans)
	rdbms.RegisterGraphQLMQs(APIConsumerMQCategory, APIProduceMQCategory, beans)
}
