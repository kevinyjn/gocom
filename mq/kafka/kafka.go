package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"

	"github.com/segmentio/kafka-go"
)

var kafkaInsts = map[string]*Kafka{}

// InitKafka init kafak
func InitKafka(mqConnName string, connCfg *mqenv.MQConnectorConfig, kafkaCfg *Config) (*Kafka, error) {
	kafkaInst, ok := kafkaInsts[mqConnName]
	if ok && !kafkaInst.Config.Equals(kafkaCfg) {
		kafkaInst.close()
		close(kafkaInst.Close)
		ok = false
	}
	if !ok {
		kafkaInst = &Kafka{
			Name:             mqConnName,
			Config:           kafkaCfg,
			ConnConfig:       connCfg,
			Publish:          make(chan *PublishingMsg),
			Consume:          make(chan *ConsumerProxy),
			Done:             make(chan error),
			Close:            make(chan interface{}),
			pendingConsumers: make([]*ConsumerProxy, 0),
			pendingPublishes: make([]*PublishingMsg, 0),
			connecting:       false,
		}
		kafkaInsts[mqConnName] = kafkaInst
		err := kafkaInst.init()
		if err == nil {
			go kafkaInst.Run()
		} else {
			return nil, err
		}
	}
	return kafkaInst, nil
}

// GetKafka get instance
func GetKafka(name string) (*Kafka, error) {
	kafkaInst, ok := kafkaInsts[name]
	if ok {
		return kafkaInst, nil
	}
	return nil, fmt.Errorf("Kafka instance by %s not found", name)
}

func (r *Kafka) init() error {
	logger.Info.Printf("Initializing kafka instance:%s", r.Name)
	return r.initConn()
}

// Run start
func (r *Kafka) Run() {
	for {
		if false == r.connecting && nil == r.Writer && nil == r.Reader {
			r.initConn()
			logger.Trace.Printf("kafka %s pre running...", r.Name)
			// 	// backstop
			// 	if r.Conn != nil && !r.Conn.IsClosed() {
			// 		if err := r.Channel.Cancel("", true); err != nil {
			// 			logger.Error.Printf("Kafka %s cancel channel failed with error:%v", r.Name, err)
			// 		}
			// 		if err := r.Conn.Close(); err != nil {
			// 			logger.Error.Printf("Kafka %s close connection failed with error:%v", r.Name, err)
			// 		}
			// 	}

			// 	// IMPORTANT: 必须清空 Notify，否则死连接不会释放
			// 	if r.channelClosed != nil {
			// 		for err := range r.channelClosed {
			// 			println(err)
			// 		}
			// 	}
			// 	if r.connClosed != nil {
			// 		for err := range r.connClosed {
			// 			println(err)
			// 		}
			// 	}
			logger.Trace.Printf("kafka %s do running...", r.Name)
		}

		select {
		case pm := <-r.Publish:
			logger.Trace.Printf("publish message: %s\n", pm.Body)
			r.publish(pm)
		case cm := <-r.Consume:
			logger.Info.Printf("consuming topic: %s\n", cm.Topic)
			r.consume(cm)
		case err := <-r.Done:
			logger.Error.Printf("Kafka connection:%s done with error:%v", r.Name, err)
			r.close()
		// case err := <-r.connClosed:
		// 	logger.Error.Printf("Kafka connection:%s closed with error:%v", r.Name, err)
		// 	r.close()
		// case err := <-r.channelClosed:
		// 	logger.Error.Printf("Kafka channel:%s closed with error:%v", r.Name, err)
		// 	r.close()
		case <-r.Close:
			r.close()
			return
		}
	}
}

func (r *Kafka) close() {
	r.connecting = false
	if r.Reader != nil {
		r.Reader.Close()
		r.Reader = nil
	}
	if r.Writer != nil {
		r.Writer.Close()
		r.Writer = nil
	}
}

func getKafkaReader(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:                brokers,
		GroupID:                groupID,
		Topic:                  topic,
		MinBytes:               10e3, // 10KB
		MaxBytes:               10e6, // 10MB
		ErrorLogger:            logger.Error,
		PartitionWatchInterval: 100 * time.Millisecond,
	})
}

