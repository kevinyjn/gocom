package logger

// Logger struct
type Logger struct {
	Level      string `yaml:"level"`
	Type       string `yaml:"type"`
	Address    string `yaml:"address"`
	RemoteType string `yaml:"udp"`
}
