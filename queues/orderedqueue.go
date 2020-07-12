package queues

import (
	"fmt"
	"strings"
	"time"
)

// OrderingMode type
type OrderingMode int

// Constants
const (
	OrderingAsc  = OrderingMode(0)
	OrderingDesc = OrderingMode(1)
)

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

// UnixTimestampToTime convert unix timestamp in seconds to time.Time
func UnixTimestampToTime(secs int64) time.Time {
	return time.Unix(secs, 0)
}

// OrderedQueue queue
type OrderedQueue struct {
	queue    []IElement
	ordering OrderingMode
}

// NewAscOrderingQueue new queue ordered by ascending
func NewAscOrderingQueue() *OrderedQueue {
	return &OrderedQueue{
		queue:    []IElement{},
		ordering: OrderingAsc,
	}
}

// NewDescOrderingQueue new queue ordered by descending
func NewDescOrderingQueue() *OrderedQueue {
	return &OrderedQueue{
		queue:    []IElement{},
		ordering: OrderingDesc,
	}
}

// Add element depending on ordered queue ordering mode
func (q *OrderedQueue) Add(item IElement) *OrderedQueue {
	q.queue = pushItemToOrderedQueue(&q.queue, len(q.queue), item, q.ordering)
	return q
}

// Push element depending on ordered queue ordering mode
func (q *OrderedQueue) Push(item IElement) bool {
	q.Add(item)
	return true
}

// Queue get all queue data
func (q *OrderedQueue) Queue() []IElement {
	if nil == q.queue {
		q.queue = []IElement{}
	}
	return q.queue
}

// Pop first item
func (q *OrderedQueue) Pop() (interface{}, bool) {
	if len(q.queue) <= 0 {
		return nil, false
	}
	item := q.queue[0]
	q.queue = append([]IElement{}, q.queue[1:]...)
	return item, true
}

// First item without pop
func (q *OrderedQueue) First() (interface{}, bool) {
	if len(q.queue) <= 0 {
		return nil, false
	}
	item := q.queue[0]
	return item, true
}

// Remove an element from queue identified by element.GetID()
func (q *OrderedQueue) Remove(item IElement) bool {
	// fmt.Printf("Removing element %s finding...\n", item.GetID())
	idx := q.findElementIndex(item)
	if 0 > idx {
		return false
	}

	q.queue = append(q.queue[0:idx], q.queue[idx+1:]...)
	return true
	// for i, e := range q.queue {
	// 	if e.GetID() == item.GetID() {
	// 		// fmt.Printf("Removing element on index:%d depend on id:%s\n", i, item.GetID())
	// 		q.queue = append(q.queue[0:i], q.queue[i+1:]...)
	// 		return true
	// 	}
	// }
	// return false
}

// Elements of all queue
func (q *OrderedQueue) Elements() []IElement {
	return q.queue
}

// GetOne an element from queue identified by element.GetID()
func (q *OrderedQueue) GetOne(item IElement) (interface{}, bool) {
	// fmt.Printf("Removing element %s finding...\n", item.GetID())
	idx := q.findElementIndex(item)
	if 0 > idx {
		return item, false
	}
	return item, true
	// for _, e := range q.queue {
	// 	if e.GetID() == item.GetID() {
	// 		return item, true
	// 	}
	// }
	// return nil, false
}

func (q *OrderedQueue) findElementIndex(item IElement) int {
	l := len(q.queue)
	if 0 >= l {
		return -1
	}
	idx := findOrderedQueueInsertingIndex(&q.queue, l, item, q.ordering)
	cursor := idx
	max := idx + 2
	min := idx - 2
	if max > l {
		max = l
	}
	if -1 > min {
		min = -1
	}
	for cursor < max {
		if item.GetID() == q.queue[cursor].GetID() {
			return cursor
		}
		cursor++
	}
	cursor = idx - 1
	for cursor > min {
		if item.GetID() == q.queue[cursor].GetID() {
			return cursor
		}
		cursor--
	}
	return -1
}

// GetElement get element by id
func (q *OrderedQueue) GetElement(ID string) (interface{}, bool) {
	for _, e := range q.queue {
		if e.GetID() == ID {
			return e, true
		}
	}
	return nil, false
}

// Dump element in queue
func (q *OrderedQueue) Dump() string {
	result := []string{}
	for _, e := range q.queue {
		result = append(result, e.DebugString())
	}
	return strings.Join(result, ", \n")
}

// CutBefore cut elements out before index
func (q *OrderedQueue) CutBefore(idx int) []IElement {
	if 0 > idx {
		return []IElement{}
	} else if len(q.queue) >= idx {
		cuts := q.queue
		q.queue = []IElement{}
		return cuts
	}
	cuts := q.queue[:idx]
	q.queue = q.queue[idx:]
	return cuts
}

// CutAfter cut elements out after index
func (q *OrderedQueue) CutAfter(idx int) []IElement {
	if 0 > idx {
		cuts := q.queue
		q.queue = []IElement{}
		return cuts
	} else if len(q.queue) >= idx {
		return []IElement{}
	}
	cuts := q.queue[idx+1:]
	q.queue = q.queue[:idx+1]
	return cuts
}

// pushItemToOrderedQueue 依据排序顺序新元素插入到已有队列中
// 由于golang的特性，数组元素任何形式的新增都需要更新插入后的数组地址，因此，执行此方法后应将返回的队列赋值到目标队列。
// 此队列考虑到所用业务队列数据规模不会太大，因此采用二分排序算法，算法效率较一般排序算法高，但并不是最高排序效率算法。
func pushItemToOrderedQueue(queue *[]IElement, l int, item IElement, ordering OrderingMode) []IElement {
	if nil == *queue || 0 >= l {
		queue := []IElement{item}
		return queue
	}

	idx := findOrderedQueueInsertingIndex(queue, l, item, ordering)

	if idx >= l {
		return append(*queue, item)
	}
	tails := append([]IElement{}, (*queue)[idx:]...)
	result := append(append((*queue)[0:idx], item), tails...)
	return result
}

func findOrderedQueueInsertingIndex(queue *[]IElement, l int, item IElement, ordering OrderingMode) int {
	if nil == *queue || 0 >= l {
		return 0
	}

	idx := (l) / 2
	originIdx := idx
	minIdx := 0
	maxIdx := l - 1
	left := false
	for idx < l {
		if OrderingDesc == ordering {
			left = item.OrderingValue() > (*queue)[idx].OrderingValue()
		} else {
			left = item.OrderingValue() < (*queue)[idx].OrderingValue()
		}

		if left {
			if idx <= 0 {
				break
			}
			maxIdx = idx - 1
		} else {
			minIdx = idx + 1
		}
		idx = (minIdx + maxIdx + 1) / 2
		if idx == originIdx {
			break
		}
		originIdx = idx
	}

	return idx
}

func checkInsertBefore(e1, e2 IElement, ordering OrderingMode) bool {
	if OrderingDesc == ordering {
		return e1.OrderingValue() > e2.OrderingValue()
	}
	return e1.OrderingValue() < e2.OrderingValue()
}

// UnitestOrderedQueue test
func UnitestOrderedQueue() error {
	queue1 := &OrderedQueue{}
	queue2 := NewDescOrderingQueue()
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
