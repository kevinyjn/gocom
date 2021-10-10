package utils

import (
	"sync"
)

// CoMapData map data for concurrency coroutine operation
type CoMapData struct {
	data map[string]interface{}
	m    sync.RWMutex
}

// NewCoMapData new a concurrency coroutine operational map data
func NewCoMapData() *CoMapData {
	return &CoMapData{
		data: map[string]interface{}{},
	}
}

// Add element
func (m *CoMapData) Add(key string, value interface{}) {
	m.m.Lock()
	if nil == m.data {
		m.data = map[string]interface{}{}
	}
	m.data[key] = value
	m.m.Unlock()
}

// Remove element
func (m *CoMapData) Delete(key string) {
	m.m.Lock()
	if nil != m.data {
		delete(m.data, key)
	}
	m.m.Unlock()
}

// Exists if element exists by key
func (m *CoMapData) Exists(key string) bool {
	m.m.RLock()
	defer m.m.RUnlock()
	if nil != m.data {
		_, ok := m.data[key]
		return ok
	}
	return false
}

// Get element by key
func (m *CoMapData) Get(key string) (interface{}, bool) {
	m.m.RLock()
	defer m.m.RUnlock()
	if nil == m.data {
		return nil, false
	}
	val, ok := m.data[key]
	return val, ok
}
