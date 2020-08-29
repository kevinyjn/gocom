package main

import (
	"time"

	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
)

type schemaDemo struct {
	ID                          int       `xorm:"'id' Int pk autoincr" json:"id"`
	Category                    string    `xorm:"'category' VARCHAR(36) notnull index" json:"category"`
	Name                        string    `xorm:"'name' VARCHAR(255) notnull index" json:"name"`
	UpdateTime                  time.Time `xorm:"'updateTime' DateTime index" json:"updateTime"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

type schemaDemo2 struct {
	ID                          int       `xorm:"'id' Int pk autoincr" json:"id"`
	Category                    string    `xorm:"'category' VARCHAR(36) notnull index" json:"category"`
	Name                        string    `xorm:"'name' VARCHAR(255) notnull index" json:"name"`
	UpdateTime                  time.Time `xorm:"'updateTime' DateTime index" json:"updateTime"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}
