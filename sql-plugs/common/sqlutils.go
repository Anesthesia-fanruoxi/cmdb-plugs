package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultLimit = 100  // 默认返回记录数（无LIMIT时自动添加）
	MaxLimit     = 1000 // 最大返回记录数（用户LIMIT超过此值会被限制）
)

// NormalizeWhitespace 规范化SQL中的空白字符
// 将所有连续的空白字符（空格、制表符、换行符等）替换为单个空格
func NormalizeWhitespace(sql string) string {
	// 正则匹配所有连续的空白字符（包括 \r \n \t 和空格）
	re := regexp.MustCompile(`[\s]+`)
	return strings.TrimSpace(re.ReplaceAllString(sql, " "))
}

// ProcessSQLLimit 处理SQL语句的LIMIT限制
// 如果没有LIMIT，自动添加LIMIT DefaultLimit (100)
// 如果LIMIT超过MaxLimit (1000)，自动修正为LIMIT MaxLimit
// 只对SELECT和WITH查询语句添加limit，SHOW等命令语句不添加
func ProcessSQLLimit(sql string) string {
	sql = strings.TrimSpace(sql)

	// 判断SQL类型，只对SELECT和WITH语句添加limit
	sqlUpper := strings.ToUpper(sql)
	isQueryStatement := strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "WITH")

	if !isQueryStatement {
		// 不是查询语句（如SHOW、DESCRIBE等），直接返回不添加limit
		return sql
	}

	// 获取用户原始LIMIT值
	userLimit := GetUserOriginalLimit(sql)

	if userLimit == -1 {
		// 没有LIMIT，自动添加默认LIMIT
		return sql + fmt.Sprintf(" LIMIT %d", DefaultLimit)
	}

	if userLimit <= MaxLimit {
		// LIMIT在允许范围内，不修改
		return sql
	}

	// LIMIT超过限制，需要修正为MaxLimit
	return replaceLimitValue(sql, MaxLimit)
}

