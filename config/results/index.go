package results

import (
	"encoding/json"

	"github.com/kevinyjn/gocom/logger"
)

// ResultObject result
type ResultObject struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestSn interface{} `json:"requestSn,omitempty"`
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
		logger.Error.Printf("Serialize result object failed with error:%s", err.Error())
		return err.Error()
	}
	return string(buf)
}

// Decode unserializer
func (r *ResultObject) Decode(s string) error {
	err := json.Unmarshal([]byte(s), r)
	return err
}
