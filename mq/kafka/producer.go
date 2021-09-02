package kafka

import (
	"context"
	"strings"
	"time"

	"github.com/kevinyjn/gocom/logger"
	k "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// Producer 生产者.
type Producer struct {
	Base
	Brokers []string // kafka 的节点
	Writer  map[string]*k.Writer
}

// Send 发送一条消息.
func (p *Producer) Send(topic string, value []byte) error {
	logger.Debug.Printf("send %s %s", topic, value)
	writer, ok := p.Writer[topic]
	if !ok {
		config := k.WriterConfig{
			Brokers:      p.Brokers,
			Topic:        topic,
			Balancer:     &k.Hash{},
			Async:        true,
			BatchTimeout: 10 * time.Millisecond,
		}
		// logger.Trace.Printf("new writer %s", topic)
		if p.Config["sasl.username"] != nil && p.Config["sasl.password"] != nil {
			logger.Debug.Println("using sasl ")
			mechanism := plain.Mechanism{
				Username: p.Config["sasl.username"].(string),
				Password: p.Config["sasl.password"].(string),
			}
			dialer := &k.Dialer{
				Timeout:       10 * time.Second,
				DualStack:     true,
				SASLMechanism: mechanism,
			}
			config.Dialer = dialer

		}
		writer = k.NewWriter(config)
		if p.Config["completion"] != nil {
			writer.Completion = p.CompletionCallback
		}

		p.Writer[topic] = writer
	}
	err := writer.WriteMessages(context.Background(),
		k.Message{
			Value: value,
		},
	)

	return err
}

// NewProducer 返回一个生产者.
func NewProducer(hosts string, partition int) *Producer {
	p := &Producer{}
	p.Config = make(map[string]interface{})
	p.Writer = make(map[string]*k.Writer)
	p.Brokers = strings.Split(hosts, ",")
	p.ConfigPartition(partition)
	p.CompletionCallback = nil
	return p
}
