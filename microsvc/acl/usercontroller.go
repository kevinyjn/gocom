package acl

import (
	"time"

	"github.com/kevinyjn/gocom/utils"

	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"

	"github.com/kataras/iris"
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/sessions"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
)

const (
	userIDKey = "UserID"
	appIDKey  = "AppID"
)

// UserController user session controller
type UserController struct {
	SessionManager *sessions.Sessions
	DefaultUsers   map[string]builtinmodels.UserModel
	initialized    bool
}

var userController = UserController{}

// GetUserController getter
func GetUserController() *UserController {
	return &userController
}

// CurrentUserID logined with session
func (c *UserController) CurrentUserID(ctx iris.Context) string {
	userID := c.Session(ctx).GetStringDefault(userIDKey, "")
	return userID
}

// IsLoggedIn with current session
func (c *UserController) IsLoggedIn(ctx iris.Context) bool {
	return c.CurrentUserID(ctx) != ""
}

// Session get session by context
func (c *UserController) Session(ctx iris.Context) *sessions.Session {
	return c.SessionManager.Start((ctx))
}

// PushUID on user login
func (c *UserController) PushUID(ctx iris.Context, uid string) {
	c.Session(ctx).Set(userIDKey, uid)
}

// PushAppID on user login
func (c *UserController) PushAppID(ctx iris.Context, appID string) {
	c.Session(ctx).Set(appIDKey, appID)
}

// CurrentUserAppID logined with session
func (c *UserController) CurrentUserAppID(ctx iris.Context) string {
	appID := c.Session(ctx).GetStringDefault(appIDKey, "")
	return appID
}

// Init with iris application
func (c *UserController) Init(app router.Party) {
	if c.initialized {
		return
	}
	// Prepare our repositories and services.
	// var err error
	// c.DefaultUsers, err = datasource.LoadUsers(datasource.Memory)
	// if err != nil {
	// 	logger.Error.Printf("error while loading the users: %v", err)
	// 	return
	// }
	// var id int64
	// for name, pwd := range DefaultUsers {
	// 	id++
	// 	hashedPassword, _ := GeneratePassword(pwd)
	// 	u := &builtinmodels.User{
	// 		ID:   id,
	// 		Name: name,
	// 	}
	// 	u.SetPasswordHash(hashedPassword)
	// 	c.DefaultUsers[u.GetUserID()] = u
	// }
	// repo := NewUserRepository(c.DefaultUsers)
	// c.Service = NewUserService(repo)
	c.SessionManager = sessions.New(sessions.Config{
		Cookie:  "sessioncookiename",
		Expires: time.Duration(SessionExpireHours) * time.Hour,
	})
	c.initialized = true

	app.Post(BaseRouteUsers+RouteUserSignin, c.Signin)
	app.Get(BaseRouteUsers+RouteUserSignout, c.Signout)
}

// Signout user
func (c *UserController) Signout(ctx iris.Context) {
	result := results.NewResultObject()
	if c.IsLoggedIn(ctx) {
		c.Session(ctx).Destroy()
		result.Code = results.OK
		result.Message = "Success"
	} else {
		result.Code = results.NotLogin
		result.Message = "Not login"
	}
	ctx.WriteString(result.Encode())
}

// Signin user
func (c *UserController) Signin(ctx iris.Context) {
	result := results.NewResultObject()
	for results.OK != result.Code {
		param := LoginParam{}
		err := ctx.ReadJSON(&param)
		if nil != err {
			logger.Error.Printf("user login while parsing user request body as json failed with error:%v", err)
			result.Code = results.InvalidInput
			result.Message = err.Error()
			break
		}
		if nil == UserVerifyManager {
			logger.Error.Printf("user %s login while user verify manager were not configured", param.Name)
			result.Code = results.InnerError
			result.Message = "User verify manager were not configured"
			break
		}

		remoteIP := utils.GetRemoteAddress(ctx.Request())
		// passwordHash, err := builtinmodels.GenerateHashedPassword(param.Password)
		// if nil != err {
		// 	logger.Error.Printf("login user:%s while process inputs password:%s failed with error:%v", param.Name, param.Password, err)
		// 	result.Code = results.InvalidInput
		// 	result.Message = err.Error()
		// 	break
		// }
		u, err := UserVerifyManager.VerifyUser(param.Name, param.Password, remoteIP)
		if nil != err {
			logger.Error.Printf("login user:%s while validate password:%s failed with error:%v", param.Name, param.Password, err)
			result.Code = results.DataNotExists
			result.Message = err.Error()
			break
		}

		c.PushUID(ctx, u.GetUserID())

		result.Code = results.OK
		result.Message = "Success"
		result.Data = u
		break
	}
	ctx.WriteString(result.Encode())
}
