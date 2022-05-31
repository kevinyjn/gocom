package pulsar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/netutils/pinger"
	"github.com/kevinyjn/gocom/netutils/sshtunnel"
	"github.com/kevinyjn/gocom/utils"
)

// Variables
var (
	pulsarInsts            = map[string]*PulsarMQ{}
	trackerQueue    string = ""
	pulsarInstMutex        = sync.RWMutex{}
)

// InitPulsarMQ init
func InitPulsarMQ(mqConnName string, connCfg *mqenv.MQConnectorConfig, pulsarCfg *Config) (*PulsarMQ, error) {
	pulsarInstMutex.RLock()
	pulsarInst, ok := pulsarInsts[mqConnName]
	pulsarInstMutex.RUnlock()
	if ok && !pulsarInst.config.Equals(pulsarCfg) {
		pulsarInst.close()
		close(pulsarInst.Close)
		ok = false
	}
	if !ok {
		pulsarInst = NewPulsarMQ(mqConnName, connCfg, pulsarCfg)
		pulsarInstMutex.Lock()
		pulsarInsts[mqConnName] = pulsarInst
		pulsarInstMutex.Unlock()
		logger.Info.Printf("Initializing pulsar instance:%s", pulsarInst.Name)
		err := pulsarInst.init()
		if err == nil {
			go pulsarInst.Run()
		} else {
			return nil, err
		}
	}
	return pulsarInst, nil
}

// GetPulsarMQ get
func GetPulsarMQ(name string) (*PulsarMQ, error) {
	pulsarInstMutex.RLock()
	pulsarInst, ok := pulsarInsts[name]
	pulsarInstMutex.RUnlock()
	if ok {
		return pulsarInst, nil
	}
	return nil, fmt.Errorf("PulsarMQ instance by %s not found", name)
}

// SetupTrackerQueue name
func SetupTrackerQueue(queueName string) {
	trackerQueue = queueName
}

// NewPulsarMQ with parameters
func NewPulsarMQ(mqConnName string, connCfg *mqenv.MQConnectorConfig, pulsarCfg *Config) *PulsarMQ {
	r := &PulsarMQ{}
	r.initWithParameters(mqConnName, connCfg, pulsarCfg)
	return r
}

func (r *PulsarMQ) initWithParameters(mqConnName string, connCfg *mqenv.MQConnectorConfig, pulsarCfg *Config) {
	r.Name = mqConnName
	r.config = pulsarCfg
	r.connConfig = connCfg
	r.Publish = make(chan *mqenv.MQPublishMessage)
	r.Consume = make(chan *mqenv.MQConsumerProxy)
	r.namespace = connCfg.Path
	r.topicName = utils.URLPathJoin(r.namespace, pulsarCfg.Topic)
	r.healthzTopicPrefix = "non-persistent://" + utils.URLPathJoin(r.namespace, "healthz")
	r.Done = make(chan error)
	r.Close = make(chan interface{})
	r.consumers = map[string]consumerWrapper{}
	r.producers = map[string]pulsar.Producer{}
	r.pendingConsumers = make([]*mqenv.MQConsumerProxy, 0)
	r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
	r.connecting = false
	r.rpcInstanceName = fmt.Sprintf("@rpc-%s", r.config.ConnConfigName)
	r.rpcCallbacks = make(map[string]*mqenv.MQPublishMessage)
	r.pendingReplies = make(map[string]mqenv.MQConsumerMessage)
	r.rpcCallbacksMutex = sync.RWMutex{}
	r.pendingRepliesMutex = sync.RWMutex{}
	hostName, err := os.Hostname()
	if nil != err {
		logger.Error.Printf("PulsarMQ %s initialize while get hostname failed with error:%v", r.Name, err)
	} else {
		r.hostName = hostName
	}
}

