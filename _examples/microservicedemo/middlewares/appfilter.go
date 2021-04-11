package middlewares

import (
	"fmt"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/filters"
)

// blackuserFilter blocks user ids that in black list
type blackuserFilter struct {
	filters.AbstractFilter
}

// NewBlackUserFilter new validate filter
func NewBlackUserFilter() filters.Filter {
	filter := blackuserFilter{}
	filter.Init("BlackUserFilter")
	return &filter
}

// Filter filters that if the event comes from black list user
func (f *blackuserFilter) Filter(event events.Event) (bool, error) {
	if "" == event.GetAppID() {
		logger.Warning.Printf("filter event %s while appId:%s userId:%s were in black list", event.GetIdentifier(), event.GetAppID(), event.GetUserID())
		return false, fmt.Errorf("User:%s by appId:%s were blocked by black list", event.GetUserID(), event.GetAppID())
	}
	return true, nil
}
