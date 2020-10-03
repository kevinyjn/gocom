package logger

import (
	"fmt"

	"github.com/fluent/fluent-logger-golang/fluent"
)

// InitFluentdLogger logger
func InitFluentdLogger(host string, port int, level LogLevel) {
	cfg := fluent.Config{
		FluentPort: port,
		FluentHost: host,
	}
	f, err := fluent.New(cfg)
	if nil != err {
		fmt.Println(err)
		return
	}
	_ = f
}