// Run start
// 1. init the pulsar conneciton
// 2. expect messages from the message hub on the Publish channel
// 3. if the connection is closed, try to restart it
func (r *PulsarMQ) Run() {
	tick := time.NewTicker(time.Second * 2)
	for {
		if r.connecting == false && r.client == nil {
			r.pendingConsumersMutex.Lock()
			if len(r.consumers) > 0 {
				if nil == r.pendingConsumers {
					r.pendingConsumers = make([]*mqenv.MQConsumerProxy, 0)
				}
				for _, c := range r.consumers {
					r.pendingConsumers = append(r.pendingConsumers, c.consumerProxy)
					c.consumer.Close()
				}
				r.consumers = make(map[string]consumerWrapper, 0)
			}
			r.pendingConsumersMutex.Unlock()
			r.producersMutex.Lock()
			if len(r.producers) > 0 {
				// if nil == r.pendingPublishes {
				// 	r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
				// }
				for _, p := range r.producers {
					p.Close()
				}
				r.producers = make(map[string]pulsar.Producer, 0)
			}
			r.producersMutex.Unlock()
			r.init()
			logger.Trace.Printf("pulsar %s pre running...", r.Name)
			// backstop
			// if r.client != nil {
			// 	r.close()
			// }

			// IMPORTANT: 必须清空 Notify，否则死连接不会释放
			logger.Trace.Printf("pulsar %s do running...", r.Name)
		}

		select {
		case pm := <-r.Publish:
			r.publish(pm)
		case cm := <-r.Consume:
			logger.Info.Printf("consuming topic: %s\n", cm.Queue)
			r.consume(cm)
		case err := <-r.Done:
			logger.Error.Printf("PulsarMQ connection:%s done with error:%v", r.Name, err)
			if r.connecting == false {
				r.close()
				break
			}
			break
		case <-r.Close:
			logger.Warning.Printf("PulsarMQ %s got an event that closing the connection", r.Name)
			r.close()
			tick.Stop()
			return
		case <-tick.C:
			// fmt.Println("pulsar tick...")
			if nil == r.client {
				break
			}
			// if r.client.IsClosed() {
			// 	r.client = nil
			// 	r.connecting = false
			// 	logger.Error.Printf("PulsarMQ connection:%s were closed on ticker checking", r.Name)
			// 	break
			// } else {
			// 	//
			// }
			break
		}
	}
}

func (r *PulsarMQ) close() {
	r.connecting = false
	logger.Info.Printf("PulsarMQ connection:%s closing", r.Name)
	if r.client != nil {
		logger.Info.Printf("PulsarMQ connection:%s closing connection", r.Name)
		r.client.Close()
	}
	if nil != r.sshTunnel {
		logger.Info.Printf("PulsarMQ connection:%s closing ssh tunnel", r.Name)
		r.sshTunnel.Stop()
		r.sshTunnel = nil
	}
	r.client = nil
	logger.Info.Printf("PulsarMQ connection:%s closing finished", r.Name)
}

// try to start a new connection, channel and deliveries channel. if failed, try again in 5 sec.
func (r *PulsarMQ) init() error {
	if mqenv.DriverTypePulsar != r.connConfig.Driver {
		logger.Error.Printf("Initialize pulsar connection by configure:%s failed, the configure driver:%s does not fit.", r.Name, r.connConfig.Driver)
		return errors.New("Invalid driver for pulsar")
	}

	r.connecting = true
	connDSN, connDescription, err := r.formatConnectionDSN()
	if nil != err {
		logger.Error.Printf("Initialize pulsar connection by configure:%s while format pulsar conneciton DSN failed with error:%v", r.Name, err)
		return err
	}

	go func() {
		ticker := time.NewTicker(mqenv.MQReconnectSeconds * time.Second)
		for nil != ticker {
			select {
			case <-ticker.C:
				client, err := pulsar.NewClient(pulsar.ClientOptions{
					URL:               connDSN,
					OperationTimeout:  30 * time.Second,
					ConnectionTimeout: 30 * time.Second,
					Logger:            NewLogger(false),
					Authentication:    r.generateAuthencation(),
				})
				if err != nil {
					// r.connecting = false
					logger.Error.Printf("Could not instantiate Pulsar client %s with %s, failed with error:%v", r.Name, connDSN, err)
					logger.Error.Printf("trying to reconnect in %d seconds...", mqenv.MQReconnectSeconds)
				} else {
					logger.Info.Printf("Connecting pulsar %s with %s succeed", r.Name, connDescription)
					r.connecting = false
					r.client = client
					ticker.Stop()
					r.ensurePendings()
				}
			}
		}
	}()
	return nil
}

