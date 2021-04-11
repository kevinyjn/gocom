package delegates

import (
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
	"github.com/kevinyjn/gocom/utils"
)

// Delegate interface
type Delegate interface {
	widgets.PatternWidget
	Execute(event events.Event) events.Event
}

// DelegatesResponsibilityChain
type DelegatesResponsibilityChain interface {
	AttachDelegate(delegate Delegate) bool
	GetDelegates(string) []Delegate
}

// DelegateOperator interface for controller operation
type DelegateOperator interface {
	AttachDelegate(delegate Delegate) bool
	RemoveDelegate(name string) bool
	GetDelegates(string) []Delegate
	InitializeDelegates(from string)
}

// AbstractDelegate abstract component of visotor
type AbstractDelegate struct {
	widgets.AbstractStrategyWidget
}

// DelegatesStrategy interface of visitors strategy
type DelegatesStrategy interface {
	Delegates() []Delegate
}

// DefaultDelegatesFactory default factory of delegates
type DefaultDelegatesFactory interface {
	Get(name string) Delegate
	Set(Delegate)
	GetAll() []Delegate
}

// DelegatesChain responsibility chain of delegates
type DelegatesChain struct {
	widgets.AbstractStrategyWidget
	delegates       widgets.WidgetsStrategyWrapper
	nestedDelegates []Delegate
}

// GetDelegate by delegate name
func GetDelegate(name string) Delegate {
	widget := _delegatesFactory.Get(name)
	if nil != widget {
		inst, _ := widget.(Delegate)
		return inst
	}
	return nil
}

// SetDelegate by delegate name
func SetDelegate(delegate Delegate) {
	_delegatesFactory.Set(delegate)
}

// GetDefaultDelegatesChain for each controllers default delegate
func GetDefaultDelegatesChain() DefaultDelegatesFactory {
	return _defaultDelegatesChain
}

var (
	_delegatesFactory      = widgets.NewAbstractStrategyWidgetFactory()
	_defaultDelegatesChain = newDefaultDelegatesFactory()
)

// NewDelegatesChain responsibility chain of delegates
func NewDelegatesChain(name string, matchPattern string) *DelegatesChain {
	chain := newDelegateResponsibilityChain(name, matchPattern, GetDefaultDelegatesChain().GetAll()...)
	return chain
}

func newDelegateResponsibilityChain(name string, matchPattern string, delegates ...Delegate) *DelegatesChain {
	chain := &DelegatesChain{
		delegates: widgets.WidgetsStrategyWrapper{},
	}
	chain.Init(name, matchPattern)
	if len(delegates) > 0 {
		for _, delegate := range delegates {
			chain.delegates.Add(delegate)
		}
	}
	return chain
}

// AttachDelegate on delegate responsibility chain
func (c *DelegatesChain) AttachDelegate(delegate Delegate) bool {
	existed := c.delegates.Get(delegate.GetName())
	if nil != existed {
		return false
	}
	c.delegates.Add(delegate)
	return true
}

// RemoveDelegate on delegate responsibility chain
func (c *DelegatesChain) RemoveDelegate(name string) bool {
	c.delegates.Remove(name)
	return true
}

// GetDelegates of observer responsibility chain
func (c *DelegatesChain) GetDelegates(pattern string) []Delegate {
	elements := c.delegates.GetWidgetsByStrategy(pattern)
	delegates := make([]Delegate, len(elements))
	if nil != elements {
		for i, ele := range elements {
			delegates[i], _ = ele.(Delegate)
		}
	}
	return delegates
}

// InitializeDelegates called from controller analyzement on loading
func (c *DelegatesChain) InitializeDelegates(from string) {
	if "" == c.GetName() {
		c.Init(utils.CamelCaseString(from)+"DelegatesChain", "*")
	}
	if c.delegates.IsEmpty() {
		for _, delegate := range GetDefaultDelegatesChain().GetAll() {
			c.AttachDelegate(delegate)
		}
	}
}

// LoadStrategies for controller handlers
func (c *DelegatesChain) LoadStrategies(strategies []string) {
	c.delegates.LoadStrategies(strategies)
}

type defaultDelegatesFactory struct {
	factory widgets.AbstractStrategyWidgetFactory
}

func newDefaultDelegatesFactory(widgets ...widgets.PatternWidget) DefaultDelegatesFactory {
	factory := defaultDelegatesFactory{}
	if nil != widgets {
		for _, widget := range widgets {
			factory.factory.Set(widget)
		}
	}
	return &factory
}

// Get delegate by name
func (f *defaultDelegatesFactory) Get(name string) Delegate {
	widget := f.factory.Get(name)
	if nil != widget {
		inst, _ := widget.(Delegate)
		return inst
	}
	return nil
}

// Set delegate
func (f *defaultDelegatesFactory) Set(delegate Delegate) {
	f.factory.Set(delegate)
}

// GetAll all registered delegates
func (f *defaultDelegatesFactory) GetAll() []Delegate {
	elements := f.factory.GetAll()
	l := len(elements)
	values := make([]Delegate, l)
	idx := 0
	for _, e := range elements {
		inst, _ := e.(Delegate)
		if nil != inst {
			values[idx] = inst
			idx++
		}
	}
	if idx < l {
		values = values[:idx]
	}
	return values
}
