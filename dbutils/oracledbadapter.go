package dbutils

import (
	"fmt"
	"strconv"
	"strings"
)

// OracleDatabaseAdapter struct
type OracleDatabaseAdapter struct {
}

// GetName database engine name
func (adapter *OracleDatabaseAdapter) GetName() string {
	return "Oracle"
}

// GetDescription database description
func (adapter *OracleDatabaseAdapter) GetDescription() string {
	return "Generates Oracle compliant SQL"
}

// GetSelectStatement for database engine
func (adapter *OracleDatabaseAdapter) GetSelectStatement(tableName string, columnNames string, whereClause string, orderByClause string, limit int, offset int, columnForPartitioningParams ...string) (string, error) {
	if "" == tableName {
		return "", fmt.Errorf("Table name cannot be empty")
	}
	columnForPartitioning := ""
	if len(columnForPartitioningParams) > 0 {
		columnForPartitioning = columnForPartitioningParams[0]
	}
	queryParts := []string{}
	nestedSelect := false
	if (0 != limit || 0 != offset) && "" == columnForPartitioning {
		nestedSelect = true
	}
	if nestedSelect {
		queryParts = append(queryParts, "SELECT")
		if "" == columnNames || strings.Trim(columnNames, " ") == "*" {
			queryParts = append(queryParts, "*")
		} else {
			queryParts = append(queryParts, columnNames)
		}
		queryParts = append(queryParts, "FROM (SELECT a.*, ROWNUM rnum FROM (")
	}
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
	if nestedSelect {
		queryParts = append(queryParts, ") a")
		if limit != 0 {
			queryParts = append(queryParts, "WHERE ROWNUM <=", strconv.Itoa(offset+limit))
		}
		queryParts = append(queryParts, ") WHERE rnum >", strconv.Itoa(offset))
	}

	return strings.Join(queryParts, " "), nil
}

// UnwrapIdentifier for database engine
func (adapter *OracleDatabaseAdapter) UnwrapIdentifier(identifier string) string {
	return UnwrapIdentifier(identifier)
}

// GetTableAliasClause for database engine
func (adapter *OracleDatabaseAdapter) GetTableAliasClause(tableName string) string {
	return tableName
}
