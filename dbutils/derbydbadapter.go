package dbutils

import (
	"fmt"
	"strconv"
	"strings"
)

// DerbyDatabaseAdapter struct
type DerbyDatabaseAdapter struct {
}

// GetName database engine name
func (adapter *DerbyDatabaseAdapter) GetName() string {
	return "Derby"
}

// GetDescription database description
func (adapter *DerbyDatabaseAdapter) GetDescription() string {
	return "Generates Derby compatible SQL"
}

// GetSelectStatement for database engine
func (adapter *DerbyDatabaseAdapter) GetSelectStatement(tableName string, columnNames string, whereClause string, orderByClause string, limit int, offset int, columnForPartitioningParams ...string) (string, error) {
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
		if 0 < offset {
			queryParts = append(queryParts, "OFFSET", strconv.Itoa(offset), "ROWS")
		}
		if 0 != limit {
			queryParts = append(queryParts, "FETCH NEXT", strconv.Itoa(limit), "ROWS ONLY")
		}
	}

	return strings.Join(queryParts, " "), nil
}

// UnwrapIdentifier for database engine
func (adapter *DerbyDatabaseAdapter) UnwrapIdentifier(identifier string) string {
	return UnwrapIdentifier(identifier)
}

// GetTableAliasClause for database engine
func (adapter *DerbyDatabaseAdapter) GetTableAliasClause(tableName string) string {
	return GetTableAliasClause(tableName)
}
