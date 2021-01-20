package kafka2

import (
	"fmt"
	"time"

	"github.com/kevinyjn/gocom/mq/mqenv"
)

// kafkaInstances kafka 实例.
var kafkaInstances = map[string]*KafkaWorker{}

// Config kafkav2 配置参数.
type Config struct {
	Hosts        string
	Partition    int
	PrivateTopic string
	GroupID      string
	// kerberos 认证需要配置
	KerberosServiceName string
	KerberosKeytab      string
	KerberosPrincipal   string
	// plain 认证需要配置
	SaslMechanisms string
	SaslUsername   string
	SaslPassword   string
}

// InitKafka 初始化kafka.
func InitKafka(mqConnName string, config Config) (*KafkaWorker, error) {
	instance, ok := kafkaInstances[mqConnName]
	if !ok {
		instance = NewKafkaWorker(config.Hosts, config.Partition, config.PrivateTopic, config.GroupID)
		if config.KerberosServiceName != "" && config.KerberosKeytab != "" && config.KerberosPrincipal != "" {
			instance.Producer.ConfigKerberosServiceName(config.KerberosServiceName)
			instance.Producer.ConfigKerberosKeyTab(config.KerberosKeytab)
			instance.Producer.ConfigKerberosPrincipal(config.KerberosPrincipal)
			instance.Producer.ConfigSecurityProtocol("sasl_plaintext")

			instance.Consumer.ConfigKerberosServiceName(config.KerberosServiceName)
			instance.Consumer.ConfigKerberosKeyTab(config.KerberosKeytab)
			instance.Consumer.ConfigKerberosPrincipal(config.KerberosPrincipal)
			instance.Consumer.ConfigSecurityProtocol("sasl_plaintext")
		}
		if config.SaslMechanisms != "" && config.SaslUsername != "" && config.SaslPassword != "" {
			instance.Producer.ConfigSaslMechanisms(config.SaslMechanisms)
			instance.Producer.ConfigSaslUserName(config.SaslUsername)
			instance.Producer.ConfigSaslPassword(config.SaslPassword)
			instance.Producer.ConfigSecurityProtocol("sasl_plaintext")

			instance.Consumer.ConfigSaslMechanisms(config.SaslMechanisms)
			instance.Consumer.ConfigSaslUserName(config.SaslUsername)
			instance.Consumer.ConfigSaslPassword(config.SaslPassword)
			instance.Consumer.ConfigSecurityProtocol("sasl_plaintext")
		}
		kafkaInstances[mqConnName] = instance
		return instance, nil
	}
	return instance, nil

}

// GetKafka 获取kafka.
func GetKafka(mqConnName string) (*KafkaWorker, error) {
	instance, ok := kafkaInstances[mqConnName]
	if ok {
		return instance, nil
	}
	return nil, fmt.Errorf("Kafka instance by %s not found", mqConnName)
}

// ConvertKafkaPacketToMQConsumerMessage 把接收到的kafkaPacket 数据转换成MQConsumerMessage.
func ConvertKafkaPacketToMQConsumerMessage(packet *KafkaPacket) mqenv.MQConsumerMessage {
	consumerMessage := mqenv.MQConsumerMessage{
		Driver:        mqenv.DriverTypeKafka2,
		Queue:         packet.SendTo,
		CorrelationID: packet.CorrelationId,
		ConsumerTag:   "",
		ReplyTo:       packet.ReplyTo,
		MessageID:     packet.MessageId,
		AppID:         packet.AppId,
		UserID:        packet.UserId,
		ContentType:   packet.ContentType,
		Exchange:      "",
		RoutingKey:    "",
		Timestamp:     time.Unix(int64(packet.Timestamp), 0),
		Body:          packet.Body,
		BindData:      &packet,
	}

	return consumerMessage
}
