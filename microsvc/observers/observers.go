package observers

import (
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
	"github.com/kevinyjn/gocom/utils"
)

// Observer interface
type Observer interface {
	widgets.PatternWidget
	Observe(event events.Event)
}

// ObserversResponsibilityChain
type ObserversResponsibilityChain interface {
	AttachObserver(observer Observer) bool
	GetObservers(string) []Observer
}

// ObserverOperator interface for controller operation
type ObserverOperator interface {
	AttachObserver(observer Observer) bool
	RemoveObserver(name string) bool
	GetObservers(string) []Observer
	InitializeObservers(from string)
}

// AbstractObserver abstract component of visotor
type AbstractObserver struct {
	widgets.AbstractStrategyWidget
}

// ObserversStrategy interface of visitors strategy
type ObserversStrategy interface {
	Observers() []Observer
}

// DefaultObserversFactory default factory of observers
type DefaultObserversFactory interface {
	Get(name string) Observer
	Set(Observer)
	GetAll() []Observer
}

// ObserversChain responsibility chain of observers
type ObserversChain struct {
	widgets.AbstractStrategyWidget
	observers       widgets.WidgetsStrategyWrapper
	nestedObservers []Observer
}

// GetObserver by observer name
func GetObserver(name string) Observer {
	widget := _observersFactory.Get(name)
	if nil != widget {
		inst, _ := widget.(Observer)
		return inst
	}
	return nil
}

// SetObserver by observer name
func SetObserver(observer Observer) {
	_observersFactory.Set(observer)
}

// GetDefaultObserversChain for each controllers default observer
func GetDefaultObserversChain() DefaultObserversFactory {
	return _defaultObserversChain
}

var (
	_observersFactory      = widgets.NewAbstractStrategyWidgetFactory(NewOperateLogObserver())
	_defaultObserversChain = newDefaultObserversFactory(_observersFactory.Get(NewOperateLogObserver().GetName()))
)

// NewObserversChain responsibility chain of observers
func NewObserversChain(name string, matchPattern string) *ObserversChain {
	chain := newObserverResponsibilityChain(name, matchPattern, GetDefaultObserversChain().GetAll()...)
	return chain
}

func newObserverResponsibilityChain(name string, matchPattern string, observers ...Observer) *ObserversChain {
	chain := &ObserversChain{
		observers: widgets.WidgetsStrategyWrapper{},
	}
	chain.Init(name, matchPattern)
	if len(observers) > 0 {
		for _, observer := range observers {
			chain.observers.Add(observer)
		}
	}
	return chain
}

// AttachObserver on observer responsibility chain
func (c *ObserversChain) AttachObserver(observer Observer) bool {
	existed := c.observers.Get(observer.GetName())
	if nil != existed {
		return false
	}
	c.observers.Add(observer)
	return true
}

// RemoveObserver on observer responsibility chain
func (c *ObserversChain) RemoveObserver(name string) bool {
	c.observers.Remove(name)
	return true
}

// GetObservers of observer responsibility chain
func (c *ObserversChain) GetObservers(pattern string) []Observer {
	elements := c.observers.GetWidgetsByStrategy(pattern)
	observers := make([]Observer, len(elements))
	if nil != elements {
		for i, ele := range elements {
			observers[i], _ = ele.(Observer)
		}
	}
	return observers
}

// InitializeObservers called from controller analyzement on loading
func (c *ObserversChain) InitializeObservers(from string) {
	if "" == c.GetName() {
		c.Init(utils.CamelCaseString(from)+"ObserversChain", "*")
	}
	if c.observers.IsEmpty() {
		for _, observer := range GetDefaultObserversChain().GetAll() {
			c.AttachObserver(observer)
		}
	}
}

// LoadStrategies for controller handlers
func (c *ObserversChain) LoadStrategies(strategies []string) {
	c.observers.LoadStrategies(strategies)
}

type defaultObserversFactory struct {
	factory widgets.AbstractStrategyWidgetFactory
}

func newDefaultObserversFactory(widgets ...widgets.PatternWidget) DefaultObserversFactory {
	factory := defaultObserversFactory{}
	if nil != widgets {
		for _, widget := range widgets {
			factory.factory.Set(widget)
		}
	}
	return &factory
}

// Get observer by name
func (f *defaultObserversFactory) Get(name string) Observer {
	widget := f.factory.Get(name)
	if nil != widget {
		inst, _ := widget.(Observer)
		return inst
	}
	return nil
}

// Set observer
func (f *defaultObserversFactory) Set(observer Observer) {
	f.factory.Set(observer)
}

// GetAll all registered observers
func (f *defaultObserversFactory) GetAll() []Observer {
	elements := f.factory.GetAll()
	l := len(elements)
	values := make([]Observer, l)
	idx := 0
	for _, e := range elements {
		inst, _ := e.(Observer)
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
