package filters

import (
	"sync"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/utils"
)

// Filter interface of visitor filter
type Filter interface {
	GetName() string
	Filter(events.Event) (bool, error)
	MatchFilter(name string) bool
}

// FiltersResponsibilityChain
type FiltersResponsibilityChain interface {
	AttachFilter(filter Filter) FiltersResponsibilityChain
	GetFilters() []Filter
	AsFilter() Filter
	MatchFilter(name string) bool
}

// FilterOperator interface for controller operation
type FilterOperator interface {
	AttachFilter(filter Filter) FiltersResponsibilityChain
	Filter(events.Event) (bool, error)
	InitializeFilters(from string)
}

// FiltersStrategy interface of filters strategy
type FiltersStrategy interface {
	Filters() []Filter
}

// filterFactory factory of filters
type filterFactory struct {
	filters map[string]Filter
	m       sync.RWMutex
}

// GetFilter by filter name
func GetFilter(name string) Filter {
	return _filtersFactory.get(name)
}

// SetFilter by filter name
func SetFilter(filter Filter) {
	if nil == filter {
		logger.Error.Printf("set filter while passing nil filter address")
		return
	}
	if "" == filter.GetName() {
		logger.Error.Printf("set filter %+v while the filter name were empty", filter)
		return
	}
	_filtersFactory.set(filter.GetName(), filter)
}

// GetDefaultFiltersChain for each controllers default filter
func GetDefaultFiltersChain() FiltersResponsibilityChain {
	return _defaultFiltersChain
}

var (
	_filtersFactory = filterFactory{
		filters: map[string]Filter{
			"ValidateFilter": NewValidateFilter(),
		},
		m: sync.RWMutex{},
	}
	_defaultFiltersChain = newFilterResponsibilityChain("DefaultFiltersChain", NewValidateFilter())
)

// FiltersChain responsibility chain of filters
type FiltersChain struct {
	AbstractFilter
	filters []Filter
	m       sync.RWMutex
}

// NewFilterChain responsibility chain of filters
func NewFilterChain(name string) *FiltersChain {
	chain := newFilterResponsibilityChain(name)
	chain.AttachFilter(GetDefaultFiltersChain().AsFilter())
	return chain
}

func newFilterResponsibilityChain(name string, filters ...Filter) *FiltersChain {
	chain := &FiltersChain{
		filters: make([]Filter, len(filters)),
		m:       sync.RWMutex{},
	}
	chain.AbstractFilter.Init(name)
	if len(filters) > 0 {
		for i, filter := range filters {
			chain.filters[i] = filter
		}
	}
	return chain
}

// AttachFilter on filter responsibility chain
func (c *FiltersChain) AttachFilter(filter Filter) FiltersResponsibilityChain {
	if c.MatchFilter(filter.GetName()) {
		logger.Warning.Printf("filter chain %s attach filter %s while the filter exists by filter name", c.name, filter.GetName())
		return c
	}
	c.m.Lock()
	if nil == c.filters {
		c.filters = []Filter{}
		if filter.GetName() != GetDefaultFiltersChain().AsFilter().GetName() {
			c.filters = append(c.filters, GetDefaultFiltersChain().AsFilter())
		}
	}
	notFound := true
	for _, o := range c.filters {
		if o.GetName() == filter.GetName() {
			notFound = false
			break
		}
	}
	if notFound {
		c.filters = append(c.filters, filter)
	}
	c.m.Unlock()
	return c
}

// GetFilters of filter responsibility chain
func (c *FiltersChain) GetFilters() []Filter {
	c.m.RLock()
	l := len(c.filters)
	filters := make([]Filter, l)
	if l > 0 {
		for i, o := range c.filters {
			filters[i] = o
		}
	}
	c.m.RUnlock()
	return filters
}

// Filter call filter responsibility chain
func (c *FiltersChain) Filter(event events.Event) (bool, error) {
	c.m.RLock()
	if nil == c.filters {
		c.m.RUnlock()
		return true, nil
	}
	filters := []Filter(c.filters)
	c.m.RUnlock()
	ok := true
	var err error
	for _, filter := range filters {
		ok, err = filter.Filter(event)
		if false == ok {
			logger.Warning.Printf("got reject message:%v when processing filter:%s", err, filter.GetName())
			break
		}
	}
	return ok, err
}

// AsFilter as filter
func (c *FiltersChain) AsFilter() Filter {
	return c
}

// MatchFilter if the name mathes self filter chain
func (c *FiltersChain) MatchFilter(name string) bool {
	if name == c.name {
		return true
	}
	if nil != c.filters {
		found := false
		c.m.RLock()
		for _, filter := range c.filters {
			if filter.MatchFilter(name) {
				found = true
				break
			}
		}
		c.m.RUnlock()
		return found
	}
	return false
}

// InitializeFilters called from controller analyzement on loading
func (c *FiltersChain) InitializeFilters(from string) {
	if "" == c.name {
		c.name = utils.CamelCaseString(from) + "FiltersChain"
	}
	if len(c.filters) <= 0 {
		c.AttachFilter(GetDefaultFiltersChain().AsFilter())
	}
}

// AbstractFilter abstract component of visitor filter
type AbstractFilter struct {
	name string
}

// GetName name of filter
func (f *AbstractFilter) GetName() string {
	return f.name
}

// Init filter
func (f *AbstractFilter) Init(name string) {
	f.name = name
}

// MatchFilter if the name mathes self filter name
func (f *AbstractFilter) MatchFilter(name string) bool {
	return name == f.name
}

func (f *filterFactory) get(name string) Filter {
	f.m.RLock()
	filter, _ := f.filters[name]
	f.m.RUnlock()
	return filter
}

func (f *filterFactory) set(name string, filter Filter) {
	if nil == filter {
		logger.Error.Printf("set filter by name:%s while passing nil filter address", name)
		return
	}
	f.m.Lock()
	f.filters[name] = filter
	f.m.Unlock()
}
