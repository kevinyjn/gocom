package testingutil

import "testing"

// AssertEquals asserts
func AssertEquals(t *testing.T, expected interface{}, value interface{}, valueName string) bool {
	if value == expected {
		return true
	}
	t.Fatalf("validate values %s:%+v not equals expected:%+v", valueName, value, expected)
	return false
}

// AssertNotEquals asserts
func AssertNotEquals(t *testing.T, expected interface{}, value interface{}, valueName string) bool {
	if value != expected {
		return true
	}
	t.Fatalf("validate values %s:%+v equals not expected:%+v", valueName, value, expected)
	return false
}

// AssertTrue asserts
func AssertTrue(t *testing.T, val bool, valueName string) bool {
	if val {
		return true
	}
	t.Fatalf("validate %s not true", valueName)
	return false
}

// AssertFalse asserts
func AssertFalse(t *testing.T, val bool, valueName string) bool {
	if false == val {
		return true
	}
	t.Fatalf("validate %s not false", valueName)
	return false
}

// AssertNil asserts
func AssertNil(t *testing.T, val interface{}, valueName string) bool {
	if nil == val {
		return true
	}
	t.Fatalf("validate %s not nil: %+v", valueName, val)
	return false
}

// AssertNotNil asserts
func AssertNotNil(t *testing.T, val interface{}, valueName string) bool {
	if nil != val {
		return true
	}
	t.Fatalf("validate %s is nil", valueName)
	return false
}
