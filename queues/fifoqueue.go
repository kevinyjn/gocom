package queues

import "strings"

// FIFOQueue queue
type FIFOQueue struct {
	queue []IElement
}

// NewFIFOQueue new queue ordered by ascending
func NewFIFOQueue() *FIFOQueue {
	return &FIFOQueue{
		queue: []IElement{},
	}
}

// Queue get all queue data
func (q *FIFOQueue) Queue() []IElement {
	if nil == q.queue {
		q.queue = []IElement{}
	}
	return q.queue
}

// Push item
func (q *FIFOQueue) Push(item IElement) bool {
	if nil == q.queue {
		q.queue = []IElement{item}
	} else {
		q.queue = append(q.queue, item)
	}
	return true
}

// Pop first item
func (q *FIFOQueue) Pop() (interface{}, bool) {
	if len(q.queue) <= 0 {
		return nil, false
	}
	item := q.queue[0]
	q.queue = append([]IElement{}, q.queue[1:]...)
	return item, true
}

// First item without pop
func (q *FIFOQueue) First() (interface{}, bool) {
	if len(q.queue) <= 0 {
		return nil, false
	}
	item := q.queue[0]
	return item, true
}

// Remove an element from queue identified by element.GetID()
func (q *FIFOQueue) Remove(item IElement) bool {
	for i, e := range q.queue {
		if e.GetID() == item.GetID() {
			q.queue = append(q.queue[0:i], q.queue[i+1:]...)
			return true
		}
	}
	return false
}

// Elements of all queue
func (q *FIFOQueue) Elements() []IElement {
	return q.queue
}

// Dump element in queue
func (q *FIFOQueue) Dump() string {
	result := []string{}
	for _, e := range q.queue {
		result = append(result, e.DebugString())
	}
	return strings.Join(result, " ")
}

// GetOne func
func (q *FIFOQueue) GetOne(item IElement) (interface{}, bool) {
	// fmt.Printf("Removing element %s finding...\n", item.GetID())
	for _, e := range q.queue {
		if e.GetID() == item.GetID() {
			return item, true
		}
	}
	return nil, false
}

// GetElement get element by id
func (q *FIFOQueue) GetElement(ID string) (interface{}, bool) {
	for _, e := range q.queue {
		if e.GetID() == ID {
			return e, true
		}
	}
	return nil, false
}

// CutBefore cut elements out before index
func (q *FIFOQueue) CutBefore(idx int) []IElement {
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
func (q *FIFOQueue) CutAfter(idx int) []IElement {
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
