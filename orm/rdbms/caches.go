package rdbms

import (
	"fmt"
	"reflect"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/queues"
	"github.com/kevinyjn/gocom/utils"
	"github.com/kevinyjn/gocom/validator/validates"
	"xorm.io/core"
)

// Constants
const (
	DefaultCachingDurationSeconds  = 300
	MaxArrayCachingDurationSeconds = 180
)

// dataCaches cacher
type dataCaches struct {
	elementGroups map[string]*cacheElementGroup // map[schemaStructureFullName]*cacheElementGroup
}

type cacheElementGroup struct {
	cachingDurationSeconds int64
	objects                map[string]*cacheElement // map[conditionBean]*cacheElement
	arrayObjects           map[string]*cacheElement // map[conditionBean]*cacheElement{data:[]rowPrimaryKeyText}
	timerObjects           queues.IQueue
	arrayTimerObjects      queues.IQueue
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
			arrayObjects:           map[string]*cacheElement{},
			timerObjects:           queues.NewAscOrderingQueue(),
			arrayTimerObjects:      queues.NewAscOrderingQueue(),
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
	setBean, err := cloneBean(bean)
	if nil != err {
		return
	}
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
		data:      setBean,
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
			data:      setBean,
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

func (c *cacheElementGroup) get(condiBean interface{}, cloneResult bool) (interface{}, string, bool) {
	key := c.serializeCondition(condiBean)
	e := c.objects[key]
	now := time.Now().Unix()
	var bean interface{} = nil
	var err error
	var ok bool = false
	var pkEle *cacheElement = nil
	for {
		if nil == e || now > e.expiresAt {
			break
		}
		if "" == e.depKey {
			if cloneResult {
				bean, err = cloneBean(e.data)
				if nil != err {
					break
				}
			} else {
				bean = e.data
			}
			ok = true
			break
		}
		pkEle = c.objects[e.depKey]
		if nil == pkEle || now > pkEle.expiresAt {
			break
		}
		if cloneResult {
			bean, err = cloneBean(pkEle.data)
			if nil != err {
				break
			}
		} else {
			bean = pkEle.data
		}
		ok = true
		break
	}
	if false == ok {
		if nil == pkEle {
			if nil != e {
				c.timerObjects.Remove(e)
				delete(c.objects, key)
			}
		} else {
			c.timerObjects.Remove(e)
			c.timerObjects.Remove(pkEle)
			delete(c.objects, e.depKey)
			delete(c.objects, key)
		}
	}
	return bean, key, ok
}

func (c *cacheElementGroup) del(condiBean interface{}) {
	key := c.serializeCondition(condiBean)
	e := c.objects[key]
	if nil != e {
		if len(e.refKeys) > 0 {
			for k := range e.refKeys {
				re := c.objects[k]
				if nil != re {
					delete(c.objects, k)
					c.timerObjects.Remove(re)
				}
			}
		}
		delete(c.objects, key)
		c.timerObjects.Remove(e)
	}
}

func (c *cacheElementGroup) setArray(condiBean interface{}, beans []interface{}, limit, offset int) {
	arrayCachingDurationSeconds := c.cachingDurationSeconds - 30
	if len(c.pkColumns) <= 0 || len(beans) <= 0 || arrayCachingDurationSeconds <= 0 {
		return
	} else if MaxArrayCachingDurationSeconds < arrayCachingDurationSeconds {
		arrayCachingDurationSeconds = MaxArrayCachingDurationSeconds
	}
	var pkCondiBean interface{}
	var pkValues int
	var err error
	key := fmt.Sprintf("%d:%d:%s", limit, offset, c.serializeCondition(condiBean))
	cacheObjects := make([]string, len(beans))
	existsEle := c.arrayObjects[key]
	if nil != existsEle {
		// clean timer queue ralated element
		c.arrayTimerObjects.Remove(existsEle)
	}
	for i, bean := range beans {
		pkCondiBean, pkValues, err = formatPkConditionBeanImpl(c.pkColumns, reflect.ValueOf(bean).Elem())
		if 0 >= pkValues {
			logger.Error.Printf("caching records while format primary key contidion on record got empty condition with error:%v", err)
			return
		}
		c.set(bean, pkCondiBean, nil)
		cacheObjects[i] = c.serializeCondition(pkCondiBean)
	}
	now := time.Now().Unix()
	ele := &cacheElement{
		expiresAt: now + c.cachingDurationSeconds,
		key:       key,
		data:      cacheObjects,
	}
	c.arrayObjects[key] = ele
	c.arrayTimerObjects.Push(ele)
}

// getArray records from cache
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies. while, this case would not
// be commonly comes out because that the new record would be inserted at the tail of rows in many
// databases
func (c *cacheElementGroup) getArray(condiBean interface{}, records *[]interface{}, limit, offset int) (string, bool) {
	var ok bool = false
	key := fmt.Sprintf("%d:%d:%s", limit, offset, c.serializeCondition(condiBean))
	e := c.arrayObjects[key]
	now := time.Now().Unix()
	if nil == e {
		return key, ok
	} else if now > e.expiresAt {
		c.arrayTimerObjects.Remove(e)
		delete(c.arrayObjects, key)
		return key, ok
	}
	pkeys := e.data.([]string)
	*records = make([]interface{}, len(pkeys))
	ok = true
	var beanType reflect.Type
	for i, pkey := range pkeys {
		one := c.objects[pkey]
		if nil == one {
			ok = false
			break
		} else if 0 == i {
			beanType = getBeanType(one.data)
		}
		bean, err := cloneBeanX(one.data, beanType)
		if nil != err {
			ok = false
			break
		}
		(*records)[i] = bean
	}
	return key, ok
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
	timeoutsIndex = 0
	for i, e := range c.arrayTimerObjects.Elements() {
		if e.OrderingValue() > now {
			timeoutsIndex = i
			break
		}
	}
	if 0 < timeoutsIndex {
		cuts := c.arrayTimerObjects.CutBefore(timeoutsIndex)
		for _, e := range cuts {
			delete(c.arrayObjects, e.GetID())
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

func cloneBean(bean interface{}) (interface{}, error) {
	newBean := reflect.New(getBeanType(bean)).Interface()
	err := utils.DeeplyCopyObject(bean, newBean)
	return newBean, err
}

func cloneBeanX(bean interface{}, beanType reflect.Type) (interface{}, error) {
	newBean := reflect.New(beanType).Interface()
	err := utils.DeeplyCopyObject(bean, newBean)
	return newBean, err
}

func getBeanType(bean interface{}) reflect.Type {
	val := reflect.ValueOf(bean)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val.Type()
}

func getBeanValue(bean interface{}) reflect.Value {
	val := reflect.ValueOf(bean)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}
