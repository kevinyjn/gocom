package dbutils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MSSQL2008DatabaseAdapter struct
type MSSQL2008DatabaseAdapter struct {
}

// GetName database engine name
func (adapter *MSSQL2008DatabaseAdapter) GetName() string {
	return "MS SQL 2008"
}

// GetDescription database description
func (adapter *MSSQL2008DatabaseAdapter) GetDescription() string {
	return "Generates MS SQL Compatible SQL for version 2008"
}

// GetSelectStatement for database engine
func (adapter *MSSQL2008DatabaseAdapter) GetSelectStatement(tableName string, columnNames string, whereClause string, orderByClause string, limit int, offset int, columnForPartitioningParams ...string) (string, error) {
	if "" == tableName {
		return "", fmt.Errorf("Table name cannot be empty")
	}
	columnForPartitioning := ""
	if len(columnForPartitioningParams) > 0 {
		columnForPartitioning = columnForPartitioningParams[0]
	}
	queryParts := []string{}
	queryParts = append(queryParts, "SELECT")
	originOffset := offset
	if offset < 0 {
		offset = 0
	}
	//If this is a limit query and not a paging query then use TOP in MS SQL
	if 0 != limit && "" == columnForPartitioning {
		if 0 <= originOffset {
			queryParts = append(queryParts, "* FROM (SELECT")
		}
		if offset+limit > 0 {
			queryParts = append(queryParts, "TOP", strconv.Itoa(offset+limit))
		}
	}
	if "" == columnNames || strings.Trim(columnNames, " ") == "*" {
		queryParts = append(queryParts, "*")
	} else {
		queryParts = append(queryParts, columnNames)
	}
	if 0 != limit && 0 <= originOffset && "" != orderByClause && "" == columnForPartitioning {
		queryParts = append(queryParts, ", ROW_NUMBER() OVER(ORDER BY", orderByClause, "asc) rnum")
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
	if 0 != limit && 0 <= originOffset && "" == columnForPartitioning {
		queryParts = append(queryParts, ") A WHERE rnum >", strconv.Itoa(offset), "AND rnum <=", strconv.Itoa(offset+limit))
	}

	return strings.Join(queryParts, " "), nil
}

// UnwrapIdentifier for database engine
func (adapter *MSSQL2008DatabaseAdapter) UnwrapIdentifier(identifier string) string {
	if "" == identifier {
		return identifier
	}
	reg := regexp.MustCompile("[\"\\[\\]]")
	return reg.ReplaceAllString(identifier, "")
}

// GetTableAliasClause for database engine
func (adapter *MSSQL2008DatabaseAdapter) GetTableAliasClause(tableName string) string {
	return GetTableAliasClause(tableName)
}
