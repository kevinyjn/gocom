package unittests

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/kevinyjn/gocom/testingutil"
	"github.com/kevinyjn/gocom/utils"
)

func TestUtilsSubUTF8Text(t *testing.T) {
	text := "Go语言是Google开发的一种静态强类型、编译型、并发型，并具有垃圾回收功能的编程语言。为了方便搜索和识别，有时会将其称为Golang。"
	v1 := utils.SubStringUTF8(text, 14)
	v2 := utils.SubStringFromUTF8(text, 14, 0, true)
	v3 := utils.SubStringUTF8(text, 4, 14)
	v4 := utils.SubStringUTF8(text, 4, 15)
	testingutil.AssertEquals(t, "Go语言是Google开发的", v1, "sub(0, 14)")
	testingutil.AssertEquals(t, "Go语言是Google开发的...", v2, "sub(0, 14)")
	testingutil.AssertEquals(t, "一种静态", v3, "sub(14, 18)")
	testingutil.AssertEquals(t, "种静态强", v4, "sub(15, 19)")
}

func TestUtilsHumanBytes(t *testing.T) {
	testingutil.AssertEquals(t, "1.21 GB", utils.HumanByteSize(1024*1024*1024+210*1024*1024+1036), "GB size")
	testingutil.AssertEquals(t, "1.21 MB", utils.HumanByteSize(1024*1024+210*1024+1036), "MB size")
	testingutil.AssertEquals(t, "10.21 KB", utils.HumanByteSize(10*1024+220), "KB size")
	testingutil.AssertEquals(t, "10240 Bytes", utils.HumanByteSize(10240), "Byte size")
}

func TestUtilsTimer(t *testing.T) {
	timer1, err1 := utils.NewTimer(10, 100, func(isnt *utils.Timer, tim time.Time, delegate interface{}) {
		fmt.Println("timer1 triggered ...")
	}, nil)
	timer2, err2 := utils.NewTimer(100, -1, func(isnt *utils.Timer, tim time.Time, delegate interface{}) {
		fmt.Println("timer2 triggered ...")
	}, nil)
	testingutil.AssertNil(t, err1, "utils.NewTimer")
	testingutil.AssertNotNil(t, timer1, "timer1")
	testingutil.AssertNil(t, err2, "utils.NewTimer")
	testingutil.AssertNotNil(t, timer2, "timer2")
	to := time.NewTimer(time.Millisecond * 500)
	<-to.C
	fmt.Println("test timer finished")
}

func TestUtilsURLPathJoin(t *testing.T) {
	url := utils.URLPathJoin("/demo", " ", "", "a/", "/b/", "/c/")
	testingutil.AssertEquals(t, "/demo/a/b/c/", url, "url")
	url = utils.URLPathJoin("http://localhost/demo/", " ", "/", "a", "/b/", "c/")
	testingutil.AssertEquals(t, "http://localhost/demo/a/b/c/", url, "url")
}

func TestUtilsConvertToString(t *testing.T) {
	v0 := []byte("abc")
	v1 := utils.ToString(v0)
	testingutil.AssertEquals(t, "abc", v1, "ToString")
}

func TestHexStringToInteger(t *testing.T) {
	val := "5c87d5c0b901858838a7d7b1"
	v, err := strconv.ParseUint(val, 16, 64)
	testingutil.AssertNotEquals(t, 0, v, "HexStringToInteger")
	fmt.Printf("hex %s integer value is %d\n", val, v)
	testingutil.AssertNotNil(t, err, "strconv.ParseUint")
}
