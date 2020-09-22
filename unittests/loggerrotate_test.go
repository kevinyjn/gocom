package unittests

import (
	"fmt"
	"testing"
	"time"

	"github.com/kevinyjn/gocom/logger"
)

func TestLoggerRotate(t *testing.T) {
	logger.LogRotatorCrontab = "*/5 * * * * ?"
	err := logger.Init(&logger.Logger{
		Level:   "DEBUG",
		Type:    "filelog",
		Address: "../log/app.log",
	})
	if nil != err {
		t.Errorf("Test logger init failed with error:%v", err)
	} else {
		fmt.Printf("Succeed.\n")
	}
	// to manually test timer, open the bellow flag
	if false {
		timer := time.NewTimer(time.Second * 6)
		select {
		case <-timer.C:
			timer.Stop()
			break
		}
	}
}
