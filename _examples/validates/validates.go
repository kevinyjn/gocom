package main

import (
	"fmt"

	"github.com/kevinyjn/gocom/validator"
)

type testValidates struct {
	requiredString  string       `validate:"required" comment:"字符串必填字段"`
	requiredInt     int          `validate:"required" comment:"整形必填字段"`
	requiredStruct  testStruct   `validate:"required" comment:"结构体必填字段"`
	RequiredStruct2 testStruct   `validate:"required" comment:"结构体必填字段"`
	RequiredArray   []testStruct `validate:"required" comment:"结构体数组必填字段"`
}

type testStruct struct {
	requiredString string   `validate:"required" comment:"字符串必填字段2"`
	requiredArray  []string `validate:"required" comment:"字符串数组必填字段"`
}

func main() {
	fmt.Println("Testing validates...")

	to := &testValidates{
		requiredString: "1",
		requiredInt:    1,
		RequiredStruct2: testStruct{
			requiredString: "a",
			requiredArray:  []string{"b"},
		},
		RequiredArray: []testStruct{
			testStruct{
				requiredString: "",
				requiredArray:  []string{"d"},
			},
		},
	}

	err := validator.Validate(to)
	fmt.Println("Validate result", err)

	fmt.Println("Finished.")
}