// format connection dsn with hosts and port
// if the hosts were configured with ssh tunnel, it will only connect first host of pulsar broker
func (r *PulsarMQ) formatConnectionDSN() (string, string, error) {
	cnf := r.connConfig
	host := cnf.Host
	port := cnf.Port
	hostAddr := ""
	if strings.Contains(host, ",") {
		hostAddr = host
		hosts := strings.Split(host, ",")
		h := strings.Split(hosts[0], ":")
		host = h[0]
		if len(h) > 1 {
			p, err := strconv.Atoi(h[1])
			if nil == err {
				port = p
			}
		}
	} else {
		if strings.Contains(host, ":") {
			hostAddr = host
		} else {
			hostAddr = fmt.Sprintf("%s:%d", host, port)
		}
	}
	var err error
	if "" != cnf.SSHTunnelDSN && !pinger.Connectable(host, port) {
		if nil != r.sshTunnel {
			r.sshTunnel.Stop()
			r.sshTunnel = nil
		}
		for {
			var sshTunnel *sshtunnel.TunnelForwarder
			sshTunnel, err = sshtunnel.NewSSHTunnel(cnf.SSHTunnelDSN, host, port)
			err = sshTunnel.ParseFromDSN(cnf.SSHTunnelDSN)
			if nil != err {
				logger.Error.Printf("format pulsar address while parse SSH Tunnel DSN:%s failed with error:%v", cnf.SSHTunnelDSN, err)
				break
			}

			err = sshTunnel.Start()
			if nil != err {
				logger.Error.Printf("format pulsar address while start SSH Tunnel failed with error:%v", err)
				break
			}
			r.sshTunnel = sshTunnel
			host = sshTunnel.LocalHost()
			port = sshTunnel.LocalPort()
			hostAddr = fmt.Sprintf("%s:%d", host, port)
			break
		}
	}

	connDSN := fmt.Sprintf("pulsar://%s", hostAddr)
	return connDSN, connDSN, err
}

// generate pulsar authencation object
func (r *PulsarMQ) generateAuthencation() pulsar.Authentication {
	var authz pulsar.Authentication
	cnf := r.connConfig
	if "" != cnf.Password {
		authz = pulsar.NewAuthenticationToken(cnf.Password)
	}
	return authz
}

func (r *PulsarMQ) prepareTopicName(topicName string) string {
	if "" == topicName {
		return r.topicName
	}
	if strings.HasPrefix(topicName, r.namespace) || strings.HasPrefix(topicName, "non-persistent://") || strings.HasPrefix(topicName, "persistent://") {
		return topicName
	}
	return utils.URLPathJoin(r.namespace, topicName)
}

func (r *PulsarMQ) ensureProducer(topicName string) (pulsar.Producer, error) {
	if "" == topicName {
		return nil, fmt.Errorf("Topic name should not be empty")
	}
	r.producersMutex.RLock()
	producer, ok := r.producers[topicName]
	r.producersMutex.RUnlock()
	if false == ok {
		var err error
		producer, err = r.client.CreateProducer(pulsar.ProducerOptions{
			Topic: topicName,
			// Name:  topicName,
		})
		if nil != err {
			logger.Error.Printf("Pulsar instance %s create producer with topic:%s failed with error:%v", r.Name, topicName, err)
			return nil, err
		}
		r.producersMutex.Lock()
		r.producers[topicName] = producer
		r.producersMutex.Unlock()
		logger.Info.Printf("Pulsar instance %s create producer with topic:%s succeed.", r.Name, topicName)
	}
	return producer, nil
}

