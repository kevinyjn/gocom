package acl

import (
	"time"

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

// Logout user
func (c *UserController) Logout(ctx iris.Context) {
	c.Session(ctx).Destroy()
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
		Expires: 24 * time.Hour,
	})
	c.initialized = true
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
