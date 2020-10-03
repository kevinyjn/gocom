package rdbms

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/netutils/dboptions"
	"github.com/kevinyjn/gocom/utils"
	"github.com/kevinyjn/gocom/validator/validates"

	"github.com/go-xorm/xorm"
	"xorm.io/core"

	// justifying
	_ "github.com/denisenkom/go-mssqldb" // sqlserver
	_ "github.com/go-sql-driver/mysql"   // mysql, tidb
	_ "github.com/godror/godror"         // oracle
	_ "github.com/lib/pq"                // postgres, cockroachdb
	_ "github.com/mattn/go-sqlite3"      // sqlite
)

// Constants
const (
	DefaultDatasourceName = "default"
	DefaultSqliteFile     = "../data/default.db"
	DefaultDatabaseName   = "default"
)

// Variables
var (
	ErrNoPrimaryKeyFields    = errors.New("No primary key filed defined")
	ErrEmptyPrimaryKeyFields = errors.New("No primary key field were set")
)

// DataAccessEngine data access layer manager
type DataAccessEngine struct {
	orms                           map[string]*xorm.Engine // map[dbDatasourceName]*xorm.Engine
	defaultDatasource              string
	datasourceNamesByTableBeanName map[string]string      // map[beanNameOrTableName]datasourceName
	structuesByTableBeanName       map[string]*xorm.Table // map[beanNameOrTableName]*core.Table
	mutex                          sync.RWMutex
	mutexDatasourceName            sync.RWMutex
	mutexTable                     sync.RWMutex
}

var _dpm = &DataAccessEngine{
	orms:                           map[string]*xorm.Engine{},
	defaultDatasource:              "",
	datasourceNamesByTableBeanName: map[string]string{},
	structuesByTableBeanName:       map[string]*xorm.Table{},
	mutex:                          sync.RWMutex{},
	mutexDatasourceName:            sync.RWMutex{},
	mutexTable:                     sync.RWMutex{},
}

// GetInstance data persistence manager
func GetInstance() *DataAccessEngine {
	return _dpm
}

// Init db instance with config
func (dan *DataAccessEngine) Init(dbDatasourceName string, dbConfig *definations.DBConnectorConfig) (*xorm.Engine, error) {
	var err error
	var connData dboptions.DBConnectionData
	if nil == dbConfig || ("" == dbConfig.Driver || "" == dbConfig.Address) {
		dan.mutex.RLock()
		eng := dan.orms[dbDatasourceName]
		dan.mutex.RUnlock()
		if nil != eng {
			logger.Warning.Printf("Initializing data access engine by instance:%s with empty or invalid db config, using existing initialized engine...", dbDatasourceName)
			return eng, nil
		}
		logger.Warning.Printf("Initializing data access engine by instance:%s with empty or invalid db config, using sqlite %s initilizing...", dbDatasourceName, DefaultSqliteFile)
		dbDir := filepath.Dir(DefaultSqliteFile)
		err = utils.EnsureDirectory(dbDir)
		if nil != err {
			logger.Error.Printf("Ensure db path:%s failed with error:%v", dbDir, err)
			return nil, err
		}
		dbConfig = &definations.DBConnectorConfig{
			Driver:  "sqlite3",
			Address: "file://" + DefaultSqliteFile,
			Db:      DefaultDatabaseName,
		}
	}
	opts := dboptions.NewDBConnectionPoolOptionsWithDSN(dbConfig.Address)
	if nil != opts {
		connData, err = opts.GetConnectionData()
		if nil != err {
			logger.Warning.Printf("Parse database connection string:%s failed with error:%v", dbConfig.Address, err)
		}
	}
	if "" == connData.Driver && "" == connData.ConnString {
		connData.Driver = dbConfig.Driver
		connData.ConnString = dbConfig.Address
		connData.ConnDescription = connData.ConnString
	}

	orm, err := xorm.NewEngine(connData.Driver, connData.ConnString)
	if nil != err {
		logger.Error.Printf("New database connection engine with driver:%s and address:%s failed with error:%v", connData.Driver, connData.ConnDescription, err)
		return nil, err
	}
	orm.SetLogger(&ormLogger{logLevel: getSysLogLevel(), showSQL: logger.IsDebugEnabled()})
	orm.TZLocation = time.Local
	orm.DatabaseTZ = time.Local

	if "" != dbConfig.TablePrefix {
		tbMapper := core.NewPrefixMapper(core.SnakeMapper{}, dbConfig.TablePrefix)
		orm.SetTableMapper(tbMapper)
	}

	dan.mutex.RLock()
	exists := dan.orms[dbDatasourceName]
	dan.mutex.RUnlock()
	if nil != exists {
		exists.Close()
	}

	dan.mutex.Lock()
	dan.orms[dbDatasourceName] = orm
	dan.mutex.Unlock()
	logger.Info.Printf("Initialize database connection engine with driver:%s and address:%s succeed", connData.Driver, connData.ConnDescription)

	return orm, nil
}

