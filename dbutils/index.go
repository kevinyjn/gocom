package dbutils

// GetAllDatabaseAdapters DatabaseAdapter
func GetAllDatabaseAdapters() []DatabaseAdapter {
	return []DatabaseAdapter{
		&GenericDatabaseAdapter{},
		&OracleDatabaseAdapter{},
		&Oracle12DatabaseAdapter{},
		&MySQLDatabaseAdapter{},
		&MSSQLDatabaseAdapter{},
		&MSSQL2008DatabaseAdapter{},
		&PhoenixDatabaseAdapter{},
		&DerbyDatabaseAdapter{},
	}
}
