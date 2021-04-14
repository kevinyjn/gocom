package autodocs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"runtime"
	"strings"
	"sync"

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
	UserVerifyManager   LoginVerifier

	_controllerGroupDescriptions   = map[string]string{}
	_controllerHandlerDescriptions = map[string]string{}

	_builtinUsers = builtinUsersWrapper{
		users: map[string]LoginUser{
			"docsuser": &builtinUser{
				Name:           "docsuser",
				UserID:         "-19900001",
				AppID:          "-1990",
				passwordOrHash: "000000",
			},
		},
	}

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

// LoginUser interface of login user information
type LoginUser interface {
	GetUserID() string
	GetAppID() string
	GetName() string
	VerifyPassword(passwordOrHash string) bool
}

// LoginVerifier interface of login user validator
type LoginVerifier interface {
	VerifyUser(userName string, passwordOrHash string) (LoginUser, error)
}

// builtinUser
type builtinUser struct {
	UserID         string `json:"uid"`
	Name           string `json:"name"`
	AppID          string `json:"appId"`
	passwordOrHash string `json:"-"`
}

type builtinUsersWrapper struct {
	users map[string]LoginUser
	mu    sync.RWMutex
}

// VerifyUser if user exists or password matches
func (buw *builtinUsersWrapper) VerifyUser(userName string, passwordOrHash string) (LoginUser, error) {
	buw.mu.RLock()
	defer buw.mu.RUnlock()
	if nil == buw.users {
		return nil, fmt.Errorf("User name does not exists")
	}
	u, _ := buw.users[userName]
	if nil == u {
		return nil, fmt.Errorf("User name does not exists")
	}
	if false == u.VerifyPassword(passwordOrHash) {
		return nil, fmt.Errorf("User password were not correct")
	}
	return u, nil
}

// GetUserID user id
func (bu *builtinUser) GetUserID() string {
	return bu.UserID
}

// GetAppID app id
func (bu *builtinUser) GetAppID() string {
	return bu.AppID
}

// GetName name of user
func (bu *builtinUser) GetName() string {
	return bu.Name
}

// VerifyPassword if password matches
func (bu *builtinUser) VerifyPassword(passwordOrHash string) bool {
	return passwordOrHash == bu.passwordOrHash
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
		APIKey: &APIKeyInfo{
			Type: "apiKey",
			Name: "Authorization",
			In:   "header",
		},
		BasicAuth: &APIKeyInfo{
			Type: "basic",
			Name: "Authorization",
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
	configs := SwaggerConfig{
		Swagger: "2.0",
		// OpenAPI: "3.0.0",
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
	if "http" == ctx.Request().URL.Scheme || strings.HasPrefix(ctx.Request().Referer(), "http:") {
		configs.Schemes = []string{"http", "https"}
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

func parseRequestAuthorization(ctx iris.Context) (LoginUser, error) {
	var u LoginUser
	var err error
	authToken := ctx.GetHeader("Authorization")
	if "" == authToken {
		authToken = ctx.GetHeader("authorization")
	}
	remoteIP := utils.GetRemoteAddress(ctx.Request())
	for {
		if "" == authToken {
			err = fmt.Errorf("No user has logged in")
			break
		}
		authSlices := strings.SplitN(authToken, " ", 2)
		if len(authSlices) < 2 {
			err = fmt.Errorf("Un recognized token")
			break
		}
		switch authSlices[0] {
		case "Basic":
			basicToken, e := base64.StdEncoding.DecodeString(authSlices[1])
			if nil != e {
				err = e
			}
			loginTokens := strings.SplitN(string(basicToken), ":", 2)
			passPart := ""
			if len(loginTokens) > 1 {
				passPart = loginTokens[1]
			}
			if nil != UserVerifyManager {
				u, err = UserVerifyManager.VerifyUser(loginTokens[0], passPart)
			}
			if nil == u && ("127.0.0.1" == remoteIP || "::1" == remoteIP) {
				u, err = _builtinUsers.VerifyUser(loginTokens[0], passPart)
			}
			break
		case "Bearer":
			// jwt token
			break
		}

		if nil == u && nil == err {
			err = fmt.Errorf("Verify user login failed")
		}
		break
	}
	return u, err
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
		loginUser, _ := parseRequestAuthorization(ctx)
		if nil != loginUser {
			msg.AppID = loginUser.GetAppID()
			msg.UserID = loginUser.GetUserID()
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
