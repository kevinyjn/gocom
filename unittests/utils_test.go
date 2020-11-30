package unittests

import (
	"fmt"
	"testing"
	"time"

	"github.com/kevinyjn/gocom/utils"
)

func TestUtilsSubUTF8Text(t *testing.T) {
	text := "Go语言是Google开发的一种静态强类型、编译型、并发型，并具有垃圾回收功能的编程语言。为了方便搜索和识别，有时会将其称为Golang。"
	v1 := utils.SubStringUTF8(text, 14)
	v2 := utils.SubStringFromUTF8(text, 14, 0, true)
	v3 := utils.SubStringUTF8(text, 4, 14)
	v4 := utils.SubStringUTF8(text, 4, 15)
	AssertEquals(t, "Go语言是Google开发的", v1, "sub(0, 14)")
	AssertEquals(t, "Go语言是Google开发的...", v2, "sub(0, 14)")
	AssertEquals(t, "一种静态", v3, "sub(14, 18)")
	AssertEquals(t, "种静态强", v4, "sub(15, 19)")
}

func TestUtilsHumanBytes(t *testing.T) {
	AssertEquals(t, "1.21 GB", utils.HumanByteSize(1024*1024*1024+210*1024*1024+1036), "GB size")
	AssertEquals(t, "1.21 MB", utils.HumanByteSize(1024*1024+210*1024+1036), "MB size")
	AssertEquals(t, "10.21 KB", utils.HumanByteSize(10*1024+220), "KB size")
	AssertEquals(t, "10240 Bytes", utils.HumanByteSize(10240), "Byte size")
}

func TestUtilsTimer(t *testing.T) {
	timer, err := utils.NewTimer(10, 100, func(isnt *utils.Timer, tim time.Time, delegate interface{}) {
		fmt.Println("timer triggered ...")
	}, nil)
	AssertNil(t, err, "utils.NewTimer")
	AssertNotNil(t, timer, "timer")
	to := time.NewTimer(time.Millisecond * 500)
	select {
	case <-to.C:
		fmt.Println("test timer finished")
		break
	}
}
