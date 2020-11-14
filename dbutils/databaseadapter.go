package dbutils

import "strings"

// DatabaseAdapter interface
type DatabaseAdapter interface {
	GetName() string
	GetDescription() string
	GetSelectStatement(tableName string, columnNames string, whereClause string, orderByClause string, limit int, offset int, columnForPartitioning ...string) (string, error)
	UnwrapIdentifier(identifier string) string
	GetTableAliasClause(tableName string) string
}

// UnwrapIdentifier common unwrap identifier
func UnwrapIdentifier(identifier string) string {
	if "" == identifier {
		return ""
	}
	identifier = strings.Replace(identifier, "\"", "", -1)
	return identifier
}

// GetTableAliasClause common table alias clause
func GetTableAliasClause(tableName string) string {
	return "AS " + tableName
}