func (dan *DataAccessEngine) ensureDbEngine(dbDatasourceName string) (*xorm.Engine, error) {
	dan.mutex.RLock()
	orm := dan.orms[dbDatasourceName]
	dan.mutex.RUnlock()
	if nil == orm {
		logger.Warning.Printf("ensuring database engine by instance:%s while the database instance were not successfully initialize, initializing it using default engine...", dbDatasourceName)
		return dan.Init(dbDatasourceName, nil)
	}
	return orm, nil
}

// FetchAll fetch all data object from database on conditionBean
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies.
func (dan *DataAccessEngine) FetchAll(condiBeans interface{}) ([]interface{}, error) {
	records := []interface{}{}
	structureName, orm, err := dan.getDbEngineWithStructureName(condiBeans)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	cacheGroup := getCaches().group(structureName)
	_, ok := cacheGroup.getArray(condiBeans, &records, 0, 0)
	if true == ok {
		logger.Trace.Printf("Fetching records by table:%s on condition:%+v from cache hitted", getTableName(condiBeans), condiBeans)
		return records, nil
	}
	err = queryAll(orm, condiBeans, &records, 0, 0)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	cacheGroup.setArray(condiBeans, records, 0, 0)
	return records, nil
}

// FetchRecords fetch all data object from database on conditionBean and limit, offset
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies. while, this case would not
// be commonly comes out because that the new record would be inserted at the tail of rows in many
// databases
func (dan *DataAccessEngine) FetchRecords(condiBeans interface{}, limit, offset int) ([]interface{}, error) {
	records := []interface{}{}
	if 0 >= limit || 0 > offset {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v with limit:%d offset:%d while limit should greater than 0 and offset should not less than 0", getTableName(condiBeans), condiBeans, limit, offset)
		return records, errors.New("limit should greater than 0 and offset should not less than 0")
	}
	structureName, orm, err := dan.getDbEngineWithStructureName(condiBeans)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	cacheGroup := getCaches().group(structureName)
	_, ok := cacheGroup.getArray(condiBeans, &records, limit, offset)
	if true == ok {
		logger.Trace.Printf("Fetching records by table:%s on condition:%+v from cache hitted", getTableName(condiBeans), condiBeans)
		return records, nil
	}
	err = queryAll(orm, condiBeans, &records, limit, offset)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	cacheGroup.setArray(condiBeans, records, limit, offset)
	return records, nil
}

// FetchOne Get retrieve one record from table, bean's non-empty fields are conditions.
// The bean should be a pointer to a struct
func (dan *DataAccessEngine) FetchOne(bean interface{}) (bool, error) {
	structureName, orm, err := dan.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return false, err
	}
	cacheGroup := getCaches().group(structureName)
	cachingBean, cachingKey, ok := cacheGroup.get(bean, false)
	if ok {
		err = utils.DeeplyCopyObject(cachingBean, bean)
		if nil == err {
			logger.Trace.Printf("Fetching record by table:%s on condition:%+v from cache hitted", getTableName(bean), cachingKey)
			return true, nil
		}
		logger.Error.Printf("Fetching record by table:%s on condition:%+v while copying from cached data failed with error:%v", getTableName(bean), bean, err)
	}
	condiBean := reflect.New(reflect.ValueOf(bean).Elem().Type()).Interface()
	utils.DeeplyCopyObject(bean, condiBean)
	has, err := orm.Get(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%+v failed with error:%v", getTableName(bean), bean, err)
	} else if has {
		pkCondiBean, _ := dan.getPkConditionBean(orm, structureName, bean)
		cacheGroup.set(bean, condiBean, pkCondiBean)
	} else {
		logger.Info.Printf("Fetching record by table:%s on condition:%+v while there is no such record", getTableName(bean), bean)
	}
	return has, err
}

