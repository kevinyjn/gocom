package main

import (
	"github.com/kevinyjn/gocom/schema/rdbms/behaviors"
	"github.com/kevinyjn/gocom/schema/rdbms/dal"
)

type schemaDemo struct {
	ID                          int    `xorm:"'id' Int pk autoincr" json:"id"`
	Category                    string `xorm:"'category' VARCHAR(36) notnull index" json:"category"`
	Name                        string `xorm:"'name' VARCHAR(255) notnull index" json:"name"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	dal.Datasource              `xorm:"-" datasource:"default"`
}
