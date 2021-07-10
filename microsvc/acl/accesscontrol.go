package acl

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/utils"
	"github.com/kevinyjn/gocom/utils/signature"
	"golang.org/x/crypto/bcrypt"
)

// IsUserLoggedIn by service myself
func IsUserLoggedIn(ctx iris.Context) bool {
	return GetUserController().IsLoggedIn(ctx)
}

// VerifyAccessControl also allows jwt, signature access
func VerifyAccessControl(ctx iris.Context) (uid string, err error) {
	err = fmt.Errorf("Not Authorized")
	for {
		remoteIP := utils.GetRemoteAddress(ctx.Request())
		if GetUserController().IsLoggedIn(ctx) {
			uid = GetUserController().CurrentUserID(ctx)
			err = nil
			break
		}
		authToken := getHeaderValue(ctx, HeaderAuthorization)
		if "" != authToken {
			uid, err = validateAuthorizationToken(authToken, remoteIP)
		}
		signValue := getHeaderValue(ctx, HeaderSignature)
		if "" != signValue && "" != getSignatureAppKey() {
			err = validateSignature(ctx, signValue)
		}
		break
	}

	return uid, err
}

// InitBuiltinRBACModels syncronize the RBAC model with database
func InitBuiltinRBACModels() error {
	var err error
	for _, tbl := range builtinmodels.GetRBACModels() {
		e := rdbms.GetInstance().EnsureTableStructures(tbl)
		if nil != e {
			err = e
		}
	}
	return err
}

func getHeaderValue(ctx iris.Context, name string) string {
	val := ctx.GetHeader(name)
	if "" == val && (name[0] >= 'A' && name[0] < 'Z') {
		val = ctx.GetHeader(strings.ToLower(name))
	}
	return val
}

func getSignatureAppKey() string {
	key := DefaultSignatureAppKey
	env := config.GetEnv()
	if nil != env.Properties {
		if "" != env.Properties["aclAppKey"] {
			key = env.Properties["aclAppKey"]
		}
	}
	return key
}

func validateAuthorizationToken(authToken string, remoteIP string) (uid string, err error) {
	authSlices := strings.SplitN(authToken, " ", 2)
	if len(authSlices) < 2 {
		err = fmt.Errorf("Un recognized token")
		return uid, err
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
		var u LoginUser
		if nil != UserVerifyManager {
			u, err = UserVerifyManager.VerifyUser(loginTokens[0], passPart, remoteIP)
		}
		if nil == u && ("127.0.0.1" == remoteIP || "::1" == remoteIP) {
			u, err = builtinUsers.VerifyUser(loginTokens[0], passPart, remoteIP)
		}
		if nil != u {
			uid = u.GetUserID()
		}
		break
	case "Bearer":
		// jwt token
		break
	}
	return uid, err
}

func validateSignature(ctx iris.Context, signValue string) error {
	signAppKey := getSignatureAppKey()
	expectedSignValue := ""
	err := fmt.Errorf("Invalid signature")
	switch ctx.Method() {
	case "GET":
		params := map[string]interface{}{}
		for k, v := range ctx.URLParams() {
			params[k] = v
		}
		expectedSignValue = signature.GenerateSignatureDataMd5(signAppKey, params, signature.WithSortedFields(signature.SignFieldsSortAsc), signature.WithSkipEmptyField(true))
		break
	case "POST", "PATCH", "PUT", "DELETE":
		// switch ctx.GetHeader("Content-Type") {
		// case "application/json":
		// 	break
		// }
		// todo
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if nil != err {
			logger.Error.Printf("verify xlab query while read post body failed with error:%v", err)
			return err
		}
		expectedSignValue = signature.GenerateSignatureDataMd5Ex(signAppKey, string(body), signature.WithSortedFields(signature.SignFieldsSortAsc), signature.WithSkipEmptyField(true))
		break
	default:
		err = fmt.Errorf("Unsupported method %s", ctx.Method())
		logger.Error.Printf("verify xlab query while query method:%s not allowed", ctx.Method())
		return err
	}
	if signValue != expectedSignValue {
		logger.Warning.Printf("verify xlab query signature:%s failed while signature value not match excepted:%s", signValue, expectedSignValue)
		return err
	}
	return err
}

// ValidatePassword 将检查密码是否匹配
func ValidatePassword(userPassword string, hashed []byte) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(hashed, []byte(userPassword)); err != nil {
		return false, err
	}
	return true, nil
}