func getKafkaWriter(brokers []string, topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
}

func formatKafkaBrokers(cnf *mqenv.MQConnectorConfig) []string {
	brokers := []string{}
	addrs := strings.Split(cnf.Host, ",")
	for _, addr := range addrs {
		if !strings.Contains(addr, ":") {
			addr = fmt.Sprintf("%s:%d", addr, cnf.Port)
		}
		brokers = append(brokers, addr)
	}
	return brokers
}

// try to start a new connection, channel and deliveries channel. if failed, try again in 5 sec.
func (r *Kafka) initConn() error {
	cnf := r.ConnConfig
	if cnf.Driver != mqenv.DriverTypeKafka {
		logger.Error.Printf("Initialize kafka connection by configure:%s failed, the configure driver:%s does not fit.", r.Name, cnf.Driver)
		return errors.New("Invalid driver for kafka")
	}

	r.connecting = true
	kafkaAddr := formatKafkaBrokers(cnf)

	go func() {
		ticker := time.NewTicker(KafkaReconnectDuration * time.Second)
		quitTiker := make(chan struct{})
		for {
			select {
			case <-ticker.C:
				reader := getKafkaReader(kafkaAddr, r.Config.Topic, r.Config.GroupID)
				writer := getKafkaWriter(kafkaAddr, r.Config.Topic)
				if nil == reader || nil == writer {
					if nil != reader {
						defer reader.Close()
					}
					if nil != writer {
						defer writer.Close()
					}
					r.connecting = false
					logger.Error.Println("node will only be able to respond to local connections")
					logger.Error.Println("trying to reconnect in 5 seconds...")
				} else {
					logger.Info.Printf("Connection kafka:%s:%d/%s by user:%s succeed", cnf.Host, cnf.Port, cnf.Path, cnf.User)
					r.connecting = false
					r.Reader = reader
					r.Writer = writer
					// defer reader.Close()
					// defer writer.Close()
					if r.pendingConsumers != nil && len(r.pendingConsumers) > 0 {
						consumers := r.pendingConsumers
						r.pendingConsumers = make([]*ConsumerProxy, 0)
						for _, cm := range consumers {
							r.consume(cm)
						}
					}
					if r.pendingPublishes != nil && len(r.pendingPublishes) > 0 {
						publishes := r.pendingPublishes
						r.pendingPublishes = make([]*PublishingMsg, 0)
						for _, pm := range publishes {
							r.publish(pm)
						}
					}
					close(quitTiker)
				}
			case <-quitTiker:
				ticker.Stop()
				return
			}
		}
	}()
	return nil
}

func (r *Kafka) publish(pm *PublishingMsg) error {
	if r.Writer == nil {
		logger.Warning.Printf("pending publishing %dB body (%s)", len(pm.Body), pm.Body)
		r.pendingPublishes = append(r.pendingPublishes, pm)
		return nil
	}
	logger.Trace.Printf("publishing %dB body (%s)", len(pm.Body), pm.Body)

	headers := []kafka.Header{}
	if nil != pm.Headers {
		for k, v := range pm.Headers {
			headers = append(headers, kafka.Header{
				Key:   k,
				Value: v,
			})
		}
	}
	err := r.Writer.WriteMessages(context.Background(), kafka.Message{
		Key:       pm.Key,
		Value:     pm.Body,
		Topic:     pm.Topic,
		Partition: pm.Partition,
		Offset:    pm.Offset,
		Headers:   headers,
	})
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
		logger.Error.Printf("Publish message key:%s value:%s failed with error: %v", string(pm.Key), string(pm.Body), err)
		return err
	}
	return nil
}

func (r *Kafka) consume(cm *ConsumerProxy) error {
	if r.Reader == nil {
		logger.Warning.Printf("Consuming topic:%s failed while the reader connection not ready, pending.", cm.Topic)
		r.pendingConsumers = append(r.pendingConsumers, cm)
		return nil
	}
	logger.Info.Printf("Consuming mq with topic:%s ...", cm.Topic)
	go r.handleConsumes(cm)
	return nil
}

