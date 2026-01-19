package common

import (
	"fmt"
	"strings"
)

// SQLFeatures SQL特性分析结果
type SQLFeatures struct {
	HasWhere     bool   `json:"has_where"`
	HasJoin      bool   `json:"has_join"`
	HasGroupBy   bool   `json:"has_group_by"`
	HasHaving    bool   `json:"has_having"`
	HasOrderBy   bool   `json:"has_order_by"`
	HasDistinct  bool   `json:"has_distinct"`
	HasSubquery  bool   `json:"has_subquery"`
	HasUnion     bool   `json:"has_union"`
	HasAggregate bool   `json:"has_aggregate"`
	HasCTE       bool   `json:"has_cte"`
	JoinType     string `json:"join_type"`
	JoinCount    int    `json:"join_count"`
}

// AnalyzeSQLFeatures 分析SQL特性
func AnalyzeSQLFeatures(sql string) *SQLFeatures {
	cleanSQL := RemoveSQLComments(sql)
	upperSQL := strings.ToUpper(cleanSQL)

	features := &SQLFeatures{}

	// 检测各种特性
	features.HasWhere = strings.Contains(upperSQL, " WHERE ")
	features.HasGroupBy = strings.Contains(upperSQL, " GROUP BY ")
	features.HasHaving = strings.Contains(upperSQL, " HAVING ")
	features.HasOrderBy = strings.Contains(upperSQL, " ORDER BY ")
	features.HasDistinct = strings.Contains(upperSQL, "DISTINCT ")
	features.HasUnion = strings.Contains(upperSQL, " UNION ")
	features.HasCTE = strings.HasPrefix(strings.TrimSpace(upperSQL), "WITH ")

	// 检测子查询
	selectCount := strings.Count(upperSQL, "SELECT")
	features.HasSubquery = selectCount > 1

	// 检测JOIN
	analyzeJoinFeatures(upperSQL, features)

	// 检测聚合函数
	features.HasAggregate = hasAggregateFunction(upperSQL)

	return features
}

// analyzeJoinFeatures 分析JOIN特性
func analyzeJoinFeatures(upperSQL string, features *SQLFeatures) {
	features.HasJoin = strings.Contains(upperSQL, " JOIN ")
	if !features.HasJoin {
		return
	}

	leftCount := strings.Count(upperSQL, " LEFT JOIN ") + strings.Count(upperSQL, " LEFT OUTER JOIN ")
	rightCount := strings.Count(upperSQL, " RIGHT JOIN ") + strings.Count(upperSQL, " RIGHT OUTER JOIN ")
	fullCount := strings.Count(upperSQL, " FULL JOIN ") + strings.Count(upperSQL, " FULL OUTER JOIN ")
	crossCount := strings.Count(upperSQL, " CROSS JOIN ")
	innerCount := strings.Count(upperSQL, " INNER JOIN ")
	totalJoin := strings.Count(upperSQL, " JOIN ")

	// 没有明确类型的JOIN默认是INNER（排除已识别的类型）
	implicitInner := totalJoin - leftCount - rightCount - fullCount - crossCount - innerCount
	if implicitInner > 0 {
		innerCount += implicitInner
	}

	features.JoinCount = totalJoin

	var joinTypes []string
	if innerCount > 0 {
		joinTypes = append(joinTypes, fmt.Sprintf("INNER:%d", innerCount))
	}
	if leftCount > 0 {
		joinTypes = append(joinTypes, fmt.Sprintf("LEFT:%d", leftCount))
	}
	if rightCount > 0 {
		joinTypes = append(joinTypes, fmt.Sprintf("RIGHT:%d", rightCount))
	}
	if fullCount > 0 {
		joinTypes = append(joinTypes, fmt.Sprintf("FULL:%d", fullCount))
	}
	if crossCount > 0 {
		joinTypes = append(joinTypes, fmt.Sprintf("CROSS:%d", crossCount))
	}
	features.JoinType = strings.Join(joinTypes, ", ")
}

// hasAggregateFunction 检测是否有聚合函数
func hasAggregateFunction(upperSQL string) bool {
	aggregateFunctions := []string{"COUNT(", "SUM(", "AVG(", "MAX(", "MIN(", "GROUP_CONCAT("}
	for _, fn := range aggregateFunctions {
		if strings.Contains(upperSQL, fn) {
			return true
		}
	}
	return false
}

// GetSQLCategory 获取SQL分类
func GetSQLCategory(sqlType string) string {
	switch sqlType {
	case "SELECT", "WITH":
		return "DQL"
	case "INSERT", "UPDATE", "DELETE":
		return "DML"
	case "CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME":
		return "DDL"
	case "GRANT", "REVOKE":
		return "DCL"
	case "COMMIT", "ROLLBACK", "SAVEPOINT":
		return "TCL"
	case "SHOW", "DESCRIBE", "DESC", "EXPLAIN":
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

// GetSQLType 获取SQL类型
func GetSQLType(sql string) string {
	sql = TrimSQL(sql)
	sql = NormalizeWhitespace(sql)
	sqlUpper := ToUpperSQL(sql)

	keywords := []string{
		"SELECT", "WITH", "INSERT", "UPDATE", "DELETE",
		"CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME",
		"GRANT", "REVOKE", "COMMIT", "ROLLBACK", "SAVEPOINT",
		"SHOW", "DESCRIBE", "DESC", "EXPLAIN", "USE",
	}

	for _, kw := range keywords {
		if strings.HasPrefix(sqlUpper, kw+" ") || sqlUpper == kw {
			if kw == "DESC" {
				return "DESCRIBE"
			}
			return kw
		}
	}

	return "UNKNOWN"
}