// SaveOne save record data to table, bean's ID field would be primary key conditions.
func (dan *DataAccessEngine) SaveOne(bean interface{}) (bool, error) {
	structureName, orm, err := dan.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Saving record by table:%s on while get database engine failed with error:%v", getTableName(bean), err)
		return false, err
	}
	condiBean, err := dan.getPkConditionBean(orm, structureName, bean)
	var ok = true
	var rc int64
	var cacheGroup *cacheElementGroup = nil
	var enableCaching bool = false
	if nil != err {
		if err == ErrNoPrimaryKeyFields {
			// insert directly
			ok = false
		} else if err == ErrEmptyPrimaryKeyFields {
			ok = false
			enableCaching = true
		} else {
			logger.Error.Printf("Saving record:%v while get primary key condition failed with error:%v", bean, err)
			return false, err
		}
	}
	if ok {
		cacheGroup = getCaches().group(structureName)
		cachingBean, _, cached := cacheGroup.get(condiBean, false)
		if cached {
			if utils.IsObjectEquals(cachingBean, bean) {
				logger.Info.Printf("Saving record for condition:%+v while the object not changed.", condiBean)
				return false, nil
			}
		}
		ok, err = orm.Exist(condiBean)
		if nil != err {
			logger.Error.Printf("Saving record:%v while check table pk:%+v exists while got error:%v", bean, condiBean, err)
			return false, err
		}
	}
	if ok {
		rc, err = orm.Update(bean, condiBean)
		if nil != err {
			logger.Error.Printf("Saving record:%+v while do update with condition:%v failed with error:%v", bean, condiBean, err)
			return false, err
		}
		logger.Info.Printf("successfully update table:%s affected %d records with item:%v", getTableName(bean), rc, bean)
	} else {
		rc, err = orm.Insert(bean)
		if nil != err {
			logger.Error.Printf("Saving record:%+v while do insert failed with error:%v", bean, err)
			return false, err
		}
		logger.Info.Printf("successfully inserted table:%s affected %d records with item:%+v", getTableName(bean), rc, bean)
	}
	if nil != cacheGroup {
		cacheGroup.set(bean, condiBean, nil)
	} else if enableCaching {
		condiBean, err = dan.getPkConditionBean(orm, structureName, bean)
		if nil == err {
			getCaches().group(structureName).set(bean, condiBean, nil)
		}
	}
	return true, nil
}

// Exists check record exists from table, bean's non-empty fields are conditions.
// The bean should be a pointer to a struct
func (dan *DataAccessEngine) Exists(bean interface{}) (bool, error) {
	structureName, orm, err := dan.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return false, err
	}
	cacheGroup := getCaches().group(structureName)
	_, _, ok := cacheGroup.get(bean, false)
	if ok {
		return true, nil
	}
	ok, err = orm.Exist(bean)
	if nil != err {
		logger.Error.Printf("Checking record:%+v exists got error:%v", bean, err)
		return false, err
	}
	return ok, err
}

// Count record counts from table, bean's non-empty fields are conditions.
func (dan *DataAccessEngine) Count(bean interface{}) (int64, error) {
	_, orm, err := dan.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Counting record by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return 0, err
	}
	counts, err := orm.Count(bean)
	if nil != err {
		logger.Error.Printf("Counting record by table:%s on condition:%+v failed with error:%v", getTableName(bean), bean, err)
	}
	return counts, err
}

// Delete record from table, bean's non-empty fields are conditions.
func (dan *DataAccessEngine) Delete(bean interface{}) (int64, error) {
	structureName, orm, err := dan.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Deleting record by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return 0, err
	}
	affected, err := orm.Delete(bean)
	getCaches().group(structureName).del(bean)
	if nil != err {
		logger.Error.Printf("Deleting record by table:%s on condition:%+v failed with error:%v", getTableName(bean), bean, err)
	}
	return affected, err
}

