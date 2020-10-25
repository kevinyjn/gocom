package healthz

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kevinyjn/gocom"
	"github.com/kevinyjn/gocom/caching"
	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/httpclient"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mongodb"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/orm/rdbms"

	"github.com/kataras/iris"
)

// LivenessCheckResult options
type LivenessCheckResult struct {
	Name    string
	Status  string
	Message string
}

// PingableSession pingable
type PingableSession interface {
	Ping() error
}

type mqCheckListWrapper struct {
	categories map[string]string
	trigger    chan LivenessCheckResult
}

var (
	livenessTickerCache     = map[string]*mqCheckListWrapper{}
	mqChecks                = mqCheckListWrapper{categories: map[string]string{}}
	customizedHealthzChecks []config.HealthzChecks
)

// InitHealhz register iris healthz handler
func InitHealhz(app *iris.Application) {
	app.Get("/healthz", handlerHealthz)
	initHealthzMQConsumer()
}

// SetCustomHealthzChecks healthz checks
func SetCustomHealthzChecks(checks []config.HealthzChecks) {
	if nil != checks {
		customizedHealthzChecks = make([]config.HealthzChecks, 0)
		for i, chk := range checks {
			customizedHealthzChecks[i] = chk
		}
	}
}

func (c *mqCheckListWrapper) CheckConsumers(trigger chan LivenessCheckResult, mqEvent chan mqenv.MQEvent) *mqCheckListWrapper {
	checkInst := &mqCheckListWrapper{
		categories: map[string]string{},
		trigger:    trigger,
	}
	for category := range c.categories {
		val := checkMQMessage(category, trigger, mqEvent)
		if "" != val {
			checkInst.categories[category] = val
			livenessTickerCache[val] = checkInst
		}
	}
	return checkInst
}

// NotEmpty check
func (c *mqCheckListWrapper) NotEmpty() bool {
	for range c.categories {
		return true
	}
	return false
}

// Remove by category
func (c *mqCheckListWrapper) Remove(category string, eventStatus string) {
	val, ok := c.categories[category]
	if ok {
		_, ok = livenessTickerCache[val]
		if ok {
			delete(livenessTickerCache, val)
		}
		delete(c.categories, category)
		checkResult := LivenessCheckResult{
			Name:   category,
			Status: eventStatus,
		}
		go pushLivenessCheckEvent(c.trigger, checkResult)
	}
}

// RemoveBySerialNo remove by value mapped by category
func (c *mqCheckListWrapper) RemoveBySerialNo(sn string, eventStatus string) {
	for category, val := range c.categories {
		if sn == val {
			c.Remove(category, eventStatus)
			break
		}
	}
}

// Clear all
func (c *mqCheckListWrapper) Clear(eventStatus string) {
	clears := map[string]string{}
	for category, val := range c.categories {
		clears[category] = val
	}
	for category := range clears {
		c.Remove(category, eventStatus)
	}
}

func handlerHealthz(ctx iris.Context) {
	results := map[string]interface{}{}
	checkResults := map[string]string{}
	messages := []string{}

	checkTrigger := make(chan LivenessCheckResult)
	mqEvent := make(chan mqenv.MQEvent)
	stillChecking := true

	checkParts := map[string]bool{}
	mqCheckObject := mqChecks.CheckConsumers(checkTrigger, mqEvent)
	checkDbConnections(checkTrigger, checkParts)
	for category := range caching.GetAllCachers() {
		checkCacherConnection(category, checkTrigger, checkParts)
	}
	for category := range mqCheckObject.categories {
		checkParts[category] = true
	}

	if nil != customizedHealthzChecks {
		for _, e := range customizedHealthzChecks {
			if "HttpGet" == e.Type {
				for _, url := range e.Endpoints {
					checkParts["healthz - "+url] = true
					go checkHTTPGetEndpoint(url, checkTrigger)
				}
			}
		}
	}

	// ticker := time.NewTicker(100 * time.Millisecond)
	timeoutTicker := time.NewTicker(2 * time.Second)
	quitTiker := make(chan struct{})

	for mqCheckObject.NotEmpty() || stillChecking {
		if !mqCheckObject.NotEmpty() {
			stillChecking = false
			for range checkParts {
				stillChecking = true
				break
			}
		}
		select {
		case <-timeoutTicker.C:
			timeoutTicker.Stop()
			close(quitTiker)
			break
		case <-quitTiker:
			logger.Warning.Printf("do healthz check while checking timeout")
			mqCheckObject.Clear("timeout")
			stillChecking = false
			// ticker.Stop()
			break
		case trigger := <-checkTrigger:
			checkResults[trigger.Name] = trigger.Status
			if "" != trigger.Message {
				messages = append(messages, trigger.Message)
			}
			if "none" != trigger.Status {
				delete(checkParts, trigger.Name)
				stillChecking = false
				for range checkParts {
					stillChecking = true
					break
				}
			}
			break
		case ev := <-mqEvent:
			// fmt.Printf("%s message sent result:%s\n", ev.Label, ev.Message)
			if mqenv.MQEventCodeOk == ev.Code {
				checkResults[ev.Label] = "healthz message sent"
			} else {
				checkResult := LivenessCheckResult{
					Name:   ev.Label,
					Status: "failed - " + ev.Message,
				}
				go pushLivenessCheckEvent(checkTrigger, checkResult)
			}
			break
		}
	}

	results["result"] = checkResults
	results["messages"] = messages
	out, err := json.MarshalIndent(results, "", "    ")
	for _, chkResult := range checkResults {
		if "failed" == chkResult {
			ctx.StatusCode(iris.StatusInternalServerError)
			break
		}
	}
	if err != nil {
		ctx.WriteString(err.Error())
	} else {
		ctx.WriteString(string(out))
	}
}

