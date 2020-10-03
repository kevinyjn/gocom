package rdbms

import (
	"github.com/kevinyjn/gocom/logger"
	"xorm.io/core"
)

type ormLogger struct {
	logLevel core.LogLevel
	showSQL  bool
}

func (l *ormLogger) Debug(v ...interface{}) {
	// logger.Debug.Print(v...)
}

func (l *ormLogger) Debugf(format string, v ...interface{}) {
	// logger.Debug.Printf(format, v...)
}
func (l *ormLogger) Error(v ...interface{}) {
	logger.Error.Print(v...)
}
func (l *ormLogger) Errorf(format string, v ...interface{}) {
	logger.Error.Printf(format, v...)
}
func (l *ormLogger) Info(v ...interface{}) {
	logger.Info.Print(v...)
}
func (l *ormLogger) Infof(format string, v ...interface{}) {
	logger.Info.Printf(format, v...)
}
func (l *ormLogger) Warn(v ...interface{}) {
	logger.Warning.Print(v...)
}
func (l *ormLogger) Warnf(format string, v ...interface{}) {
	logger.Warning.Printf(format, v...)
}

func (l *ormLogger) Level() core.LogLevel {
	return l.logLevel
}
func (l *ormLogger) SetLevel(level core.LogLevel) {
	l.logLevel = level
}

func (l *ormLogger) ShowSQL(show ...bool) {
	if nil != show && len(show) > 0 {
		l.showSQL = show[0]
	}
	l.showSQL = true
}
func (l *ormLogger) IsShowSQL() bool {
	return l.showSQL
}

func getSysLogLevel() core.LogLevel {
	switch logger.Level {
	case logger.LogLevelDebug, logger.LogLevelTrace:
		return core.LOG_INFO
	case logger.LogLevelInfo:
		return core.LOG_INFO
	case logger.LogLevelWarning:
		return core.LOG_WARNING
	case logger.LogLevelError, logger.LogLevelFatal:
		return core.LOG_ERR
	}
	return core.LOG_INFO
}