func (r *PulsarMQ) ensureConsumer(topicName string, cm *mqenv.MQConsumerProxy) (pulsar.Consumer, error) {
	r.consumersMutex.RLock()
	consumer, ok := r.consumers[topicName]
	r.consumersMutex.RUnlock()
	if ok {
		return consumer.consumer, nil
	} else {
		subscriptionName := cm.ConsumerTag
		subscriptionType := pulsar.Shared
		if "fanout" == r.config.MessageType || "broadcast" == r.config.MessageType {
			subscriptionName = cm.ConsumerTag + "-" + utils.GenLoweruuid()
			subscriptionType = pulsar.Exclusive
		}
		if r.isInstanceRPC {
			subscriptionName = "rpc-consumer"
			subscriptionType = pulsar.Exclusive
		}
		if "" == subscriptionName {
			subscriptionName = topicName
		}
		pulsarConsumer, err := r.client.Subscribe(pulsar.ConsumerOptions{
			Name:             r.Name,
			Topic:            topicName,
			SubscriptionName: subscriptionName,
			Type:             subscriptionType,
		})
		if nil != err {
			if nil != cm.Ready {
				cm.Ready <- false
			}
			logger.Error.Printf("Pulsar instance %s create consumer with topic:%s failed with error:%v", r.Name, topicName, err)
			return nil, err
		}
		logger.Info.Printf("Pulsar instance %s create consumer with topic:%s succeed.", r.Name, topicName)
		r.consumersMutex.Lock()
		r.consumers[topicName] = consumerWrapper{
			consumer:      pulsarConsumer,
			consumerProxy: cm,
		}
		r.consumersMutex.Unlock()
		if nil != cm.Ready {
			cm.Ready <- true
		}
		return pulsarConsumer, nil
	}
}

func (r *PulsarMQ) publish(pm *mqenv.MQPublishMessage) error {
	if r.client == nil {
		logger.Warning.Printf("pending publishing %dB body (%s)", len(pm.Body), pm.Body)
		r.pendingPublishesMutex.Lock()
		if nil == r.pendingPublishes {
			r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
		}
		r.pendingPublishes = append(r.pendingPublishes, pm)
		r.pendingPublishesMutex.Unlock()
		return nil
	}
	topicName := r.prepareTopicName(pm.RoutingKey)
	producer, err := r.ensureProducer(topicName)
	if nil != err {
		logger.Error.Printf("Pulsar instance %s ensure producer with topic:%s failed with error:%v", r.Name, topicName, err)
		return err
	}

	if nil != pm.Response {
		rpc, err := r.getRPCInstance()
		if nil != err {
			logger.Error.Printf("PulsarMQ %s publish message while get rpc callback instance failed with error:%v", r.Name, err)
		} else {
			rpc.ensureRPCMessage(pm)
		}
	}

	properties := map[string]string{}
	properties = preparePublishMessageProperties(properties, pm)
	m := pulsar.ProducerMessage{
		Payload:    pm.Body,
		Key:        pm.MessageID,
		Properties: properties,
	}
	if logger.IsDebugEnabled() {
		if false == strings.HasPrefix(topicName, r.healthzTopicPrefix) {
			logger.Trace.Printf("PulsarMQ %s publishing message(%s) to %s with %dB body (%s)", r.Name, pm.CorrelationID, topicName, len(pm.Body), pm.Body)
		}
	}
	ctx := context.Background()
	producer.SendAsync(ctx, &m, r.handlePublishMessage)
	if "" != pm.CorrelationID {
		r.answerReplyNeededMessage(pm.CorrelationID)
		if "" != trackerQueue {
			r.publishTrackerMQMessage(pm, strconv.Itoa(int(mqenv.MQTypePublisher)))
		}
	}
	// TODO
	if nil != pm.PublishStatus {
		status := mqenv.MQEvent{
			Code:    mqenv.MQEventCodeOk,
			Label:   pm.EventLabel,
			Message: "Publish success",
		}
		if nil != err {
			status.Code = mqenv.MQEventCodeFailed
			status.Message = err.Error()
		}
		pm.PublishStatus <- status
	}
	if err != nil {
		return fmt.Errorf("Topic :%s publish failed: %s", topicName, err)
	}
	return nil
}

func (r *PulsarMQ) consume(cm *mqenv.MQConsumerProxy) error {
	topicName := r.prepareTopicName(cm.Queue)
	if r.client == nil {
		logger.Warning.Printf("PulsarMQ %s consuming queue:%s failed while the client not ready, pending.", r.Name, topicName)
		r.pendingConsumersMutex.Lock()
		if nil == r.pendingConsumers {
			r.pendingConsumers = make([]*mqenv.MQConsumerProxy, 0)
		}
		r.pendingConsumers = append(r.pendingConsumers, cm)
		r.pendingConsumersMutex.Unlock()
		return nil
	}

	consumer, err := r.ensureConsumer(topicName, cm)
	if nil != err {
		return err
	}

	logger.Info.Printf("Now consuming mq(%s) with topic:%s ...", r.Name, topicName)
	go r.handleConsumes(cm.Callback, cm.AutoAck, consumer, topicName, cm.ConsumerTag)
	return nil
}

