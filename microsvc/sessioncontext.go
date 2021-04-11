package microsvc

import (
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/mq/mqenv"
)

// SessionContext carries application info and user session info
type SessionContext struct {
	AppID                    string
	UserID                   string
	RequestSn                string
	CorrelationID            string
	ContentSerializationType string
}

// CreateSessionContextWithEvent SessionContext
func CreateSessionContextWithEvent(event events.Event) SessionContext {
	ctx := SessionContext{
		AppID:         event.GetAppID(),
		UserID:        event.GetUserID(),
		RequestSn:     event.GetIdentifier(),
		CorrelationID: event.GetCorrelationID(),
	}
	return ctx
}

// CreateSessionContextWithConsumerMessage SessionContext
func CreateSessionContextWithConsumerMessage(msg mqenv.MQConsumerMessage) SessionContext {
	ctx := SessionContext{
		AppID:         msg.AppID,
		UserID:        msg.UserID,
		RequestSn:     msg.MessageID,
		CorrelationID: msg.CorrelationID,
	}
	return ctx
}

// HandlerError handler execute error code
type HandlerError interface {
	StatusCode() int
	Error() string
}

// StatusCodeResultType interface
type StatusCodeResultType interface {
	StatusCode() int
}

// HandlerStatus handler execute error code
type HandlerStatus struct {
	Code    int
	Message string
}

// NewHandlerError HandlerError
func NewHandlerError(code int, message string) HandlerError {
	he := HandlerStatus{
		Code:    code,
		Message: message,
	}
	return &he
}

// Error message
func (he *HandlerStatus) Error() string {
	return he.Message
}

// Status code
func (he *HandlerStatus) StatusCode() int {
	return he.Code
}
