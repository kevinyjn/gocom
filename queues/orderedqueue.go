package queues

import (
	"strings"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/definations"
)

// OrderingMode type
type OrderingMode int

// Constants
const (
	OrderingAsc  = OrderingMode(0)
	OrderingDesc = OrderingMode(1)
)

// UnixTimestampToTime convert unix timestamp in seconds to time.Time
func UnixTimestampToTime(secs int64) time.Time {
	return time.Unix(secs, 0)
}

// OrderedQueue queue
type OrderedQueue struct {
	queue    []IElement
	ordering OrderingMode
	m        sync.RWMutex
}

// NewAscOrderingQueue new queue ordered by ascending
func NewAscOrderingQueue() *OrderedQueue {
	return &OrderedQueue{
		queue:    []IElement{},
		ordering: OrderingAsc,
		m:        sync.RWMutex{},
	}
}

// NewDescOrderingQueue new queue ordered by descending
func NewDescOrderingQueue() *OrderedQueue {
	return &OrderedQueue{
		queue:    []IElement{},
		ordering: OrderingDesc,
		m:        sync.RWMutex{},
	}
}

// Add element depending on ordered queue ordering mode
func (q *OrderedQueue) Add(item IElement) *OrderedQueue {
	q.m.Lock()
	ql := len(q.queue)
	q.queue = pushItemToOrderedQueue(&q.queue, ql, item, q.ordering)
	q.m.Unlock()
	return q
}

// Push element depending on ordered queue ordering mode
func (q *OrderedQueue) Push(item IElement) bool {
	q.Add(item)
	return true
}

// Pop first item
func (q *OrderedQueue) Pop() (interface{}, bool) {
	if q.GetSize() <= 0 {
		return nil, false
	}
	q.m.Lock()
	item := q.queue[0]
	q.queue = append([]IElement{}, q.queue[1:]...)
	q.m.Unlock()
	return item, true
}

// PopMany head elements from queue limited by maxResults, the element would be deleted from queue
func (q *OrderedQueue) PopMany(maxResults int) ([]interface{}, int) {
	maxLen := q.GetSize()
	if 0 >= maxLen || 0 >= maxResults {
		return nil, 0
	}

	if maxLen > maxResults {
		maxLen = maxResults
	}
	q.m.Lock()
	items := make([]interface{}, maxLen)
	for i := 0; i < maxLen; i++ {
		items[i] = q.queue[i]
	}
	q.queue = append([]IElement{}, q.queue[maxLen:]...)
	q.m.Unlock()
	return items, maxLen
}

// First item without pop
func (q *OrderedQueue) First() (interface{}, bool) {
	if q.GetSize() <= 0 {
		return nil, false
	}
	q.m.RLock()
	item := q.queue[0]
	q.m.RUnlock()
	return item, true
}

// Remove an element from queue identified by element.GetID()
func (q *OrderedQueue) Remove(item IElement) bool {
	// fmt.Printf("Removing element %s finding...\n", item.GetID())
	idx := q.findElementIndex(item)
	if 0 > idx {
		return false
	}
	q.m.Lock()
	q.queue = append(q.queue[0:idx], q.queue[idx+1:]...)
	q.m.Unlock()
	return true
}

// Elements of all queue
func (q *OrderedQueue) Elements() []IElement {
	q.m.RLock()
	elements := append([]IElement{}, q.queue...)
	q.m.RUnlock()
	return elements
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

// FindElements by compaire condition
func (q *OrderedQueue) FindElements(cmp *definations.ComparisonObject) []IElement {
	elements := []IElement{}
	if nil == cmp {
		return elements
	}
	q.m.RLock()
	for _, e := range q.queue {
		if cmp.Evaluate(e) {
			elements = append(elements, e)
		}
	}
	q.m.RUnlock()
	return elements
}

func (q *OrderedQueue) findElementIndex(item IElement) int {
	q.m.Lock()
	l := len(q.queue)
	if 0 >= l {
		q.m.Unlock()
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
			q.m.Unlock()
			return cursor
		}
		cursor++
	}
	cursor = idx - 1
	for cursor > min {
		if item.GetID() == q.queue[cursor].GetID() {
			q.m.Unlock()
			return cursor
		}
		cursor--
	}
	q.m.Unlock()
	return -1
}

// GetElement get element by id
func (q *OrderedQueue) GetElement(ID string) (interface{}, bool) {
	q.m.RLock()
	for _, e := range q.queue {
		if e.GetID() == ID {
			q.m.RUnlock()
			return e, true
		}
	}
	q.m.RUnlock()
	return nil, false
}

// Dump element in queue
func (q *OrderedQueue) Dump() string {
	result := []string{}
	q.m.RLock()
	for _, e := range q.queue {
		result = append(result, e.DebugString())
	}
	q.m.RUnlock()
	return strings.Join(result, ", \n")
}

// CutBefore cut elements out before index
func (q *OrderedQueue) CutBefore(idx int) []IElement {
	if 0 > idx {
		return []IElement{}
	} else if q.GetSize() >= idx {
		q.m.Lock()
		cuts := q.queue
		q.queue = []IElement{}
		q.m.Unlock()
		return cuts
	}
	q.m.Lock()
	cuts := q.queue[:idx]
	q.queue = q.queue[idx:]
	q.m.Unlock()
	return cuts
}

// CutAfter cut elements out after index
func (q *OrderedQueue) CutAfter(idx int) []IElement {
	if 0 > idx {
		q.m.Lock()
		cuts := q.queue
		q.queue = []IElement{}
		q.m.Unlock()
		return cuts
	} else if q.GetSize() >= idx {
		return []IElement{}
	}
	q.m.Lock()
	cuts := q.queue[idx+1:]
	q.queue = q.queue[:idx+1]
	q.m.Unlock()
	return cuts
}

// GetSize of queue
func (q *OrderedQueue) GetSize() int {
	q.m.RLock()
	n := len(q.queue)
	q.m.RUnlock()
	return n
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
