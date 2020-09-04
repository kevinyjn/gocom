package caching

import (
	"time"

	"github.com/kevinyjn/gocom/caching/cachingenv"
	"github.com/kevinyjn/gocom/caching/memcacheadapter"
	"github.com/kevinyjn/gocom/caching/redisadapter"
	"github.com/kevinyjn/gocom/logger"
)

// CacheProxySession cache proxy session interface
type CacheProxySession interface {
	GetName() string
	GetConnectionString() string
	Initialized() bool
	Get(key string) ([]byte, error)
	Set(key string, value []byte, expire time.Duration) bool
}

var cachesessions = make(map[string]CacheProxySession)

// InitCacheProxy initialize cache proxy using configuration
func InitCacheProxy(cachesConfig map[string]cachingenv.CacheConnectorConfig) bool {
	for name, cnf := range cachesConfig {
		cs, ok := cachesessions[name]
		if ok && cs.Initialized() {
			continue
		}
		if cachingenv.CachingDriverRedis == cnf.Driver {
			rc, err := redisadapter.NewRedisSession(name, cnf)
			if err != nil {
				logger.Error.Printf("connect %s %s failed with error:%v", cnf.Driver, rc.GetConnectionString(), err)
			}
			if rc != nil {
				cachesessions[name] = rc
			}
		} else if cachingenv.CachingDriverMemcache == cnf.Driver {
			rc, err := memcacheadapter.NewMemcacheSession(name, cnf)
			if err != nil {
				logger.Error.Printf("connect %s %s failed with error:%v", cnf.Driver, rc.GetConnectionString(), err)
			}
			if rc != nil {
				cachesessions[name] = rc
			}
		}
	}
	return true
}

// GetCacher get cacher session by category name
func GetCacher(name string) CacheProxySession {
	return cachesessions[name]
}

// GetAllCachers get all cacher sessions
func GetAllCachers() map[string]CacheProxySession {
	return cachesessions
}
