package visitors

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
)

// accessVisitor records the inputs access action
type accessVisitor struct {
	widgets.AbstractStrategyWidget
	accessLogModel builtinmodels.AccessLogModel
}

// NewAccessVisitor new access visitor
func NewAccessVisitor() Visitor {
	visitor := accessVisitor{}
	visitor.Init("AccessVisitor", "*")
	visitor.accessLogModel = &builtinmodels.AccessLog{}
	return &visitor
}

// SetAccessLogModel set customized access log model
func (f *accessVisitor) SetAccessLogModel(model builtinmodels.AccessLogModel) {
	f.accessLogModel = model
}

// Visit records the access action
func (f *accessVisitor) Visit(event events.Event) {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on access with %s from %s.%s appId:%s userId:%s\n", f.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
	if nil != f.accessLogModel {
		log := f.accessLogModel.NewRecord()
		log.LoadFromEvent(event)
		builtinmodels.GetAsyncRecordSaver().Push(log)
	}
}
