package logger

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
)

// Constants
const (
	deleteFileOnExit = false

	AccessLogFileSize20M  int64 = 20 * 1024 * 1024
	AccessLogFileSize50M  int64 = 50 * 1024 * 1024
	AccessLogFileSize200M int64 = 200 * 1024 * 1024
)

func todayFilename() string {
	// today := time.Now().Format("20060102")
	// return "../log/access-" + today + ".log"
	return "../log/access.log"
}

func newAccessLogFile() *os.File {
	logPath := todayFilename()
	logDir, _ := path.Split(logPath)
	if logDir != "" {
		os.MkdirAll(logDir, 0776)
	}
	// 打开一个输出文件，如果重新启动服务器，它将追加到今天的文件中
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return f
}

// Variables
var (
	excludeExtensions = [...]string{
		".js",
		".css",
		".jpg",
		".png",
		".ico",
		".svg",
	}

	accessLogFile         *os.File = nil
	lastAccessLogStatTime time.Time
	AccessLogFileMaxBytes = AccessLogFileSize20M
)

// NewAccessLogger logger
func NewAccessLogger() (h iris.Handler, closer func() error) {
	c := logger.Config{
		Status:  true,
		IP:      true,
		Method:  true,
		Path:    true,
		Query:   true,
		Columns: false,
	}
	accessLogFile = newAccessLogFile()
	checkAccessLogRotate(time.Now())
	closer = func() error {
		err := accessLogFile.Close()
		if deleteFileOnExit {
			err = os.Remove(accessLogFile.Name())
		}
		return err
	}
	c.LogFunc = func(now time.Time, latency time.Duration, status, ip, method, path string, message interface{}, headerMessage interface{}) {
		// line := fmt.Sprintf("%s | %v | %4v | %s | %s | %s", now.Format("2006/01/02T15:04:05"), status, latency, ip, method, path)
		// output := columnize.SimpleFormat([]string{line}) + "\n"
		output := fmt.Sprintf("%s %v %4v %s %s %s\n", now.Format("2006/01/02T15:04:05"), status, latency, ip, method, path)
		//output := logger.Columnize(now.Format("2006/01/02T15:04:05"), latency, status, ip, method, path, message, headerMessage)
		checkAccessLogRotate(now)
		if nil != accessLogFile {
			accessLogFile.WriteString(output)
		}
	}
	//我们不想使用记录器，一些静态请求等
	c.AddSkipper(func(ctx iris.Context) bool {
		path := ctx.Path()
		for _, ext := range excludeExtensions {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	})
	h = logger.New(c)
	return h, closer
}

func checkAccessLogRotate(now time.Time) {
	if AccessLogFileMaxBytes > 0 && now.Sub(lastAccessLogStatTime).Seconds() > 300 {
		lastAccessLogStatTime = now
		stat, err := accessLogFile.Stat()
		if nil == err {
			if stat.Size() > AccessLogFileMaxBytes {
				// rotate log
				accessLogFile, err = archiveAndNewLogFile(accessLogFile)
			}
		}
	}
}

func archiveAndNewLogFile(lf *os.File) (*os.File, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	fbasename := path.Base(lf.Name())
	fname := lf.Name()
	f, err := w.Create(fbasename)
	for {
		if nil != err {
			w.Close()
			Error.Printf("archive access log file while create zip element file:%s failed with error:%v", fbasename, err)
			break
		}

		lf.Close()
		fbuf, err := ioutil.ReadFile(fname)
		if nil == err {
			_, err = f.Write(fbuf)
			if nil != err {
				Error.Printf("archive access log file:%s.gz while write zip buffer failed with error:%v", fname, err)
			}
		} else {
			Error.Printf("archive access log file:%s.gz while read origin file %s failed with error:%v", fname, fname, err)
		}
		if w.Close() == nil {
			zf, err := os.OpenFile(fname+".gz", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			if nil == err {
				_, err = buf.WriteTo(zf)
				if nil != err {
					Error.Printf("archive access log file:%s.gz while create file failed with error:%v", fname, err)
				}
				zf.Close()
			} else {
				Error.Printf("archive access log file:%s.gz while create file failed with error:%v", fname, err)
			}
		}
		break
	}
	os.Remove(fname)
	newFile, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if nil != err {
		Error.Printf("archived access log file:%s.gz and recreate access log file:%s failed with error:%v", fname, fname, err)
	}

	return newFile, err
}
