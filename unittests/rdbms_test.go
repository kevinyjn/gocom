package unittests

import (
	"os"
	"testing"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/microsvc/acl"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/testingutil"
	"github.com/kevinyjn/gocom/utils"
)

const (
	testingDB = "default"
)

func TestRDBMS_Orm(t *testing.T) {
	initRDBMSTestingDB(t)
	testingutil.AssertNil(t, acl.InitBuiltinRBACModels(), "EnsureTableStructures error")
	user := builtinmodels.User{}
	rows, err := rdbms.GetInstance().FetchAll(&user)
	testingutil.AssertNil(t, err, "rdbms.FetchAll")
	testingutil.AssertNotNil(t, rows, "rdbms.FetchAll rows")
	rows, err = rdbms.GetInstance().FetchAll(&user)
	testingutil.AssertNil(t, err, "rdbms.FetchAll")
}

func initRDBMSTestingDB(t *testing.T) {
	dbFile := "./testing.db.test"
	dbConfig := definations.DBConnectorConfig{
		Driver:  "sqlite3",
		Address: "file://" + dbFile,
		Db:      testingDB,
	}
	if utils.IsPathExists(dbFile) {
		err := os.Remove(dbFile)
		testingutil.AssertNil(t, err, "Clean RDBMS testing db error")
	}
	engine, err := rdbms.GetInstance().Init(testingDB, &dbConfig)
	testingutil.AssertNil(t, err, "Init RDBMS error")
	testingutil.AssertNotNil(t, engine, "Init RDBMS engine")
}
