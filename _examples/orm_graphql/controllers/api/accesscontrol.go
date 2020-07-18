package api

import (
	"github.com/kataras/iris"
)

func beforeAPIAuthMiddlewareHandler(ctx iris.Context) {
	ctx.Next()
}
