package observers

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
)

// operateLogObserver records the operation log action
type operateLogObserver struct {
	widgets.AbstractStrategyWidget
}

// NewOperateLogObserver new access visitor
func NewOperateLogObserver() Observer {
	observer := operateLogObserver{}
	observer.Init("OperateLogObserver", "*")
	return &observer
}

// Visit records the access action
func (s *operateLogObserver) Observe(event events.Event) {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on access with %s from %s.%s appId:%s userId:%s\n", s.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
}
