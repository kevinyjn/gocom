package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/testingutil"
	"github.com/kevinyjn/gocom/validator"
)

func TestValidates(t *testing.T) {
	err := testValidatesRequired()
	if nil != err {
		t.Errorf("Validates required failed with error:%v", err)
	} else {
		fmt.Printf("Succeed.\n")
	}
}

func TestValidateDefault(t *testing.T) {
	tst := &testStruct{
		requiredString: "a",
		requiredArray:  []string{"b"},
	}
	err := validator.Validate(tst)
	testingutil.AssertNil(t, err, "Validate result")
	fmt.Printf("validate result:%+v\n", tst)
}

type testValidates struct {
	requiredString  string       `validate:"required" comment:"字符串必填字段"`
	requiredInt     int          `validate:"required" comment:"整形必填字段"`
	requiredStruct  testStruct   `validate:"required" comment:"结构体必填字段"`
	RequiredStruct2 *testStruct  `validate:"required" comment:"结构体必填字段2"`
	RequiredStruct3 *testStruct  `validate:"optional" comment:"结构体选填字段3"`
	RequiredArray   []testStruct `validate:"required" comment:"结构体数组必填字段"`
}

type testStruct struct {
	requiredString string   `validate:"required" comment:"字符串必填字段2"`
	requiredArray  []string `validate:"required" comment:"字符串数组必填字段"`
	OptionalValue  string   `validate:"optional" comment:"默认值字段21" default:"abcde"`
	OptionalInt    int      `validate:"optional" comment:"默认值字段22" default:"10000"`
	OptionalFloat  int      `validate:"optional" comment:"默认值字段23" default:"1"`
	OptionalBool   bool     `validate:"optional" comment:"默认值字段24" default:"true"`
}

func testValidatesRequired() error {
	fmt.Println("Testing validates...")

	to := &testValidates{
		requiredString: "1",
		requiredInt:    1,
		requiredStruct: testStruct{
			requiredString: "a",
			requiredArray:  []string{"b"},
		},
		RequiredStruct2: &testStruct{
			requiredString: "a",
			requiredArray:  []string{"b"},
		},
		RequiredArray: []testStruct{
			{
				requiredString: "a",
				requiredArray:  []string{"d"},
			},
		},
	}

	err := validator.Validate(to)
	fmt.Println("Validate result", err)

	fmt.Println("Finished.")
	return err
}
