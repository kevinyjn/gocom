package main

import (
	"fmt"
	"time"

	"github.com/kevinyjn/gocom/config"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/orm/rdbms"

	"github.com/kataras/iris"
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

	testDemo1()
	testDemo2()

	timer1 := time.NewTicker(5 * time.Second)
	timer2 := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-timer1.C:
				testDemo1()
				break
			}
		}
	}()
	go func() {
		for {
			select {
			case <-timer2.C:
				testDemo2()
				break
			}
		}
	}()

	app := iris.New()
	app.Get("/demo", testDemo0)
	app.Run(iris.Addr(":8081"))
}

func testDemo0(ctx iris.Context) {
	ele2 := &SchemaDemo{ID: 1}
	ele2.Fetch(ele2)
	ele2.Name = "origin_demo"
	ele2.Save(ele2)

	rows, err := rdbms.GetInstance().FetchRecords(&SchemaDemo{}, 20, 0)
	if nil != err {
		ctx.WriteString(err.Error())
	} else {
		ctx.WriteString(fmt.Sprintf("%+v", rows))
	}
}

func testDemo1() {
	ele := SchemaDemo{
		Name:     "demo",
		Category: "debug",
	}

	ele.Save(&ele)

	rdbms.GetInstance().FetchRecords(&SchemaDemo{}, 20, 0)

	elex := &SchemaDemo{Name: "demo"}
	elex.Fetch(elex)

	ele2 := &SchemaDemo{ID: 1}
	ele2.Fetch(ele2)
	ele3 := &SchemaDemo{ID: 2}
	ele3.Fetch(ele3)
	ele2.Name = "origin_demo"
	ele2.Save(ele2)

	elex = &SchemaDemo{Name: "demo"}
	elex.Fetch(elex)

	elex = &SchemaDemo{Name: "origin_demo"}
	elex.Fetch(elex)
	elex = &SchemaDemo{Name: "origin_demo"}
	elex.Fetch(elex)
}

func testDemo2() {
	ele := SchemaDemo2{}
	ele.Name = "demo"
	ele.Category = "debug"

	ele.Save(&ele)

	rdbms.GetInstance().FetchRecords(&SchemaDemo2{}, 20, 0)

	elex := &SchemaDemo2{}
	elex.Name = "demo"
	elex.Fetch(elex)

	ele2 := &SchemaDemo2{}
	ele2.ID = 1
	ele2.Fetch(ele2)
	ele3 := &SchemaDemo2{}
	ele3.ID = 2
	ele3.Fetch(ele3)
	ele2.Name = "origin_demo"
	ele2.Save(ele2)

	elex = &SchemaDemo2{}
	elex.Name = "demo"
	elex.Fetch(elex)

	elex = &SchemaDemo2{}
	elex.Name = "origin_demo"
	elex.Fetch(elex)
	elex = &SchemaDemo2{}
	elex.Name = "origin_demo"
	elex.Fetch(elex)
}
