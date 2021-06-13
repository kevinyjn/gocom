package microsvc

import (
	"fmt"
	"sync"

	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/autodocs"
	"github.com/kevinyjn/gocom/microsvc/delegates"
	"github.com/kevinyjn/gocom/microsvc/filters"
	"github.com/kevinyjn/gocom/microsvc/observers"
	"github.com/kevinyjn/gocom/microsvc/visitors"
)

// Controller interface
type Controller interface {
	filters.FilterOperator
	visitors.VisitorOperator
	observers.ObserverOperator
	delegates.DelegateOperator
	GetName() string
	SetName(name string)
	GetTopicCategory() string
	SetTopicCategory(topicCategory string)
	IsDisablesAuthorization() bool
	HasConsumedMQ() bool
	SetConsumedMQ(consumed bool)
	afterAnalyzedHandlers([]string)
}

// AbstractController base controller
type AbstractController struct {
	filters.FiltersChain
	visitors.VisitorsChain
	observers.ObserversChain
	delegates.DelegatesChain
	name          string
	topicCategory string
	hasConsumedMQ bool
}

// GetName controller name
func (c *AbstractController) GetName() string {
	return c.name
}

// SetName mq category
func (c *AbstractController) SetName(name string) {
	c.name = name
}

// GetTopicCategory mq category
func (c *AbstractController) GetTopicCategory() string {
	return c.topicCategory
}

// SetTopicCategory mq category
func (c *AbstractController) SetTopicCategory(topicCategory string) {
	c.topicCategory = topicCategory
}

// IsDisablesAuthorization boolean
func (c *AbstractController) IsDisablesAuthorization() bool {
	return false
}

// HasConsumedMQ boolean
func (c *AbstractController) HasConsumedMQ() bool {
	return c.hasConsumedMQ
}

// SetConsumedMQ boolean
func (c *AbstractController) SetConsumedMQ(consumed bool) {
	c.hasConsumedMQ = consumed
}

func (c *AbstractController) afterAnalyzedHandlers(handlerNames []string) {
	if nil != handlerNames {
		c.VisitorsChain.LoadStrategies(handlerNames)
		c.ObserversChain.LoadStrategies(handlerNames)
		c.DelegatesChain.LoadStrategies(handlerNames)
	}
}

// GetController by controller name
func GetController(name string) Controller {
	return _controllerManager.get(name)
}

// SetController by controller name and controller instance
func SetController(name string, controller Controller) {
	_controllerManager.set(name, controller)
}

// mqRegisterController for registering mq outside system middlewares
type mqRegisterController struct {
	AbstractController
	SerializationStrategy interface{} `serialization:"json"`
}

type controllersManager struct {
	controllers map[string]Controller
	mu          sync.RWMutex
}

var (
	_mqRegisterController Controller
	_controllerManager    = controllersManager{controllers: map[string]Controller{}, mu: sync.RWMutex{}}
)

func (cm *controllersManager) get(name string) Controller {
	cm.mu.RLock()
	controller, _ := cm.controllers[name]
	cm.mu.RUnlock()
	return controller
}

func (cm *controllersManager) set(name string, controller Controller) {
	if cm.get(name) != nil {
		return
	}
	cm.mu.Lock()
	cm.controllers[name] = controller
	cm.mu.Unlock()
}

type registerResult struct {
	Group string `json:"group" label:"处理器组"`
}

type registerBase struct {
	Group        string `json:"group" validate:"required" label:"处理器组" comment:"注册目标处理器组"`
	Topic        string `json:"topic" validate:"required" label:"通知队列" comment:"事件触发时向该队列通知"`
	RoutingKey   string `json:"routingKey" validate:"required" label:"通知队列子路由" comment:"事件触发时向该队列通知，该子路由用于接受者的事件分发逻辑"`
	MatchPattern string `json:"pattern" validate:"required" label:"路由通配符" comment:"是否命中访问者由路由通配符匹配具体路由"`
}

type registerVisitor struct {
	registerBase
	Name string `json:"name" validate:"required" label:"访问者名称" comment:"访问者类型的全局唯一名称"`
}

type registerObserver struct {
	registerBase
	Name        string `json:"name" validate:"required" label:"观察者名称" comment:"观察者类型的全局唯一名称"`
	PayloadFrom string `json:"payloadFrom" validate:"anyof:request,result" label:"事件带载数据来源" comment:"事件带载数据来源，请求方或处理器的执行结果"`
}

