package rdbms

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/netutils/dboptions"
	"github.com/kevinyjn/gocom/utils"
	"github.com/kevinyjn/gocom/validator/validates"

	"xorm.io/xorm"
	"xorm.io/xorm/names"
	"xorm.io/xorm/schemas"

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

	PingTimeoutSeconds = 5
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
	datasourceNamesByTableBeanName map[string]string         // map[beanNameOrTableName]datasourceName
	structuesByTableBeanName       map[string]*schemas.Table // map[beanNameOrTableName]*schemas.Table
	mutex                          sync.RWMutex
	mutexDatasourceName            sync.RWMutex
	mutexTable                     sync.RWMutex
	keepaliveTicker                *time.Ticker
	noAutoTime                     bool // this feature could be removed at xorm@v1.3.0
}

var _dpm = &DataAccessEngine{
	orms:                           map[string]*xorm.Engine{},
	defaultDatasource:              "",
	datasourceNamesByTableBeanName: map[string]string{},
	structuesByTableBeanName:       map[string]*schemas.Table{},
	mutex:                          sync.RWMutex{},
	mutexDatasourceName:            sync.RWMutex{},
	mutexTable:                     sync.RWMutex{},
	keepaliveTicker:                nil,
	noAutoTime:                     true,
}

type dbSavingSession interface {
	Insert(beans ...interface{}) (int64, error)
	InsertOne(bean interface{}) (int64, error)
	Update(bean interface{}, condiBeans ...interface{}) (int64, error)
	Delete(bean ...interface{}) (int64, error)
	Close() error
}

// GetInstance data persistence manager
func GetInstance() *DataAccessEngine {
	return _dpm
}

// Init db instance with config
func (dae *DataAccessEngine) Init(dbDatasourceName string, dbConfig *definations.DBConnectorConfig) (*xorm.Engine, error) {
	var err error
	var connData dboptions.DBConnectionData
	if nil == dbConfig || ("" == dbConfig.Driver || "" == dbConfig.Address) {
		dae.mutex.RLock()
		eng := dae.orms[dbDatasourceName]
		dae.mutex.RUnlock()
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
		tbMapper := names.NewPrefixMapper(names.SnakeMapper{}, dbConfig.TablePrefix)
		orm.SetTableMapper(tbMapper)
	}

	dae.mutex.RLock()
	exists := dae.orms[dbDatasourceName]
	dae.mutex.RUnlock()
	if nil != exists {
		exists.Close()
	}

	dae.mutex.Lock()
	dae.orms[dbDatasourceName] = orm
	dae.mutex.Unlock()
	logger.Info.Printf("Initialize database connection engine with driver:%s and address:%s succeed", connData.Driver, connData.ConnDescription)

	if nil == dae.keepaliveTicker {
		go dae.StartKeepAlive()
	}
	return orm, nil
}

// EnableAutoTime enable auto time for created, updated event
// if enabled, the model BeforeInsert and BeforeUpdate event would not be effective for saving to db
func (dae *DataAccessEngine) EnableAutoTime(enabled bool) {
	dae.noAutoTime = !enabled
}

// StartKeepAlive keepalive
func (dae *DataAccessEngine) StartKeepAlive() {
	if nil != dae.keepaliveTicker {
		return
	}
	dae.keepaliveTicker = time.NewTicker(time.Second * 30)
	for nil != dae.keepaliveTicker {
		select {
		case curTime := <-dae.keepaliveTicker.C:
			dae.pingEngines(curTime)
			break
		}
	}
	if nil != dae.keepaliveTicker {
		dae.keepaliveTicker.Stop()
	}
	dae.keepaliveTicker = nil
}

// Ping all engines
func (dae *DataAccessEngine) Ping() error {
	return dae.pingEngines(time.Now())
}

// IsValid is any engine valid
func (dae *DataAccessEngine) IsValid() bool {
	dae.mutex.RLock()
	ormLen := len(dae.orms)
	dae.mutex.RUnlock()
	return ormLen > 0
}

