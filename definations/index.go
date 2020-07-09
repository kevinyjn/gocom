package definations

import (
	"strings"
)

// TLSOptions options about TLS
type TLSOptions struct {
	Enabled      bool   `yaml:"enabled"`
	CertFile     string `yaml:"certFile"`
	KeyFile      string `yaml:"keyFile"`
	CaFile       string `yaml:"caFile"`
	SkipVerify   bool   `yaml:"skipVerify"`
	VerifyClient bool   `yaml:"verifyClient"`
}

// Proxies options about http proxy
type Proxies struct {
	HTTP  string `yaml:"http"`
	HTTPS string `yaml:"https"`
}

// Valid check if proxies configuration is valid
func (n *Proxies) Valid() bool {
	return n.HTTP != "" || n.HTTPS != ""
}

// FetchProxyURL fetch proxy url
func (n *Proxies) FetchProxyURL(endpointURL string) string {
	if strings.HasPrefix(endpointURL, "https") {
		return n.HTTPS
	} else {
		return n.HTTP
	}
}

// DBConnectorConfig db connector configuration
type DBConnectorConfig struct {
	Driver          string `yaml:"driver"`
	Address         string `yaml:"address"`
	Db              string `yaml:"db"`
	Mechanism       string `yaml:"mechanism"`
	TableNamePrefix string `yaml:"tableNamePrefix"`
}
