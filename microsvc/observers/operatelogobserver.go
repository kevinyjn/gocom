package observers

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
)

// operateLogObserver records the operation log action
type operateLogObserver struct {
	widgets.AbstractStrategyWidget
	operationLogModel builtinmodels.OperationLogModel
}

// NewOperateLogObserver new operation observer
func NewOperateLogObserver() Observer {
	observer := operateLogObserver{}
	observer.Init("OperateLogObserver", "*")
	observer.operationLogModel = &builtinmodels.OperationLog{}
	return &observer
}

// Visit records the observe action
func (o *operateLogObserver) Observe(event events.Event) {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on observer with %s from %s.%s appId:%s userId:%s\n", o.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
	if nil != o.operationLogModel {
		log := o.operationLogModel.NewRecord()
		log.LoadFromEvent(event)
		builtinmodels.GetAsyncRecordSaver().Push(log)
	}
}