func initHealthzMQConsumer() {
	mqconfs := mq.GetAllMQDriverConfigs()
	for instName := range mqconfs {
		category := fmt.Sprintf("amqp-%s-healthz", instName)
		mqconf := mq.Config{
			Instance: instName,
			Queue:    fmt.Sprintf("healthz-%s", gocom.GlobalUUID),
			Exchange: mq.Exchange{
				Type:    "topic",
				Name:    "",
				Durable: false,
			},
			BindingKey:  "",
			RoutingKeys: map[string]string{},
			Durable:     false,
			RPCEnabled:  false,
			Topic:       fmt.Sprintf("healthz-%s", gocom.GlobalUUID),
			GroupID:     "",
		}
		err := mq.InitMQTopic(category, &mqconf, mqconfs)
		if nil == err {
			consumer1 := mqenv.MQConsumerProxy{
				Queue:    mqconf.Queue,
				Callback: handleHealthzConsumer,
			}
			err := mq.ConsumeMQ(category, &consumer1)
			if nil != err {
				logger.Error.Printf("Initialize %s healthz consumer failed with error:%v", category, err)
			} else {
				logger.Info.Printf("MQ healthz consumer %s initialized.", category)
			}
			mqChecks.categories[category] = ""
		} else {
			logger.Error.Printf("Initialize %s healthz topic failed with error:%v", category, err)
		}
	}
}

func handleHealthzConsumer(msg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	logger.Trace.Printf("Got mq %s %s check key:%s value:%s", msg.Driver, msg.Queue, msg.RoutingKey, string(msg.Body))
	cacherKey := string(msg.Body)
	ev, ok := livenessTickerCache[cacherKey]
	if ok {
		// fmt.Printf("trigger liveness event for key:%s", cacherKey)
		delete(livenessTickerCache, cacherKey)
		ev.RemoveBySerialNo(cacherKey, "success")
	} else {
		// fmt.Printf("got mq %s %s message while could not find trigger by key:%s", msg.Driver, msg.Queue, cacherKey)
	}
	return nil
}

func checkDbConnections(trigger chan LivenessCheckResult, checkParts map[string]bool) {
	for name, mongoSession := range mongodb.GetAllMongoDBs() {
		checkDbConnection("mongo-"+name, mongoSession, trigger, checkParts)
	}
	if rdbms.GetInstance().IsValid() {
		checkDbConnection("orm/rdbms", rdbms.GetInstance(), trigger, checkParts)
	}
}

func checkDbConnection(name string, dbSession PingableSession, trigger chan LivenessCheckResult, checkParts map[string]bool) {
	if nil != dbSession {
		checkResult := LivenessCheckResult{
			Name:   name,
			Status: "failed",
		}
		checkParts[checkResult.Name] = true
		go func() {
			err := dbSession.Ping()
			if nil != err {
				checkResult.Status = "failed"
				checkResult.Message = err.Error()
			} else {
				checkResult.Status = "success"
			}
			trigger <- checkResult
		}()
	}
}

func checkCacherConnection(name string, trigger chan LivenessCheckResult, checkParts map[string]bool) {
	cacher := caching.GetCacher(name)
	if nil != cacher {
		checkResult := LivenessCheckResult{
			Name:   "cacher-" + name,
			Status: "failed",
		}
		checkParts[checkResult.Name] = true
		go func() {
			cacherTestResult := cacher.Set(gocom.GlobalUUID+":hittest", []byte("whoami"), time.Second*300)
			if cacherTestResult {
				checkResult.Status = "success"
			} else {
				checkResult.Status = "failed"
			}
			trigger <- checkResult
		}()
	}
}

func checkMQMessage(name string, trigger chan LivenessCheckResult, mqEvent chan mqenv.MQEvent) string {
	checkResult := LivenessCheckResult{
		Name:   name,
		Status: "none",
	}
	mqconf := mq.GetMQConfig(name)
	healthzKeyValue := ""
	if mqconf == nil {
		checkResult.Message = fmt.Sprintf("mq route config by %s not exists", name)
	} else {
		healthzKeyValue = fmt.Sprintf("healthz:%s:%s:%d", name, gocom.GlobalUUID, time.Now().Unix())
		ms := mqenv.MQPublishMessage{
			Exchange:      "",
			RoutingKey:    mqconf.Queue,
			Body:          []byte(healthzKeyValue),
			PublishStatus: mqEvent,
			EventLabel:    name,
		}
		checkResult.Message = fmt.Sprintf("sending mq msg:%v", ms)
		err := mq.PublishMQ(name, &ms)
		if nil != err {
			checkResult.Status = "failed"
			checkResult.Message = err.Error()
		}
	}
	go pushLivenessCheckEvent(trigger, checkResult)
	return healthzKeyValue
}

func pushLivenessCheckEvent(trigger chan LivenessCheckResult, event LivenessCheckResult) {
	trigger <- event
}

func checkHTTPGetEndpoint(url string, trigger chan LivenessCheckResult) {
	resp, err := httpclient.HTTPGet(url, nil)
	checkResult := LivenessCheckResult{
		Name:   "healthz - " + url,
		Status: "timeout",
	}
	if nil != err {
		checkResult.Status = err.Error()
		checkResult.Message = err.Error()
	} else {
		checkResult.Status = fmt.Sprintf("success [%s]", string(resp))
	}
	trigger <- checkResult
}
