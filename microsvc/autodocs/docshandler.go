package autodocs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/utils"
)

// DocsHandler interface of handler defination
type DocsHandler interface {
	GetGroup() string
	GetName() string
	GetRoutingKey() string
	GetSummary() string
	GetDescription() string
	GetOperationID() string
	GetAllowsMethod() string
	GetParametersDocsInfo() []ParameterInfo
	GetRequestBodyDocsInfo() RequestBodyInfo
	GetResponsesDocsInfo() map[string]SchemaInfo
}

// DocsHandlersManager interfaces of docs handler manager
type DocsHandlersManager interface {
	AllDocHandlers() []DocsHandler
}

// Docs constants
const (
	DocsRoute              = "/docs"
	DocsIndexRoute         = "/index"
	DocsSwaggerConfigRoute = "/swagger-config"
)

var (
	DocumentTitle       = "Document"
	DocumentDescription = "Document Description"
	DocumentVersion     = "0.0.1"
	OAuthURL            = ""

	_controllerGroupDescriptions   = map[string]string{}
	_controllerHandlerDescriptions = map[string]string{}
	_apiBaseRoute                  = "/api/"
	_docsHandlers                  DocsHandlersManager
)

// IsDocsHandlerLoaded boolean
func IsDocsHandlerLoaded() bool {
	return _docsHandlers != nil
}

// LoadDocsHandler docs handler, the method would affects unique load
func LoadDocsHandler(app *iris.Application, apiBaseRoute string, handlersManager DocsHandlersManager) {
	if nil != _docsHandlers || nil == app {
		return
	}
	_docsHandlers = handlersManager
	_apiBaseRoute = apiBaseRoute
	swaggerPath := getSwaggerPath()
	app.RegisterView(iris.HTML(swaggerPath, config.TemplateViewEndfix))
	app.StaticWeb(DocsRoute, swaggerPath)

	app.Get(DocsRoute, handlerRedirectToDocsPage)
	party := app.Party(DocsRoute)
	party.Get(DocsIndexRoute, handlerDocsPage)
	party.Get(DocsSwaggerConfigRoute, handlerGetSwaggerConfig)
}

// SetControllerGroupDescription for controller
func SetControllerGroupDescription(controllerName string, description string) {
	_controllerGroupDescriptions[controllerName] = description
}

// SetControllerHandlerDescription for controller name and handler name
func SetControllerHandlerDescription(controllerName string, handlerName string, description string) {
	_controllerHandlerDescriptions[controllerName+"."+handlerName] = description
}

// getSwaggerPath : find valid swagger static file path
func getSwaggerPath() string {
	checkPaths := []string{"../swagger", "./swagger"}
	swaggerPath := checkPaths[0]
	isSwaggerPathNotExists := true
	for _, p := range checkPaths {
		if utils.IsPathExists(p) {
			swaggerPath = p
			isSwaggerPathNotExists = false
			break
		}
	}

	if isSwaggerPathNotExists {
		_, file, _, ok := runtime.Caller(0)
		if ok {
			curDir := path.Dir(file)
			swaggerPath = path.Join(curDir, "swagger")
		}
	}
	return swaggerPath
}

// getBaseRoutePath calculate document base path, supports for nginx and kubernetes+istio deployment
func getBaseRoutePath(ctx iris.Context) string {
	routes := strings.Split(ctx.Path(), DocsRoute)
	return routes[0]
}

// handlerDocsPage document index page
func handlerDocsPage(ctx iris.Context) {
	ctx.ViewData("Title", DocumentTitle)
	ctx.View("index.html")
}

// handlerRedirectToDocsPage : redirects /docs to /docs/index
func handlerRedirectToDocsPage(ctx iris.Context) {
	ctx.Redirect(getBaseRoutePath(ctx) + DocsRoute + DocsIndexRoute)
}

func getAPISecurityInfo() (SecurityDefinitions, []interface{}) {
	secNames := []interface{}{}
	sec := SecurityDefinitions{
		APIKey: &BasicAuthInfo{
			Type: "apiKey",
			Name: "Authorization",
			In:   "header",
		},
		BasicAuth: &BasicAuthInfo{
			Type:   "http",
			Scheme: "basic",
			Name:   "Authorization",
		},
	}
	secNames = append(secNames, map[string][]string{
		"jwt":       {},
		"basicAuth": {},
	})
	if "" != OAuthURL {
		sec.PrestoreAuth = &PrestoreAuthInfo{
			Type:             "oauth2",
			AuthorizationUrl: OAuthURL,
			Flow:             "implicit",
		}
		secNames = append(secNames, map[string][]string{
			"prestoreAuth": {},
		})
	}
	return sec, secNames
}