func (dae *DataAccessEngine) pingEngines(curTime time.Time) error {
	engs := map[string]*xorm.Engine{}
	dae.mutex.RLock()
	for cate, eg := range dae.orms {
		engs[cate] = eg
	}
	dae.mutex.RUnlock()
	errMessages := []string{}
	for cate, eg := range engs {
		pingCtx, cancel := context.WithTimeout(context.Background(), PingTimeoutSeconds*time.Second)
		defer cancel()
		err := eg.PingContext(pingCtx)
		if nil != err {
			logger.Error.Printf("Ping database %s failed with error:%v", cate, err)
			// todo: find reconnect method
			errMessages = append(errMessages, fmt.Sprintf("Ping database %s failed with error:%v", cate, err))
		}
	}
	if len(errMessages) > 0 {
		return fmt.Errorf(strings.Join(errMessages, ";"))
	}
	return nil
}

func (dae *DataAccessEngine) ensureDbEngine(dbDatasourceName string) (*xorm.Engine, error) {
	dae.mutex.RLock()
	orm := dae.orms[dbDatasourceName]
	dae.mutex.RUnlock()
	if nil == orm {
		logger.Warning.Printf("ensuring database engine by instance:%s while the database instance were not successfully initialize, initializing it using default engine...", dbDatasourceName)
		return dae.Init(dbDatasourceName, nil)
	}
	return orm, nil
}

// FetchAll fetch all data object from database on conditionBean
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies.
func (dae *DataAccessEngine) FetchAll(condiBeans interface{}, customConds ...map[string]interface{}) ([]interface{}, error) {
	records := []interface{}{}
	_, orm, err := dae.getDbEngineWithStructureName(condiBeans)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	// cacheGroup := getCaches().group(structureName)
	// _, ok := cacheGroup.getArray(condiBeans, &records, 0, 0)
	// if true == ok {
	// 	logger.Trace.Printf("Fetching records by table:%s on condition:%+v from cache hitted", getTableName(condiBeans), condiBeans)
	// 	return records, nil
	// }
	err = queryAll(orm, condiBeans, &records, 0, 0, customConds...)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	// cacheGroup.setArray(condiBeans, records, 0, 0)
	return records, nil
}

// FetchRecords fetch all data object from database on conditionBean and limit, offset
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies. while, this case would not
// be commonly comes out because that the new record would be inserted at the tail of rows in many
// databases
func (dae *DataAccessEngine) FetchRecords(condiBeans interface{}, limit, offset int, customConds ...map[string]interface{}) ([]interface{}, error) {
	records := []interface{}{}
	if 0 >= limit || 0 > offset {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v with limit:%d offset:%d while limit should greater than 0 and offset should not less than 0", getTableName(condiBeans), condiBeans, limit, offset)
		return records, errors.New("limit should greater than 0 and offset should not less than 0")
	}
	_, orm, err := dae.getDbEngineWithStructureName(condiBeans)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	// cacheGroup := getCaches().group(structureName)
	// _, ok := cacheGroup.getArray(condiBeans, &records, limit, offset)
	// if true == ok {
	// 	logger.Trace.Printf("Fetching records by table:%s on condition:%+v from cache hitted", getTableName(condiBeans), condiBeans)
	// 	return records, nil
	// }
	err = queryAll(orm, condiBeans, &records, limit, offset, customConds...)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%+v failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	// cacheGroup.setArray(condiBeans, records, limit, offset)
	return records, nil
}

// FetchRecordsAndCountTotal fetch all data object from database on conditionBean and limit, offset
// Nodes: there would be a max {MaxArrayCachingDurationSeconds} seconds non-syncronized to db data
// in case inserted some records in rows that conditionBean specifies. while, this case would not
// be commonly comes out because that the new record would be inserted at the tail of rows in many
// databases
// returns total count of records in database by condiBeans and error object if gots error
func (dae *DataAccessEngine) FetchRecordsAndCountTotal(condiBeans interface{}, limit, offset int, results interface{}, customConds ...map[string]interface{}) (int64, error) {
	resultsValue := reflect.ValueOf(results)
	if resultsValue.Type().Kind() != reflect.Ptr {
		return 0, fmt.Errorf("The results parameter should not be non-pointer array")
	}
	if resultsValue.Elem().Type().Kind() != reflect.Slice {
		return 0, fmt.Errorf("The results parameter were not array type")
	}
	count, err := dae.Count(condiBeans, customConds...)
	if nil != err {
		return 0, err
	}
	records, err := dae.FetchRecords(condiBeans, limit, offset, customConds...)
	if nil != err {
		return 0, err
	}
	var resultsElem reflect.Value
	if !resultsValue.IsValid() || resultsValue.IsNil() {
		resultsElem = reflect.New(reflect.TypeOf(results).Elem())
	} else {
		resultsElem = resultsValue.Elem()
	}
	rowsValue := make([]reflect.Value, len(records))
	for i, row := range records {
		rowsValue[i] = reflect.ValueOf(row)
		// resultsElem.Index(i).Set(reflect.ValueOf(row))
	}
	resultsElem.Set(reflect.Append(resultsElem, rowsValue...))
	return count, nil
}

