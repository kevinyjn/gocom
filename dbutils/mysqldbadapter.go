package dbutils

import "regexp"

// MySQLDatabaseAdapter struct
type MySQLDatabaseAdapter struct {
	GenericDatabaseAdapter
}

// GetName database engine name
func (adapter *MySQLDatabaseAdapter) GetName() string {
	return "MySQL"
}

// GetDescription database description
func (adapter *MySQLDatabaseAdapter) GetDescription() string {
	return "Generates MySQL compatible SQL"
}

// UnwrapIdentifier for database engine
func (adapter *MySQLDatabaseAdapter) UnwrapIdentifier(identifier string) string {
	if "" == identifier {
		return identifier
	}
	reg := regexp.MustCompile("[\"`]")
	return reg.ReplaceAllString(identifier, "")
}
