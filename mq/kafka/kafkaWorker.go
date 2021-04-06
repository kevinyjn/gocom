package kafka

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

// Worker 订阅topic 后处理收到信息的回调函数.
type Worker func(*KafkaPacket) []byte

// KafkaWorker 把生产者、消费者结合起来，实现请求响应模式.
type KafkaWorker struct {
	Producer            *Producer                         // 生产者
	Consumer            *Consumer                         // 消费者
	consumerRegisters   map[string]*mqenv.MQConsumerProxy // 已经订阅的topic
	methodRegisters     map[string]*mqenv.MQConsumerProxy // 处理函数字典
	PrivateTopic        string                            // 私有topic，用于发出信息后收到回复
	waitResponseMessage map[string]chan *KafkaPacket      //发出信息后，会以消息id为key 保存在字典中，值是通道。通过通道来接收信息
	availableChannels   []chan *KafkaPacket               // 可用于接收的通道切片
	openTopicChannel    map[string]string                 // 记录已经打开的topic通道
	ContentType         string                            //序列化类型，如json
	ContentEncoding     string                            // 编码格式
	GroupID             string                            //组id，会包含在 kafkapacket 数据包中
	MsgType             string                            // 消息类型
	Stats               Stats                             // 统计信息
	UseOriginalContent  bool                              // 是否使用原始的方式序列化(使用json 序列化，而不是protobuf)
}

// newChannel 返回一个新的 字节数组通道.
func (worker *KafkaWorker) newChannel() chan *KafkaPacket {
	c := make(chan *KafkaPacket)
	return c
}

// obtainChannel 获取一个通道.
func (worker *KafkaWorker) obtainChannel() chan *KafkaPacket {
	channelLength := len(worker.availableChannels)
	if channelLength == 0 {
		return worker.newChannel()
	}
	c := worker.availableChannels[channelLength-1]
	// 从可用的数组从移除
	worker.availableChannels = worker.availableChannels[:channelLength-1]
	return c
}

// recycleChannel 回收用过的旧通道.
func (worker *KafkaWorker) recycleChannel(c chan *KafkaPacket) {
	worker.availableChannels = append(worker.availableChannels, c)
}

// sendWorker 对发送的操作做额外的操作.
func (worker *KafkaWorker) sendWorker(topic string, message []byte) error {
	_, ok := worker.openTopicChannel[topic]
	if !ok {
		err := worker.sendOpenChannel(topic)
		if err != nil {
			return err
		}
		worker.openTopicChannel[topic] = "1"
	}
	worker.Stats.Producer.Bytes += int64(len(message))
	worker.Stats.Producer.Messages++
	err := worker.Producer.Send(topic, message)
	return err
}