func (r *PulsarMQ) ensurePendings() {
	// subscribe pending consumers
	r.pendingConsumersMutex.Lock()
	pendingConsumers := make([]*mqenv.MQConsumerProxy, len(r.pendingConsumers))
	if nil != r.pendingConsumers {
		for i, cm := range r.pendingConsumers {
			pendingConsumers[i] = cm
		}
	}
	r.pendingConsumers = make([]*mqenv.MQConsumerProxy, 0)
	r.pendingConsumersMutex.Unlock()
	for _, cm := range pendingConsumers {
		r.consume(cm)
	}

	// send pending publish messages
	r.pendingPublishesMutex.Lock()
	pendingPublishes := make([]*mqenv.MQPublishMessage, len(r.pendingPublishes))
	if nil != r.pendingPublishes {
		for i, pm := range r.pendingPublishes {
			pendingPublishes[i] = pm
		}
	}
	r.pendingPublishes = make([]*mqenv.MQPublishMessage, 0)
	r.pendingPublishesMutex.Unlock()
	for _, pm := range pendingPublishes {
		r.publish(pm)
	}
}

func (r *PulsarMQ) handlePublishMessage(msgID pulsar.MessageID, pm *pulsar.ProducerMessage, err error) {
	if nil == err {
		// logger.Trace.Printf("PulsarMQ %s publishing message(%s) %dB succeed %+v", r.Name, pm.Properties[PropertyCorrelationID], len(pm.Payload), pm)
	} else {
		logger.Error.Printf("PulsarMQ %s publishing message %dB failed with error:%v message:%s", r.Name, len(pm.Payload), err, pm.Payload)
	}
}

func preparePublishMessageProperties(properties map[string]string, pm *mqenv.MQPublishMessage) map[string]string {
	if nil == properties {
		properties = map[string]string{}
	}
	if nil != pm.Headers {
		for k, v := range pm.Headers {
			properties[k] = v
		}
	}
	if "" != pm.AppID {
		properties[PropertyAppID] = pm.AppID
	}
	if "" != pm.UserID {
		properties[PropertyUserID] = pm.UserID
	}
	if "" != pm.MessageID {
		properties[PropertyMessageID] = pm.MessageID
	}
	if "" != pm.CorrelationID {
		properties[PropertyCorrelationID] = pm.CorrelationID
	}
	if "" != pm.ReplyTo {
		properties[PropertyReplyTo] = pm.ReplyTo
	}
	if "" != pm.ContentType {
		properties[PropertyContentType] = pm.ContentType
	}
	return properties
}

func (r *PulsarMQ) handleConsumes(cb mqenv.MQConsumerCallback, autoAck bool, consumer pulsar.Consumer, topicName string, consumerTag string) {
	ctx := context.Background()
	for {
		msg, err := consumer.Receive(ctx)
		if nil == err {
			logger.Debug.Printf("PulsarMQ %s topic:%s received msg %s", r.Name, topicName, msg.Payload())
			go r.handleConsumeCallback(consumer, msg, cb, autoAck, consumerTag)
		} else {
			logger.Error.Printf("PulsarMQ %s consuming topic:%s failed with error:%v", r.Name, topicName, err)
		}
		// select {
		// case cm := <-consumer.Chan():
		// 	go r.handleConsumeCallback(cm.Consumer, cm.Message, cb, autoAck, consumerTag)
		// }
	}
	// r.Done <- fmt.Errorf("error: deliveries channel closed")
}