// handlerGetSwaggerConfig : auto generates the handlers documents
func handlerGetSwaggerConfig(ctx iris.Context) {
	docsServerAPIPath := getBaseRoutePath(ctx) + _apiBaseRoute
	configs := SwaggerConfig{
		// Swagger: "2.0",
		OpenAPI: "3.0.0",
		Info: DocsInfo{
			Title:       DocumentTitle,
			Description: DocumentDescription,
			Version:     DocumentVersion,
		},
		Host:     ctx.Host(),
		BasePath: docsServerAPIPath,
		Servers: []ServerInfo{
			{
				URL:         docsServerAPIPath,
				Description: "Development Server",
			},
		},
		Tags:       []TagInfo{},
		Schemes:    []string{"https", "http"},
		Paths:      map[string]*PathInfo{},
		Components: Components{},
	}
	if "http" == ctx.Request().URL.Scheme || strings.HasPrefix(ctx.Request().Referer(), "http:") {
		configs.Schemes = []string{"http", "https"}
	}
	secs, secNames := getAPISecurityInfo()
	// configs.SecurityDefinitions = secs
	configs.Components.SecuritySchemes = secs
	controllerGroupDescriptions := _controllerGroupDescriptions
	controllerHandlerDescriptions := _controllerHandlerDescriptions
	if nil == controllerGroupDescriptions {
		controllerGroupDescriptions = map[string]string{}
	}
	if nil == controllerHandlerDescriptions {
		controllerHandlerDescriptions = map[string]string{}
	}

	if nil != _docsHandlers {
		handlers := _docsHandlers.AllDocHandlers()
		for _, h := range handlers {
			handlerDescription := controllerHandlerDescriptions[h.GetRoutingKey()]
			if "" == handlerDescription {
				handlerDescription = h.GetDescription()
			}
			qi := QueryInfo{
				Tags:        []string{h.GetGroup()},
				Summary:     h.GetSummary(),
				Consumes:    []string{"application/json"},
				Produces:    []string{"application/json"},
				Description: handlerDescription,
				OperationID: h.GetOperationID(),
				Responses:   h.GetResponsesDocsInfo(),
				Security:    secNames,
			}
			pn := fmt.Sprintf("/%s/%s", h.GetGroup(), h.GetName())
			pi, exists := configs.Paths[pn]
			if false == exists {
				pi = &PathInfo{}
				configs.Paths[pn] = pi
			}
			switch h.GetAllowsMethod() {
			case "GET":
				qi.Parameters = h.GetParametersDocsInfo()
				pi.Get = &qi
				break
			case "POST":
				qi.RequestBody = h.GetRequestBodyDocsInfo()
				pi.Post = &qi
				break
			case "PUT":
				qi.RequestBody = h.GetRequestBodyDocsInfo()
				pi.Put = &qi
				break
			case "PATCH":
				qi.RequestBody = h.GetRequestBodyDocsInfo()
				pi.Patch = &qi
				break
			case "DELETE":
				qi.RequestBody = h.GetRequestBodyDocsInfo()
				pi.Delete = &qi
				break
			case "HEAD":
				qi.Parameters = h.GetParametersDocsInfo()
				pi.Head = &qi
				break
			case "OPTION":
				qi.Parameters = h.GetParametersDocsInfo()
				pi.Option = &qi
				break
			default:
				qi.RequestBody = h.GetRequestBodyDocsInfo()
				pi.Post = &qi
				break
			}

			if "" == controllerGroupDescriptions[h.GetGroup()] {
				controllerGroupDescriptions[h.GetGroup()] = fmt.Sprintf("【%s】的接口清单", h.GetGroup())
			}
		}

		leftsTags := []TagInfo{}
		for k, v := range controllerGroupDescriptions {
			tag := TagInfo{
				Name:        k,
				Description: v,
			}
			if "mq-register" == tag.Name {
				leftsTags = append(leftsTags)
			} else {
				configs.Tags = append(configs.Tags, tag)
			}
		}
		for _, tag := range leftsTags {
			configs.Tags = append(configs.Tags, tag)
		}
	}

	body, err := json.Marshal(configs)
	if nil != err {
		logger.Error.Printf("get docs swagger config while serialize config failed with error:%+v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.WriteString(err.Error())
		return
	}
	ctx.Header("Content-Type", "application/json")
	ctx.WriteString(string(body))
}
