// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pulsar

import (
	"fmt"
	"strings"

	pulsarlog "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/kevinyjn/gocom/logger"
)

// pulsarLogWrapper implements Logger interface
// based on underlying pulsarLog.FieldLogger
type pulsarLogWrapper struct {
	enableDebugLogger bool
}

type pulsarLogEntry struct {
	fields            pulsarlog.Fields
	enableDebugLogger bool
}

// NewLogger creates a new logger which wraps
func NewLogger(enableDebugLogger bool) pulsarlog.Logger {
	return &pulsarLogWrapper{
		enableDebugLogger: enableDebugLogger,
	}
}

func (l *pulsarLogWrapper) SubLogger(fs pulsarlog.Fields) pulsarlog.Logger {
	sl := NewLogger(l.enableDebugLogger)
	sl.WithFields(fs)
	return sl
}

func (l *pulsarLogWrapper) WithFields(fs pulsarlog.Fields) pulsarlog.Entry {
	return l.entry().WithFields(fs)
}

func (l *pulsarLogWrapper) WithField(name string, value interface{}) pulsarlog.Entry {
	return l.entry().WithField(name, value)
}

func (l *pulsarLogWrapper) WithError(err error) pulsarlog.Entry {
	return l.entry().WithField("error", err)
}

func (l *pulsarLogWrapper) entry() pulsarlog.Entry {
	e := &pulsarLogEntry{
		fields:            pulsarlog.Fields{},
		enableDebugLogger: l.enableDebugLogger,
	}
	return e
}

func (l *pulsarLogWrapper) Debug(args ...interface{}) {
	if l.enableDebugLogger {
		logger.Debug.Output(3, fmt.Sprint(args...))
	}
}

func (l *pulsarLogWrapper) Info(args ...interface{}) {
	logger.Info.Output(3, fmt.Sprint(args...))
}

func (l *pulsarLogWrapper) Warn(args ...interface{}) {
	logger.Warning.Output(3, fmt.Sprint(args...))
}

func (l *pulsarLogWrapper) Error(args ...interface{}) {
	logger.Error.Output(3, fmt.Sprint(args...))
}

func (l *pulsarLogWrapper) Debugf(format string, args ...interface{}) {
	if l.enableDebugLogger {
		logger.Debug.Output(3, fmt.Sprintf(format, args...))
	}
}

func (l *pulsarLogWrapper) Infof(format string, args ...interface{}) {
	logger.Info.Output(3, fmt.Sprintf(format, args...))
}

func (l *pulsarLogWrapper) Warnf(format string, args ...interface{}) {
	logger.Warning.Output(3, fmt.Sprintf(format, args...))
}

func (l *pulsarLogWrapper) Errorf(format string, args ...interface{}) {
	logger.Error.Output(3, fmt.Sprintf(format, args...))
}

func (l pulsarLogEntry) WithFields(fs pulsarlog.Fields) pulsarlog.Entry {
	e := pulsarLogEntry{
		fields:            pulsarlog.Fields{},
		enableDebugLogger: l.enableDebugLogger,
	}
	if nil != l.fields {
		for k, v := range l.fields {
			e.fields[k] = v
		}
	}
	if nil != fs {
		for k, v := range fs {
			e.fields[k] = v
		}
	}
	return e
}

func (l pulsarLogEntry) WithField(name string, value interface{}) pulsarlog.Entry {
	e := pulsarLogEntry{
		fields:            pulsarlog.Fields{},
		enableDebugLogger: l.enableDebugLogger,
	}
	if nil != l.fields {
		for k, v := range l.fields {
			e.fields[k] = v
		}
	}
	e.fields[name] = value
	return e
}

func (l pulsarLogEntry) fieldsContent() string {
	if nil != l.fields {
		arr := []string{}
		for k, v := range l.fields {
			arr = append(arr, fmt.Sprintf("%s: %v", k, v))
		}
		return strings.Join(arr, " ")
	}
	return ""
}

func (l pulsarLogEntry) Debug(args ...interface{}) {
	if l.enableDebugLogger {
		logger.Debug.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprint(args...)))
	}
}

func (l pulsarLogEntry) Info(args ...interface{}) {
	logger.Info.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprint(args...)))
}

func (l pulsarLogEntry) Warn(args ...interface{}) {
	logger.Warning.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprint(args...)))
}

func (l pulsarLogEntry) Error(args ...interface{}) {
	logger.Error.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprint(args...)))
}

func (l pulsarLogEntry) Debugf(format string, args ...interface{}) {
	if l.enableDebugLogger {
		logger.Debug.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprintf(format, args...)))
	}
}

func (l pulsarLogEntry) Infof(format string, args ...interface{}) {
	logger.Info.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprintf(format, args...)))
}

func (l pulsarLogEntry) Warnf(format string, args ...interface{}) {
	logger.Warning.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprintf(format, args...)))
}

func (l pulsarLogEntry) Errorf(format string, args ...interface{}) {
	logger.Error.Output(3, fmt.Sprint(l.fieldsContent(), " ", fmt.Sprintf(format, args...)))
}
