package rdbms

import (
	"fmt"
	"reflect"

	"github.com/kevinyjn/gocom/logger"
)

// Datasource an struct that specifies datasource name
// the database schema model should be like this:
// ```
// type SchemaDemo struct {
//     Name string      `xorm:"'name' VARCHAR(50) default('')"`
//     rdbms.Datasource `xorm:"'-' datasource:"demo"`
// }
// ```
type Datasource struct {
}

// the bellow 3 interfaces gives another way to specify datasource on schema model,
// the model can optionally implements one of them

// IDatasource interface
type IDatasource interface {
	Datasource() string
}

// IDatasourceName interface
type IDatasourceName interface {
	DatasourceName() string
}

// IGetDatasource interface
type IGetDatasource interface {
	GetDatasource() string
}

// Fetch retrieve record data from table, bean's non-empty fields are conditions
func (s *Datasource) Fetch(bean interface{}) (bool, error) {
	if reflect.TypeOf(bean).Kind() != reflect.Ptr {
		logger.Error.Printf("Fetching bean:%+v failed, the operation needs a pointer passive", bean)
		return false, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().FetchOne(bean)
}

// Save record data to table
func (s *Datasource) Save(bean interface{}) (bool, error) {
	if reflect.TypeOf(bean).Kind() != reflect.Ptr {
		logger.Error.Printf("Saving bean:%+v failed, the operation needs a pointer passive", bean)
		return false, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().SaveOne(bean)
}

// Exists by record
func (s *Datasource) Exists(bean interface{}) (bool, error) {
	if reflect.TypeOf(bean).Kind() != reflect.Ptr {
		logger.Error.Printf("Exists checking bean:%+v failed, the operation needs a pointer passive", bean)
		return false, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().Exists(bean)
}

// Count record
func (s *Datasource) Count(bean interface{}) (int64, error) {
	if reflect.TypeOf(bean).Kind() != reflect.Ptr {
		logger.Error.Printf("Count bean:%+v failed, the operation needs a pointer passive", bean)
		return 0, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().Count(bean)
}

// Insert record data to table
func (s *Datasource) Insert(beans ...interface{}) (int64, error) {
	if len(beans) <= 0 {
		logger.Error.Printf("Insert records by passing no records")
		return 0, fmt.Errorf("Passing no records")
	}
	if reflect.TypeOf(beans[0]).Kind() != reflect.Ptr {
		logger.Error.Printf("Inserting bean:%+v failed, the operation needs a pointer passive", beans)
		return 0, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().Insert(beans...)
}

// InsertMulti records data to table
func (s *Datasource) InsertMulti(beans []interface{}) (int64, error) {
	if len(beans) <= 0 {
		logger.Error.Printf("Insert records by passing no records")
		return 0, fmt.Errorf("Passing no records")
	}
	if reflect.TypeOf(beans[0]).Kind() != reflect.Ptr {
		logger.Error.Printf("Inserting multi beans failed, the operation needs a pointer passive")
		return 0, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().InsertMulti(beans)
}

// Delete record
func (s *Datasource) Delete(bean interface{}) (int64, error) {
	if reflect.TypeOf(bean).Kind() != reflect.Ptr {
		logger.Error.Printf("Delete bean:%+v failed, the operation needs a pointer passive", bean)
		return 0, fmt.Errorf("Non pointer bean")
	}
	return GetInstance().Delete(bean)
}
