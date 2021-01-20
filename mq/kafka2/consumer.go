package kafka2

import (
	"time"

	"github.com/kevinyjn/gocom/logger"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// CallBack .回调函数
type CallBack func([]byte)

// Consumer 消费者.
type Consumer struct {
	Base
	Consumer      *kafka.Consumer
	IsInitialized bool //是否已经初始化
}

// ConfigGroupID 配置group id.
func (c *Consumer) ConfigGroupID(groupID string) {
	c.Config["group.id"] = groupID
}

// StopConsumer 停止消费.
func (c *Consumer) StopConsumer() {
	c.IsInitialized = false
}

// Receive 订阅topic，处理消息.
// @title Receive
// @param topic 订阅的topic
// @param callback ,处理接收到的信息，入参是 接收到的[]byte
func (c *Consumer) Receive(topic string, callback CallBack) error {

	if !c.IsInitialized {
		consumer, err := kafka.NewConsumer(&c.Config)
		if err != nil {
			logger.Error.Panicln(err)
			return err
		}
		c.Consumer = consumer
		c.IsInitialized = true
		logger.Debug.Println("Receive init " + topic + " success")
	}
	c.Consumer.Subscribe(topic, nil)
	go func() {
		for {
			if c.IsInitialized {
				logger.Debug.Println("Receive begin to receive data")
				msg, err := c.Consumer.ReadMessage(-1)
				if err == nil {
					// 执行回调函数的时候进行异常捕捉，避免退出循环
					func() {
						defer func() {
							if err := recover(); err != nil {
								logger.Error.Println(err)
							}
						}()

						callback(msg.Value)
					}()

				} else {
					logger.Error.Printf("Consumer error: %v (%v)\n", err, msg)
				}
			} else {
				// 停止消费了，所以退出
				c.Consumer.Close()
				break
			}
		}
	}()
	//周期性的提交偏移量
	go func() {
		for c.IsInitialized {
			time.Sleep(1000 * time.Millisecond)
			c.Consumer.Commit()
		}
	}()
	return nil
}

// NewConsumer 返回消费者.
func NewConsumer(hosts string, groupID string) *Consumer {
	c := &Consumer{}
	c.Config = kafka.ConfigMap{}
	c.ConfigServers(hosts)
	c.ConfigGroupID(groupID)
	//c.ConfigPartition(0)
	c.ConfigHeartbeatInterval(2000)
	c.ConfigSessionTimeout(6000)
	return c
}
