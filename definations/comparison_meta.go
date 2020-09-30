package definations

import (
	"reflect"
	"strings"
)

// CompareType type
type CompareType int

// Constants
const (
	CompareEquals        = CompareType(0)
	ConpareNotEquals     = CompareType(1)
	CompareLessThan      = CompareType(10)
	CompareLessEquals    = CompareType(11)
	CompareGreaterThan   = CompareType(12)
	CompareGreaterEquals = CompareType(13)
	CompareContains      = CompareType(21)
	CompareInArray       = CompareType(22)
	CompareNotInArray    = CompareType(23)
	CompareBetween       = CompareType(24)
	CompareNotBetween    = CompareType(25)
)

// comparisonMeta struct
type comparisonMeta struct {
	Comparison CompareType
	Field      string
	Value      interface{}
}

// ComparisonObject struct
type ComparisonObject struct {
	ands             []comparisonMeta
	ors              []comparisonMeta
	nestedComparison *ComparisonObject
}

// NewComparisonObject new comparison
func NewComparisonObject() *ComparisonObject {
	return &ComparisonObject{}
}

// And contion
func (c *ComparisonObject) And(compareType CompareType, field string, value interface{}) *ComparisonObject {
	if nil == c.ands {
		c.ands = []comparisonMeta{}
	}
	c.ands = append(c.ands, comparisonMeta{compareType, field, value})
	return c
}

// Or condition
func (c *ComparisonObject) Or(compareType CompareType, field string, value interface{}) *ComparisonObject {
	if nil == c.ors {
		c.ors = []comparisonMeta{}
	}
	c.ors = append(c.ors, comparisonMeta{compareType, field, value})
	return c
}

// Evaluate the comparison
func (c *ComparisonObject) Evaluate(element interface{}) bool {
	ev := reflect.ValueOf(element)
	var r bool = false
	if ev.IsValid() && ev.Type().Kind() == reflect.Ptr {
		ev = ev.Elem()
	}
	if false == ev.IsValid() || ev.Type().Kind() != reflect.Struct {
		return r
	}
	if nil != c.ands {
		r = true
		for _, a := range c.ands {
			fv := ev.FieldByName(a.Field)
			if fv.IsValid() == false {
				r = false
				break
			}
			if a.compareValue(fv) == false {
				r = false
				break
			}
		}
		if false == r {
			return r
		}
	}
	if nil != c.ors {
		r = false
		for _, o := range c.ors {
			fv := ev.FieldByName(o.Field)
			if fv.IsValid() && o.compareValue(fv) {
				r = true
				break
			}
		}
		if false == r {
			return r
		}
	}
	if nil != c.nestedComparison {
		r = c.nestedComparison.Evaluate(element)
	}
	return r
}

// compareValue with value
func (c *comparisonMeta) compareValue(value reflect.Value) bool {
	return compareValue(value, c.Value, c.Comparison)
}

func compareValue(value reflect.Value, right interface{}, cmp CompareType) bool {
	var r bool
	switch value.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		r = compareInt(cmp, value, right)
		break
	case reflect.String:
		r = compareString(cmp, value, right)
		break
	case reflect.Float32, reflect.Float64:
		r = compareFloat(cmp, value, right)
		break
	default:
		break
	}
	return r
}

func compareInt(cmp CompareType, value reflect.Value, right interface{}) bool {
	var r bool
	left := value.Int()
	rv := reflect.ValueOf(right)
	switch cmp {
	case CompareEquals:
		r = isIntKind(rv.Type().Kind()) && left == rv.Int()
		break
	case ConpareNotEquals:
		r = isIntKind(rv.Type().Kind()) && left != rv.Int()
		break
	case CompareLessThan:
		r = isIntKind(rv.Type().Kind()) && left < rv.Int()
		break
	case CompareLessEquals:
		r = isIntKind(rv.Type().Kind()) && left <= rv.Int()
		break
	case CompareGreaterThan:
		r = isIntKind(rv.Type().Kind()) && left > rv.Int()
		break
	case CompareGreaterEquals:
		r = isIntKind(rv.Type().Kind()) && left >= rv.Int()
		break
	case CompareContains, CompareInArray:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) {
			rl := rv.Len()
			for i := 0; i < rl; i++ {
				ev := rv.Index(i)
				if ev.IsValid() && isIntKind(ev.Type().Kind()) {
					if left == ev.Int() {
						r = true
						break
					}
				}
			}
		}
		break
	case CompareNotInArray:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) {
			r = true
			rl := rv.Len()
			for i := 0; i < rl; i++ {
				ev := rv.Index(i)
				if ev.IsValid() && isIntKind(ev.Type().Kind()) {
					if left == ev.Int() {
						r = false
						break
					}
				}
			}
		} else {
			r = true
		}
		break
	case CompareBetween:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) && rv.Len() > 1 {
			ev1 := rv.Index(0)
			ev2 := rv.Index(1)
			if ev1.IsValid() && isIntKind(ev1.Type().Kind()) && ev2.IsValid() && isIntKind(ev2.Type().Kind()) {
				if left >= ev1.Int() && left <= ev2.Int() {
					r = true
				}
			}
		}
		break
	case CompareNotBetween:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) && rv.Len() > 1 {
			ev1 := rv.Index(0)
			ev2 := rv.Index(1)
			if ev1.IsValid() && isIntKind(ev1.Type().Kind()) && ev2.IsValid() && isIntKind(ev2.Type().Kind()) {
				if left >= ev1.Int() && left <= ev2.Int() {
					r = false
				} else {
					r = true
				}
			}
		}
		break
	default:
		break
	}
	return r
}

func isIntKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

func isFloatKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func isArrayKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Array, reflect.Slice:
		return true
	}
	return false
}

func compareString(cmp CompareType, value reflect.Value, right interface{}) bool {
	var r bool
	left := value.String()
	rv := reflect.ValueOf(right)
	switch cmp {
	case CompareEquals:
		r = rv.Type().Kind() == reflect.String && strings.Compare(left, rv.String()) == 0
		break
	case ConpareNotEquals:
		r = rv.Type().Kind() == reflect.String && strings.Compare(left, rv.String()) != 0
		break
	case CompareLessThan:
		r = rv.Type().Kind() == reflect.String && strings.Compare(left, rv.String()) < 0
		break
	case CompareLessEquals:
		r = rv.Type().Kind() == reflect.String && strings.Compare(left, rv.String()) <= 0
		break
	case CompareGreaterThan:
		r = rv.Type().Kind() == reflect.String && strings.Compare(left, rv.String()) > 0
		break
	case CompareGreaterEquals:
		r = rv.Type().Kind() == reflect.String && strings.Compare(left, rv.String()) >= 0
		break
	case CompareContains:
		r = rv.Type().Kind() == reflect.String && strings.Contains(rv.String(), left)
		break
	case CompareInArray:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) {
			rl := rv.Len()
			for i := 0; i < rl; i++ {
				ev := rv.Index(i)
				if ev.IsValid() && ev.Type().Kind() == reflect.String {
					if strings.Compare(left, ev.String()) == 0 {
						r = true
						break
					}
				}
			}
		}
		break
	case CompareNotInArray:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) {
			r = true
			for i := 0; i < rv.Len(); i++ {
				ev := rv.Index(i)
				if ev.IsValid() && ev.Type().Kind() == reflect.String {
					if strings.Compare(left, ev.String()) == 0 {
						r = false
						break
					}
				}
			}
		} else {
			r = true
		}
		break
	case CompareBetween:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) && rv.Len() > 1 {
			ev1 := rv.Index(0)
			ev2 := rv.Index(1)
			if ev1.IsValid() && ev1.Type().Kind() == reflect.String && ev2.IsValid() && ev2.Type().Kind() == reflect.String {
				if strings.Compare(left, ev1.String()) >= 0 && strings.Compare(left, ev2.String()) <= 0 {
					r = true
				}
			}
		}
		break
	case CompareNotBetween:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) && rv.Len() > 1 {
			ev1 := rv.Index(0)
			ev2 := rv.Index(1)
			if ev1.IsValid() && ev1.Type().Kind() == reflect.String && ev2.IsValid() && ev2.Type().Kind() == reflect.String {
				if strings.Compare(left, ev1.String()) >= 0 && strings.Compare(left, ev2.String()) <= 0 {
					r = false
				} else {
					r = true
				}
			}
		}
		break
	default:
		break
	}
	return r
}

func compareFloat(cmp CompareType, value reflect.Value, right interface{}) bool {
	var r bool
	left := value.Float()
	rv := reflect.ValueOf(right)
	switch cmp {
	case CompareEquals:
		r = isFloatKind(rv.Type().Kind()) && left == rv.Float()
		break
	case ConpareNotEquals:
		r = isFloatKind(rv.Type().Kind()) && left != rv.Float()
		break
	case CompareLessThan:
		r = isFloatKind(rv.Type().Kind()) && left < rv.Float()
		break
	case CompareLessEquals:
		r = isFloatKind(rv.Type().Kind()) && left <= rv.Float()
		break
	case CompareGreaterThan:
		r = isFloatKind(rv.Type().Kind()) && left > rv.Float()
		break
	case CompareGreaterEquals:
		r = isFloatKind(rv.Type().Kind()) && left >= rv.Float()
		break
	case CompareContains, CompareInArray:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) {
			for i := 0; i < rv.Len(); i++ {
				ev := rv.Index(i)
				if ev.IsValid() && isFloatKind(ev.Type().Kind()) {
					if left == ev.Float() {
						r = true
						break
					}
				}
			}
		}
		break
	case CompareNotInArray:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) {
			r = true
			for i := 0; i < rv.Len(); i++ {
				ev := rv.Index(i)
				if ev.IsValid() && isFloatKind(ev.Type().Kind()) {
					if left == ev.Float() {
						r = false
						break
					}
				}
			}
		} else {
			r = true
		}
		break
	case CompareBetween:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) && rv.Len() > 1 {
			ev1 := rv.Index(0)
			ev2 := rv.Index(1)
			if ev1.IsValid() && isFloatKind(ev1.Type().Kind()) && ev2.IsValid() && isFloatKind(ev2.Type().Kind()) {
				if left >= ev1.Float() && left <= ev2.Float() {
					r = true
				}
			}
		}
		break
	case CompareNotBetween:
		if rv.IsValid() && isArrayKind(rv.Type().Kind()) && rv.Len() > 1 {
			ev1 := rv.Index(0)
			ev2 := rv.Index(1)
			if ev1.IsValid() && isFloatKind(ev1.Type().Kind()) && ev2.IsValid() && isFloatKind(ev2.Type().Kind()) {
				if left >= ev1.Float() && left <= ev2.Float() {
					r = false
				} else {
					r = true
				}
			}
		}
		break
	default:
		break
	}
	return r
}
