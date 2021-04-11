package visitors

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
)

// accessVisitor records the inputs access action
type accessVisitor struct {
	widgets.AbstractStrategyWidget
}

// NewAccessVisitor new access visitor
func NewAccessVisitor() Visitor {
	visitor := accessVisitor{}
	visitor.Init("AccessVisitor", "*")
	return &visitor
}

// Visit records the access action
func (f *accessVisitor) Visit(event events.Event) {
	if logger.IsDebugEnabled() {
		logger.Debug.Printf("%s on access with %s from %s.%s appId:%s userId:%s\n", f.GetName(), event.GetIdentifier(), event.GetCategory(), event.GetFrom(), event.GetAppID(), event.GetUserID())
	}
}
