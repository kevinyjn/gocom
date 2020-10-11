package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/utils"
)

func TestConCurrencyGenUUID(t *testing.T) {
	err := testGenUUIDInCoroutine()
	if nil != err {
		t.Errorf("Testing gen uuid in coroutine failed with error:%v", err)
	}
	fmt.Println("Testing gen uuid in coroutine finished")
}

func testGenUUIDInCoroutine() error {
	finish := make(chan string)
	finishCount := 0
	go func() {
		for i := 0; i < 5000; i++ {
			utils.RandomString(23)
			// time.Sleep(time.Microsecond * 1)
		}
		finish <- "go1"
	}()
	go func() {
		for i := 0; i < 5000; i++ {
			utils.GenLoweruuid()
			// time.Sleep(time.Microsecond * 1)
		}
		finish <- "go2"
	}()

	for finishCount < 2 {
		select {
		case <-finish:
			finishCount++
			break
		}
	}
	return nil
}
