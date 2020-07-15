package config

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kevinyjn/gocom/caching/cachingenv"
	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/yamlutils"
)

// DBRestfulConfig config block
type DBRestfulConfig struct {
	API      string                 `yaml:"api"`
	Enabled  bool                   `yaml:"enabled"`
	TLS      definations.TLSOptions `yaml:"tls"`
	UserName string                 `yaml:"username"`
	Password string                 `yaml:"password"`
}

// Server config block
type Server struct {
	Host       string                 `yaml:"host"`
	Port       int                    `yaml:"port"`
	Scheme     string                 `yaml:"scheme"`
	WhiteList  []string               `yaml:"whiteList"`
	TLS        definations.TLSOptions `yaml:"tls"`
	DeployAddr string                 `yaml:"depoyAddr"`
	Dev        bool                   `yaml:"dev"`
	DevAddr    string                 `yaml:"devAddr"`
}

// DefaultACL config block
type DefaultACL struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	AppKey string `yaml:"appKey"`
}

// RestfulLoader config block
type RestfulLoader struct {
	Enabled bool                   `yaml:"enabled"`
	Address string                 `yaml:"address"`
	AppIds  []string               `yaml:"appId"`
	TLS     definations.TLSOptions `yaml:"tls"`
}

// HealthzChecks config block
type HealthzChecks struct {
	Name      string   `yaml:"name"`
	Type      string   `yaml:"type"`
	Endpoints []string `yaml:"endpoints"`
}

// Env config block
type Env struct {
	MQs           map[string]mqenv.MQConnectorConfig         `yaml:"mq"`
	Caches        map[string]cachingenv.CacheConnectorConfig `yaml:"cache"`
	DBs           map[string]definations.DBConnectorConfig   `yaml:"db"`
	DBRestfuls    map[string]DBRestfulConfig                 `yaml:"restful"`
	Logger        logger.Logger                              `yaml:"logger"`
	Server        Server                                     `yaml:"server"`
	DefaultAcls   []DefaultACL                               `yaml:"defaultAcl"`
	RestfulLoader RestfulLoader                              `yaml:"restLoader"`
	Proxies       *definations.Proxies                       `yaml:"proxies"`
	HealthzChecks []HealthzChecks                            `yaml:"healthzChecks"`
}

var env = Env{}

// GetEnv getter
func GetEnv() *Env {
	return &env
}

// Init initializer
func Init(filePath string) (*Env, error) {
	cfgLoaded := true
	cfgDir, cfgFile := path.Split(filePath)
	err := yamlutils.LoadConfig(filePath, &env)
	if err != nil {
		cfgLoaded = false
	}
	err = yamlutils.LoadConfig(path.Join(cfgDir, "local."+cfgFile), &env)
	if !cfgLoaded && err != nil {
		log.Println("Please check the configure file and restart.")
		return nil, err
	}

	curPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return &env, nil
	}
	chkPaths := [][]*string{
		{&env.RestfulLoader.TLS.CaFile, &env.RestfulLoader.TLS.CertFile, &env.RestfulLoader.TLS.KeyFile},
	}
	for _, svr := range []Server{env.Server} {
		chkPaths = append(chkPaths, []*string{&svr.TLS.CertFile, &svr.TLS.CaFile, &svr.TLS.KeyFile})
	}
	for _, subChkPath := range chkPaths {
		for _, chkPath := range subChkPath {
			if *chkPath != "" && strings.HasPrefix(*chkPath, ".") {
				// _d, _f := path.Split(*chkPath)
				*chkPath = path.Join(curPath, *chkPath)
			}
		}
	}
	return &env, nil
}

// FormatCallbackURL formatter
func (n Server) FormatCallbackURL(uri string) string {
	if n.Dev && n.DevAddr != "" {
		return n.DevAddr + uri
	}
	return n.DeployAddr + uri
}
