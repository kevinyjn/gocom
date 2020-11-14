package dbutils

import (
	"fmt"
	"strconv"
	"strings"
)

// PhoenixDatabaseAdapter struct
type PhoenixDatabaseAdapter struct {
}

// GetName database engine name
func (adapter *PhoenixDatabaseAdapter) GetName() string {
	return "Phoenix"
}

// GetDescription database description
func (adapter *PhoenixDatabaseAdapter) GetDescription() string {
	return "Generates Phoenix compliant SQL"
}

// GetSelectStatement for database engine
func (adapter *PhoenixDatabaseAdapter) GetSelectStatement(tableName string, columnNames string, whereClause string, orderByClause string, limit int, offset int, columnForPartitioningParams ...string) (string, error) {
	if "" == tableName {
		return "", fmt.Errorf("Table name cannot be empty")
	}
	columnForPartitioning := ""
	if len(columnForPartitioningParams) > 0 {
		columnForPartitioning = columnForPartitioningParams[0]
	}
	queryParts := []string{}
	queryParts = append(queryParts, "SELECT")
	if "" == columnNames || strings.Trim(columnNames, " ") == "*" {
		queryParts = append(queryParts, "*")
	} else {
		queryParts = append(queryParts, columnNames)
	}
	queryParts = append(queryParts, "FROM", tableName)

	if "" != whereClause {
		queryParts = append(queryParts, "WHERE", whereClause)
		if "" != columnForPartitioning {
			queryParts = append(queryParts, "AND", columnForPartitioning, ">=", strconv.Itoa(offset))
			if 0 != limit {
				queryParts = append(queryParts, "AND", columnForPartitioning, "<", strconv.Itoa(offset+limit))
			}
		}
	}
	if "" != orderByClause && "" == columnForPartitioning {
		queryParts = append(queryParts, "ORDER BY", orderByClause)
	}
	if "" == columnForPartitioning {
		if 0 != limit {
			queryParts = append(queryParts, "LIMIT", strconv.Itoa(limit))
		}
		if 0 != offset {
			queryParts = append(queryParts, "OFFSET", strconv.Itoa(offset))
		}
	}

	return strings.Join(queryParts, " "), nil
}

// UnwrapIdentifier for database engine
func (adapter *PhoenixDatabaseAdapter) UnwrapIdentifier(identifier string) string {
	return UnwrapIdentifier(identifier)
}

// GetTableAliasClause for database engine
func (adapter *PhoenixDatabaseAdapter) GetTableAliasClause(tableName string) string {
	return GetTableAliasClause(tableName)
}
