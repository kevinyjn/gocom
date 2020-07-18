package main

import (
	"fmt"

	"github.com/kevinyjn/gocom/orm/rdbms/scaffolds"
)

func main() {
	scaf := scaffolds.NewProjectScaffolds("example_orms", "../examples")
	scaf.DockerfileTemplate = "../etc/Dockerfile.tpl"
	scaf.JenkinsfileTemplate = "../etc/Jenkinsfile.tpl"
	err := scaf.Generate()
	if nil != err {
		fmt.Printf("generate project:%s failed with error:%v", scaf.Name, err)
		return
	}

	m := scaffolds.ModelScaffolds{
		Path:      "orm_example_model.go",
		Name:      "ExampleModel",
		TableName: "example_model",
		Fields: []scaffolds.ModelField{
			{
				Name:           "ID",
				Column:         "id",
				Type:           "int",
				Length:         32,
				IsPrimaryKey:   true,
				AutoIncreament: true,
			},
			{
				Name:    "name",
				Column:  "Name",
				Type:    "string",
				Length:  64,
				IsIndex: true,
			},
			{
				Name:    "UpdateTime",
				Column:  "updateTime",
				Type:    "time.Time",
				Length:  0,
				IsIndex: true,
			},
		},
	}
	fmt.Sprintf("formatted model:\n========\n%s\n", m.Encode())
	scaf.GenerateModel(&m)
}
