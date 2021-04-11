package filters

import (
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/validator"
)

// validateFilter validates the inputs data
type validateFilter struct {
	AbstractFilter
}

// NewValidateFilter new validate filter
func NewValidateFilter() Filter {
	filter := validateFilter{}
	filter.Init("ValidateFilter")
	return &filter
}

// Filter filters that if the event payload were valid on tag validate specification
func (f *validateFilter) Filter(event events.Event) (bool, error) {
	payload := event.GetPayload()
	if nil != payload {
		err := validator.Validate(payload)
		if nil != err {
			return false, err
		}
	}
	return true, nil
}
