package weixin

import (
	"encoding/json"

	"github.com/kevinyjn/gocom/httpclient"
)

const (
	// WeixinAuthURLCode2Session request url
	WeixinAuthURLCode2Session = "https://api.weixin.qq.com/sns/jscode2session"
	// WeixinAuthGrantTypeDefault grant type
	WeixinAuthGrantTypeDefault = "authorization_code"
)

// WeixinLoginResult login result data
type WeixinLoginResult struct {
	Code       int    `json:"errcode"`
	Openid     string `json:"openid"`
	SessionKey string `json:"session_key"`
	Unionid    string `json:"unionid"`
	Message    string `json:"errmsg"`
}

// IsSuccess if the result code is success
func (r *WeixinLoginResult) IsSuccess() bool {
	return 0 == r.Code
}

// VerifyWeixinLogin verify weixin auth login
func VerifyWeixinLogin(wxAppID, wxAppSecret, wxCode string, options ...httpclient.ClientOption) WeixinLoginResult {
	result := WeixinLoginResult{}
	params := map[string]string{
		"appid":      wxAppID,
		"secret":     wxAppSecret,
		"js_code":    wxCode,
		"grant_type": "authorization_code",
	}

	body, err := httpclient.HTTPGet(WeixinAuthURLCode2Session, &params, options...)
	if nil != err {
		result.Code = -1
		result.Message = err.Error()
	} else {
		err = json.Unmarshal(body, &result)
		if nil != err {
			result.Code = -1
			result.Message = err.Error()
		}
	}
	return result
}
