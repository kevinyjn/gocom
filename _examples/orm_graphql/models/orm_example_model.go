package models

import (
	"time"

	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
)

// ExampleModel model
type ExampleModel struct {
	ID                          int       `xorm:"'id' Int pk autoincr" json:"id"`
	Name                        string    `xorm:"'Name' Varchar(64) index" json:"name"`
	UpdateTime                  time.Time `xorm:"'updateTime' DateTime index" json:"updateTime"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// TableName returns table name in database
func (m *ExampleModel) TableName() string {
	return "example_model"
}

// Fetch retrieve one record by self condition
func (m *ExampleModel) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *ExampleModel) Save() (bool, error) {
	return m.Datasource.Save(m)
}