// EnsureTableStructures check if the table named in {beanOrTableName} struct exists in database, create it if not exixts.
func (dan *DataAccessEngine) EnsureTableStructures(beanOrTableName interface{}) error {
	var ok bool
	_, datasourceName := dan.getDatasourceName(beanOrTableName, dan.defaultDatasource)
	if "" == datasourceName {
		dan.defaultDatasource = DefaultDatasourceName
		datasourceName = dan.defaultDatasource
	}
	orm, err := dan.ensureDbEngine(datasourceName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database while ensure the database engine failed with error:%v", getTableName(beanOrTableName), err)
		return err
	}

	ok, err = orm.IsTableExist(beanOrTableName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database while check table exists failed with error:%v", getTableName(beanOrTableName), err)
		return err
	}

	if false == ok {
		err = orm.CreateTables(beanOrTableName)
		if nil != err {
			logger.Error.Printf("Create tables faield with error:%v", err)
			return err
		}

		err = orm.CreateIndexes(beanOrTableName)
		if nil != err {
			logger.Warning.Printf("Create table %s indexes faield with error:%v", getTableName(beanOrTableName), err)
		}
		err = orm.CreateUniques(beanOrTableName)
		if nil != err {
			logger.Warning.Printf("Create table %s unique indexes faield with error:%v", getTableName(beanOrTableName), err)
		}
	}

	return nil
}

func getTableName(beanOrTableName interface{}) string {
	tblName, ok := beanOrTableName.(xorm.TableName)
	if ok {
		return tblName.TableName()
	}
	val := reflect.ValueOf(beanOrTableName)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Type().Kind() == reflect.Struct {
		return val.Type().Name()
	}
	return fmt.Sprintf("%v", beanOrTableName)
}

// GetDbEngine get table datasource name
func (dan *DataAccessEngine) GetDbEngine(beanOrTableName interface{}) (*xorm.Engine, error) {
	_, orm, err := dan.getDbEngineWithStructureName(beanOrTableName)
	return orm, err
}

func (dan *DataAccessEngine) getDbEngineWithStructureName(beanOrTableName interface{}) (string, *xorm.Engine, error) {
	structureName, datasourceName := dan.getDatasourceName(beanOrTableName, dan.defaultDatasource)
	if "" == datasourceName {
		dan.defaultDatasource = DefaultDatasourceName
		datasourceName = dan.defaultDatasource
	}
	orm, err := dan.ensureDbEngine(datasourceName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database sepcified by instance:%s while ensure the database engine failed with error:%v", getTableName(beanOrTableName), datasourceName, err)
		return structureName, nil, err
	}

	if "" != structureName {
		dan.mutexTable.RLock()
		tbl, ok := dan.structuesByTableBeanName[structureName]
		dan.mutexTable.RUnlock()
		if nil == tbl || !ok {
			dan.EnsureTableStructures(beanOrTableName)
		}
	}
	return structureName, orm, nil
}

func (dan *DataAccessEngine) getDatasourceName(beanOrTableName interface{}, defaultDatasourceName string) (string, string) {
	val := getBeanValue(beanOrTableName)
	if val.Type().Kind() == reflect.Struct {
		structureName := fmt.Sprintf("%s.%s", val.Type().PkgPath(), val.Type().Name())
		dan.mutexDatasourceName.RLock()
		dataSourceName := dan.datasourceNamesByTableBeanName[structureName]
		dan.mutexDatasourceName.RUnlock()
		for "" == dataSourceName {
			dsField, ok := val.Type().FieldByName("Datasource")
			if ok {
				tagName := dsField.Tag.Get("datasource")
				if "" != tagName {
					dataSourceName = tagName
					break
				}
			}

			ds1, ok := beanOrTableName.(IDatasource)
			if ok {
				dataSourceName = ds1.Datasource()
				break
			}
			ds2, ok := beanOrTableName.(IDatasourceName)
			if ok {
				dataSourceName = ds2.DatasourceName()
				break
			}
			ds3, ok := beanOrTableName.(IGetDatasource)
			if ok {
				dataSourceName = ds3.GetDatasource()
				break
			}
		}
		if "" != dataSourceName {
			dan.mutexDatasourceName.Lock()
			dan.datasourceNamesByTableBeanName[structureName] = dataSourceName
			dan.mutexDatasourceName.Unlock()
			return structureName, dataSourceName
		}
	} else if val.Type().Kind() == reflect.String {
		return "", defaultDatasourceName
	}
	return "", defaultDatasourceName
}

