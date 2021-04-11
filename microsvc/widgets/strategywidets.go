package widgets

import (
	"reflect"
	"regexp"
	"sync"

	"github.com/kevinyjn/gocom/logger"
)

// PatternWidget interface
type PatternWidget interface {
	GetName() string
	MatchPattern() string
}

// WidgetsStrategyWrapper container of eatch strategy widgets manager
type WidgetsStrategyWrapper struct {
	widgetsByName map[string]PatternWidget
	strategies    map[string][]string
	m             sync.RWMutex
	m2            sync.RWMutex
}

// Add widget element
func (w *WidgetsStrategyWrapper) Add(widget PatternWidget) {
	if nil == widget {
		logger.Error.Printf("add widget %s with nil pointer", reflect.TypeOf(widget).Elem().Name())
		return
	}
	w.m.Lock()
	if nil == w.widgetsByName {
		w.widgetsByName = map[string]PatternWidget{}
	}
	v, ok := w.widgetsByName[widget.GetName()]
	if nil != v {
		logger.Warning.Printf("add widget %s while the widget already registered", reflect.TypeOf(widget).Elem().Name())
	} else {
		w.widgetsByName[widget.GetName()] = widget
	}
	w.m.Unlock()
	if false == ok {
		w.loadStrategy(widget)
	}
}

// Get widget element
func (w *WidgetsStrategyWrapper) Get(name string) PatternWidget {
	var widget PatternWidget
	w.m.RLock()
	if nil != w.widgetsByName {
		widget = w.widgetsByName[name]
	}
	w.m.RUnlock()
	return widget
}

// Remove widget element
func (w *WidgetsStrategyWrapper) Remove(name string) bool {
	var removed bool
	var removedWidget PatternWidget
	w.m.Lock()
	if nil != w.widgetsByName {
		removedWidget, removed = w.widgetsByName[name]
		if removed {
			delete(w.widgetsByName, name)
		}
	}
	w.m.Unlock()
	if removed {
		w.removeStrategy(removedWidget)
	}
	return removed
}

// IsEmpty if widget element were empty
func (w *WidgetsStrategyWrapper) IsEmpty() bool {
	w.m.RLock()
	empty := nil == w.widgetsByName || len(w.widgetsByName) < 1
	w.m.RUnlock()
	return empty
}

// GetWidgetsByStrategy widgets mathed by strategy pattern
func (w *WidgetsStrategyWrapper) GetWidgetsByStrategy(strategy string) []PatternWidget {
	w.m2.RLock()
	carrier, ok := w.strategies[strategy]
	w.m2.RUnlock()
	l := len(carrier)
	widgets := make([]PatternWidget, l)
	if ok {
		w.m.RLock()
		idx := 0
		for _, name := range carrier {
			widget, _ := w.widgetsByName[name]
			if nil != widget {
				widgets[idx] = widget
				idx++
			}
		}
		w.m.RUnlock()
		if idx < l {
			widgets = widgets[:idx]
		}
		return widgets
	}
	return widgets
}

// LoadStrategies with strategy names
func (w *WidgetsStrategyWrapper) LoadStrategies(patterns []string) {
	if nil == patterns {
		return
	}
	widgets := map[string]PatternWidget{}
	w.m.RLock()
	for name, widget := range w.widgetsByName {
		widgets[name] = widget
	}
	w.m.RUnlock()
	w.m2.Lock()
	if nil == w.strategies {
		w.strategies = map[string][]string{}
	}
	for _, p := range patterns {
		v, ok := w.strategies[p]
		if ok && len(v) > 0 {
			continue
		}
		w.strategies[p] = []string{}
	}
	w.m2.Unlock()
	for _, widget := range widgets {
		w.loadStrategy(widget)
	}
}

