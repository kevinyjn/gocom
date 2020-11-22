package queues

import "github.com/kevinyjn/gocom/definations"

// IElement queue element
type IElement interface {
	GetID() string
	GetName() string
	OrderingValue() int64
	DebugString() string
}

// IQueue interface
type IQueue interface {
	// Get an element from queue identified by element.GetID()
	GetOne(IElement) (interface{}, bool)
	// Pop first element from queue, the element would be deleted from queue
	Pop() (interface{}, bool)
	// PopMany head elements from queue limited by maxResults, the element would be deleted from queue
	PopMany(maxResults int) ([]interface{}, bool)
	// First element of queue would be returned, the element would not be deleted from queue
	First() (interface{}, bool)
	// Push an element into queue
	Push(IElement) bool
	// Remove an element from queue identified by element.GetID()
	Remove(IElement) bool
	// Elements of all queue
	Elements() []IElement
	// FindElements by compaire condition
	FindElements(*definations.ComparisonObject) []IElement
	//Dump all elements from queue
	Dump() string
	// GetElement
	GetElement(ID string) (interface{}, bool)
	// CutBefore cut elements out before index
	CutBefore(idx int) []IElement
	// CutAfter cut elements out after index
	CutAfter(idx int) []IElement
}
