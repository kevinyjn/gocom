package queues

import (
	"strings"
	"sync"
)

// FIFOQueue queue
type FIFOQueue struct {
	queue []IElement
	m     sync.RWMutex
}

// NewFIFOQueue new queue ordered by ascending
func NewFIFOQueue() *FIFOQueue {
	return &FIFOQueue{
		queue: []IElement{},
		m:     sync.RWMutex{},
	}
}

// Push item
func (q *FIFOQueue) Push(item IElement) bool {
	q.m.Lock()
	if nil == q.queue {
		q.queue = []IElement{item}
	} else {
		q.queue = append(q.queue, item)
	}
	q.m.Unlock()
	return true
}

// Pop first item
func (q *FIFOQueue) Pop() (interface{}, bool) {
	if q.GetSize() <= 0 {
		return nil, false
	}
	q.m.Lock()
	item := q.queue[0]
	q.queue = append([]IElement{}, q.queue[1:]...)
	q.m.Unlock()
	return item, true
}

// First item without pop
func (q *FIFOQueue) First() (interface{}, bool) {
	if q.GetSize() <= 0 {
		return nil, false
	}
	q.m.RLock()
	item := q.queue[0]
	q.m.RUnlock()
	return item, true
}

// Remove an element from queue identified by element.GetID()
func (q *FIFOQueue) Remove(item IElement) bool {
	var r = false
	q.m.Lock()
	for i, e := range q.queue {
		if e.GetID() == item.GetID() {
			q.queue = append(q.queue[0:i], q.queue[i+1:]...)
			r = true
			break
		}
	}
	q.m.Unlock()
	return r
}

// Elements of all queue
func (q *FIFOQueue) Elements() []IElement {
	q.m.RLock()
	elements := append([]IElement{}, q.queue...)
	q.m.RUnlock()
	return elements
}

// Dump element in queue
func (q *FIFOQueue) Dump() string {
	result := []string{}
	q.m.RLock()
	for _, e := range q.queue {
		result = append(result, e.DebugString())
	}
	q.m.RUnlock()
	return strings.Join(result, " ")
}

// GetOne func
func (q *FIFOQueue) GetOne(item IElement) (interface{}, bool) {
	// fmt.Printf("Removing element %s finding...\n", item.GetID())
	q.m.RLock()
	for _, e := range q.queue {
		if e.GetID() == item.GetID() {
			q.m.RUnlock()
			return item, true
		}
	}
	q.m.RUnlock()
	return nil, false
}

// GetElement get element by id
func (q *FIFOQueue) GetElement(ID string) (interface{}, bool) {
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

// CutBefore cut elements out before index
func (q *FIFOQueue) CutBefore(idx int) []IElement {
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
func (q *FIFOQueue) CutAfter(idx int) []IElement {
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
func (q *FIFOQueue) GetSize() int {
	q.m.RLock()
	n := len(q.queue)
	q.m.RUnlock()
	return n
}
