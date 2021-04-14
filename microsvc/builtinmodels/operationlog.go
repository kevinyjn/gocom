package builtinmodels

import (
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc/events"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
)

// OperationLogModel interface
type OperationLogModel interface {
	Save() (bool, error)
	LoadFromEvent(events.Event)
	NewRecord() OperationLogModel
}

// OperationLog label element
type OperationLog struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id"`
	AppID                       string `xorm:"'app_id' VARCHAR(42) index" json:"appId"`                 // 应用标识
	Module                      string `xorm:"'module' VARCHAR(64) notnull index" json:"module"`        // 模块分组
	Name                        string `xorm:"'name' VARCHAR(64) notnull index" json:"name"`            // 模块功能名称
	UserID                      string `xorm:"'user_id' VARCHAR(64) index" json:"userId"`               // 用户ID
	RequestID                   string `xorm:"'request_id' VARCHAR(64) index" json:"requestId"`         // 请求唯一标识
	CorrelationID               string `xorm:"'correlation_id' VARCHAR(64) index" json:"currelationId"` // 调用链唯一标识
	DeviceID                    string `xorm:"'device_id' VARCHAR(64) index" json:"deviceId"`           // 设备标识
	RemoteIP                    string `xorm:"'remote_ip' VARCHAR(64) index" json:"remoteIp"`           // 源IP
	Agent                       string `xorm:"'agent' VARCHAR(128)" json:"agent"`                       // 源客户端信息
	Status                      int    `xorm:"'status' Int index" json:"status"`                        // 响应状态
	Message                     string `xorm:"'message' TEXT" json:"message"`                           // 响应提示
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// TableName table name
func (m *OperationLog) TableName() string {
	return "operation_log"
}

// Fetch retrieve one record by self condition
func (m *OperationLog) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *OperationLog) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *OperationLog) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *OperationLog) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *OperationLog) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// GetID primary id
func (m *OperationLog) GetID() int64 {
	return m.ID
}

// InsertMany records
func (m *OperationLog) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}

// LoadFromEvent load record fields by event data
func (m *OperationLog) LoadFromEvent(event events.Event) {
	if nil == event {
		logger.Error.Printf("access log model load from event while giving empty event")
		return
	}
	m.AppID = event.GetAppID()
	m.UserID = event.GetUserID()
	m.CorrelationID = event.GetCorrelationID()
	m.RequestID = event.GetIdentifier()
	m.Module = event.GetCategory()
	m.Name = event.GetFrom()
	m.Status = event.GetStatus()
	m.Message = event.GetDescription()
	headers := event.GetHeaders()
	if nil != headers {
		m.Agent = headers["agent"]
		m.RemoteIP = headers["remoteip"]
		m.DeviceID = headers["deviceid"]
	}
}

// NewRecord of access log model
func (m *OperationLog) NewRecord() OperationLogModel {
	log := OperationLog{}
	return &log
}
