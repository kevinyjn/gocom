package rdbms

import (
	"github.com/kevinyjn/gocom/logger"
	"xorm.io/xorm/log"
)

type ormLogger struct {
	logLevel log.LogLevel
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

func (l *ormLogger) Level() log.LogLevel {
	return l.logLevel
}
func (l *ormLogger) SetLevel(level log.LogLevel) {
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

func getSysLogLevel() log.LogLevel {
	switch logger.Level {
	case logger.LogLevelDebug, logger.LogLevelTrace:
		return log.LOG_INFO
	case logger.LogLevelInfo:
		return log.LOG_INFO
	case logger.LogLevelWarning:
		return log.LOG_WARNING
	case logger.LogLevelError, logger.LogLevelFatal:
		return log.LOG_ERR
	}
	return log.LOG_INFO
}