func (r *PulsarMQ) getRPCInstance() (*PulsarMQ, error) {
	if nil == r.connConfig || nil == r.config {
		logger.Error.Printf("%s pulsar instance get RPC instance failed that connection configuration empty", r.Name)
		return nil, fmt.Errorf("connection configuration nil")
	}
	pulsarInstMutex.RLock()
	rpcInst, ok := pulsarInsts[r.rpcInstanceName]
	pulsarInstMutex.RUnlock()
	if ok {
		return rpcInst, nil
	}

	config := &Config{
		ConnConfigName: r.config.ConnConfigName,
		Topic:          fmt.Sprintf("rpc-%s-%s", r.hostName, utils.RandomString(8)),
		MessageType:    "topic",
	}
	rpcInst = NewPulsarMQ(r.rpcInstanceName, r.connConfig, config)
	rpcInst.topicName = "non-persistent://" + rpcInst.topicName
	rpcInst.isInstanceRPC = true
	err := rpcInst.init()
	if err == nil {
		go rpcInst.Run()
	} else {
		return nil, err
	}
	pxy := &mqenv.MQConsumerProxy{
		Queue:       rpcInst.topicName,
		Callback:    rpcInst.handleRPCCallback,
		ConsumerTag: rpcInst.Name,
		AutoAck:     true,
		Exclusive:   false,
		NoLocal:     false,
		NoWait:      false,
		Ready:       make(chan bool),
		// Arguments:   consumeProxy.Arguments,
	}
	err = rpcInst.consume(pxy)
	if nil != err {
		logger.Error.Printf("%s consume rpc failed with error:%v", rpcInst.Name, err)
	}
	pulsarInstMutex.Lock()
	pulsarInsts[r.rpcInstanceName] = rpcInst
	pulsarInstMutex.Unlock()

	// waiting for rpc consumer ready, in case producer message sent before subscription
	ticker := time.NewTicker(5 * time.Second)
	for nil != ticker {
		select {
		case <-ticker.C:
			ticker.Stop()
			ticker = nil
			break
		case <-pxy.Ready:
			ticker.Stop()
			ticker = nil
			break
		}
	}

	return rpcInst, nil
}

func (r *PulsarMQ) ensureRPCMessage(pm *mqenv.MQPublishMessage) {
	if "" == pm.CorrelationID {
		pm.CorrelationID = utils.GenLoweruuid()
	}
	pm.ReplyTo = r.topicName
	r.rpcCallbacksMutex.Lock()
	if nil == r.rpcCallbacks {
		r.rpcCallbacks = make(map[string]*mqenv.MQPublishMessage)
	}
	originPm, _ := r.rpcCallbacks[pm.CorrelationID]
	if nil == originPm || originPm.Response == nil {
		r.rpcCallbacks[pm.CorrelationID] = pm
	}
	r.rpcCallbacksMutex.Unlock()
}

func (r *PulsarMQ) handleConsumeCallback(consumer pulsar.Consumer, msg pulsar.Message, cb mqenv.MQConsumerCallback, autoAck bool, consumerTag string) {
	if cb != nil {
		properties := msg.Properties()
		m := mqenv.MQConsumerMessage{
			Driver:      r.connConfig.Driver,
			Queue:       msg.Topic(),
			Timestamp:   msg.PublishTime(),
			Body:        msg.Payload(),
			Headers:     properties,
			BindData:    msg,
			ConsumerTag: consumerTag,
		}
		if nil != properties {
			m.CorrelationID, _ = properties[PropertyCorrelationID]
			m.ReplyTo, _ = properties[PropertyReplyTo]
			m.MessageID, _ = properties[PropertyMessageID]
			m.AppID, _ = properties[PropertyAppID]
			m.UserID, _ = properties[PropertyUserID]
			m.ContentType, _ = properties[PropertyContentType]
			m.RoutingKey, _ = properties["RoutingKey"]
		}

		logger.Debug.Printf("PulsarMQ %s received msg(%s) %s", r.Name, m.CorrelationID, m.Body)
		if "" != m.CorrelationID && "" != m.ReplyTo {
			r.setReplyNeededMessage(m)
			resp := cb(m)
			if nil != resp {
				if false == r.isReplyNeededMessageAnswered(m.CorrelationID) {
					// publish response
					resp.ReplyTo = m.ReplyTo
					r.publish(resp)
				}
			}
			if "" != trackerQueue {
				cmsg := r.generatePublishMessageByDelivery(&m, trackerQueue, []byte{})
				r.publishTrackerMQMessage(cmsg, strconv.Itoa(int(mqenv.MQTypeConsumer)))
			}
		} else {
			cb(m)
		}
	}
	// if autoAck {
	consumer.Ack(msg)
	// }
}

