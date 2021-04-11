package middlewares

import (
	"fmt"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/filters"
)

// appFilter blocks app that in black list
type appFilter struct {
	filters.AbstractFilter
}

// NewBlackAppFilter new validate filter
func NewBlackAppFilter() filters.Filter {
	filter := appFilter{}
	filter.Init("AppFilter")
	return &filter
}

// Filter filters that if the event comes from black list apps
func (f *appFilter) Filter(event events.Event) (bool, error) {
	if "" == event.GetUserID() {
		logger.Warning.Printf("filter event %s while appId:%s were in black list", event.GetIdentifier(), event.GetAppID())
		return false, fmt.Errorf("app:%s were blocked by black list", event.GetAppID())
	}
	return true, nil
}
