package resthandlers

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/sessions"
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/acl"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

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

var (
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
func LoadRestfulHandlers(app router.Party, baseRoute string, handlersManager HandlersManager) {
	_handlersManager = handlersManager
	app.Any(baseRoute+"{groupName:string}/{apiName:string}", handlerRestfulAPI)
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
		id := utils.GenLoweruuid()
		msg := mqenv.MQConsumerMessage{
			RoutingKey:    routingKey,
			AppID:         "",
			UserID:        "",
			MessageID:     id,
			CorrelationID: id,
			ReplyTo:       "rpc-" + id,
		}
		switch ctx.Method() {
		case "GET", "HEAD":
			msg.Body = []byte(ctx.Request().URL.RawQuery)
			msg.ContentType = "application/x-www-form-urlencoded"
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
			msg.Body = make([]byte, buff.Len())
			copy(msg.Body, buff.Bytes())
			buff.Reset()
			bufferPool.Put(buff)
			break
		}
		if logger.IsDebugEnabled() {
			logger.Trace.Printf("got %s api query %s", routingKey, string(msg.Body))
		}
		uid, err := acl.VerifyAccessControl(ctx)
		// loginUser, err := parseRequestAuthorization(ctx)
		if false == handler.IsDisablesAuthorization() && nil != err {
			logger.Error.Printf("verify request %s authorization failed with error:%v", routingKey, err)
			resObj.Code = results.Unauthorized
			resObj.Message = err.Error()
			break
		}
		if "" != uid {
			msg.UserID = uid
		}
		appID := acl.GetUserController().CurrentUserAppID(ctx)
		if "" == appID {
			appID = ctx.GetHeader("AppID")
		}
		if "" != appID {
			msg.AppID = appID
		}
		// if nil != loginUser {
		// 	msg.AppID = loginUser.GetAppID()
		// 	msg.UserID = loginUser.GetUserID()
		// }

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