func (dan *DataAccessEngine) getPkConditionBean(orm *xorm.Engine, structureName string, bean interface{}) (interface{}, error) {
	var condiBean interface{}
	var err error
	var pkValues int
	val := reflect.ValueOf(bean)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Type().Kind() != reflect.Struct {
		logger.Error.Printf("get bean:%v primary keys condition while the bean were not struct type.", bean)
		return condiBean, errors.New("condition bean were not struct type")
	}
	var pkCols []*core.Column
	dan.mutexTable.RLock()
	tbl, ok := dan.structuesByTableBeanName[structureName]
	dan.mutexTable.RUnlock()
	if nil == tbl || !ok {
		tbl = orm.TableInfo(bean)
		if nil == tbl {
			logger.Error.Printf("get table:%s primary keys condition while get table info empty", getTableName(bean))
			return condiBean, errors.New("Table info empty")
		}
		dan.mutexTable.Lock()
		dan.structuesByTableBeanName[structureName] = tbl
		dan.mutexTable.Unlock()
		pkCols = tbl.PKColumns()
		getCaches().group(structureName).pkColumns = pkCols
	} else {
		pkCols = tbl.PKColumns()
	}

	if len(pkCols) <= 0 {
		logger.Error.Printf("get table:%s primary keys condition while schema struct defines no primary key", getTableName(bean))
		return condiBean, ErrNoPrimaryKeyFields
	}

	condiBean, pkValues, err = formatPkConditionBeanImpl(pkCols, val)
	if pkValues > 0 {
		err = nil
	} else {
		err = ErrEmptyPrimaryKeyFields
	}
	return condiBean, err
}

func formatPkConditionBeanImpl(pkColumns []*core.Column, value reflect.Value) (interface{}, int, error) {
	condiValue := reflect.New(value.Type())
	pkValues := 0
	for _, col := range pkColumns {
		pkField := condiValue.Elem().FieldByName(col.FieldName)
		if !pkField.IsValid() {
			logger.Error.Printf("get table:%s primary keys condition while schema struct primary key field:%s for column:%s not defined", getTableName(value.Interface()), col.FieldName, col.Name)
			return nil, pkValues, ErrNoPrimaryKeyFields
		}
		pkValue := value.FieldByName(col.FieldName)
		pkField.Set(pkValue)

		if validates.ValidateRequired(pkField, "") == nil {
			pkValues++
		}
	}
	return condiValue.Interface(), pkValues, nil
}

func formatPkConditionBeanX(structureName string, bean interface{}) (interface{}, error) {
	val := reflect.ValueOf(bean)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	condiBean, pkValues, err := formatPkConditionBeanImpl(getCaches().group(structureName).pkColumns, val)
	if pkValues > 0 {
		err = nil
	} else {
		err = ErrEmptyPrimaryKeyFields
	}
	return condiBean, err
}

func queryAll(orm *xorm.Engine, condiBeans interface{}, records *[]interface{}, limit, offset int) error {
	var typ = reflect.ValueOf(condiBeans).Elem().Type()
	var rows *xorm.Rows
	var err error
	if 0 > limit || 0 > offset {
		logger.Error.Printf("Fetching records by table:%s on condition:%v with limit:%d offset:%d while limit and offset should not less than 0", getTableName(condiBeans), condiBeans, limit, offset)
		return errors.New("limit and offset should not less than 0")
	}
	if 0 < limit {
		rows, err = orm.Limit(limit, offset).Rows(condiBeans)
	} else {
		rows, err = orm.Rows(condiBeans)
	}
	if nil != err {
		logger.Error.Printf("Query all results by table %v failed with error:%v", condiBeans, err)
		return err
	}
	for rows.Next() {
		one := reflect.New(typ).Interface()
		err2 := rows.Scan(one)
		if nil != err2 {
			logger.Error.Printf("Parse database row for table %v failed with error:%v", condiBeans, err)
			err = err2
		} else {
			*records = append(*records, one)
		}
	}
	return err
}
