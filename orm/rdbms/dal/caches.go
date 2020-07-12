package dal

import (
	"fmt"
	"reflect"
	"time"

	"github.com/kevinyjn/gocom/queues"
	"github.com/kevinyjn/gocom/validator/validates"
	"xorm.io/core"
)

// Constants
const (
	DefaultCachingDurationSeconds = 300
)

// dataCaches cacher
type dataCaches struct {
	elementGroups map[string]*cacheElementGroup // map[schemaStructureFullName]*cacheElementGroup
}

type cacheElementGroup struct {
	cachingDurationSeconds int64
	objects                map[string]*cacheElement // map[conditionBean]*cacheElement
	timerObjects           queues.IQueue
	timeoutsTicker         *time.Ticker
	pkColumns              []*core.Column
}

type cacheElement struct {
	expiresAt int64
	key       string
	depKey    string
	refKeys   map[string]bool
	data      interface{}
}

var _caches = &dataCaches{
	elementGroups: map[string]*cacheElementGroup{},
}

func getCaches() *dataCaches {
	return _caches
}

func (c *dataCaches) group(schemaStructureFullName string) *cacheElementGroup {
	eg := c.elementGroups[schemaStructureFullName]
	if nil == eg {
		eg = &cacheElementGroup{
			cachingDurationSeconds: DefaultCachingDurationSeconds,
			objects:                map[string]*cacheElement{},
			timerObjects:           queues.NewAscOrderingQueue(),
			timeoutsTicker:         nil,
			pkColumns:              nil,
		}
		c.elementGroups[schemaStructureFullName] = eg
		go eg.runTimeoutsCheck()
	}
	return eg
}

func (c *cacheElementGroup) set(bean interface{}, condiBean interface{}, pkCondiBean interface{}) {
	key := c.serializeCondition(condiBean)
	pkKey := c.serializeCondition(pkCondiBean)
	now := time.Now().Unix()
	if len(c.pkColumns) <= 0 && "" == pkKey {
		c.objects = map[string]*cacheElement{}
		c.timerObjects = queues.NewAscOrderingQueue()
	}
	existsEle := c.objects[key]
	if nil != existsEle {
		// clean timer queue ralated element
		c.timerObjects.Remove(existsEle)
	}
	ele := &cacheElement{
		expiresAt: now + c.cachingDurationSeconds,
		key:       key,
		data:      bean,
	}
	if "" == pkKey || pkKey == key {
		// Many times would caused by saving record
		if nil != existsEle && nil != existsEle.refKeys {
			for k := range existsEle.refKeys {
				existsEle2 := c.objects[k]
				if nil != existsEle2 {
					delete(c.objects, k)
					c.timerObjects.Remove(existsEle2)
				}
			}
		}
		ele.refKeys = map[string]bool{}
	} else {
		existsEle = c.objects[pkKey]
		var refKeys map[string]bool
		if nil != existsEle {
			refKeys = existsEle.refKeys
			if nil == refKeys {
				refKeys = map[string]bool{}
			}
			c.timerObjects.Remove(existsEle)
		} else {
			refKeys = map[string]bool{}
		}
		refKeys[key] = true
		pkEle := &cacheElement{
			expiresAt: now + c.cachingDurationSeconds,
			key:       pkKey,
			data:      bean,
			refKeys:   refKeys,
		}
		c.objects[pkKey] = pkEle
		c.timerObjects.Push(pkEle)

		ele.data = nil
		ele.depKey = pkKey
	}
	c.objects[key] = ele
	c.timerObjects.Push(ele)
}

func (c *cacheElementGroup) get(condiBean interface{}) (interface{}, string, bool) {
	key := c.serializeCondition(condiBean)
	e := c.objects[key]
	now := time.Now().Unix()
	if nil == e {
		return nil, key, false
	} else if now > e.expiresAt {
		delete(c.objects, key)
		return nil, key, false
	}
	if "" == e.depKey {
		return e.data, key, true
	}
	pkEle := c.objects[e.depKey]
	if nil == pkEle {
		return nil, key, false
	} else if now > pkEle.expiresAt {
		delete(c.objects, e.depKey)
		delete(c.objects, key)
		return nil, key, false
	}
	return pkEle.data, key, true
}

func (c *cacheElementGroup) del(condiBean interface{}) {
	key := c.serializeCondition(condiBean)
	e := c.objects[key]
	if nil != e {
		delete(c.objects, key)
		c.timerObjects.Remove(e)
	}
}

func (c *cacheElementGroup) serializeCondition(condiBean interface{}) string {
	if nil == condiBean {
		return ""
	}
	val := reflect.ValueOf(condiBean)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Type().Kind() == reflect.Struct {
		return _searlizeConditionBean(val)
	}
	return fmt.Sprintf("%v", condiBean)
}

func _searlizeConditionBean(condiBeanValue reflect.Value) string {
	var key string
	if condiBeanValue.Type().Kind() == reflect.Struct {
		fl := condiBeanValue.Type().NumField()
		d := map[string]interface{}{}
		for i := 0; i < fl; i++ {
			f := condiBeanValue.Field(i)
			if f.IsValid() == false {
				continue
			}
			if f.Type().Kind() == reflect.Struct {
				if f.Type().String() == "time.Time" {
					continue
				}
				ft := condiBeanValue.Type().Field(i)
				tagName := ft.Tag.Get("xorm")
				if "" == tagName || "-" == tagName {
					continue
				}
				skey := _searlizeConditionBean(f)
				if "" != skey {
					d[ft.Name] = skey
				}
			} else if validates.ValidateRequired(f, "") == nil {
				d[condiBeanValue.Type().Field(i).Name] = f.Interface()
			}
		}
		if len(d) > 0 {
			key = fmt.Sprintf("%v", d)
		}
	} else {
		key = fmt.Sprintf("%v", condiBeanValue.Interface())
	}
	return key
}

func (c *cacheElementGroup) runTimeoutsCheck() {
	if nil != c.timeoutsTicker {
		return
	}
	c.timeoutsTicker = time.NewTicker(time.Duration(DefaultCachingDurationSeconds / 2))
	for nil != c.timeoutsTicker {
		select {
		case tim := <-c.timeoutsTicker.C:
			c.cleanTimeouts(tim)
			break
		}
	}
}

func (c *cacheElementGroup) cleanTimeouts(tim time.Time) {
	now := tim.Unix()
	timeoutsIndex := 0
	for i, e := range c.timerObjects.Elements() {
		if e.OrderingValue() > now {
			timeoutsIndex = i
			break
		}
	}
	if 0 < timeoutsIndex {
		cuts := c.timerObjects.CutBefore(timeoutsIndex)
		for _, e := range cuts {
			delete(c.objects, e.GetID())
		}
	}
}

// GetID caching key
func (e *cacheElement) GetID() string {
	return e.key
}

// GetName get name
func (e *cacheElement) GetName() string {
	return e.key
}

// OrderingValue get expire time
func (e *cacheElement) OrderingValue() int64 {
	return e.expiresAt
}

// DebugString text
func (e *cacheElement) DebugString() string {
	return fmt.Sprintf("caching element %s expires:%d data: %v", e.key, e.expiresAt, e.data)
}

// GetData value
func (e *cacheElement) GetData() interface{} {
	return e.data
}