// sendOpenChannel 用于创建私有topic 或第一次发topic发送信息，用于把通道打开.
func (worker *KafkaWorker) sendOpenChannel(topic string) error {
	registerValue := []byte("{'_register_private': 'open'}")
	for i := 0; i < 20; i++ {
		err := worker.Producer.Send(topic, registerValue)
		if nil != err {
			logger.Error.Println(err)
			// 因为开启kafka 自动创建topic后，第一次向private_topic 发送信息会报错
			// return err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// registerPrivateTopic 注册私有的topic，接收发送出去的信息的回复.
func (worker *KafkaWorker) registerPrivateTopic() {
	if worker.PrivateTopic == "" {
		return
	}
	_, ok := worker.consumerRegisters[worker.PrivateTopic]
	if !ok {
		worker.sendOpenChannel(worker.PrivateTopic)
		// PrivateTopic 收到的信息是发出信息后的回复，会在onMessage被拦截用chan 返回给发送方
		// 所以这里传的回调函数不做任何处理
		time.Sleep(100 * time.Millisecond)
		proxy := &mqenv.MQConsumerProxy{}
		worker.Subscribe(worker.PrivateTopic, proxy)
	}
}

// Send 发送信息.
func (worker *KafkaWorker) Send(topic string, publishMsg *mqenv.MQPublishMessage, needReply bool) (*mqenv.MQConsumerMessage, error) {

	worker.registerPrivateTopic()
	headers := make([]*KafkaPacket_Header, 0)
	for k, v := range publishMsg.Headers {
		h := &KafkaPacket_Header{
			Name:  k,
			Value: v,
		}
		headers = append(headers, h)
	}
	replyTo := worker.PrivateTopic
	if worker.PrivateTopic == "" {
		replyTo = publishMsg.ReplyTo
	}
	p := &KafkaPacket{
		ContentType:     publishMsg.ContentType,
		ContentEncoding: worker.ContentEncoding,
		SendTo:          topic,
		GroupId:         worker.GroupID,
		CorrelationId:   publishMsg.CorrelationID,
		ReplyTo:         replyTo,
		Timestamp:       uint64(utils.CurrentMillisecond()),
		Type:            worker.MsgType,
		UserId:          publishMsg.UserID,
		AppId:           publishMsg.AppID,
		StatusCode:      200,
		ErrorMessage:    "success",
		Body:            publishMsg.Body,
		Headers:         headers,
		RoutingKey:      publishMsg.RoutingKey,
		ConsumerTag:     publishMsg.RoutingKey,
		Exchange:        publishMsg.Exchange,
	}
	var sendBytes []byte
	var err error
	if worker.UseOriginalContent {
		sendBytes, err = json.Marshal(p)
	} else {
		sendBytes, err = proto.Marshal(p)
	}

	if err != nil {
		logger.Error.Println(err)
		return nil, err
	}
	// 注册通道，等待回复
	if needReply {
		ch := worker.obtainChannel()
		worker.waitResponseMessage[p.CorrelationId] = ch
		worker.sendWorker(topic, sendBytes)
		responsePacket := <-ch
		// 回收通道
		worker.recycleChannel(ch)

		consumerMessage := ConvertKafkaPacketToMQConsumerMessage(responsePacket)
		return &consumerMessage, nil
	}
	worker.sendWorker(topic, sendBytes)
	return nil, nil

}

// reply 服务端收到信息处理完后进行回复.
func (worker *KafkaWorker) reply(topic string, message *mqenv.MQPublishMessage, msgID string) {

	headers := make([]*KafkaPacket_Header, 0)
	for k, v := range message.Headers {
		h := &KafkaPacket_Header{
			Name:  k,
			Value: v,
		}
		headers = append(headers, h)
	}
	p := &KafkaPacket{
		ContentType:     message.ContentType,
		ContentEncoding: worker.ContentEncoding,
		SendTo:          topic,
		GroupId:         worker.GroupID,
		CorrelationId:   msgID,
		ReplyTo:         worker.PrivateTopic,
		Timestamp:       uint64(time.Now().UTC().Unix()),
		Type:            worker.MsgType,
		UserId:          message.UserID,
		AppId:           message.AppID,
		StatusCode:      200,
		ErrorMessage:    "success",
		Body:            message.Body,
		Headers:         headers,
	}
	var sendBytes []byte
	var err error
	if worker.UseOriginalContent {
		sendBytes, err = json.Marshal(p)
	} else {
		sendBytes, err = proto.Marshal(p)
	}

	if err != nil {
		logger.Error.Println(err)
	}
	worker.sendWorker(topic, sendBytes)
	// logger.Debug.Println("reply " + string(message.Body))

}

// onMessage 处理收到的信息的逻辑.
func (worker *KafkaWorker) onMessage(packet *KafkaPacket) {
	// 收到信息有两种情况
	// 1 发出信息后收到回复,在waitResponseMessage 有通道，把信息发送过去就可以了
	// 2 订阅topic 后收到的回复
	// logger.Debug.Println("onMessage body=" + string(packet.Body))
	ch, ok := worker.waitResponseMessage[packet.CorrelationId]
	if ok {
		delete(worker.waitResponseMessage, packet.CorrelationId)
		ch <- packet
	} else {
		//consumerProxy, isExits := worker.consumerRegisters[packet.SendTo]
		// 先从包含具体方法的字典(methodRegisters)查找处理函数
		//如果找不到就从订阅topic的字典(consumerRegisters)查找
		key := fmt.Sprintf("%s-%s", packet.SendTo, packet.RoutingKey)
		consumerProxy, isExits := worker.methodRegisters[key]
		if !isExits {
			consumerProxy, isExits = worker.consumerRegisters[packet.SendTo]
		}
		if isExits {
			func() {
				defer func() {
					if err := recover(); err != nil {
						logger.Error.Println(err)
					}
				}()
				consumerMessage := ConvertKafkaPacketToMQConsumerMessage(packet)
				if consumerProxy.Callback != nil {
					result := consumerProxy.Callback(consumerMessage)
					// if result != nil {
					// 	logger.Debug.Println("onMessage result=" + string(result.Body))
					// 	logger.Debug.Println("onMessage packet.ReplyTo=" + packet.ReplyTo)
					// }
					if result != nil && packet.ReplyTo != "" {
						// logger.Debug.Println("onMessage begin to reply")
						worker.reply(packet.ReplyTo, result, packet.CorrelationId)
					}
				}
			}()
		}
	}

}

// bindToOnMessage 绑定接收到的数据.
func (worker *KafkaWorker) bindToOnMessage(data []byte) {
	// 私有topic不一定存在，所以worker 会发送信息来创建。
	// 收到创建的信息忽略掉
	logger.Debug.Println("bindToOnMessage: " + string(data))
	if strings.Contains(string(data), "_register_private") {
		return
	}
	worker.Stats.Consumer.Bytes += int64(len(data))
	worker.Stats.Consumer.Messages++
	p := &KafkaPacket{}
	var err error
	if worker.UseOriginalContent {
		err = json.Unmarshal(data, p)

	} else {
		err = proto.Unmarshal(data, p)
	}
	if err != nil {
		logger.Error.Println(err)
	} else {
		worker.extractRoutingKey(p)
		worker.onMessage(p)
	}

}

// Subscribe 订阅topic.
func (worker *KafkaWorker) Subscribe(topic string, consumeProxy *mqenv.MQConsumerProxy) error {

	_, ok := worker.consumerRegisters[topic]
	if !ok {
		logger.Info.Println("Subscribe subscribing topic " + topic)
		worker.Consumer.Receive(topic, worker.bindToOnMessage)
		worker.consumerRegisters[topic] = consumeProxy

	}
	key := fmt.Sprintf("%s-%s", consumeProxy.Queue, consumeProxy.ConsumerTag)
	_, ok = worker.methodRegisters[key]
	if !ok {
		worker.methodRegisters[key] = consumeProxy
	}
	return nil
}

// extractRoutingKey 尝试从body 里面提取routing key.
func (worker *KafkaWorker) extractRoutingKey(packet *KafkaPacket) error {

	if packet.RoutingKey != "" {
		return nil
	}
	var params struct {
		Method     string `json:"method"`
		RoutingKey string `json:"routingKey"`
	}
	err := json.Unmarshal(packet.Body, &params)
	if nil != err {
		logger.Error.Println(err)
		return err
	}
	if params.Method != "" {
		packet.RoutingKey = params.Method
		return nil
	}
	if params.RoutingKey != "" {
		packet.RoutingKey = params.RoutingKey
		return nil
	}
	return fmt.Errorf("cannot found method value")
}

// NewKafkaWorker 实例化一个kafka worker.
func NewKafkaWorker(hosts string, partition int, privateTopic, groupID string) *KafkaWorker {
	worker := &KafkaWorker{}
	worker.Producer = NewProducer(hosts, partition)
	worker.Consumer = NewConsumer(hosts, groupID)
	worker.PrivateTopic = privateTopic
	worker.waitResponseMessage = make(map[string]chan *KafkaPacket)
	worker.availableChannels = []chan *KafkaPacket{}
	worker.consumerRegisters = make(map[string]*mqenv.MQConsumerProxy)
	worker.methodRegisters = make(map[string]*mqenv.MQConsumerProxy)
	worker.openTopicChannel = make(map[string]string)
	worker.Stats.Consumer = InstStats{}
	worker.Stats.Producer = InstStats{}

	return worker
}
