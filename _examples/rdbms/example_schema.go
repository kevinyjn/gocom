package main

import (
	"time"

	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
)

// SchemaDemo demo
type SchemaDemo struct {
	ID                          int       `xorm:"'id' Int pk autoincr" json:"id"`
	Category                    string    `xorm:"'category' VARCHAR(36) notnull index" json:"category"`
	Name                        string    `xorm:"'name' VARCHAR(255) notnull index" json:"name"`
	Tag                         string    `xorm:"'tag' VARCHAR(255) notnull index" json:"tag"`
	Flag                        string    `xorm:"'flag' VARCHAR(255) index" json:"flag"`
	UpdateTime                  time.Time `xorm:"'updateTime' DateTime index" json:"updateTime"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// SchemaDemo2 demo
type SchemaDemo2 struct {
	SchemaDemo `xorm:"extends"`
	NameX      string `xorm:"'namex' VARCHAR(255) notnull index" json:"namex"`
}
