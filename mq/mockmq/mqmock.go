package mockmq

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

// Config mock mq 配置参数.
type Config struct {
	Hosts        string
	Partition    int
	PrivateTopic string
	GroupID      string
}

type mockQueue struct {
	topic      string
	messages   []mqenv.MQPublishMessage
	subscribes []mqenv.MQConsumerCallback
}

type mockQueueManager struct {
	topicQueues map[string]*mockQueue
	m           sync.RWMutex
}

var mockInstances = map[string]*MockMQ{}
var mockMQ = mockQueueManager{
	topicQueues: map[string]*mockQueue{},
	m:           sync.RWMutex{},
}

// MockMQ for test when programing
type MockMQ struct {
	Name                    string
	consumerRegisters       map[string]*mqenv.MQConsumerProxy // 处理函数字典
	waitingResponseMessages map[string]chan mqenv.MQConsumerMessage
	waitingResponseTopic    string
	m1                      sync.RWMutex
	m2                      sync.RWMutex
}

// InitMockMQ init
func InitMockMQ(mqConnName string, connCfg *mqenv.MQConnectorConfig, mqCfg *Config) (*MockMQ, error) {
	var err error
	instance, ok := mockInstances[mqConnName]
	if false == ok {
		instance = &MockMQ{
			Name:              mqConnName,
			consumerRegisters: map[string]*mqenv.MQConsumerProxy{},
			m1:                sync.RWMutex{},
			m2:                sync.RWMutex{},
		}
		instance.registerWaitingResponseTopic()
		mockInstances[mqConnName] = instance
	}
	return instance, err
}

// GetMockMQ 获取MockMQ.
func GetMockMQ(mqConnName string) (*MockMQ, error) {
	instance, ok := mockInstances[mqConnName]
	if ok {
		return instance, nil
	}
	return nil, fmt.Errorf("MockMQ instance by %s not found", mqConnName)
}

// Subscribe 订阅topic.
func (worker *MockMQ) Subscribe(topic string, consumeProxy *mqenv.MQConsumerProxy) {
	worker.m1.Lock()
	_, ok := worker.consumerRegisters[topic]
	if !ok {
		logger.Info.Println("Subscribe subscribing topic " + topic)
		mockMQ.subscribe(topic, worker.bindToOnMessage)
		worker.consumerRegisters[topic] = consumeProxy
	}
	worker.m1.Unlock()
}

// Send 发送信息.
func (worker *MockMQ) Send(topic string, pm *mqenv.MQPublishMessage, withReply bool) (*mqenv.MQConsumerMessage, error) {
	var waiter chan mqenv.MQConsumerMessage
	if withReply {
		waiter = worker.prepareRepliablePubilshMessage(pm)
	}
	mockMQ.publish(topic, *pm)
	if nil != waiter {
		var timer *time.Timer
		var resp *mqenv.MQConsumerMessage
		var err error
		if pm.TimeoutSeconds > 0 {
			timer = time.NewTimer(time.Duration(pm.TimeoutSeconds) * time.Second)
		} else {
			timer = time.NewTimer(30 * time.Second)
		}
		select {
		case res := <-waiter:
			if logger.IsDebugEnabled() {
				logger.Trace.Printf("MockMQ %s Got response %s", worker.Name, string(res.Body))
			}
			resp = &res
			timer.Stop()
			break
		case <-timer.C:
			pm.OnClosed()
			err = fmt.Errorf("Query timeout")
			timer.Stop()
			break
		}
		return resp, err
	}
	return nil, nil

}

func (worker *MockMQ) bindToOnMessage(msg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	if nil != worker.consumerRegisters {
		worker.m1.RLock()
		consumer, ok := worker.consumerRegisters[msg.ConsumerTag]
		worker.m1.RUnlock()
		if ok {
			return consumer.Callback(msg)
		}
	}
	return nil
}

