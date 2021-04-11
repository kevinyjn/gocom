package results

import (
	"encoding/json"

	"github.com/kevinyjn/gocom/logger"
)

// ResultObject result
type ResultObject struct {
	Code      int         `json:"code" validate:"required" comment:"结果编码"`
	Message   string      `json:"message" validate:"required" comment:"结果描述"`
	Data      interface{} `json:"data,omitempty" validate:"optional" comment:"响应结果"`
	RequestSn interface{} `json:"requestSn,omitempty" validate:"optional" comment:"请求序列号"`
}

// NewResultObject result
func NewResultObject() ResultObject {
	return ResultObject{
		Code:    Failed,
		Message: "Unknown.",
	}
}

// NewResultObjectWithRequestSn result
func NewResultObjectWithRequestSn(requestSn interface{}) ResultObject {
	return ResultObject{
		Code:      Failed,
		Message:   "Unknown.",
		RequestSn: requestSn,
	}
}

// Encode serializer
func (r *ResultObject) Encode() string {
	buf, err := json.Marshal(r)
	if err != nil {
		logger.Error.Printf("Serialize result object failed with error:%v", err)
		return err.Error()
	}
	return string(buf)
}

// Decode unserializer
func (r *ResultObject) Decode(s string) error {
	err := json.Unmarshal([]byte(s), r)
	return err
}

// StatusCode of result
func (r *ResultObject) StatusCode() int {
	return r.Code
}
