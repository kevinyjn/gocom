package dbutils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MSSQLDatabaseAdapter struct
type MSSQLDatabaseAdapter struct {
}

// GetName database engine name
func (adapter *MSSQLDatabaseAdapter) GetName() string {
	return "MS SQL 2012+"
}

// GetDescription database description
func (adapter *MSSQLDatabaseAdapter) GetDescription() string {
	return "Generates MS SQL Compatible SQL, for version 2012 or greater"
}

// GetSelectStatement for database engine
func (adapter *MSSQLDatabaseAdapter) GetSelectStatement(tableName string, columnNames string, whereClause string, orderByClause string, limit int, offset int, columnForPartitioningParams ...string) (string, error) {
	if "" == tableName {
		return "", fmt.Errorf("Table name cannot be empty")
	}
	columnForPartitioning := ""
	if len(columnForPartitioningParams) > 0 {
		columnForPartitioning = columnForPartitioningParams[0]
	}
	queryParts := []string{}
	queryParts = append(queryParts, "SELECT")
	//If this is a limit query and not a paging query then use TOP in MS SQL
	if 0 != limit && 0 > offset && "" == columnForPartitioning {
		queryParts = append(queryParts, "TOP", strconv.Itoa(limit))
	}
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
		if 0 < limit && 0 <= offset {
			if "" == orderByClause {
				return "", fmt.Errorf("Order by clause cannot be null or empty when using row paging")
			}
			queryParts = append(queryParts, "OFFSET", strconv.Itoa(offset), "ROWS", "FETCH NEXT", strconv.Itoa(limit), "ROWS ONLY")
		}
	}

	return strings.Join(queryParts, " "), nil
}

// UnwrapIdentifier for database engine
func (adapter *MSSQLDatabaseAdapter) UnwrapIdentifier(identifier string) string {
	if "" == identifier {
		return identifier
	}
	reg := regexp.MustCompile("[\"\\[\\]]")
	return reg.ReplaceAllString(identifier, "")
}

// GetTableAliasClause for database engine
func (adapter *MSSQLDatabaseAdapter) GetTableAliasClause(tableName string) string {
	return GetTableAliasClause(tableName)
}