type registerDelegate struct {
	registerBase
	Name string `json:"name" validate:"required" label:"委托者名称" comment:"委托者类型的全局唯一名称"`
}

// InitMQRegisterController for mq register
func InitMQRegisterController(topicCategory string) Controller {
	if nil == _mqRegisterController {
		_mqRegisterController = &mqRegisterController{}
		LoadController(topicCategory, _mqRegisterController)
		mqGroupName := "mq-register"
		autodocs.SetControllerGroupDescription(mqGroupName, "MQ注册服务")
		autodocs.SetControllerHandlerDescription(mqGroupName, "add-visitor", "注册MQ访问者")
		autodocs.SetControllerHandlerDescription(mqGroupName, "add-observer", "注册MQ观察者")
		autodocs.SetControllerHandlerDescription(mqGroupName, "add-delegate", "注册MQ委托者")
		autodocs.SetControllerHandlerDescription(mqGroupName, "remove-visitor", "移除MQ观察者")
		autodocs.SetControllerHandlerDescription(mqGroupName, "remove-observer", "移除MQ访问者")
		autodocs.SetControllerHandlerDescription(mqGroupName, "remove-delegate", "移除MQ委托者")
	}
	return _mqRegisterController
}

func getControllerForRegister(controllerName string) (Controller, HandlerError) {
	controller := GetController(controllerName)
	if nil == controller {
		return nil, NewHandlerError(results.NotFound, fmt.Sprintf("Controller by %s not found", controllerName))
	}
	return controller, nil
}

func (c *mqRegisterController) HandleAddVisitor(param registerVisitor) (*registerResult, HandlerError) {
	controller, err := getControllerForRegister(param.Group)
	if nil == controller {
		return nil, err
	}
	visitor := visitors.NewStdMQVisitor(c.GetTopicCategory(), param.Topic, param.RoutingKey, param.MatchPattern)
	controller.AttachVisitor(visitor)
	logger.Info.Printf("controller %s registerd visitor %s on pattern:%s", controller.GetName(), visitor.GetName(), visitor.MatchPattern())
	return &registerResult{param.Group}, nil
}

func (c *mqRegisterController) HandleAddObserver(param registerObserver) (*registerResult, HandlerError) {
	controller, err := getControllerForRegister(param.Group)
	if nil == controller {
		return nil, err
	}
	observer := observers.NewStdMQObserver(param.PayloadFrom, c.GetTopicCategory(), param.Topic, param.RoutingKey, param.MatchPattern)
	controller.AttachObserver(observer)
	logger.Info.Printf("controller %s registerd observer %s on pattern:%s", controller.GetName(), observer.GetName(), observer.MatchPattern())
	return &registerResult{param.Group}, nil
}

func (c *mqRegisterController) HandleAddDelegate(param registerDelegate) (*registerResult, HandlerError) {
	controller, err := getControllerForRegister(param.Group)
	if nil == controller {
		return nil, err
	}
	delegate := delegates.NewStdMQDelegate(c.GetTopicCategory(), param.Topic, param.RoutingKey, param.MatchPattern)
	controller.AttachDelegate(delegate)
	logger.Info.Printf("controller %s registerd delegate %s on pattern:%s", controller.GetName(), delegate.GetName(), delegate.MatchPattern())
	return &registerResult{param.Group}, nil
}

func (c *mqRegisterController) HandleRemoveVisitor(param registerVisitor) (*registerResult, HandlerError) {
	controller, err := getControllerForRegister(param.Group)
	if nil == controller {
		return nil, err
	}
	controller.RemoveVisitor(param.Name)
	return &registerResult{param.Group}, nil
}

func (c *mqRegisterController) HandleRemoveObserver(param registerObserver) (*registerResult, HandlerError) {
	controller, err := getControllerForRegister(param.Group)
	if nil == controller {
		return nil, err
	}
	controller.RemoveObserver(param.Name)
	return &registerResult{param.Group}, nil
}

func (c *mqRegisterController) HandleRemoveDelegate(param registerDelegate) (*registerResult, HandlerError) {
	controller, err := getControllerForRegister(param.Group)
	if nil == controller {
		return nil, err
	}
	controller.RemoveDelegate(param.Name)
	return &registerResult{param.Group}, nil
}
