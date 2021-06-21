package builtinmodels

import (
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
)

// Oauths 内置第三方登录授权模型。
type Oauths struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id" form:"id"`
	UserID                      int64  `xorm:"'user_id' BigInt index" json:"user_id" form:"user_id"`
	OauthType                   string `xorm:"'oauth_type' VARCHAR(64) notnull index" json:"oauth_type" form:"oauth_type"`
	OauthID                     string `xorm:"'oauth_id' VARCHAR(64) null index" json:"oauth_id" form:"oauth_id"`
	UnionID                     string `xorm:"'union_id' VARCHAR(64) null index" json:"union_id" form:"union_id"`
	Credential                  string `xorm:"'credential' VARCHAR(64) null" json:"credential" form:"credential"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// IsValid 可以做一些非常简单的“低级”数据验证
func (m *Oauths) IsValid() bool {
	return m.ID > 0
}

// TableName table name
func (m *Oauths) TableName() string {
	return "oauths"
}

// Fetch retrieve one record by self condition
func (m *Oauths) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *Oauths) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *Oauths) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *Oauths) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *Oauths) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// GetID primary id
func (m *Oauths) GetID() int64 {
	return m.ID
}

// InsertMany records
func (m *Oauths) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}