// FetchOne Get retrieve one record from table, bean's non-empty fields are conditions.
// The bean should be a pointer to a struct
func (dae *DataAccessEngine) FetchOne(bean interface{}, customConds ...map[string]interface{}) (bool, error) {
	_, orm, err := dae.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return false, err
	}
	// cacheGroup := getCaches().group(structureName)
	// cachingBean, cachingKey, ok := cacheGroup.get(bean, false)
	// if ok {
	// 	err = utils.DeeplyCopyObject(cachingBean, bean)
	// 	if nil == err {
	// 		logger.Trace.Printf("Fetching record by table:%s on condition:%+v from cache hitted", getTableName(bean), cachingKey)
	// 		return true, nil
	// 	}
	// 	logger.Error.Printf("Fetching record by table:%s on condition:%+v while copying from cached data failed with error:%v", getTableName(bean), bean, err)
	// }
	// condiBean := reflect.New(reflect.ValueOf(bean).Elem().Type()).Interface()
	// utils.DeeplyCopyObject(bean, condiBean)
	s := prepareDbSession(orm, customConds...)
	has, err := s.Get(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%+v failed with error:%v", getTableName(bean), bean, err)
	} else if has {
		// pkCondiBean, _ := dae.getPkConditionBean(orm, structureName, bean)
		// cacheGroup.set(bean, condiBean, pkCondiBean)
	} else {
		logger.Info.Printf("Fetching record by table:%s on condition:%+v while there is no such record", getTableName(bean), bean)
	}
	return has, err
}

// SaveOne save record data to table, bean's ID field would be primary key conditions.
func (dae *DataAccessEngine) SaveOne(bean interface{}) (bool, error) {
	structureName, orm, err := dae.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Saving record by table:%s on while get database engine failed with error:%v", getTableName(bean), err)
		return false, err
	}
	condiBean, err := dae.getPkConditionBean(orm, structureName, bean)
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
		// cacheGroup = getCaches().group(structureName)
		// cachingBean, _, cached := cacheGroup.get(condiBean, false)
		// if cached {
		// 	if utils.IsObjectEquals(cachingBean, bean) {
		// 		logger.Info.Printf("Saving record for condition:%+v while the object not changed.", condiBean)
		// 		return false, nil
		// 	}
		// }
		ok, err = orm.Exist(condiBean)
		if nil != err {
			logger.Error.Printf("Saving record:%v while check table pk:%+v exists while got error:%v", bean, condiBean, err)
			return false, err
		}
	}
	var session dbSavingSession = orm
	if dae.noAutoTime {
		session = orm.NoAutoTime()
		defer session.Close()
	}
	if ok {
		rc, err = session.Update(bean, condiBean)
		if nil != err {
			logger.Error.Printf("Saving record:%+v while do update with condition:%v failed with error:%v", bean, condiBean, err)
			return false, err
		}
		logger.Info.Printf("successfully update table:%s affected %d records with item:%v", getTableName(bean), rc, bean)
	} else {
		rc, err = session.Insert(bean)
		if nil != err {
			logger.Error.Printf("Saving record:%+v while do insert failed with error:%v", bean, err)
			return false, err
		}
		logger.Info.Printf("successfully inserted table:%s affected %d records with item:%+v", getTableName(bean), rc, bean)
	}
	if nil != cacheGroup {
		cacheGroup.set(bean, condiBean, nil)
	} else if enableCaching {
		condiBean, err = dae.getPkConditionBean(orm, structureName, bean)
		if nil == err {
			getCaches().group(structureName).set(bean, condiBean, nil)
		}
	}
	return true, nil
}

