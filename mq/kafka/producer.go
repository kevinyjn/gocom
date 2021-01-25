package kafka

import (
	"time"

	"github.com/kevinyjn/gocom/logger"
	confluentKafka "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// Producer 生产者.
type Producer struct {
	Base
	Producer      *confluentKafka.Producer // 生产者
	IsInitialized bool                     //是否已经初始化
}

// SendReport 打印发送报告.
func (p *Producer) SendReport() {
	if p.IsInitialized {
		for e := range p.Producer.Events() {
			switch ev := e.(type) {
			case *confluentKafka.Message:
				if ev.TopicPartition.Error != nil {
					logger.Error.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					logger.Debug.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}
}

// StartFlush 为确保消息成功发出，周期性刷新.
func (p *Producer) StartFlush() {
	if p.IsInitialized {
		for {
			time.Sleep(10 * time.Second)
			p.Producer.Flush(200)
		}
	}
}

// Send 发送信息.
func (p *Producer) Send(topic string, value []byte) error {
	if !p.IsInitialized {
		// 初始化生产者
		logger.Info.Println("初始化生产者")
		producer, err := confluentKafka.NewProducer(&p.Config)
		if err != nil {
			logger.Error.Println(err)
			panic(err)
		}
		p.Producer = producer
		p.IsInitialized = true
		go p.StartFlush()
		go p.SendReport()
	}

	err := p.Producer.Produce(&confluentKafka.Message{
		TopicPartition: confluentKafka.TopicPartition{Topic: &topic, Partition: int32(p.Base.Partition)},
		Value:          value,
	}, nil)
	return err

}

// NewProducer 返回一个生产者.
func NewProducer(hosts string, partition int) *Producer {
	p := &Producer{}
	p.Config = confluentKafka.ConfigMap{}
	p.ConfigServers(hosts)
	p.ConfigPartition(partition)
	return p
}
