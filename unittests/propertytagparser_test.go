package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/utils"
)

func TestPropertyTagParser(t *testing.T) {
	err := testPropertyTagParser()
	if nil != err {
		t.Errorf("Parse properties failed with error:%v", err)
	} else {
		fmt.Printf("Succeed.\n")
	}
}

func testPropertyTagParser() error {
	texts := []string{
		`Value Separator,default:\,`,
		"Escape Character,default:\\",
		`Record Separator,default:\\n`,
		"csv-reader-csv-parser,default:commons-csv",
		"Trim Fields,default:false",
		"Timestamp Format:default:2020/08/08 22\\:33\\:50,format:YYYY/MM/dd HH\\:ii\\:ss",
	}

	for _, text := range texts {
		name, attributes := utils.ParsePropertyTagValue(text)
		fmt.Printf("- [%s]\n  extracted: name:[%s] attributes:%+v\n", text, name, attributes)
	}
	return nil
}
