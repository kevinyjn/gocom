package resthandlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

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

// Handler interface of handler defination
type Handler interface {
	GetGroup() string
	GetName() string
	GetRoutingKey() string
	GetSummary() string
	GetDescription() string
	GetOperationID() string
	GetAllowsMethod() string
	Execute(mqenv.MQConsumerMessage) *mqenv.MQPublishMessage
	IsDisablesAuthorization() bool
}

// HandlersManager interfaces of handler manager
type HandlersManager interface {
	GetHandler(routingKey string) Handler
}

const (
	userIDKey = "UserID"
	appIDKey  = "AppID"
)

var (
	UserVerifyManager LoginVerifier
	_builtinUsers     = builtinUsersWrapper{
		users: map[string]LoginUser{
			"docsuser": &builtinUser{
				Name:           "docsuser",
				UserID:         "-19900001",
				AppID:          "-1990",
				passwordOrHash: "000000",
			},
		},
	}

	_handlersManager HandlersManager = nil
	_sessionManager                  = sessions.New(sessions.Config{
		Cookie:  "sessioncookiename",
		Expires: 24 * time.Hour,
	})
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 4096))
		},
	}
)

// IsRestfulHandlersLoaded boolean
func IsRestfulHandlersLoaded() bool {
	return _handlersManager != nil
}

// LoadRestfulHandlers resthandlers handler, the method would affects unique load
func LoadRestfulHandlers(app *iris.Application, baseRoute string, handlersManager HandlersManager) {
	_handlersManager = handlersManager
	app.Any(baseRoute+"{groupName:string}/{apiName:string}", handlerRestfulAPI)
}

// GetSession get session by context
func GetSession(ctx iris.Context) *sessions.Session {
	return _sessionManager.Start(ctx)
}

// PushUID on user login
func PushUID(ctx iris.Context, uid string) {
	GetSession(ctx).Set(userIDKey, uid)
}

// PushAppID on user login
func PushAppID(ctx iris.Context, appID string) {
	GetSession(ctx).Set(appIDKey, appID)
}

// GetCurrentUserID logined with session
func GetCurrentUserID(ctx iris.Context) string {
	userID := GetSession(ctx).GetStringDefault(userIDKey, "")
	return userID
}

// GetCurrentUserAppID logined with session
func GetCurrentUserAppID(ctx iris.Context) string {
	userID := GetSession(ctx).GetStringDefault(appIDKey, "")
	return userID
}

// builtinUser
type builtinUser struct {
	UserID         string `json:"uid"`
	Name           string `json:"name"`
	AppID          string `json:"appId"`
	passwordOrHash string
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

func parseRequestAuthorization(ctx iris.Context) (LoginUser, error) {
	var u LoginUser
	var err error
	remoteIP := utils.GetRemoteAddress(ctx.Request())
	for {
		authToken := ctx.GetHeader("Authorization")
		if "" == authToken {
			authToken = ctx.GetHeader("authorization")
		}
		if "" == authToken {
			uid := GetCurrentUserID(ctx)
			if uid != "" {
				u = &builtinUser{
					UserID: uid,
					AppID:  GetCurrentUserAppID(ctx),
				}
				break
			}
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

// handlerRestfulAPI : Restful API handlers entry
func handlerRestfulAPI(ctx iris.Context) {
	groupName := ctx.Params().Get("groupName")
	apiName := ctx.Params().Get("apiName")
	routingKey := fmt.Sprintf("%s.%s", groupName, apiName)
	routingKeyEndfix := "." + strings.ToLower(ctx.Method())
	resObj := results.NewResultObject()
	for {
		if nil == _handlersManager {
			resObj.Code = results.InnerError
			resObj.Message = "处理单元未加载。"
			break
		}
		handler := _handlersManager.GetHandler(routingKey + routingKeyEndfix)
		if nil == handler {
			handler = _handlersManager.GetHandler(routingKey)
			if nil != handler && ctx.Method() != "POST" {
				resObj.Code = results.MethodNotAllowed
				resObj.Message = fmt.Sprintf("路由键：%s 的处理器不允许 %s 行为", routingKey, ctx.Method())
				break
			}
		}
		if nil == handler {
			resObj.Code = results.NotFound
			resObj.Message = fmt.Sprintf("未找到路由键为：%s 的处理器", routingKey)
			break
		}
		var body []byte
		switch ctx.Method() {
		case "GET", "HEAD":
			body, _ = json.Marshal(ctx.URLParams())
			break
		default:
			buff := bufferPool.Get().(*bytes.Buffer)
			buff.Reset()
			_, err := io.Copy(buff, ctx.Request().Body)
			if nil != err {
				bufferPool.Put(buff)
				logger.Error.Printf("read request %s content body failed with error:%v", routingKey, err)
				resObj.Code = results.InnerError
				resObj.Message = err.Error()
				break
			}
			body = make([]byte, buff.Len())
			copy(body, buff.Bytes())
			buff.Reset()
			bufferPool.Put(buff)
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
		loginUser, err := parseRequestAuthorization(ctx)
		if false == handler.IsDisablesAuthorization() && nil != err {
			logger.Error.Printf("verify request %s authorization failed with error:%v", routingKey, err)
			resObj.Code = results.Unauthorized
			resObj.Message = err.Error()
			break
		}
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
	if resObj.Code >= 400 && resObj.Code < 510 {
		ctx.StatusCode(resObj.Code)
		ctx.WriteString(resObj.Message)
	} else {
		ctx.WriteString(resObj.Encode())
	}
}