// GetUserOriginalLimit 获取用户原始SQL中的LIMIT值
// 返回-1表示没有LIMIT
func GetUserOriginalLimit(sql string) int {
	// LIMIT count OFFSET offset
	re1 := regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)\s+OFFSET\s+\d+`)
	if match := re1.FindStringSubmatch(sql); match != nil {
		limit, _ := strconv.Atoi(match[1])
		return limit
	}

	// LIMIT offset, count (MySQL风格)
	re2 := regexp.MustCompile(`(?i)\bLIMIT\s+\d+\s*,\s*(\d+)`)
	if match := re2.FindStringSubmatch(sql); match != nil {
		limit, _ := strconv.Atoi(match[1])
		return limit
	}

	// LIMIT count
	re3 := regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)
	if match := re3.FindStringSubmatch(sql); match != nil {
		limit, _ := strconv.Atoi(match[1])
		return limit
	}

	return -1
}

// replaceLimitValue 替换SQL中的LIMIT值为新值
func replaceLimitValue(sql string, newLimit int) string {
	// LIMIT count OFFSET offset
	re1 := regexp.MustCompile(`(?i)(\bLIMIT\s+)\d+(\s+OFFSET\s+\d+)`)
	if re1.MatchString(sql) {
		return re1.ReplaceAllString(sql, fmt.Sprintf("${1}%d${2}", newLimit))
	}

	// LIMIT offset, count
	re2 := regexp.MustCompile(`(?i)(\bLIMIT\s+\d+\s*,\s*)\d+`)
	if re2.MatchString(sql) {
		return re2.ReplaceAllString(sql, fmt.Sprintf("${1}%d", newLimit))
	}

	// LIMIT count
	re3 := regexp.MustCompile(`(?i)(\bLIMIT\s+)\d+`)
	return re3.ReplaceAllString(sql, fmt.Sprintf("${1}%d", newLimit))
}

// TrimSQL 去除SQL首尾空白和开头的注释
func TrimSQL(sql string) string {
	sql = strings.TrimSpace(sql)

	// 循环去除开头的注释（可能有多行注释）
	for {
		// 去除开头的单行注释 --
		if strings.HasPrefix(sql, "--") {
			// 找到换行符，去除这一行
			if idx := strings.Index(sql, "\n"); idx >= 0 {
				sql = strings.TrimSpace(sql[idx+1:])
				continue
			} else {
				// 整个SQL都是注释
				return ""
			}
		}

		// 去除开头的单行注释 # (MySQL特有)
		if strings.HasPrefix(sql, "#") {
			// 找到换行符，去除这一行
			if idx := strings.Index(sql, "\n"); idx >= 0 {
				sql = strings.TrimSpace(sql[idx+1:])
				continue
			} else {
				// 整个SQL都是注释
				return ""
			}
		}

		// 去除开头的多行注释 /* */
		if strings.HasPrefix(sql, "/*") {
			if idx := strings.Index(sql, "*/"); idx >= 0 {
				sql = strings.TrimSpace(sql[idx+2:])
				continue
			} else {
				// 注释没有结束
				return ""
			}
		}

		// 没有更多注释了
		break
	}

	return sql
}

// ToUpperSQL 转换SQL为大写（用于判断类型）
func ToUpperSQL(sql string) string {
	return strings.ToUpper(sql)
}

// HasPrefix 判断SQL是否以指定关键字开头（关键字后必须是空白字符或结束）
func HasPrefix(sql string, prefix string) bool {
	if !strings.HasPrefix(sql, prefix) {
		return false
	}
	// 如果SQL长度等于关键字长度，说明整个SQL就是这个关键字
	if len(sql) == len(prefix) {
		return true
	}
	// 检查关键字后面的字符是否是空白字符
	nextChar := sql[len(prefix)]
	return nextChar == ' ' || nextChar == '\t' || nextChar == '\n' || nextChar == '\r'
}

// IsReadOnlySQL 检查SQL是否为只读查询语句（DQL）
// 只允许：SELECT、SHOW、DESCRIBE、DESC、EXPLAIN
// 拒绝：INSERT、UPDATE、DELETE、DROP、CREATE、ALTER、TRUNCATE等
func IsReadOnlySQL(sql string) bool {
	sql = TrimSQL(sql)
	if sql == "" {
		return false
	}

	upperSQL := strings.ToUpper(sql)

	// 允许的只读语句前缀
	allowedPrefixes := []string{
		"SELECT",
		"SHOW",
		"DESCRIBE",
		"DESC",
		"EXPLAIN",
		"WITH", // CTE查询
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(upperSQL, prefix) {
			return true
		}
	}

	return false
}

// HasFilterConditions 检测 SQL 是否包含过滤条件或结果集限制
// 有过滤条件的查询通常结果集较小，可以直接返回全部数据
// 无过滤条件的查询（如 SELECT * FROM table）需要强制 LIMIT
//
// 判定为"有过滤条件"的情况：
//  1. WHERE/HAVING/JOIN - 过滤数据
//  2. GROUP BY/DISTINCT - 聚合/去重
//  3. LIMIT - 用户指定限制
//  4. 聚合函数（无 GROUP BY）- 结果只有1行
func HasFilterConditions(sql string) bool {
	upperSQL := strings.ToUpper(sql)

	// 检测是否包含过滤条件
	// 使用正则表达式匹配,支持前后有空格、换行、制表符等
	filterPatterns := []string{
		`\bWHERE\b`,      // WHERE 条件
		`\bHAVING\b`,     // HAVING 条件
		`\bLIMIT\b`,      // 用户指定的 LIMIT
		`\bGROUP\s+BY\b`, // GROUP BY 聚合
		`\bDISTINCT\b`,   // DISTINCT 去重
	}

	for _, pattern := range filterPatterns {
		if matched, _ := regexp.MatchString(pattern, upperSQL); matched {
			return true
		}
	}

	// 检测是否包含 JOIN（JOIN 通常会过滤数据）
	joinPatterns := []string{
		`\bJOIN\b`,
		`\bLEFT\s+JOIN\b`,
		`\bRIGHT\s+JOIN\b`,
		`\bINNER\s+JOIN\b`,
		`\bOUTER\s+JOIN\b`,
	}

	for _, pattern := range joinPatterns {
		if matched, _ := regexp.MatchString(pattern, upperSQL); matched {
			return true
		}
	}

	// 检测聚合函数（无 GROUP BY）：结果只有1行，不是危险查询
	// 例如：SELECT COUNT(*) FROM users, SELECT AVG(age) FROM users
	if !strings.Contains(upperSQL, "GROUP BY") && !regexp.MustCompile(`\bGROUP\s+BY\b`).MatchString(upperSQL) {
		// 提取 SELECT 和 FROM 之间的内容
		selectIdx := strings.Index(upperSQL, "SELECT")
		fromIdx := strings.Index(upperSQL, "FROM")

		if selectIdx != -1 && fromIdx != -1 && selectIdx < fromIdx {
			selectClause := upperSQL[selectIdx+6 : fromIdx] // +6 跳过 "SELECT"

			// 检测常见聚合函数
			aggregateFunctions := []string{
				"COUNT(", "SUM(", "AVG(", "MAX(", "MIN(",
			}

			for _, fn := range aggregateFunctions {
				if strings.Contains(selectClause, fn) {
					return true // 聚合查询，结果只有1行
				}
			}
		}
	}

	return false
}

// BuildCountSQL 构建 COUNT 查询
// 将 SELECT * FROM users ORDER BY id LIMIT 100 转换为 SELECT COUNT(*) FROM users
func BuildCountSQL(sql string) string {
	sql = strings.TrimSpace(sql)

	// 移除 LIMIT（使用字符串拼接避免正则被损坏）
	limitPattern := `(?i)\s+LIMIT\s+\d+(\s*,\s*\d+)?(\s+OFFSET\s+\d+)?\s*` + "$"
	limitRegex := regexp.MustCompile(limitPattern)
	sql = limitRegex.ReplaceAllString(sql, "")

	// 移除 ORDER BY
	orderByPattern := `(?i)\s+ORDER\s+BY\s+[^;]+` + "$"
	orderByRegex := regexp.MustCompile(orderByPattern)
	sql = orderByRegex.ReplaceAllString(sql, "")
	sql = strings.TrimSpace(sql)

	upperSQL := strings.ToUpper(sql)
	selectIdx := strings.Index(upperSQL, "SELECT")
	fromIdx := strings.Index(upperSQL, "FROM")

	if selectIdx == -1 || fromIdx == -1 || selectIdx >= fromIdx {
		return sql
	}

	return "SELECT COUNT(*) " + sql[fromIdx:]
}

// IsValidDatabaseName 验证数据库名是否合法
// 只允许字母、数字、下划线
func IsValidDatabaseName(dbName string, maxLen int) bool {
	if len(dbName) == 0 || len(dbName) > maxLen {
		return false
	}
	pattern := `^[a-zA-Z0-9_]+` + "$"
	matched, _ := regexp.MatchString(pattern, dbName)
	return matched
}
