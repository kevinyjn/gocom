package dal

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"
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
	ErrNoPrimaryKeyFields = errors.New("No primary key filed defined")
)

// DataAccessEngine data access layer manager
type DataAccessEngine struct {
	orms                           map[string]*xorm.Engine // map[dbDatasourceName]*xorm.Engine
	defaultDatasource              string
	datasourceNamesByTableBeanName map[string]string      // map[beanNameOrTableName]datasourceName
	structuesByTableBeanName       map[string]*xorm.Table // map[beanNameOrTableName]*core.Table
}

var _dpm = &DataAccessEngine{
	orms:                           map[string]*xorm.Engine{},
	defaultDatasource:              "",
	datasourceNamesByTableBeanName: map[string]string{},
	structuesByTableBeanName:       map[string]*xorm.Table{},
}

// GetInstance data persistence manager
func GetInstance() *DataAccessEngine {
	return _dpm
}

// Init db instance with config
func (dan *DataAccessEngine) Init(dbDatasourceName string, dbConfig *definations.DBConnectorConfig) (*xorm.Engine, error) {
	var err error
	if nil == dbConfig || (dbConfig.Driver == "" || dbConfig.Address == "") {
		logger.Warning.Printf("Initializing data access engine by instance:%s with empty or invalid db config, using sqlite %s initilizing...", dbDatasourceName, DefaultSqliteFile)
		dbDir := filepath.Dir(DefaultSqliteFile)
		err = utils.EnsureDirectory(dbDir)
		if nil != err {
			logger.Error.Printf("Ensure db path:%s failed with error:%v", dbDir, err)
			return nil, err
		}
		dbConfig = &definations.DBConnectorConfig{
			Driver:  "sqlite3",
			Address: DefaultSqliteFile,
			Db:      DefaultDatabaseName,
		}
	}

	orm, err := xorm.NewEngine(dbConfig.Driver, dbConfig.Address)
	if nil != err {
		logger.Error.Printf("New database connection engine with driver:%s and address:%s failed with error:%v", err, dbConfig.Driver, dbConfig.Address)
		return nil, err
	}

	if "" != dbConfig.TableNamePrefix {
		tbMapper := core.NewPrefixMapper(core.SnakeMapper{}, dbConfig.TableNamePrefix)
		orm.SetTableMapper(tbMapper)
	}

	exists := dan.orms[dbDatasourceName]
	if nil != exists {
		exists.Close()
	}

	dan.orms[dbDatasourceName] = orm

	return orm, nil
}

func (dan *DataAccessEngine) ensureDbEngine(dbDatasourceName string) (*xorm.Engine, error) {
	orm := dan.orms[dbDatasourceName]
	if nil == orm {
		logger.Warning.Printf("ensuring database engine by instance:%s while the database instance were not successfully initialize, initializing it using default engine...", dbDatasourceName)
		return dan.Init(dbDatasourceName, nil)
	}
	return orm, nil
}

// FetchAll fetch all data object from database on conditionBean
func (dan *DataAccessEngine) FetchAll(condiBeans interface{}) ([]interface{}, error) {
	records := []interface{}{}
	orm, err := dan.GetDbEngine(condiBeans)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%v while get database engine failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	err = queryAll(orm, condiBeans, &records, 0, 0)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%v failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	return records, nil
}

// FetchRecords fetch all data object from database on conditionBean and limit, offset
func (dan *DataAccessEngine) FetchRecords(condiBeans interface{}, limit, offset int) ([]interface{}, error) {
	records := []interface{}{}
	if 0 >= limit || 0 > offset {
		logger.Error.Printf("Fetching records by table:%s on condition:%v with limit:%d offset:%d while limit should greater than 0 and offset should not less than 0", getTableName(condiBeans), condiBeans, limit, offset)
		return records, errors.New("limit should greater than 0 and offset should not less than 0")
	}
	orm, err := dan.GetDbEngine(condiBeans)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%v while get database engine failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	err = queryAll(orm, condiBeans, &records, limit, offset)
	if nil != err {
		logger.Error.Printf("Fetching records by table:%s on condition:%v failed with error:%v", getTableName(condiBeans), condiBeans, err)
		return records, err
	}
	return records, nil
}

// FetchOne Get retrieve one record from table, bean's non-empty fields are conditions.
// The bean should be a pointer to a struct
func (dan *DataAccessEngine) FetchOne(bean interface{}) (bool, error) {
	orm, err := dan.GetDbEngine(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return false, err
	}
	has, err := orm.Get(bean)
	if nil != err {
		logger.Error.Printf("Fetching record by table:%s on condition:%v failed with error:%v", getTableName(bean), bean, err)
	}
	return has, err
}

// SaveOne save record data to table, bean's ID field would be primary key conditions.
func (dan *DataAccessEngine) SaveOne(bean interface{}) (bool, error) {
	orm, err := dan.GetDbEngine(bean)
	if nil != err {
		logger.Error.Printf("Saving record by table:%s on while get database engine failed with error:%v", getTableName(bean), err)
		return false, err
	}
	condiBeans, err := dan.getPkConditionBean(orm, bean)
	var ok = true
	var rc int64
	if nil != err {
		if err == ErrNoPrimaryKeyFields {
			// insert directly
			ok = false
		} else {
			logger.Error.Printf("Saving record:%v while get primary key condition failed with error:%v", bean, err)
			return false, err
		}
	}
	if ok {
		ok, err = orm.Exist(condiBeans)
		if nil != err {
			logger.Error.Printf("Saving record:%v while check table pk:%v exists while got error:%v", bean, condiBeans, err)
			return false, err
		}
	}
	if ok {
		rc, err = orm.Update(bean, condiBeans)
		if nil != err {
			logger.Error.Printf("Saving record:%v while do update with condition:%v failed with error:%v", bean, condiBeans, err)
			return false, err
		}
		logger.Info.Printf("successfully update table:%s affected %d records with item:%v", getTableName(bean), rc, bean)
	} else {
		rc, err = orm.Insert(bean)
		if nil != err {
			logger.Error.Printf("Saving record:%v while do insert failed with error:%v", bean, err)
			return false, err
		}
		logger.Info.Printf("successfully inserted table:%s affected %d records with item:%v", getTableName(bean), rc, bean)
	}
	// orm.AllCols().Commit()
	return false, nil
}

