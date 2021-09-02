package kafka

import (
	k "github.com/segmentio/kafka-go"
)

// Base .
type Base struct {
	Partition          int                                   // partition 分区
	Config             map[string]interface{}                // kafka 的配置字典
	CompletionCallback func(messages []k.Message, err error) // 发送状态通知函数
}

// ConfigServers 配置连接的服务器,如"localhost:9092,localhost:9093".
func (b *Base) ConfigServers(servers string) {
	b.Config["bootstrap.servers"] = servers
}

// ConfigPartition 配置分区，如partition 为0.
func (b *Base) ConfigPartition(partition int) {
	b.Partition = partition
}

// ConfigKerberosServiceName 使用kerberos 认证需要配置.
func (b *Base) ConfigKerberosServiceName(name string) {
	b.Config["kerberos.service.name"] = name
}

// ConfigKerberosKeyTab 使用kerberos 认证需要配置.
func (b *Base) ConfigKerberosKeyTab(kerberosKeyTab string) {
	b.Config["kerberos.keytab"] = kerberosKeyTab
}

// ConfigKerberosPrincipal 使用kerberos 认证需要配置.
func (b *Base) ConfigKerberosPrincipal(kerberosPrincipal string) {
	b.Config["kerberos.principal"] = kerberosPrincipal
}

// ConfigSecurityProtocol 使用plain 和kerberos 认证需要配置,如sasl_plaintext.
func (b *Base) ConfigSecurityProtocol(securityProtocol string) {
	b.Config["security.protocol"] = securityProtocol
}

// ConfigSaslMechanisms 使用plain 认证需要配置,可以使用PLAIN.
func (b *Base) ConfigSaslMechanisms(saslMechanisms string) {
	b.Config["sasl.mechanisms"] = saslMechanisms

}

// ConfigSaslUserName 使用plain 认证需要配置.
func (b *Base) ConfigSaslUserName(saslUsername string) {
	b.Config["sasl.username"] = saslUsername
}

// ConfigSaslPassword 使用plain 认证需要配置.
func (b *Base) ConfigSaslPassword(saslPassword string) {
	b.Config["sasl.password"] = saslPassword
}

// ConfigReconnectInterval 配置断线重连的时间间隔，单位是毫秒.
func (b *Base) ConfigReconnectInterval(interval int) {
	b.Config["reconnect.backoff.ms"] = interval
}

// ConfigSessionTimeout 配置会话超时.
func (b *Base) ConfigSessionTimeout(timeout int) {
	b.Config["session.timeout.ms"] = timeout
}

// ConfigHeartbeatInterval 配置心跳检测间隔.
func (b *Base) ConfigHeartbeatInterval(interval int) {
	b.Config["heartbeat.interval.ms"] = interval
}

// SetCompletionCallback 消息发送状态通知回调
func (b *Base) SetCompletionCallback(callback func(messages []k.Message, err error)) {
	b.CompletionCallback = callback
}