func (w *WidgetsStrategyWrapper) loadStrategy(widget PatternWidget) {
	if nil == widget || nil == w.strategies {
		return
	}
	p := widget.MatchPattern()
	r, _ := regexp.Compile(p)
	w.m2.Lock()
	for strategy, carrier := range w.strategies {
		matched := false
		if "*" == p || "#" == p {
			matched = true
		} else if nil != r {
			if r.MatchString(p) {
				matched = true
			}
		}

		if matched {
			addWidget := true
			for _, name := range carrier {
				if name == widget.GetName() {
					addWidget = false
					break
				}
			}
			if addWidget {
				w.strategies[strategy] = append(carrier, widget.GetName())
			}
		}
	}
	w.m2.Unlock()
}

func (w *WidgetsStrategyWrapper) removeStrategy(widget PatternWidget) {
	if nil == widget || nil == w.strategies {
		return
	}
	p := widget.MatchPattern()
	r, _ := regexp.Compile(p)
	w.m2.Lock()
	for strategy, carrier := range w.strategies {
		matched := false
		if "*" == p || "#" == p {
			matched = true
		} else if nil != r {
			if r.MatchString(p) {
				matched = true
			}
		}

		if matched {
			for i, name := range carrier {
				if name == widget.GetName() {
					w.strategies[strategy] = append(carrier[0:i], carrier[i+1:]...)
					break
				}
			}
		}
	}
	w.m2.Unlock()
}

// AbstractStrategyWidget abstract component of pattern widget
type AbstractStrategyWidget struct {
	name         string
	matchPattern string
}

// GetName name of widget
func (f *AbstractStrategyWidget) GetName() string {
	return f.name
}

// MatchPattern pattern that which handle would match
func (f *AbstractStrategyWidget) MatchPattern() string {
	return f.matchPattern
}

// Init widget
func (f *AbstractStrategyWidget) Init(name string, matchPattern string) {
	f.name = name
	f.matchPattern = matchPattern
}

// GetAll() []PatternWidget

// GetAllPatternWidgetWrapper interface of visitors strategy
type VisitorsStrategy interface {
	Visitors() []PatternWidget
}

// AbstractStrategyWidgetFactory factory of pattern widgets
type AbstractStrategyWidgetFactory struct {
	widgets map[string]PatternWidget
	m       sync.RWMutex
}

// Get pattern widget by name
func (f *AbstractStrategyWidgetFactory) Get(name string) PatternWidget {
	var widget PatternWidget
	f.m.RLock()
	if nil != f.widgets {
		widget, _ = f.widgets[name]
	}
	f.m.RUnlock()
	return widget
}

// Get pattern widget
func (f *AbstractStrategyWidgetFactory) Set(widget PatternWidget) {
	if nil == widget {
		logger.Error.Printf("set %s by name:%s while passing nil widget address", reflect.TypeOf(widget).Elem().Name(), widget.GetName())
		return
	}
	if "" == widget.GetName() {
		logger.Error.Printf("set %s %+v while the widget name were empty", reflect.TypeOf(widget).Elem().Name(), widget)
		return
	}
	f.m.Lock()
	if nil == f.widgets {
		f.widgets = make(map[string]PatternWidget)
	}
	f.widgets[widget.GetName()] = widget
	f.m.Unlock()
}

// GetAll all registered widgets
func (f *AbstractStrategyWidgetFactory) GetAll() []PatternWidget {
	f.m.RLock()
	widgets := make([]PatternWidget, len(f.widgets))
	idx := 0
	for _, widget := range f.widgets {
		if nil != widget {
			widgets[idx] = widget
			idx++
		}
	}
	f.m.RUnlock()
	return widgets
}

// NewAbstractStrategyWidgetFactory with widgets
func NewAbstractStrategyWidgetFactory(widgets ...PatternWidget) *AbstractStrategyWidgetFactory {
	factory := AbstractStrategyWidgetFactory{
		widgets: map[string]PatternWidget{},
		m:       sync.RWMutex{},
	}
	if nil != widgets {
		for _, widget := range widgets {
			factory.Set(widget)
		}
	}
	return &factory
}
