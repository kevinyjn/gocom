package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/queues"
	"github.com/kevinyjn/gocom/testingutil"
	"github.com/kevinyjn/gocom/utils"
)

func TestQueuesOperate(t *testing.T) {
	err := testOrderedQueue()
	testingutil.AssertNil(t, err, "testOrderedQueue")
	fmt.Println("Testing ordered queue finished")
	err = testQueueOperatesInCoroutine()
	testingutil.AssertNil(t, err, "testQueueOperatesInCoroutine")
	fmt.Println("Testing operates queue in coroutine finished")
}

// demoElement demo element
type demoElement struct {
	val      string
	ordering int64
}

// GetID get id
func (e *demoElement) GetID() string {
	return e.val
}

// GetName get name
func (e *demoElement) GetName() string {
	return e.val
}

// OrderingValue get expire time
func (e *demoElement) OrderingValue() int64 {
	return e.ordering
}

// DebugString text
func (e *demoElement) DebugString() string {
	return e.val
}

// testOrderedQueue test
func testOrderedQueue() error {
	queue1 := queues.NewAscOrderingQueue()
	queue2 := queues.NewDescOrderingQueue()
	items := []*demoElement{
		{val: "3", ordering: 3},
		{val: "5", ordering: 5},
		{val: "2", ordering: 2},
		{val: "9", ordering: 9},
		{val: "6", ordering: 6},
		{val: "7", ordering: 7},
		{val: "1", ordering: 1},
		{val: "10", ordering: 10},
		{val: "8", ordering: 8},
		{val: "4", ordering: 4},
	}

	for _, e := range items {
		queue1.Add(e)
		queue2.Add(e)
	}
	queue1.Remove(&demoElement{val: "10", ordering: 10})
	fmt.Println("Testing... asceding queue  ->", queue1.Dump())
	fmt.Println("Testing... desceding queue ->", queue2.Dump())

	return nil
}

func testQueueOperatesInCoroutine() error {
	var err error
	queue1 := queues.NewAscOrderingQueue()
	queue2 := queues.NewFIFOQueue()
	finish := make(chan string)
	finishCount := 0

	go func() {
		for i := 0; i < 50000; i++ {
			queue1.Push(&demoElement{val: utils.RandomString(4), ordering: int64(utils.RandomInt(990))})
			queue2.First()
			queue2.Pop()
		}
		finish <- "go1"
	}()
	go func() {
		for i := 0; i < 50000; i++ {
			queue2.Push(&demoElement{val: utils.RandomString(4), ordering: int64(utils.RandomInt(990))})
			queue1.First()
			queue1.Pop()
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
	return err
}

func TestQueuesFindElements(t *testing.T) {
	queue1 := queues.NewAscOrderingQueue()
	items := []*demoElement{
		{val: "3", ordering: 3},
		{val: "5", ordering: 5},
		{val: "2", ordering: 2},
		{val: "9", ordering: 9},
		{val: "6", ordering: 6},
		{val: "7", ordering: 7},
		{val: "1", ordering: 1},
		{val: "10", ordering: 10},
		{val: "8", ordering: 8},
		{val: "4", ordering: 4},
	}
	for _, e := range items {
		queue1.Add(e)
	}

	results1 := queue1.FindElements(definations.NewComparisonObject().And(definations.CompareGreaterEquals, "val", "3").And(definations.CompareLessEquals, "val", "5"))
	results2 := queue1.FindElements(definations.NewComparisonObject().And(definations.CompareContains, "ordering", []int{3, 4, 5}))
	results3 := queue1.FindElements(definations.NewComparisonObject().And(definations.CompareContains, "val", "345"))
	results4 := queue1.FindElements(definations.NewComparisonObject().And(definations.CompareBetween, "val", []string{"3", "5"}))

	testingutil.AssertEquals(t, 3, len(results1), "results1.length")
	testingutil.AssertEquals(t, len(results1), len(results2), "results2.length")
	testingutil.AssertEquals(t, len(results1), len(results3), "results3.length")
	testingutil.AssertEquals(t, len(results1), len(results4), "results4.length")
	for i, v := range results1 {
		testingutil.AssertEquals(t, v, results2[i], fmt.Sprintf("results2[%d]", i))
		testingutil.AssertEquals(t, v, results3[i], fmt.Sprintf("results3[%d]", i))
		testingutil.AssertEquals(t, v, results4[i], fmt.Sprintf("results4[%d]", i))
	}

	results, n := queue1.PopMany(3)
	testingutil.AssertEquals(t, 3, n, "queue1.PopMany(3) length")
	testingutil.AssertEquals(t, 3, len(results), "queue1.PopMany(3) results length")

	fmt.Println("Testing queue find elements finished")
}
