package acl

import (
	"fmt"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
)

// LoginUser interface of login user information
type LoginUser interface {
	GetUserID() string
	GetName() string
	VerifyPassword(passwordOrHash string) bool
}

// LoginVerifier interface of login user validator
type LoginVerifier interface {
	VerifyUser(userName string, passwordOrHash string, remoteIP string) (LoginUser, error)
	PushBlocked(remoteIP string, expireTs int64)
}

// PushBuiltinUser for localhost login
func PushBuiltinUser(user LoginUser) {
	builtinUsers.PushUser(user)
}

var (
	UserVerifyManager LoginVerifier = &BuiltinUserVerifyManager{blacklist: NewBlacklistPool()}
	builtinUsers                    = builtinUsersWrapper{
		users: map[string]LoginUser{},
	}
)

// builtinUserVerifyManager of LoginVerifier
type builtinUserVerifyManager struct {
	users map[string]LoginUser
	mu    sync.RWMutex
}

// BuiltinUser
type BuiltinUser struct {
	UserID         string `json:"uid"`
	Name           string `json:"name"`
	AppID          string `json:"appId"`
	PasswordOrHash string `json:"-"`
}

// GetUserID user id
func (bu *BuiltinUser) GetUserID() string {
	return bu.UserID
}

// GetAppID app id
func (bu *BuiltinUser) GetAppID() string {
	return bu.AppID
}

// GetName name of user
func (bu *BuiltinUser) GetName() string {
	return bu.Name
}

// VerifyPassword if password matches
func (bu *BuiltinUser) VerifyPassword(passwordOrHash string) bool {
	return passwordOrHash == bu.PasswordOrHash
}

type builtinUsersWrapper struct {
	users map[string]LoginUser
	mu    sync.RWMutex
}

// VerifyUser if user exists or password matches
func (buw *builtinUsersWrapper) VerifyUser(userName string, passwordOrHash string, remoteIP string) (LoginUser, error) {
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

// PushUser for localhost login
func (buw *builtinUsersWrapper) PushUser(user LoginUser) {
	if nil == user {
		return
	}
	buw.mu.Lock()
	buw.users[user.GetName()] = user
	buw.mu.Unlock()
}

// PushBlocked by remote IP
func (buw *builtinUserVerifyManager) PushBlocked(remoteIP string, expireTs int64) {

}

type BuiltinUserVerifyManager struct {
	userModel builtinmodels.UserModel
	blacklist BlacklistPool
	failures  map[string]failureUserElement
	mu        sync.RWMutex
}

type failureUserElement struct {
	name          string
	count         int
	lastTimestamp int64
}

// VerifyUser if user exists or password matches
func (buv *BuiltinUserVerifyManager) VerifyUser(userName string, passwordOrHash string, remoteIP string) (LoginUser, error) {
	if nil != buv.blacklist {
		if buv.blacklist.IsBlocked(remoteIP) {
			return nil, fmt.Errorf("Client were blocked")
		}
	}
	u, err := buv.ensureUserModel().FindByName(userName)
	if nil != err || nil == u {
		if nil == err {
			err = fmt.Errorf("User name does not exists")
		}
		buv.onInvalid(remoteIP, time.Now().Unix(), nil)
		return nil, err
	}
	if false == u.VerifyPassword(passwordOrHash) {
		now := time.Now().Unix()
		buv.onInvalid(remoteIP, now, nil)
		buv.onInvalid(userName, now, u)
		return nil, fmt.Errorf("User password were not correct")
	}
	return u, nil
}

// PushUser for localhost login
func (buv *BuiltinUserVerifyManager) PushUser(user LoginUser) {
	if nil == user {
		return
	}
}

// PushBlocked by remote IP
func (buv *BuiltinUserVerifyManager) PushBlocked(remoteIP string, expireTs int64) {
	if nil == buv.blacklist {
		buv.blacklist = NewBlacklistPool()
	}
	buv.blacklist.PushBlocked(remoteIP, expireTs)
}

func (buv *BuiltinUserVerifyManager) ensureUserModel() builtinmodels.UserModel {
	if nil == buv.userModel {
		buv.userModel = &builtinmodels.User{}
	}
	return buv.userModel
}

func (buv *BuiltinUserVerifyManager) onInvalid(name string, timestamp int64, user builtinmodels.UserModel) int {
	buv.mu.Lock()
	if nil == buv.failures {
		buv.failures = map[string]failureUserElement{}
	}
	ele, ok := buv.failures[name]
	if ok && ele.lastTimestamp+int64(ValidateFailureDurationSecondsThreshold) > timestamp {
		ele.count = ele.count + 1
	} else {
		ele = failureUserElement{
			name:          name,
			count:         1,
			lastTimestamp: timestamp,
		}
	}
	buv.failures[name] = ele
	buv.mu.Unlock()

	if ele.count > ValidateFailureCountThreshold && nil == user {
		if nil == buv.blacklist {
			buv.blacklist = NewBlacklistPool()
		}
		buv.blacklist.PushBlocked(name, timestamp+int64(BlockingIPDurationSeconds))
	}
	return ele.count
}