// QueryRPC publishes a message and waiting the response
func (r *PulsarMQ) QueryRPC(pm *mqenv.MQPublishMessage) (*mqenv.MQConsumerMessage, error) {
	pm.Response = make(chan mqenv.MQConsumerMessage)
	r.Publish <- pm

	var timer *time.Timer
	var resp *mqenv.MQConsumerMessage
	var err error
	if pm.TimeoutSeconds > 0 {
		timer = time.NewTimer(time.Duration(pm.TimeoutSeconds) * time.Second)
	} else {
		timer = time.NewTimer(30 * time.Second)
	}
	select {
	case res := <-pm.Response:
		if logger.IsDebugEnabled() {
			logger.Trace.Printf("PulsarMQ %s Got response %s", r.Name, string(res.Body))
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

func (r *PulsarMQ) handleRPCCallback(msg mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	if nil == r.rpcCallbacks {
		return nil
	}
	correlationID := msg.CorrelationID
	r.rpcCallbacksMutex.RLock()
	pm, pmExists := r.rpcCallbacks[correlationID]
	r.rpcCallbacksMutex.RUnlock()
	if pmExists {
		if pm.CallbackEnabled() {
			// fmt.Printf("====> push back response data %s %+v\n", correlationID, pm)
			// resp := generateMQResponseMessage(msg, r.Config.ExchangeName)
			// resp.Queue = r.queueName
			// resp.ReplyTo = pm.ReplyTo
			pm.Response <- msg
		}
		r.rpcCallbacksMutex.Lock()
		delete(r.rpcCallbacks, correlationID)
		r.rpcCallbacksMutex.Unlock()
	}
	return nil
}

func (r *PulsarMQ) setReplyNeededMessage(m mqenv.MQConsumerMessage) {
	if "" != m.CorrelationID && "" != m.ReplyTo {
		r.pendingRepliesMutex.Lock()
		if nil == r.pendingReplies {
			r.pendingReplies = make(map[string]mqenv.MQConsumerMessage)
		}
		r.pendingReplies[m.CorrelationID] = m
		r.pendingRepliesMutex.Unlock()
	}
}

func (r *PulsarMQ) answerReplyNeededMessage(correlationID string) {
	r.pendingRepliesMutex.Lock()
	if "" != correlationID && nil != r.pendingReplies {
		_, exists := r.pendingReplies[correlationID]
		if exists {
			delete(r.pendingReplies, correlationID)
		}
	}
	r.pendingRepliesMutex.Unlock()
}

func (r *PulsarMQ) isReplyNeededMessageAnswered(correlationID string) bool {
	if "" == correlationID || nil == r.pendingReplies {
		return true
	}
	r.pendingRepliesMutex.RLock()
	_, exists := r.pendingReplies[correlationID]
	r.pendingRepliesMutex.RUnlock()
	return false == exists
}

func (r *PulsarMQ) publishTrackerMQMessage(pm *mqenv.MQPublishMessage, typ string) {
	if "" == trackerQueue {
		return
	}
	producer, err := r.ensureProducer(trackerQueue)
	if nil != err {
		logger.Error.Printf("PulsarMQ %s ensure tracker queue:%s producer failed with error:%v", r.Name, trackerQueue, err)
		return
	}
	properties := map[string]string{}
	if nil != pm.Headers {
		for k, v := range pm.Headers {
			properties[k] = v
		}
	}
	properties["ReplyTo"] = pm.ReplyTo
	properties["Type"] = typ
	m := pulsar.ProducerMessage{
		Payload:    pm.Body,
		Key:        pm.MessageID,
		Properties: properties,
	}
	ctx := context.Background()
	producer.SendAsync(ctx, &m, r.handlePublishMessage)
}

func (r *PulsarMQ) generatePublishMessageByDelivery(msg *mqenv.MQConsumerMessage, queueName string, body []byte) *mqenv.MQPublishMessage {
	headers := map[string]string{}
	if nil != msg.Headers {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}
	pub := &mqenv.MQPublishMessage{
		Body:          body,
		RoutingKey:    queueName,
		CorrelationID: msg.CorrelationID,
		ReplyTo:       msg.ReplyTo,
		MessageID:     msg.MessageID,
		AppID:         msg.AppID,
		UserID:        msg.UserID,
		ContentType:   msg.ContentType,
		Headers:       headers,
	}
	return pub
}