// Exists check record exists from table, bean's non-empty fields are conditions.
// The bean should be a pointer to a struct
func (dae *DataAccessEngine) Exists(bean interface{}, customConds ...map[string]interface{}) (bool, error) {
	_, orm, err := dae.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return false, err
	}
	// cacheGroup := getCaches().group(structureName)
	// _, _, ok := cacheGroup.get(bean, false)
	// if ok {
	// 	return true, nil
	// }
	s := prepareDbSession(orm, customConds...)
	ok, err := s.Exist(bean)
	if nil != err {
		logger.Error.Printf("Checking record:%+v exists got error:%v", bean, err)
		return false, err
	}
	return ok, err
}

// Count record counts from table, bean's non-empty fields are conditions.
func (dae *DataAccessEngine) Count(bean interface{}, customConds ...map[string]interface{}) (int64, error) {
	_, orm, err := dae.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Counting record by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return 0, err
	}
	s := prepareDbSession(orm, customConds...)
	counts, err := s.Count(bean)
	if nil != err {
		logger.Error.Printf("Counting record by table:%s on condition:%+v failed with error:%v", getTableName(bean), bean, err)
	}
	return counts, err
}

// Insert save record data to table, bean's ID field would be primary key conditions.
func (dae *DataAccessEngine) Insert(beans ...interface{}) (int64, error) {
	if len(beans) <= 0 {
		logger.Error.Printf("Insert records by passing no records")
		return 0, fmt.Errorf("Passing no records")
	}
	structureName, orm, err := dae.getDbEngineWithStructureName(beans[0])
	if nil != err {
		logger.Error.Printf("Insert record by table:%s on while get database engine failed with error:%v", getTableName(beans[0]), err)
		return 0, err
	}
	var rc int64
	var enableCaching bool = false

	var session dbSavingSession = orm
	if dae.noAutoTime {
		session = orm.NoAutoTime()
		defer session.Close()
	}
	rc, err = session.Insert(beans...)
	if nil != err {
		logger.Error.Printf("Insert records while do insert failed with error:%v", err)
		return rc, err
	}
	logger.Info.Printf("successfully inserted table:%s affected %d records", getTableName(beans[0]), rc)

	if enableCaching {
		for _, bean := range beans {
			condiBean, err := dae.getPkConditionBean(orm, structureName, bean)
			if nil == err {
				getCaches().group(structureName).set(bean, condiBean, nil)
			}
		}
	}
	return rc, nil
}

// InsertMulti insert records data to table, bean's ID field would be primary key conditions.
func (dae *DataAccessEngine) InsertMulti(beans []interface{}) (int64, error) {
	if len(beans) <= 0 {
		logger.Error.Printf("InsertMulti records by passing no records")
		return 0, fmt.Errorf("Passing no records")
	}
	structureName, orm, err := dae.getDbEngineWithStructureName(beans[0])
	if nil != err {
		logger.Error.Printf("InsertMulti records by table:%s on while get database engine failed with error:%v", getTableName(beans[0]), err)
		return 0, err
	}
	var rc int64
	var enableCaching bool = false

	var session dbSavingSession = orm
	if dae.noAutoTime {
		session = orm.NoAutoTime()
		defer session.Close()
	}
	rc, err = session.Insert(beans...)
	if nil != err {
		logger.Error.Printf("InsertMulti record while do insert failed with error:%v", err)
		return rc, err
	}
	logger.Info.Printf("successfully inserted table:%s affected %d records", getTableName(beans[0]), rc)

	if enableCaching {
		for _, bean := range beans {
			condiBean, err := dae.getPkConditionBean(orm, structureName, bean)
			if nil == err {
				getCaches().group(structureName).set(bean, condiBean, nil)
			}
		}
	}
	return rc, nil
}

