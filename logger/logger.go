package logger

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinyjn/gocom/logger/fluentdlogger"
)

// Constant
const (
	RecordingTypeFilelog = "filelog"
	RecordingTypeEFK     = "efk"
)

// Variables
var (
	Trace   *log.Logger = log.New(os.Stdout, "[TRACE] ", log.Ldate|log.Ltime|log.Lshortfile)
	Info    *log.Logger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning *log.Logger = log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	Error   *log.Logger = log.New(io.MultiWriter(os.Stdout, os.Stderr), "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	Fatal   *log.Logger = log.New(io.MultiWriter(os.Stdout, os.Stderr), "[FATAL] ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Init initializer
func Init(loggerConfig *Logger) {
	if RecordingTypeFilelog == loggerConfig.Type {
		initFilelog(loggerConfig.Address, loggerConfig.Level)
	} else if RecordingTypeEFK == loggerConfig.Type {
		initEfkLogger(loggerConfig.Address, loggerConfig.Level)
	}
}

// IsDebugEnabled boolean
func IsDebugEnabled() bool {
	if ioutil.Discard == Trace.Writer() {
		return false
	}
	return true
}

func convertLogLevel(logLevel string) int {
	logLevel = strings.ToUpper(logLevel)
	actLogLevel := 1
	if strings.ToUpper(logLevel) == "INFO" {
		actLogLevel = 2
	} else if strings.ToUpper(logLevel) == "WARN" {
		actLogLevel = 3
	} else if strings.ToUpper(logLevel) == "ERROR" {
		actLogLevel = 4
	} else if strings.ToUpper(logLevel) == "FATAL" {
		actLogLevel = 5
	}
	return actLogLevel
}

func selectIobufferByLevel(file *os.File, level int, limitLevel int) io.Writer {
	if level < limitLevel {
		return ioutil.Discard
	} else if level < 5 {
		if file != nil {
			return file
		}
		return os.Stdout
	} else {
		if file != nil {
			return io.MultiWriter(file, os.Stderr)
		}
		return io.MultiWriter(os.Stdout, os.Stderr)
	}
}

func initFilelog(logPath string, logLevel string) {
	curPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		curPath = ""
	}
	if logPath == "" {
		_, fName := path.Split(os.Args[0])
		fSlices := strings.Split(fName, ".")
		logPath = "../log/" + fSlices[0] + ".log"
	}
	if strings.HasPrefix(logPath, ".") {
		logPath = path.Join(curPath, logPath)
	}

	logDir, _ := path.Split(logPath)
	if logDir != "" {
		os.MkdirAll(logDir, 0776)
	}

	actLogLevel := convertLogLevel(logLevel)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Open logger file:%s failed with error:%s", logPath, err.Error())
	}

	loggerFlag := log.Ldate | log.Ltime
	if actLogLevel < 2 {
		loggerFlag += log.Lshortfile
	}

	Trace = log.New(selectIobufferByLevel(file, 1, actLogLevel), "[TRACE] ", loggerFlag)
	Info = log.New(selectIobufferByLevel(file, 2, actLogLevel), "[INFO] ", loggerFlag)
	Warning = log.New(selectIobufferByLevel(file, 3, actLogLevel), "[WARN] ", loggerFlag)
	Error = log.New(selectIobufferByLevel(file, 4, actLogLevel), "[ERROR] ", loggerFlag)
	Fatal = log.New(selectIobufferByLevel(file, 5, actLogLevel), "[FATAL] ", loggerFlag)
}

func initEfkLogger(logPath string, logLevel string) {
	hosts := strings.Split(logPath, ":")
	port := 80
	actLogLevel := convertLogLevel(logLevel)
	if len(hosts) > 1 {
		port, _ = strconv.Atoi(hosts[1])
	}
	fluentdlogger.InitFluentdLogger(hosts[0], port, actLogLevel)
}
