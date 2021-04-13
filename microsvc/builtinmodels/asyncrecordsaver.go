package builtinmodels

import (
	"sync"
	"time"

	"github.com/kevinyjn/gocom/orm/rdbms"
)

// AsyncRecordSaver struct
type AsyncRecordSaver struct {
	logs []RecordModel
	t    *time.Ticker
	m    sync.RWMutex
}

var _lm = &AsyncRecordSaver{
	logs: []RecordModel{},
	t:    nil,
	m:    sync.RWMutex{},
}

// GetAsyncRecordSaver AsyncRecordSaver
func GetAsyncRecordSaver() *AsyncRecordSaver {
	return _lm
}

// Push log
func (m *AsyncRecordSaver) Push(log RecordModel) error {
	m.m.Lock()
	m.logs = append(m.logs, log)
	m.m.Unlock()
	if nil == m.t {
		go m.startPersistentTimer()
	}
	return nil
}

func (m *AsyncRecordSaver) startPersistentTimer() {
	if nil != m.t {
		return
	}
	m.t = time.NewTicker(1 * time.Second)
	for nil != m.t {
		select {
		case <-m.t.C:
			// do persistent
			m.persistLogs()
			break
		}
	}
	m.t.Stop()
	m.t = nil
}

func (m *AsyncRecordSaver) persistLogs() {
	m.m.Lock()
	logs := make([]interface{}, len(m.logs))
	for i, row := range m.logs {
		logs[i] = row
	}
	m.logs = []RecordModel{}
	m.m.Unlock()
	ll := len(logs)
	l := ll
	if l > 0 {
		offset := 0
		if 100 < l {
			l = 100
		}
		pendings := [][]interface{}{}
		curGroup := make([]interface{}, l)
		for i, row := range logs {
			if i-offset >= 100 {
				pendings = append(pendings, curGroup)
				offset += l
				l = ll - offset
				if 100 < l {
					l = 100
				}
				curGroup = make([]interface{}, l)
			}
			curGroup[i-offset] = row
		}
		if len(curGroup) > 0 {
			pendings = append(pendings, curGroup)
		}

		for _, rows := range pendings {
			InsertRecords(rows)
		}
	}
}

// InsertRecords records
func InsertRecords(records []interface{}) (int64, error) {
	return rdbms.GetInstance().Insert(records...)
}
