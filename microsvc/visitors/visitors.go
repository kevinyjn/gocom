package visitors

import (
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/microsvc/widgets"
	"github.com/kevinyjn/gocom/utils"
)

// Visitor interface
type Visitor interface {
	widgets.PatternWidget
	Visit(event events.Event)
}

// VisitorsResponsibilityChain
type VisitorsResponsibilityChain interface {
	AttachVisitor(visitor Visitor) bool
	GetVisitors(string) []Visitor
}

// VisitorOperator interface for controller operation
type VisitorOperator interface {
	AttachVisitor(visitor Visitor) bool
	RemoveVisitor(name string) bool
	GetVisitors(string) []Visitor
	InitializeVisitors(from string)
}

// AbstractVisitor abstract component of visotor
type AbstractVisitor struct {
	widgets.AbstractStrategyWidget
}

// VisitorsStrategy interface of visitors strategy
type VisitorsStrategy interface {
	Visitors() []Visitor
}

// DefaultVisitorsFactory default factory of visitors
type DefaultVisitorsFactory interface {
	Get(name string) Visitor
	Set(Visitor)
	GetAll() []Visitor
}

// VisitorsChain responsibility chain of visitors
type VisitorsChain struct {
	widgets.AbstractStrategyWidget
	visitors       widgets.WidgetsStrategyWrapper
	nestedVisitors []Visitor
}

// GetVisitor by visitor name
func GetVisitor(name string) Visitor {
	widget := _visitorsFactory.Get(name)
	if nil != widget {
		inst, _ := widget.(Visitor)
		return inst
	}
	return nil
}

// SetVisitor by visitor name
func SetVisitor(visitor Visitor) {
	_visitorsFactory.Set(visitor)
}

// GetDefaultVisitorsChain for each controllers default visitor
func GetDefaultVisitorsChain() DefaultVisitorsFactory {
	return _defaultVisitorsChain
}

var (
	_visitorsFactory      = widgets.NewAbstractStrategyWidgetFactory(NewAccessVisitor())
	_defaultVisitorsChain = newDefaultVisitorsFactory(_visitorsFactory.Get(NewAccessVisitor().GetName()))
)

// NewVisitorsChain responsibility chain of visitors
func NewVisitorsChain(name string, matchPattern string) *VisitorsChain {
	chain := newVisitorResponsibilityChain(name, matchPattern, GetDefaultVisitorsChain().GetAll()...)
	return chain
}

func newVisitorResponsibilityChain(name string, matchPattern string, visitors ...Visitor) *VisitorsChain {
	chain := &VisitorsChain{
		visitors: widgets.WidgetsStrategyWrapper{},
	}
	chain.Init(name, matchPattern)
	if len(visitors) > 0 {
		for _, visitor := range visitors {
			chain.visitors.Add(visitor)
		}
	}
	return chain
}

// AttachVisitor on visitor responsibility chain
func (c *VisitorsChain) AttachVisitor(visitor Visitor) bool {
	existed := c.visitors.Get(visitor.GetName())
	if nil != existed {
		return false
	}
	c.visitors.Add(visitor)
	return true
}

// RemoveVisitor on visitor responsibility chain
func (c *VisitorsChain) RemoveVisitor(name string) bool {
	c.visitors.Remove(name)
	return true
}

// GetVisitors of observer responsibility chain
func (c *VisitorsChain) GetVisitors(pattern string) []Visitor {
	elements := c.visitors.GetWidgetsByStrategy(pattern)
	visitors := make([]Visitor, len(elements))
	if nil != elements {
		for i, ele := range elements {
			visitors[i], _ = ele.(Visitor)
		}
	}
	return visitors
}

// InitializeVisitors called from controller analyzement on loading
func (c *VisitorsChain) InitializeVisitors(from string) {
	if "" == c.GetName() {
		c.Init(utils.CamelCaseString(from)+"VisitorsChain", "*")
	}
	if c.visitors.IsEmpty() {
		for _, visitor := range GetDefaultVisitorsChain().GetAll() {
			c.AttachVisitor(visitor)
		}
	}
}

// LoadStrategies for controller handlers
func (c *VisitorsChain) LoadStrategies(strategies []string) {
	c.visitors.LoadStrategies(strategies)
}

type defaultVisitorsFactory struct {
	factory widgets.AbstractStrategyWidgetFactory
}

func newDefaultVisitorsFactory(widgets ...widgets.PatternWidget) DefaultVisitorsFactory {
	factory := defaultVisitorsFactory{}
	if nil != widgets {
		for _, widget := range widgets {
			factory.factory.Set(widget)
		}
	}
	return &factory
}

// Get visitor by name
func (f *defaultVisitorsFactory) Get(name string) Visitor {
	widget := f.factory.Get(name)
	if nil != widget {
		inst, _ := widget.(Visitor)
		return inst
	}
	return nil
}

// Set visitor
func (f *defaultVisitorsFactory) Set(visitor Visitor) {
	f.factory.Set(visitor)
}

// GetAll all registered visitors
func (f *defaultVisitorsFactory) GetAll() []Visitor {
	elements := f.factory.GetAll()
	l := len(elements)
	values := make([]Visitor, l)
	idx := 0
	for _, e := range elements {
		inst, _ := e.(Visitor)
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
