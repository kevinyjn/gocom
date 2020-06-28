package cachingenv

// Constants
const (
	CachingDriverRedis    = "redis"
	CachingDriverMemcache = "memcache"
)

// CacheConnectorConfig connector config
type CacheConnectorConfig struct {
	Driver      string `yaml:"driver"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Password    string `yaml:"password"`
	Index       int    `yaml:"db"`
	ClusterMode bool   `yaml:"clusterMode"`
}
