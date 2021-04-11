package unittests

import (
	"testing"

	"github.com/kevinyjn/gocom/dbutils"
	"github.com/kevinyjn/gocom/testingutil"
)

// OracleDatabaseAdapter tests
func TestOracleDatabaseAdapterGeneration(t *testing.T) {
	db := dbutils.OracleDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value'", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT * FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql4, "sql4")
}

func TestOracleDatabaseAdapterPagingQuery(t *testing.T) {
	db := dbutils.OracleDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 100, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM (SELECT a.*, ROWNUM rnum FROM ( SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain ) a WHERE ROWNUM <= 100 ) WHERE rnum > 0", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM (SELECT a.*, ROWNUM rnum FROM ( SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain ) a WHERE ROWNUM <= 133456 ) WHERE rnum > 123456", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM (SELECT a.*, ROWNUM rnum FROM ( SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' ORDER BY contain ) a WHERE ROWNUM <= 133456 ) WHERE rnum > 123456", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM (SELECT a.*, ROWNUM rnum FROM ( SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ) a WHERE ROWNUM <= 100 ) WHERE rnum > 0", sql4, "sql4")
}

func TestOracleDatabaseAdapterPagingQueryUsingColumnValuesForPartitioning(t *testing.T) {
	db := dbutils.OracleDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 100, 0, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 0 AND contain < 100", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 123456 AND contain < 133456", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' AND contain >= 123456 AND contain < 133456", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, 0, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql4, "sql4")
}

// Oracle12DatabaseAdapter tests
func TestOracle12DatabaseAdapterGeneration(t *testing.T) {
	db := dbutils.Oracle12DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value'", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT * FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql4, "sql4")
}

func TestOracle12DatabaseAdapterPagingQuery(t *testing.T) {
	db := dbutils.Oracle12DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 100, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain FETCH NEXT 100 ROWS ONLY", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain OFFSET 123456 ROWS FETCH NEXT 10000 ROWS ONLY", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' ORDER BY contain OFFSET 123456 ROWS FETCH NEXT 10000 ROWS ONLY", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename FETCH NEXT 100 ROWS ONLY", sql4, "sql4")
}

func TestOracle12DatabaseAdapterPagingQueryUsingColumnValuesForPartitioning(t *testing.T) {
	db := dbutils.Oracle12DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 100, 0, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 0 AND contain < 100", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 123456 AND contain < 133456", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' AND contain >= 123456 AND contain < 133456", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, 0, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql4, "sql4")
}

// MSSQLDatabaseAdapter tests
func TestMSSQLDatabaseAdapterGeneration(t *testing.T) {
	db := dbutils.MSSQLDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value'", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT * FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql4, "sql4")
}

func TestMSSQLDatabaseAdapterPagingNoOrderBy(t *testing.T) {
	db := dbutils.MSSQLDatabaseAdapter{}
	_, err := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, 0)
	testingutil.AssertNotNil(t, err, "err")
}

func TestMSSQLDatabaseAdapterTOPQuery(t *testing.T) {
	db := dbutils.MSSQLDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, -1)
	testingutil.AssertEquals(t, "SELECT TOP 100 some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 100, -1)
	testingutil.AssertEquals(t, "SELECT TOP 100 some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain", sql2, "sql2")

	sql4, _ := db.GetSelectStatement("database.tablename", "", "that='some\"' value'", "might DESC", 123456, -1)
	testingutil.AssertEquals(t, "SELECT TOP 123456 * FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql4, "sql4")
}

func TestMSSQLDatabaseAdapterPagingQuery(t *testing.T) {
	db := dbutils.MSSQLDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 100, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain OFFSET 123456 ROWS FETCH NEXT 10000 ROWS ONLY", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' ORDER BY contain OFFSET 123456 ROWS FETCH NEXT 10000 ROWS ONLY", sql3, "sql3")
}

func TestMSSQLDatabaseAdapterPagingQueryUsingColumnValuesForPartitioning(t *testing.T) {
	db := dbutils.MSSQLDatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 100, 0, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 0 AND contain < 100", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 123456 AND contain < 133456", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' AND contain >= 123456 AND contain < 133456", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, -1, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql4, "sql4")
}

// MSSQL2008DatabaseAdapter tests
func TestMSSQL2008DatabaseAdapterGeneration(t *testing.T) {
	db := dbutils.MSSQL2008DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value'", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "", "that='some\"' value'", "might DESC", 0, 0)
	testingutil.AssertEquals(t, "SELECT * FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql4, "sql4")
}

func TestMSSQL2008DatabaseAdapterTOPQuery(t *testing.T) {
	db := dbutils.MSSQL2008DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, -1)
	testingutil.AssertEquals(t, "SELECT TOP 100 some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 100, -1)
	testingutil.AssertEquals(t, "SELECT TOP 100 some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename ORDER BY contain", sql2, "sql2")

	sql4, _ := db.GetSelectStatement("database.tablename", "", "that='some\"' value'", "might DESC", 123456, -1)
	testingutil.AssertEquals(t, "SELECT TOP 123456 * FROM database.tablename WHERE that='some\"' value' ORDER BY might DESC", sql4, "sql4")
}

func TestMSSQL2008DatabaseAdapterPagingQuery(t *testing.T) {
	db := dbutils.MSSQL2008DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 100, 0)
	testingutil.AssertEquals(t, "SELECT * FROM (SELECT TOP 100 some(set),of(columns),that,might,contain,methods,a.* , ROW_NUMBER() OVER(ORDER BY contain asc) rnum FROM database.tablename ORDER BY contain ) A WHERE rnum > 0 AND rnum <= 100", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT * FROM (SELECT TOP 133456 some(set),of(columns),that,might,contain,methods,a.* , ROW_NUMBER() OVER(ORDER BY contain asc) rnum FROM database.tablename ORDER BY contain ) A WHERE rnum > 123456 AND rnum <= 133456", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456)
	testingutil.AssertEquals(t, "SELECT * FROM (SELECT TOP 133456 some(set),of(columns),that,might,contain,methods,a.* , ROW_NUMBER() OVER(ORDER BY contain asc) rnum FROM database.tablename WHERE methods='strange' ORDER BY contain ) A WHERE rnum > 123456 AND rnum <= 133456", sql3, "sql3")
}

func TestMSSQL2008DatabaseAdapterPagingQueryUsingColumnValuesForPartitioning(t *testing.T) {
	db := dbutils.MSSQL2008DatabaseAdapter{}
	sql1, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 100, 0, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 0 AND contain < 100", sql1, "sql1")

	sql2, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "1=1", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE 1=1 AND contain >= 123456 AND contain < 133456", sql2, "sql2")

	sql3, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "methods='strange'", "contain", 10000, 123456, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename WHERE methods='strange' AND contain >= 123456 AND contain < 133456", sql3, "sql3")

	sql4, _ := db.GetSelectStatement("database.tablename", "some(set),of(columns),that,might,contain,methods,a.*", "", "", 100, -1, "contain")
	testingutil.AssertEquals(t, "SELECT some(set),of(columns),that,might,contain,methods,a.* FROM database.tablename", sql4, "sql4")
}
