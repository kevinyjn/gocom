package mq

import (
	"fmt"
	"path"
	"sync"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/yamlutils"
)

// Exchange struct
type Exchange struct {
	Type    string `yaml:"type" json:"type"`
	Name    string `yaml:"name" json:"name"`
	Durable bool   `yaml:"durable" json:"durable"`
}

// Config struct
type Config struct {
	Instance string `yaml:"instance" json:"instance"`
	// RabbitMQ parameters
	Queue       string            `yaml:"queue" json:"queue"`
	Exchange    Exchange          `yaml:"exchange" json:"exchange"`
	BindingKey  string            `yaml:"bindingKey" json:"bindingKey"`
	RoutingKeys map[string]string `yaml:"routingKeys" json:"routingKeys"`
	Durable     bool              `yaml:"durable" json:"durable"`
	AutoDelete  bool              `yaml:"autoDelete" json:"autoDelete"`
	RPCEnabled  bool              `yaml:"rpcEnabled"`
	// Kafka parameters
	Topic             string `yaml:"topic" json:"topic"`
	GroupID           string `yaml:"groupId" json:"groupId"`
	Partition         int    `yaml:"partition" json:"partition"`
	MaxPollIntervalMS int    `yaml:"maxPollIntervalMs" json:"maxPollIntervalMs"`
	// 消息类型:
	//direct:组播,订阅同一个topic，消费者组会相同，一条消息只会被组内一个消费者接收
	//fanout:广播,订阅同一个topic，但是消费者组会使用uuid，所有组都会收到信息
	MessageType        string `yaml:"messageType" json:"messageType"`
	UseOriginalContent bool   `yaml:"useOriginalContent" json:"useOriginalContent"`
}

// RoutesEnv struct
type RoutesEnv struct {
	MQs map[string]Config `yaml:"mq"`
	m   sync.RWMutex      `yaml:"-"`
}

var mqRoutesEnv = RoutesEnv{
	MQs: map[string]Config{},
	m:   sync.RWMutex{},
}

// GetMQConfig config
func GetMQConfig(category string) *Config {
	mqRoutesEnv.m.RLock()
	cnf, ok := mqRoutesEnv.MQs[category]
	mqRoutesEnv.m.RUnlock()
	if ok {
		return &cnf
	}
	return nil
}

// SetMQConfig with category
func SetMQConfig(category string, cnf Config) {
	mqRoutesEnv.m.Lock()
	mqRoutesEnv.MQs[category] = cnf
	mqRoutesEnv.m.Unlock()
}

// GetMQRoutes config map
func GetMQRoutes() map[string]Config {
	results := map[string]Config{}
	mqRoutesEnv.m.RLock()
	for category, cnf := range mqRoutesEnv.MQs {
		results[category] = cnf
	}
	mqRoutesEnv.m.RUnlock()
	return results
}

// InitMQRoutesEnv initialize with configure file
func InitMQRoutesEnv(configFile string) (*RoutesEnv, error) {
	cfgLoaded := true
	cfgDir, cfgFile := path.Split(configFile)
	mqRoutesEnv.m.Lock()
	err := yamlutils.LoadConfig(configFile, &mqRoutesEnv)
	mqRoutesEnv.m.Unlock()
	if err != nil {
		cfgLoaded = false
	}
	mqRoutesEnv.m.Lock()
	err = yamlutils.LoadConfig(path.Join(cfgDir, "local."+cfgFile), &mqRoutesEnv)
	mqRoutesEnv.m.Unlock()
	if !cfgLoaded && err != nil {
		logger.Error.Println("Please check the mq configure file and restart.")
	} else {
		cfgLoaded = true
	}

	if cfgLoaded {
		return &mqRoutesEnv, nil
	}
	return nil, fmt.Errorf("Load mq routes config with config file:%s failed", configFile)
}
