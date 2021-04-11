package autodocs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

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
	APIBaseURI          = "/api/"
	OAuthURL            = ""

	_controllerGroupDescriptions   = map[string]string{}
	_controllerHandlerDescriptions = map[string]string{}

	_docsHandlerLoaded                 = false
	_handlersManager   HandlersManager = nil
)

// Handler interface of handler defination
type Handler interface {
	GetGroup() string
	GetName() string
	GetRoutingKey() string
	GetSummary() string
	GetDescription() string
	GetOperationID() string
	GetParametersDocsInfo() []ParameterInfo
	// GetRequestBodyDocsInfo() RequestBodyInfo
	GetResponsesDocsInfo() map[string]SchemaInfo
	Execute(mqenv.MQConsumerMessage) *mqenv.MQPublishMessage
}

// HandlersManager interfaces of handler manager
type HandlersManager interface {
	AllDocHandlers() []Handler
	GetHandler(routingKey string) Handler
}

// IsDocsHandlerLoaded boolean
func IsDocsHandlerLoaded() bool {
	return _docsHandlerLoaded
}

// LoadDocsHandler docs handler, the method would affects unique load
func LoadDocsHandler(app *iris.Application, handlersManager HandlersManager) {
	if _docsHandlerLoaded || nil == app {
		return
	}
	_docsHandlerLoaded = true
	_handlersManager = handlersManager
	swaggerPath := getSwaggerPath()
	app.RegisterView(iris.HTML(swaggerPath, config.TemplateViewEndfix))
	app.StaticWeb(DocsRoute, swaggerPath)

	app.Get(DocsRoute, handlerRedirectToDocsPage)
	app.Post(APIBaseURI+"{groupName:string}/{apiName:string}", handlerTryoutAPICase)
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
	ctx.View("index.html")
}

// handlerRedirectToDocsPage : redirects /docs to /docs/index
func handlerRedirectToDocsPage(ctx iris.Context) {
	ctx.Redirect(getBaseRoutePath(ctx) + DocsRoute + DocsIndexRoute)
}

func getAPISecurityInfo() (SecurityDefinitions, []interface{}) {
	secNames := []interface{}{}
	sec := SecurityDefinitions{
		APIKey: APIKeyInfo{
			Type: "apiKey",
			Name: "X-API-Key",
			In:   "header",
		},
	}
	secNames = append(secNames, map[string][]string{
		"apiKey": {},
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
	configs := SwaggerConfig{
		Swagger: "2.0",
		Info: DocsInfo{
			Title:       DocumentTitle,
			Description: DocumentDescription,
			Version:     DocumentVersion,
		},
		Host:     ctx.Host(),
		BasePath: getBaseRoutePath(ctx) + APIBaseURI,
		Tags:     []TagInfo{},
		Schemes:  []string{"https", "http"},
		Paths:    map[string]PathInfo{},
	}
	secs, secNames := getAPISecurityInfo()
	configs.SecurityDefinitions = secs
	controllerGroupDescriptions := _controllerGroupDescriptions
	controllerHandlerDescriptions := _controllerHandlerDescriptions
	if nil == controllerGroupDescriptions {
		controllerGroupDescriptions = map[string]string{}
	}
	if nil == controllerHandlerDescriptions {
		controllerHandlerDescriptions = map[string]string{}
	}

	if nil != _handlersManager {
		handlers := _handlersManager.AllDocHandlers()
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
				Parameters:  h.GetParametersDocsInfo(),
				Responses:   h.GetResponsesDocsInfo(),
				Security:    secNames,
			}
			configs.Paths[fmt.Sprintf("/%s/%s", h.GetGroup(), h.GetName())] = PathInfo{
				Post: &qi,
			}

			if "" == controllerGroupDescriptions[h.GetGroup()] {
				controllerGroupDescriptions[h.GetGroup()] = fmt.Sprintf("【%s】的接口清单", h.GetGroup())
			}
		}

		for k, v := range controllerGroupDescriptions {
			configs.Tags = append(configs.Tags, TagInfo{
				Name:        k,
				Description: v,
			})
		}
		// todo
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

// handlerTryoutAPICase : 测试API支持用例
func handlerTryoutAPICase(ctx iris.Context) {
	groupName := ctx.Params().Get("groupName")
	apiName := ctx.Params().Get("apiName")
	routingKey := fmt.Sprintf("%s.%s", groupName, apiName)
	resObj := results.NewResultObject()
	for {
		if nil == _handlersManager {
			resObj.Code = results.InnerError
			resObj.Message = "处理单元未加载。"
			break
		}
		handler := _handlersManager.GetHandler(routingKey)
		if nil == handler {
			resObj.Code = results.NotFound
			resObj.Message = fmt.Sprintf("未找到路由键为：%s 的处理器", routingKey)
			break
		}
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if nil != err {
			logger.Error.Printf("read content body failed with error:%v", err)
			resObj.Code = results.InnerError
			resObj.Message = err.Error()
			break
		}
		if logger.IsDebugEnabled() {
			logger.Trace.Printf("got %s api query %s", routingKey, string(body))
		}
		id := utils.GenLoweruuid()
		msg := mqenv.MQConsumerMessage{
			RoutingKey:    routingKey,
			AppID:         "",
			UserID:        "",
			MessageID:     id,
			CorrelationID: id,
			ReplyTo:       "rpc-" + id,
			Body:          body,
		}

		pubMsg := handler.Execute(msg)
		if nil == pubMsg {
			resObj.Code = results.InnerError
			resObj.Message = fmt.Sprintf("处理器 %s 未返回响应", routingKey)
			break
		}

		ctx.Write(pubMsg.Body)
		return
	}
	ctx.WriteString(resObj.Encode())
}
