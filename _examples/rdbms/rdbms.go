package main

import (
	"fmt"

	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/orm/rdbms"
)

func main() {
	env, err := config.Init("../etc/config.yaml")
	if err != nil {
		return
	}

	logger.Init(&env.Logger)
	dbConfig := env.DBs["default"]

	_, err = rdbms.GetInstance().Init("default", &dbConfig)
	if nil != err {
		fmt.Sprintf("initialize database entine failed with error:%v", err)
		return
	}

	ele := schemaDemo{
		Name:     "demo",
		Category: "debug",
	}

	ele.Save(&ele)

	rdbms.GetInstance().FetchRecords(&schemaDemo{}, 20, 0)

	elex := &schemaDemo{Name: "demo"}
	elex.Fetch(elex)

	ele2 := &schemaDemo{ID: 1}
	ele2.Fetch(ele2)
	ele3 := &schemaDemo{ID: 2}
	ele3.Fetch(ele3)
	ele2.Name = "origin_demo"
	ele2.Save(ele2)

	elex = &schemaDemo{Name: "demo"}
	elex.Fetch(elex)

	elex = &schemaDemo{Name: "origin_demo"}
	elex.Fetch(elex)
	elex = &schemaDemo{Name: "origin_demo"}
	elex.Fetch(elex)

	// test cleaning the outdated data

}
