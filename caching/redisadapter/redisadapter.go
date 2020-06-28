package redisadapter

import (
	"fmt"
	"time"

	"github.com/kevinyjn/gocom/caching/cachingenv"
	"github.com/kevinyjn/gocom/logger"

	"github.com/go-redis/redis"
)

// Constants
const (
	RedisPingInterval = 5
)

// RedisCacheSession instance
type RedisCacheSession struct {
	Client redis.UniversalClient
	Name   string
	Addr   string
	ticker *time.Ticker
}

// NewRedisSession new session
func NewRedisSession(name string, conf cachingenv.CacheConnectorConfig) (*RedisCacheSession, error) {
	redisAddr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	s := &RedisCacheSession{
		Name: name,
		Addr: fmt.Sprintf("%s/%d", redisAddr, conf.Index),
	}
	if conf.ClusterMode {
		s.Client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:     []string{redisAddr},
			Password:  conf.Password,
			OnConnect: s.OnConnected,
		})
	} else {
		s.Client = redis.NewClient(&redis.Options{
			Addr:      redisAddr,
			Password:  conf.Password,
			DB:        conf.Index,
			OnConnect: s.OnConnected,
		})
	}

	_, err := s.Client.Ping().Result()
	if err != nil {
		logger.Error.Printf("connect redis %s %s failed with error:%s", name, redisAddr, err.Error())
		go s.StartKeepalive()
		return s, err
	}
	logger.Info.Printf("initialized redis connection %s %s on db:%d\n", name, redisAddr, conf.Index)
	return s, nil
}

// GetName getter
func (s *RedisCacheSession) GetName() string {
	return s.Name
}

// GetConnectionString getter
func (s *RedisCacheSession) GetConnectionString() string {
	return s.Addr
}

// Initialized getter
func (s *RedisCacheSession) Initialized() bool {
	return s.Client != nil
}

// OnConnected event
func (s *RedisCacheSession) OnConnected(*redis.Conn) error {
	logger.Info.Printf("redis %s connection:%s connected.", s.GetName(), s.GetConnectionString())
	if s.ticker == nil {
		go s.StartKeepalive()
	}
	return nil
}

// StartKeepalive keepalive
func (s *RedisCacheSession) StartKeepalive() {
	if s.ticker != nil {
		return
	}
	s.ticker = time.NewTicker(time.Second * RedisPingInterval)
	for {
		select {
		case <-s.ticker.C:
			_, err := s.Client.Ping().Result()
			if err != nil {
				logger.Error.Printf("ping redis %s %s failed error:%s", s.GetName(), s.GetConnectionString(), err)
			} else {
				// logger.Trace.Printf("ping redis %s %s %s", s.GetName(), s.GetConnectionString(), txt)
			}
		}
	}
}

// Get value by key
func (s *RedisCacheSession) Get(key string) ([]byte, error) {
	val, err := s.Client.Get(key).Bytes()
	if nil != err {
		logger.Warning.Printf("get %s failed with error:%s", key, err.Error())
		return nil, err
	}
	return val, err
}

// Set value by key
func (s *RedisCacheSession) Set(key string, value []byte, expire time.Duration) bool {
	err := s.Client.Set(key, value, expire).Err()
	if nil != err {
		logger.Error.Printf("set %s failed with error:%s", key, err.Error())
		return false
	}
	return true
}