// Delete record from table, bean's non-empty fields are conditions.
func (dae *DataAccessEngine) Delete(bean interface{}) (int64, error) {
	_, orm, err := dae.getDbEngineWithStructureName(bean)
	if nil != err {
		logger.Error.Printf("Deleting record by table:%s on condition:%+v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return 0, err
	}
	var session dbSavingSession = orm
	if dae.noAutoTime {
		session = orm.NoAutoTime()
		defer session.Close()
	}
	affected, err := session.Delete(bean)
	// getCaches().group(structureName).del(bean)
	if nil != err {
		logger.Error.Printf("Deleting record by table:%s on condition:%+v failed with error:%v", getTableName(bean), bean, err)
	}
	return affected, err
}

// EnsureTableStructures check if the table named in {beanOrTableName} struct exists in database, create it if not exixts.
func (dae *DataAccessEngine) EnsureTableStructures(beanOrTableName interface{}) error {
	var ok bool
	_, datasourceName := dae.getDatasourceName(beanOrTableName, dae.defaultDatasource)
	if "" == datasourceName {
		dae.defaultDatasource = DefaultDatasourceName
		datasourceName = dae.defaultDatasource
	}
	orm, err := dae.ensureDbEngine(datasourceName)
	if nil != err {
		logger.Error.Printf("ensuring table '%s' on database while ensure the database engine failed with error:%v", getTableName(beanOrTableName), err)
		return err
	}

	ok, err = orm.IsTableExist(beanOrTableName)
	if nil != err {
		logger.Error.Printf("ensuring table '%s' on database while check table exists failed with error:%v", getTableName(beanOrTableName), err)
		return err
	}

	if false == ok {
		err = orm.CreateTables(beanOrTableName)
		if nil != err {
			logger.Error.Printf("Create table '%s' faield with error:%v", getTableName(beanOrTableName), err)
			return err
		}

		err = orm.CreateIndexes(beanOrTableName)
		if nil != err {
			logger.Warning.Printf("Create table '%s' indexes faield with error:%v", getTableName(beanOrTableName), err)
		}
		err = orm.CreateUniques(beanOrTableName)
		if nil != err {
			logger.Warning.Printf("Create table '%s' unique indexes faield with error:%v", getTableName(beanOrTableName), err)
		}

		if false {
			beanValue := reflect.ValueOf(beanOrTableName)
			beanType := reflect.TypeOf(beanOrTableName)
			if beanValue.IsValid() && beanValue.Type().Kind() == reflect.Ptr {
				beanValue = beanValue.Elem()
				beanType = beanType.Elem()
			}
			if beanValue.IsValid() && beanValue.Type().Kind() == reflect.Struct {
				firstField := beanValue.Field(0)
				if firstField.IsValid() && firstField.Type().Kind() == reflect.Struct {
					if beanType.Field(0).Tag.Get("xorm") == "extends" {
						// todo
					}
				}
			}
		}
	} else if reflect.TypeOf(beanOrTableName).Kind() != reflect.String {
		err = orm.Sync(beanOrTableName)
		if nil != err {
			logger.Error.Printf("Syncronize table '%s' structure with bean:%+v faield with error:%v", getTableName(beanOrTableName), beanOrTableName, err)
			return err
		}
	}

	return nil
}

func getTableName(beanOrTableName interface{}) string {
	tblName, ok := beanOrTableName.(names.TableName)
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
func (dae *DataAccessEngine) GetDbEngine(beanOrTableName interface{}) (*xorm.Engine, error) {
	_, orm, err := dae.getDbEngineWithStructureName(beanOrTableName)
	return orm, err
}

// Table sepcify table session of db engine
func (dae *DataAccessEngine) Table(beanOrTableName interface{}) (*xorm.Session, error) {
	_, orm, err := dae.getDbEngineWithStructureName(beanOrTableName)
	if nil != err {
		return nil, err
	}
	return orm.Table(beanOrTableName), nil
}

// QueryRelationsAsMappedInterface query the relationship mapper includes data
func (dae *DataAccessEngine) QueryRelationsAsMappedInterface(relationQuery RelationQuery, args ...interface{}) ([]map[string]interface{}, error) {
	orm, err := dae.GetDbEngine(relationQuery.TargetTable)
	if nil != err {
		return nil, err
	}
	argc := len(args)
	seps := make([]string, argc)
	placeholder := GetSQLParamPlaceholderString(orm.DriverName())
	for i := 0; i < argc; i++ {
		seps[i] = placeholder
	}
	rows, err := orm.Table(relationQuery.TargetTable).Select(relationQuery.Select).
		Join("INNER", relationQuery.RelationTable, fmt.Sprintf("%s.%s = %s.%s", relationQuery.RelationTable.TableName(), relationQuery.TargetRelationField, relationQuery.TargetTable.TableName(), relationQuery.TargetPrimaryKey)).
		Where(fmt.Sprintf("%s.%s IN (%s)", relationQuery.RelationTable.TableName(), relationQuery.SelfRelationField, strings.Join(seps, ", ")), args...).
		QueryInterface()
	return rows, err
}

// QueryRelationData query the table relation data
func (dae *DataAccessEngine) QueryRelationData(relationQuery RelationQuery, structRowParameterField string, structRowRelationsField string, structRows interface{}) ([]map[string]interface{}, error) {
	if nil == structRows || "" == structRowParameterField || "" == structRowRelationsField {
		return nil, fmt.Errorf("No argument specified")
	}
	args := organizeQueryParameters(structRowParameterField, structRows)
	rows, err := dae.QueryRelationsAsMappedInterface(relationQuery, args...)
	if nil != err {
		return nil, err
	}
	return rows, nil
}

func (dae *DataAccessEngine) getDbEngineWithStructureName(beanOrTableName interface{}) (string, *xorm.Engine, error) {
	structureName, datasourceName := dae.getDatasourceName(beanOrTableName, dae.defaultDatasource)
	if "" == datasourceName {
		dae.defaultDatasource = DefaultDatasourceName
		datasourceName = dae.defaultDatasource
	}
	orm, err := dae.ensureDbEngine(datasourceName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database sepcified by instance:%s while ensure the database engine failed with error:%v", getTableName(beanOrTableName), datasourceName, err)
		return structureName, nil, err
	}

	if "" != structureName {
		dae.mutexTable.RLock()
		tbl, ok := dae.structuesByTableBeanName[structureName]
		dae.mutexTable.RUnlock()
		if nil == tbl || !ok {
			dae.EnsureTableStructures(beanOrTableName)
		}
	}
	return structureName, orm, nil
}

func (dae *DataAccessEngine) getDatasourceName(beanOrTableName interface{}, defaultDatasourceName string) (string, string) {
	val := getBeanValue(beanOrTableName)
	if val.Type().Kind() == reflect.Struct {
		structureName := fmt.Sprintf("%s.%s", val.Type().PkgPath(), val.Type().Name())
		dae.mutexDatasourceName.RLock()
		dataSourceName := dae.datasourceNamesByTableBeanName[structureName]
		dae.mutexDatasourceName.RUnlock()
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
			dae.mutexDatasourceName.Lock()
			dae.datasourceNamesByTableBeanName[structureName] = dataSourceName
			dae.mutexDatasourceName.Unlock()
			return structureName, dataSourceName
		}
	} else if val.Type().Kind() == reflect.String {
		return "", defaultDatasourceName
	}
	return "", defaultDatasourceName
}

func (dae *DataAccessEngine) getPkConditionBean(orm *xorm.Engine, structureName string, bean interface{}) (interface{}, error) {
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
	var pkCols []*schemas.Column
	dae.mutexTable.RLock()
	tbl, ok := dae.structuesByTableBeanName[structureName]
	dae.mutexTable.RUnlock()
	if nil == tbl || !ok {
		tbl, err = orm.TableInfo(bean)
		if nil != err {
			logger.Error.Printf("get table:%s primary keys condition while get table info empty", getTableName(bean))
			return condiBean, errors.New("Table info empty")
		}
		if nil == tbl {
			logger.Error.Printf("get table:%s primary keys condition while get table info empty", getTableName(bean))
			return condiBean, errors.New("Table info empty")
		}
		dae.mutexTable.Lock()
		dae.structuesByTableBeanName[structureName] = tbl
		dae.mutexTable.Unlock()
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

func formatPkConditionBeanImpl(pkColumns []*schemas.Column, value reflect.Value) (interface{}, int, error) {
	condiValue := reflect.New(value.Type())
	pkValues := 0
	var pkField reflect.Value
	var pkValue reflect.Value
	for _, col := range pkColumns {
		pkField = condiValue.Elem().FieldByName(col.FieldName)
		if false == pkField.IsValid() {
			fieldNames := strings.Split(col.FieldName, ".")
			if len(fieldNames) > 1 {
				pkField = condiValue.Elem()
				pkValue = value
				for _, fieldName := range fieldNames {
					pkField = pkField.FieldByName(fieldName)
					pkValue = pkValue.FieldByName(fieldName)

					if false == pkField.IsValid() {
						break
					}
					if pkField.Type().Kind() == reflect.Ptr {
						pkField = pkField.Elem()
						pkValue = pkValue.Elem()
					}
					if pkField.Type().Kind() != reflect.Struct {
						break
					}
				}
			}
			if false == pkField.IsValid() {
				logger.Error.Printf("get table:%s primary keys condition while schema struct primary key field:%s for column:%s not defined", getTableName(value.Interface()), col.FieldName, col.Name)
				return nil, pkValues, ErrNoPrimaryKeyFields
			}
		} else {
			pkValue = value.FieldByName(col.FieldName)
		}
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

func queryAll(orm *xorm.Engine, condiBeans interface{}, records *[]interface{}, limit, offset int, customConds ...map[string]interface{}) error {
	var typ = reflect.ValueOf(condiBeans).Elem().Type()
	var rows *xorm.Rows
	var err error
	if 0 > limit || 0 > offset {
		logger.Error.Printf("Fetching records by table:%s on condition:%v with limit:%d offset:%d while limit and offset should not less than 0", getTableName(condiBeans), condiBeans, limit, offset)
		return errors.New("limit and offset should not less than 0")
	}
	s := prepareDbSession(orm, customConds...)
	if 0 < limit {
		rows, err = s.Limit(limit, offset).Rows(condiBeans)
	} else {
		rows, err = s.Rows(condiBeans)
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

func prepareDbSession(orm *xorm.Engine, args ...map[string]interface{}) xorm.Interface {
	if nil == args || len(args) <= 0 {
		return orm
	}
	var s xorm.Interface = orm
	for _, c := range args {
		s = s.Where(c)
	}
	return s
}

// GetSQLQuoteString get sql quote
func GetSQLQuoteString(driverName string) string {
	quote := "'"
	switch driverName {
	case "mysql":
		quote = "`"
		break
	case "postgres":
		quote = "\""
		break
	}
	return quote
}

// GetSQLParamPlaceholderString get sql parameter placeholder
func GetSQLParamPlaceholderString(driverName string) string {
	quote := "?"
	switch driverName {
	case "postgres":
		quote = "$"
		break
	}
	return quote
}

func organizeQueryParameters(fieldName string, params interface{}) []interface{} {
	pv := reflect.ValueOf(params)
	if reflect.Ptr == pv.Type().Kind() {
		pv = pv.Elem()
	}
	var argc int
	if pv.Type().Kind() != reflect.Slice {
		nv := []interface{}{params}
		pv = reflect.ValueOf(nv)
		argc = 1
	} else {
		argc = pv.Len()
	}
	argsMap := map[interface{}]bool{}
	for i := 0; i < argc; i++ {
		v := pv.Index(i)
		if v.Type().Kind() == reflect.Ptr {
			v = v.Elem()
		}
		switch v.Type().Kind() {
		case reflect.Struct:
			fv := v.FieldByName(fieldName)
			if fv.IsValid() {
				argsMap[fv.Interface()] = true
			}
			break
		case reflect.Map:
			mv, ok := v.Interface().(map[string]interface{})
			if ok {
				fv, ok := mv[fieldName]
				if ok {
					argsMap[fv] = true
				}
			}
			break
		case reflect.String, reflect.Int, reflect.Uint, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Bool:
			argsMap[v.Interface()] = true
			break
		}
	}
	args := make([]interface{}, len(argsMap))
	i := 0
	for k := range argsMap {
		args[i] = k
		i++
	}
	return args
}