func (r *Kafka) handleConsumes(cm *ConsumerProxy) {
	for {
		msg, err := r.Reader.FetchMessage(context.Background())
		if nil != err {
			logger.Error.Printf("Consume kafka message with topic:%s failed with error:%v", cm.Topic, err)
			// TODO error process, weather should reconnect
			r.Done <- fmt.Errorf("Kafka reader by topic:%s closed", cm.Topic)
		} else {
			if nil != cm.Callback {
				headers := map[string][]byte{}
				if nil != msg.Headers {
					for _, h := range msg.Headers {
						headers[h.Key] = h.Value
					}
				}

				cm.Callback(ConsumerMessage{
					Topic:     msg.Topic,
					Key:       msg.Key,
					Value:     msg.Value,
					Headers:   headers,
					Partition: msg.Partition,
					Offset:    msg.Offset,
					Time:      msg.Time,
				})

				r.Reader.CommitMessages(context.Background(), msg)
			}
		}
	}
	r.Done <- fmt.Errorf("Kafka reader by topic:%s closed", cm.Topic)
}

// Stats stats
func (r *Kafka) Stats() Stats {
	stats := Stats{}
	if nil != r.Reader {
		readerStats := r.Reader.Stats()
		stats.Consumer = InstStats{
			Bytes:         readerStats.Bytes,
			ClientID:      readerStats.ClientID,
			Topic:         readerStats.Topic,
			Dials:         readerStats.Dials,
			Messages:      readerStats.Messages,
			Errors:        readerStats.Errors,
			Rebalances:    readerStats.Rebalances,
			Timeouts:      readerStats.Timeouts,
			QueueCapacity: readerStats.QueueCapacity,
			QueueLength:   readerStats.QueueLength,
		}

	}
	if nil != r.Writer {
		writerStats := r.Writer.Stats()
		stats.Producer = InstStats{
			Bytes:         writerStats.Bytes,
			ClientID:      writerStats.ClientID,
			Topic:         writerStats.Topic,
			Dials:         writerStats.Dials,
			Messages:      writerStats.Messages,
			Errors:        writerStats.Errors,
			Rebalances:    writerStats.Rebalances,
			QueueCapacity: writerStats.QueueCapacity,
			QueueLength:   writerStats.QueueLength,
		}
	}
	return stats
}

// GenerateKafkaConsumerProxy geenrate kafak consumer proxy
func GenerateKafkaConsumerProxy(consumeProxy *mqenv.MQConsumerProxy) *ConsumerProxy {
	cb := func(msg ConsumerMessage) {
		mqMsg := mqenv.MQConsumerMessage{
			Driver:        mqenv.DriverTypeKafka,
			Queue:         msg.Topic,
			CorrelationID: "",
			ConsumerTag:   "",
			ReplyTo:       "",
			RoutingKey:    string(msg.Key),
			Body:          msg.Value,
			BindData:      &msg,
		}

		if nil != consumeProxy.Callback {
			consumeProxy.Callback(mqMsg)
		}
	}
	pxy := ConsumerProxy{
		Topic:       consumeProxy.Queue,
		Callback:    cb,
		ConsumerTag: consumeProxy.ConsumerTag,
		AutoAck:     consumeProxy.AutoAck,
		Exclusive:   consumeProxy.Exclusive,
		NoLocal:     consumeProxy.NoLocal,
		NoWait:      consumeProxy.NoWait,
		// Arguments:   consumeProxy.Arguments,
	}
	return &pxy
}

// GenerateKafkaPublishMessage generate publish message
func GenerateKafkaPublishMessage(publishMsg *mqenv.MQPublishMessage, topic string) *PublishingMsg {
	msg := &PublishingMsg{
		Body:          publishMsg.Body,
		Key:           []byte(publishMsg.RoutingKey),
		Topic:         topic,
		PublishStatus: publishMsg.PublishStatus,
		EventLabel:    publishMsg.EventLabel,
	}
	return msg
}
