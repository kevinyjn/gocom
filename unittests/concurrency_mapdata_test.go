package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/utils"
)

func TestConCurrencyMapDataOperations(t *testing.T) {
	err := testOperateMapDataInCoroutine(t)
	if nil != err {
		t.Errorf("Testing gen uuid in coroutine failed with error:%v", err)
	}
	fmt.Println("Testing gen uuid in coroutine finished")
}

func testOperateMapDataInCoroutine(t *testing.T) error {
	mapData := utils.CoMapData{}
	finish := make(chan string)
	finishCount := 0
	go func() {
		for i := 0; i < 5000; i++ {
			key1 := fmt.Sprintf("go1:%d", i)
			key2 := fmt.Sprintf("go2:%d", i)
			mapData.Add(key1, 1)
			if mapData.Exists(key2) {
				mapData.Delete(key2)
			}
			// time.Sleep(time.Microsecond * 1)
		}
		finish <- "go1"
	}()
	go func() {
		for i := 0; i < 5000; i++ {
			key1 := fmt.Sprintf("go1:%d", i)
			key2 := fmt.Sprintf("go2:%d", i)
			mapData.Add(key2, 1)
			if mapData.Exists(key1) {
				mapData.Delete(key1)
			}
			// time.Sleep(time.Microsecond * 1)
		}
		finish <- "go2"
	}()

	for finishCount < 2 {
		<-finish
		finishCount++
	}
	return nil
}