func (worker *MockMQ) bindToReplyMessage(msg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	if nil != worker.waitingResponseMessages {
		worker.m2.RLock()
		waiter, ok := worker.waitingResponseMessages[msg.CorrelationID]
		worker.m2.RUnlock()
		if ok {
			worker.m2.Lock()
			delete(worker.waitingResponseMessages, msg.CorrelationID)
			worker.m2.Unlock()
			if nil != waiter {
				go func() {
					waiter <- msg
				}()
			}
		}
	}
	return nil
}

func (worker *MockMQ) prepareRepliablePubilshMessage(publishMsg *mqenv.MQPublishMessage) chan mqenv.MQConsumerMessage {
	publishMsg.ReplyTo = worker.waitingResponseTopic
	if "" == publishMsg.CorrelationID {
		publishMsg.CorrelationID = utils.GenLoweruuid()
	}
	waiter := make(chan mqenv.MQConsumerMessage)
	worker.m2.Lock()
	if nil == worker.waitingResponseMessages {
		worker.registerWaitingResponseTopic()
	}
	worker.waitingResponseMessages[publishMsg.CorrelationID] = waiter
	worker.m2.Unlock()
	return waiter
}

func (worker *MockMQ) registerWaitingResponseTopic() {
	if "" == worker.waitingResponseTopic {
		worker.waitingResponseTopic = fmt.Sprintf("rpc-%s", utils.GenLoweruuid())
	}
	if nil == worker.waitingResponseMessages {
		worker.m2.Lock()
		worker.waitingResponseMessages = make(map[string]chan mqenv.MQConsumerMessage)
		worker.m2.Unlock()
	}
	mockMQ.subscribe(worker.waitingResponseTopic, worker.bindToReplyMessage)
}

func (q *mockQueue) subscribe(callback mqenv.MQConsumerCallback) {
	if nil == q.subscribes {
		q.subscribes = []mqenv.MQConsumerCallback{}
	}
	nofFound := true
	for _, cb := range q.subscribes {
		if reflect.ValueOf(cb).Pointer() == reflect.ValueOf(callback).Pointer() {
			nofFound = false
			break
		}
	}
	if nofFound {
		q.subscribes = append(q.subscribes, callback)
		if len(q.messages) > 0 {
			for _, msg := range q.messages {
				cm := mqenv.NewConsumerMessageFromPublishMessage(&msg)
				cm.ConsumerTag = q.topic
				mockDoCallback(callback, cm)
			}
		}
	}
}

func (q *mockQueue) publish(msg mqenv.MQPublishMessage) {
	if len(q.subscribes) < 1 {
		q.messages = append(q.messages, msg)
		return
	}
	cm := mqenv.NewConsumerMessageFromPublishMessage(&msg)
	cm.ConsumerTag = q.topic
	for _, subscriber := range q.subscribes {
		mockDoCallback(subscriber, cm)
	}
}

func mockDoCallback(callback mqenv.MQConsumerCallback, msg mqenv.MQConsumerMessage) {
	resp := callback(msg)
	if nil != resp && "" != msg.ReplyTo {
		mockMQ.publish(msg.ReplyTo, *resp)
	}
}

func (qm *mockQueueManager) publish(topic string, msg mqenv.MQPublishMessage) {
	qm.m.RLock()
	q, ok := qm.topicQueues[topic]
	qm.m.RUnlock()
	if ok {
		q.publish(msg)
	}
}

func (qm *mockQueueManager) subscribe(topic string, callback mqenv.MQConsumerCallback) {
	qm.m.Lock()
	q, ok := qm.topicQueues[topic]
	if !ok {
		q = &mockQueue{
			topic:      topic,
			messages:   []mqenv.MQPublishMessage{},
			subscribes: []mqenv.MQConsumerCallback{},
		}
		qm.topicQueues[topic] = q
	}
	qm.m.Unlock()

	q.subscribe(callback)
}