// Count record counts from table, bean's non-empty fields are conditions.
func (dan *DataAccessEngine) Count(bean interface{}) (int64, error) {
	orm, err := dan.GetDbEngine(bean)
	if nil != err {
		logger.Error.Printf("Counting record by table:%s on condition:%v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return 0, err
	}
	counts, err := orm.Count(bean)
	if nil != err {
		logger.Error.Printf("Counting record by table:%s on condition:%v failed with error:%v", getTableName(bean), bean, err)
	}
	return counts, err
}

// Delete record from table, bean's non-empty fields are conditions.
func (dan *DataAccessEngine) Delete(bean interface{}) (int64, error) {
	orm, err := dan.GetDbEngine(bean)
	if nil != err {
		logger.Error.Printf("Deleting record by table:%s on condition:%v while get database engine failed with error:%v", getTableName(bean), bean, err)
		return 0, err
	}
	affected, err := orm.Delete(bean)
	if nil != err {
		logger.Error.Printf("Deleting record by table:%s on condition:%v failed with error:%v", getTableName(bean), bean, err)
	}
	return affected, err
}

// EnsureTableStructures check if the table named in {beanOrTableName} struct exists in database, create it if not exixts.
func (dan *DataAccessEngine) EnsureTableStructures(beanOrTableName interface{}) error {
	datasourceName := dan.defaultDatasource
	var ok bool
	orm, err := dan.ensureDbEngine(datasourceName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database sepcified by instance:%s while ensure the database engine failed with error:%v", getTableName(beanOrTableName), datasourceName, err)
		return err
	}

	ok, err = orm.IsTableExist(beanOrTableName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database sepcified by instance:%s while check table exists failed with error:%v", getTableName(beanOrTableName), datasourceName, err)
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
	structName, datasourceName := dan.getDatasourceName(beanOrTableName, dan.defaultDatasource)
	if "" == datasourceName {
		dan.defaultDatasource = DefaultDatasourceName
		datasourceName = dan.defaultDatasource
	}
	orm, err := dan.ensureDbEngine(datasourceName)
	if nil != err {
		logger.Error.Printf("ensuring table %s on database sepcified by instance:%s while ensure the database engine failed with error:%v", getTableName(beanOrTableName), datasourceName, err)
		return nil, err
	}

	if "" != structName {
		tbl, ok := dan.structuesByTableBeanName[structName]
		if nil == tbl || !ok {
			dan.EnsureTableStructures(beanOrTableName)
		}
	}
	return orm, nil
}

func (dan *DataAccessEngine) getDatasourceName(beanOrTableName interface{}, defaultDatasourceName string) (string, string) {
	val := reflect.ValueOf(beanOrTableName)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Type().Kind() == reflect.Struct {
		structName := fmt.Sprintf("%s.%s", val.Type().PkgPath(), val.Type().Name())
		dataSourceName := dan.datasourceNamesByTableBeanName[structName]
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
			dan.datasourceNamesByTableBeanName[structName] = dataSourceName
			return structName, dataSourceName
		}
	} else if val.Type().Kind() == reflect.String {
		return "", defaultDatasourceName
	}
	return "", defaultDatasourceName
}

func (dan *DataAccessEngine) getPkConditionBean(orm *xorm.Engine, bean interface{}) (interface{}, error) {
	var condiBeans interface{}
	var err error
	val := reflect.ValueOf(bean)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Type().Kind() != reflect.Struct {
		logger.Error.Printf("get bean:%v primary keys condition while the bean were not struct type.", bean)
		return condiBeans, errors.New("condition bean were not struct type")
	}
	structName := val.Type().Name()
	tbl, ok := dan.structuesByTableBeanName[structName]
	if nil == tbl || !ok {
		tbl = orm.TableInfo(bean)
		if nil == tbl {
			logger.Error.Printf("get table:%s primary keys condition while get table info empty", getTableName(bean))
			return condiBeans, errors.New("Table info empty")
		}
		dan.structuesByTableBeanName[structName] = tbl
	}

	pkCols := tbl.PKColumns()
	if len(pkCols) <= 0 {
		logger.Error.Printf("get table:%s primary keys condition while schema struct defines no primary key", getTableName(bean))
		return condiBeans, ErrNoPrimaryKeyFields
	}

	condVal := reflect.New(val.Type())
	settingFields := 0
	for _, col := range pkCols {
		pkField := condVal.Elem().FieldByName(col.FieldName)
		if !pkField.IsValid() {
			logger.Error.Printf("get table:%s primary keys condition while schema struct primary key field:%s for column:%s not defined", getTableName(bean), col.FieldName, col.Name)
			return condiBeans, ErrNoPrimaryKeyFields
		}
		pkValue := val.FieldByName(col.FieldName)
		pkField.Set(pkValue)

		if validates.ValidateRequired(pkField, "") == nil {
			settingFields++
		}
	}
	condiBeans = condVal.Interface()
	if settingFields > 0 {
		err = nil
	} else {
		err = ErrNoPrimaryKeyFields
	}
	return condiBeans, err
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
