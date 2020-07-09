package dal

import (
	"reflect"

	"github.com/kevinyjn/gocom/logger"
)

// Datasource an struct that specifies datasource name
// the database schema model should be like this:
// ```
// type SchemaDemo struct {
//     Name string    `xorm:"'name' VARCHAR(50) default('')"`
//     dal.Datasource `xorm:"'-' datasource:"demo"`
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
		logger.Error.Printf("Fetching bean:%v failed, the operation needs a pointer passive", bean)
	}
	return GetInstance().FetchOne(bean)
}

// Save record data to table
func (s *Datasource) Save(bean interface{}) (bool, error) {
	if reflect.TypeOf(bean).Kind() != reflect.Ptr {
		logger.Error.Printf("Saving bean:%v failed, the operation needs a pointer passive", bean)
	}
	return GetInstance().SaveOne(bean)
}
