package rdbms

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/queues"
	"github.com/kevinyjn/gocom/utils"
	"github.com/kevinyjn/gocom/validator/validates"
	"xorm.io/xorm/schemas"
)

// Constants
const (
	DefaultCachingDurationSeconds  = 300
	MaxArrayCachingDurationSeconds = 180
)

// dataCaches cacher
type dataCaches struct {
	elementGroups map[string]*cacheElementGroup // map[schemaStructureFullName]*cacheElementGroup
	mutex         sync.RWMutex
}

var _caches = &dataCaches{
	elementGroups: map[string]*cacheElementGroup{},
	mutex:         sync.RWMutex{},
}

type cacheElementGroup struct {
	cachingDurationSeconds int64
	objects                *cacheElementWrapper // map[conditionBean]*cacheElement
	arrayObjects           *cacheElementWrapper // map[conditionBean]*cacheElement{data:[]rowPrimaryKeyText}
	timeoutsTicker         *time.Ticker
	pkColumns              []*schemas.Column
}

type cacheElement struct {
	expiresAt int64
	key       string
	depKey    string
	refKeys   map[string]bool
	data      interface{}
}

type cacheElementWrapper struct {
	objects      map[string]*cacheElement // map[conditionBean]*cacheElement
	timerObjects queues.IQueue
	mutex        sync.RWMutex
}

func newCacheElementWrapper() *cacheElementWrapper {
	return &cacheElementWrapper{
		objects:      map[string]*cacheElement{},
		timerObjects: queues.NewAscOrderingQueue(),
		mutex:        sync.RWMutex{},
	}
}

func (w *cacheElementWrapper) reset() {
	w.mutex.Lock()
	w.objects = map[string]*cacheElement{}
	w.timerObjects = queues.NewAscOrderingQueue()
	w.mutex.Unlock()
}

func (w *cacheElementWrapper) get(key string, removeTimerIfExists bool) *cacheElement {
	w.mutex.RLock()
	item := w.objects[key]
	w.mutex.RUnlock()
	if removeTimerIfExists && nil != item {
		w.mutex.Lock()
		delete(w.objects, key)
		w.timerObjects.Remove(item)
		w.mutex.Unlock()
	}
	return item
}

func (w *cacheElementWrapper) remove(key string) *cacheElement {
	w.mutex.RLock()
	item := w.objects[key]
	w.mutex.RUnlock()
	if nil != item {
		w.mutex.Lock()
		delete(w.objects, key)
		w.timerObjects.Remove(item)
		w.mutex.Unlock()
	}
	return item
}

func (w *cacheElementWrapper) set(key string, item *cacheElement) {
	w.mutex.Lock()
	w.objects[key] = item
	w.timerObjects.Push(item)
	w.mutex.Unlock()
}

func (w *cacheElementWrapper) cleanTimeouts(now int64) {
	w.mutex.Lock()
	timeoutsIndex := 0
	for i, e := range w.timerObjects.Elements() {
		if e.OrderingValue() > now {
			timeoutsIndex = i
			break
		}
	}
	if 0 < timeoutsIndex {
		cuts := w.timerObjects.CutBefore(timeoutsIndex)
		for _, e := range cuts {
			delete(w.objects, e.GetID())
		}
	}
	w.mutex.Unlock()
}

func getCaches() *dataCaches {
	return _caches
}

func (c *dataCaches) group(schemaStructureFullName string) *cacheElementGroup {
	c.mutex.RLock()
	eg := c.elementGroups[schemaStructureFullName]
	c.mutex.RUnlock()
	if nil == eg {
		eg = &cacheElementGroup{
			cachingDurationSeconds: DefaultCachingDurationSeconds,
			objects:                newCacheElementWrapper(),
			arrayObjects:           newCacheElementWrapper(),
			timeoutsTicker:         nil,
			pkColumns:              nil,
		}
		c.mutex.Lock()
		c.elementGroups[schemaStructureFullName] = eg
		c.mutex.Unlock()
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
		c.objects.reset()
	}
	existsEle := c.objects.get(key, true)
	ele := &cacheElement{
		expiresAt: now + c.cachingDurationSeconds,
		key:       key,
		data:      setBean,
	}
	if "" == pkKey || pkKey == key {
		// Many times would caused by saving record
		if nil != existsEle && nil != existsEle.refKeys {
			for k := range existsEle.refKeys {
				c.objects.remove(k)
			}
		}
		ele.refKeys = map[string]bool{}
	} else {
		existsEle = c.objects.get(pkKey, false)
		var refKeys map[string]bool
		if nil != existsEle {
			refKeys = existsEle.refKeys
			if nil == refKeys {
				refKeys = map[string]bool{}
			}
			c.objects.remove(pkKey)
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
		c.objects.set(pkKey, pkEle)

		ele.data = nil
		ele.depKey = pkKey
	}
	c.objects.set(key, ele)
}

func (c *cacheElementGroup) get(condiBean interface{}, cloneResult bool) (interface{}, string, bool) {
	key := c.serializeCondition(condiBean)
	e := c.objects.get(key, false)
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
		pkEle = c.objects.get(e.depKey, false)
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
				c.objects.remove(key)
			}
		} else {
			c.objects.remove(key)
			c.objects.remove(e.depKey)
		}
	}
	return bean, key, ok
}

func (c *cacheElementGroup) del(condiBean interface{}) {
	key := c.serializeCondition(condiBean)
	e := c.objects.get(key, false)
	if nil != e {
		if len(e.refKeys) > 0 {
			for k := range e.refKeys {
				c.objects.remove(k)
			}
		}
		c.objects.remove(key)
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
	c.arrayObjects.get(key, true)
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
	c.arrayObjects.set(key, ele)
}

// getArray records from cache
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies. while, this case would not
// be commonly comes out because that the new record would be inserted at the tail of rows in many
// databases
func (c *cacheElementGroup) getArray(condiBean interface{}, records *[]interface{}, limit, offset int) (string, bool) {
	var ok bool = false
	key := fmt.Sprintf("%d:%d:%s", limit, offset, c.serializeCondition(condiBean))
	e := c.arrayObjects.get(key, false)
	now := time.Now().Unix()
	if nil == e {
		return key, ok
	} else if now > e.expiresAt {
		c.arrayObjects.remove(key)
		return key, ok
	}
	pkeys := e.data.([]string)
	*records = make([]interface{}, len(pkeys))
	ok = true
	var beanType reflect.Type
	for i, pkey := range pkeys {
		one := c.objects.get(pkey, false)
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
	c.timeoutsTicker = time.NewTicker(time.Duration(DefaultCachingDurationSeconds/2) * time.Second)
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
	c.objects.cleanTimeouts(now)
	c.arrayObjects.cleanTimeouts(now)
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
