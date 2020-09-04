package memcacheadapter

import (
	"fmt"
	"strings"
	"time"

	"github.com/kevinyjn/gocom/caching/cachingenv"
	"github.com/kevinyjn/gocom/logger"

	"github.com/bradfitz/gomemcache/memcache"
)

// Constant
const (
	MemCachePingInterval = 5
)

// MemCacheSession session struct
type MemCacheSession struct {
	Client *memcache.Client
	Name   string
	Addr   string
	ticker *time.Ticker
}

// NewMemcacheSession new session struct
func NewMemcacheSession(name string, conf cachingenv.CacheConnectorConfig) (*MemCacheSession, error) {
	memcacheAddrs := []string{}
	addrs := strings.Split(conf.Host, ",")
	for _, addr := range addrs {
		if !strings.Contains(addr, ":") {
			addr = fmt.Sprintf("%s:%d", addr, conf.Port)
		}
		memcacheAddrs = append(memcacheAddrs, addr)
	}
	s := &MemCacheSession{
		Name: name,
		Addr: strings.Join(memcacheAddrs, ","),
	}
	s.Client = memcache.New(memcacheAddrs...)

	err := s.Client.Ping()
	if err != nil {
		logger.Error.Printf("connect memcache %s %s failed with error:%v", name, s.Addr, err)
		go s.StartKeepalive()
		return s, err
	}
	logger.Info.Printf("initialized memcache connection %s %s", name, s.Addr)
	return s, nil
}

// GetName getter
func (s *MemCacheSession) GetName() string {
	return s.Name
}

// GetConnectionString getter
func (s *MemCacheSession) GetConnectionString() string {
	return s.Addr
}

// Initialized if initialized
func (s *MemCacheSession) Initialized() bool {
	return s.Client != nil
}

// StartKeepalive start keepalive
func (s *MemCacheSession) StartKeepalive() {
	if s.ticker != nil {
		return
	}
	s.ticker = time.NewTicker(time.Second * MemCachePingInterval)
	for {
		select {
		case <-s.ticker.C:
			err := s.Client.Ping()
			if err != nil {
				logger.Error.Printf("ping memcache %s %s failed error:%s", s.GetName(), s.GetConnectionString(), err)
			} else {
				// logger.Trace.Printf("ping memcache %s %s %s", s.GetName(), s.GetConnectionString(), txt)
			}
		}
	}
}

// Get value by key
func (s *MemCacheSession) Get(key string) ([]byte, error) {
	val, err := s.Client.Get(key)
	if err != nil {
		logger.Warning.Printf("get %s failed with error:%v", key, err)
		return nil, err
	}
	return val.Value, err
}

// Set value by key
func (s *MemCacheSession) Set(key string, value []byte, expire time.Duration) bool {
	err := s.Client.Set(&memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: int32(expire.Seconds()),
	})
	if err != nil {
		logger.Error.Printf("set %s failed with error:%v", key, err)
		return false
	}
	return true
}
