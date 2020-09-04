package mq

import (
	"fmt"
	"path"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/yamlutils"
)

// Exchange struct
type Exchange struct {
	Type    string `yaml:"type"`
	Name    string `yaml:"name"`
	Durable bool   `yaml:"durable"`
}

// Config struct
type Config struct {
	Instance string `yaml:"instance"`
	// RabbitMQ parameters
	Queue       string            `yaml:"queue"`
	Exchange    Exchange          `yaml:"exchange"`
	BindingKey  string            `yaml:"bindingKey"`
	RoutingKeys map[string]string `yaml:"routingKeys"`
	Durable     bool              `yaml:"durable"`
	RPCEnabled  bool              `yaml:"rpcEnabled"`
	// Kafka parameters
	Topic   string `yaml:"topic"`
	GroupID string `yaml:"groupId"`
}

// RoutesEnv struct
type RoutesEnv struct {
	MQs map[string]Config `yaml:"mq"`
}

var mqRoutesEnv = RoutesEnv{
	MQs: map[string]Config{},
}

// GetMQConfig config
func GetMQConfig(category string) *Config {
	cnf, ok := mqRoutesEnv.MQs[category]
	if ok {
		return &cnf
	}
	return nil
}

// GetMQRoutes config map
func GetMQRoutes() map[string]Config {
	return mqRoutesEnv.MQs
}

// InitMQRoutesEnv initialize with configure file
func InitMQRoutesEnv(configFile string) (*RoutesEnv, error) {
	cfgLoaded := true
	cfgDir, cfgFile := path.Split(configFile)
	err := yamlutils.LoadConfig(configFile, &mqRoutesEnv)
	if err != nil {
		cfgLoaded = false
	}
	err = yamlutils.LoadConfig(path.Join(cfgDir, "local."+cfgFile), &mqRoutesEnv)
	if !cfgLoaded && err != nil {
		logger.Error.Println("Please check the ess configure file and restart.")
	} else {
		cfgLoaded = true
	}

	if cfgLoaded {
		return &mqRoutesEnv, nil
	}
	return nil, fmt.Errorf("Load mq routes config with config file:%s failed", configFile)
}
