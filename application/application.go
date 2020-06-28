package application

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

var _apps = make(map[string]*iris.Application)

// GetApplication application
func GetApplication(appName string) *iris.Application {
	app := _apps[appName]
	if app == nil {
		app = iris.New()
		_apps[appName] = app
	}
	return app
}

// HandleGet handler
func HandleGet(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Get(relativePath, handlers...)
}

// HandlePost handler
func HandlePost(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Post(relativePath, handlers...)
}

// HandlePut handler
func HandlePut(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Put(relativePath, handlers...)
}

// HandleDelete handler
func HandleDelete(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Delete(relativePath, handlers...)
}

// HandleConnect handler
func HandleConnect(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Connect(relativePath, handlers...)
}

// HandleHead handler
func HandleHead(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Head(relativePath, handlers...)
}

// HandleOptions handler
func HandleOptions(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Options(relativePath, handlers...)
}

// HandlePatch handler
func HandlePatch(app *iris.Application, relativePath string, handlers ...context.Handler) {
	app.Patch(relativePath, handlers...)
}
