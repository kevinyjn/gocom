package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron"
)

// LogLevel type
type LogLevel int

// Constant
const (
	RecordingTypeFilelog = "filelog"
	RecordingTypeEFK     = "efk"

	LogRotatorCronDaily   = "0 0 0 * * ?"
	LogRotatorCronWeekly  = "0 0 0 ? * 1"
	LogRotatorCronMonthly = "0 0 0 1 * ?"

	LogRotatorExpiresMonthly = 30
	LogRotatorExpiresWeekly  = 7
	LogRotatorExpiresSeason  = 90
	LogRotatorExpiresYearly  = 365
)

// LogLevels
const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

// Variables
var (
	Trace                 *log.Logger = log.New(os.Stdout, "[TRACE] ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug                 *log.Logger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	Info                  *log.Logger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning               *log.Logger = log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	Error                 *log.Logger = log.New(io.MultiWriter(os.Stdout, os.Stderr), "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	Fatal                 *log.Logger = log.New(io.MultiWriter(os.Stdout, os.Stderr), "[FATAL] ", log.Ldate|log.Ltime|log.Lshortfile)
	LogRotatorCrontab     string      = LogRotatorCronDaily
	LogRotatorExpiresDays int         = LogRotatorExpiresMonthly
	baseLogFileName       string      = ""
	rotatorTimer          *cron.Cron  = nil
	originLogFile         *os.File    = nil
	Level                 LogLevel    = LogLevelDebug
)

// Init initializer
func Init(loggerConfig *Logger) error {
	if RecordingTypeFilelog == loggerConfig.Type {
		return initFilelog(loggerConfig.Address, loggerConfig.Level)
	} else if RecordingTypeEFK == loggerConfig.Type {
		return initEfkLogger(loggerConfig.Address, loggerConfig.Level)
	}
	return nil
}

// IsDebugEnabled boolean
func IsDebugEnabled() bool {
	return Level <= LogLevelDebug
}

func convertLogLevel(logLevel string) LogLevel {
	logLevel = strings.ToUpper(logLevel)
	actLogLevel := LogLevelDebug
	switch strings.ToUpper(logLevel) {
	case "TRACE":
		actLogLevel = LogLevelTrace
		break
	case "DEBUG":
		actLogLevel = LogLevelDebug
		break
	case "INFO":
		actLogLevel = LogLevelInfo
		break
	case "WARN":
		actLogLevel = LogLevelWarning
		break
	case "ERROR":
		actLogLevel = LogLevelError
		break
	case "FATAL":
		actLogLevel = LogLevelFatal
		break
	}
	return actLogLevel
}

func selectIobufferByLevel(file *os.File, level LogLevel, limitLevel LogLevel) io.Writer {
	if level < limitLevel {
		return ioutil.Discard
	} else if level < LogLevelFatal {
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

func initFilelog(logPath string, logLevel string) error {
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
	Level = actLogLevel

	baseLogFileName = logPath
	file, _, err := generateLogFile(logPath)
	if err != nil {
		log.Fatalf("Open logger file:%s failed with error:%v", logPath, err)
		return err
	}

	loggerFlag := log.Ldate | log.Ltime
	if actLogLevel < LogLevelWarning {
		loggerFlag += log.Lshortfile
	}

	Trace = log.New(selectIobufferByLevel(file, LogLevelTrace, actLogLevel), "[TRACE] ", loggerFlag)
	Debug = log.New(selectIobufferByLevel(file, LogLevelDebug, actLogLevel), "[DEBUG] ", loggerFlag)
	Info = log.New(selectIobufferByLevel(file, LogLevelWarning, actLogLevel), "[INFO] ", loggerFlag)
	Warning = log.New(selectIobufferByLevel(file, LogLevelWarning, actLogLevel), "[WARN] ", loggerFlag)
	Error = log.New(selectIobufferByLevel(file, LogLevelError, actLogLevel), "[ERROR] ", loggerFlag)
	Fatal = log.New(selectIobufferByLevel(file, LogLevelFatal, actLogLevel), "[FATAL] ", loggerFlag)

	Info.Printf("logger initialized.")
	if nil == rotatorTimer {
		rotatorTimer = cron.New()
		err = rotatorTimer.AddFunc(LogRotatorCrontab, logRotator)
		if nil != err {
			Error.Printf("add log rotator timer failed with error:%v", err)
		} else {
			rotatorTimer.Start()
		}
	}
	if 0 < LogRotatorExpiresDays {
		cleanExpiredLogFiles(baseLogFileName)
	}
	return nil
}

func initEfkLogger(logPath string, logLevel string) error {
	hosts := strings.Split(logPath, ":")
	port := 80
	actLogLevel := convertLogLevel(logLevel)
	if len(hosts) > 1 {
		port, _ = strconv.Atoi(hosts[1])
	}
	InitFluentdLogger(hosts[0], port, actLogLevel)
	return nil
}

func logRotator() {
	file, endfix, err := generateLogFile(baseLogFileName)
	if nil != err {
		Error.Printf("rotate log file:%s failed with error:%v", baseLogFileName, err)
		return
	}
	if file == originLogFile {
		return
	}
	Trace.SetOutput(file)
	Debug.SetOutput(file)
	Info.SetOutput(file)
	Warning.SetOutput(file)
	Error.SetOutput(file)
	Fatal.SetOutput(file)
	if nil != originLogFile {
		originLogFile.Close()
	}
	originLogFile = file
	Info.Printf("log rotated at %s", endfix)

	if 0 < LogRotatorExpiresDays {
		cleanExpiredLogFiles(baseLogFileName)
	}
}

func generateLogFile(logPath string) (*os.File, string, error) {
	ext := path.Ext(logPath)
	now := time.Now()
	var endfix string
	logBaseName := strings.Split(logPath, ext)[0]
	switch LogRotatorCrontab {
	case LogRotatorCronDaily:
		endfix = now.Format("2006-01-02")
		break
	case LogRotatorCronMonthly:
		endfix = now.Format("2006-01")
		break
	case LogRotatorCronWeekly:
		endfix = now.Format("2006-01-02")
		break
	default:
		endfix = now.Format("2006-01-02-150405")
		break
	}
	logFileName := fmt.Sprintf("%s-%s%s", logBaseName, endfix, ext)
	if nil != originLogFile {
		if originLogFile.Name() == logFileName {
			return originLogFile, endfix, nil
		}
	}
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		Fatal.Fatalf("Open logger file:%s failed with error:%v", logFileName, err)
	}
	return file, endfix, err
}

func cleanExpiredLogFiles(logPath string) {
	logdir := path.Dir(logPath)
	fbasename := path.Base(logPath)
	fbasename = strings.Split(fbasename, path.Ext(fbasename))[0]
	datereg := regexp.MustCompile("-(\\d{4}-\\d{2}(?:-\\d{2})?)(?:-\\d+)?$")
	r := datereg.FindAllStringSubmatch(fbasename, -1)
	if len(r) > 0 {
		fbasename = strings.Split(fbasename, r[0][0])[0]
	}
	files, err := ioutil.ReadDir(logdir)
	if nil == err {
		expiresDate := time.Now().Add(-(time.Hour * 24 * time.Duration(LogRotatorExpiresDays)))
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			fname := path.Base(f.Name())
			fname = strings.Split(fname, path.Ext(fname))[0]
			r = datereg.FindAllStringSubmatch(fname, -1)
			if len(r) > 0 {
				datestr := r[0][1]
				if len(datestr) < 8 {
					datestr = datestr + "-01"
				}
				createDate, err := time.Parse("2006-01-02", datestr)
				if nil == err && createDate.Before(expiresDate) {
					delFileName := path.Join(logdir, f.Name())
					Info.Printf("cleaning expired log file:%s", delFileName)
					os.Remove(delFileName)
				}
			}
		}
	}
}
