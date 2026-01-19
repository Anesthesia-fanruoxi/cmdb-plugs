package common

import (
	"fmt"
	"regexp"
	"strings"
)

// ExtractDatabases 提取SQL中涉及的数据库名
func ExtractDatabases(sql string) []string {
	dbMap := make(map[string]bool)

	re := regexp.MustCompile(`(?i)(?:FROM|JOIN|INTO|UPDATE)\s+(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?\s*\.\s*(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) > 1 {
			dbName := match[1]
			dbMap[dbName] = true
		}
	}

	databases := make([]string, 0, len(dbMap))
	for db := range dbMap {
		databases = append(databases, db)
	}

	return databases
}

// ExtractTables 提取SQL中涉及的表名（已废弃，使用 ExtractTablesWithAlias）
func ExtractTables(sql string) []string {
	tables := ExtractTablesWithAlias(sql)
	result := make([]string, 0, len(tables))
	for _, t := range tables {
		result = append(result, t.Name)
	}
	return result
}

// ExtractTablesWithAlias 提取SQL中涉及的表名和别名
// 返回所有表，包括真实表和 CTE 表
// 注意：同一张表的不同别名会分别记录
func ExtractTablesWithAlias(sql string) []TableInfo {
	var tables []TableInfo
	seen := make(map[string]bool) // 用于去重：表名-别名组合

	sql = RemoveSQLComments(sql)

	// 提取CTE名称
	cteNames := extractCTENames(sql)

	// 调试日志:打印提取到的 CTE 名称
	//if len(cteNames) > 0 {
	//	common.Logger.Infof("提取到的 CTE 名称: %v", cteNames)
	//}

	// 先移除函数调用中的内容，避免函数参数中的 FROM 关键字被误识别
	// 例如：SUBSTRING(field FROM position) 中的 FROM 不是表引用
	cleanSQL := removeFunctionCalls(sql)

	// 正则表达式：匹配 FROM/JOIN 后的表名和可选别名
	// 别名部分排除 SQL 关键字（包括 JOIN、ON、WHERE 等）
	re := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+` +
		`(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?` + // 表名或数据库名
		`(?:\.(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?)?` + // 可选的 .表名
		`(?:\s+(?:AS\s+)?(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?)?` + // 可选的别名
		`(?:\s|,|$|;|\)|JOIN|ON|WHERE|GROUP|ORDER|LIMIT|UNION|HAVING)`) // 别名后必须跟这些分隔符

	matches := re.FindAllStringSubmatch(cleanSQL, -1)

	// 额外提取子查询别名：FROM (...) AS alias 或 JOIN (...) AS alias
	// 需要正确处理嵌套括号
	subqueryAliases := extractSubqueryAliases(sql)

	// 先处理子查询别名（这些是派生表）
	for _, alias := range subqueryAliases {
		if alias != "" && !isKeyword(alias) {
			key := "subquery-" + alias
			if !seen[key] {
				seen[key] = true
				tables = append(tables, TableInfo{
					Name:       alias, // 子查询没有真实表名，使用别名作为名称
					Alias:      alias,
					IsCTE:      false,
					CTEName:    "",
					IsSubquery: true, // 标记为子查询派生表
				})
			}
		}
	}

	for _, match := range matches {
		var tableName, alias string
		var fullName string
		var isCTE bool

		if len(match) > 2 && match[2] != "" {
			// 有数据库前缀：db.table（一定是真实表）
			dbName := match[1]
			tableName = match[2]
			fullName = dbName + "." + tableName
			isCTE = false
			if len(match) > 3 {
				alias = strings.TrimSpace(match[3])
			}
		} else {
			// 无数据库前缀：table（可能是 CTE 或真实表）
			tableName = match[1]
			fullName = tableName
			// 检查是否为 CTE
			isCTE = cteNames[tableName] || cteNames[strings.ToLower(tableName)] || cteNames[strings.ToUpper(tableName)]
			if len(match) > 3 {
				alias = strings.TrimSpace(match[3])
			}
		}

		// 过滤掉无效表名
		if !IsValidTableName(tableName) {
			continue
		}

		// 如果别名是SQL关键字，说明没有别名
		if isKeyword(alias) {
			alias = ""
		}

		// 使用 表名-别名 作为唯一键（允许同一张表有多个别名）
		key := fullName
		if alias != "" {
			key = fullName + "-" + alias
		}

		if !seen[key] {
			seen[key] = true
			tables = append(tables, TableInfo{
				Name:  fullName,
				Alias: alias,
				IsCTE: isCTE,
				CTEName: func() string {
					if isCTE {
						return tableName
					} else {
						return ""
					}
				}(),
				IsSubquery: false,
			})
		}
	}

	return tables
}

// isKeyword 判断是否为SQL关键字
func isKeyword(word string) bool {
	keywords := map[string]bool{
		"ON": true, "WHERE": true, "AND": true, "OR": true,
		"LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
		"JOIN": true, "GROUP": true, "ORDER": true, "LIMIT": true,
		"HAVING": true, "UNION": true, "SELECT": true, "FROM": true,
	}
	return keywords[strings.ToUpper(word)]
}

// extractSubqueryAliases 提取子查询别名
// 正确处理嵌套括号：FROM (SELECT ... FROM (SELECT ...) ...) AS alias
func extractSubqueryAliases(sql string) []string {
	var aliases []string
	upperSQL := strings.ToUpper(sql)

	// 遍历 SQL,查找 FROM 或 JOIN 后跟 (
	i := 0
	for i < len(sql) {
		// 查找 FROM 或 JOIN
		fromIdx := strings.Index(upperSQL[i:], "FROM")
		joinIdx := strings.Index(upperSQL[i:], "JOIN")

		var keywordIdx int
		var keywordLen int

		if fromIdx >= 0 && (joinIdx < 0 || fromIdx < joinIdx) {
			keywordIdx = i + fromIdx
			keywordLen = 4
		} else if joinIdx >= 0 {
			keywordIdx = i + joinIdx
			keywordLen = 4
		} else {
			break
		}

		// 跳过关键字
		pos := keywordIdx + keywordLen

		// 跳过空格
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}

		// 检查是否是左括号（子查询）
		if pos < len(sql) && sql[pos] == '(' {
			// 找到匹配的右括号
			depth := 1
			pos++
			for pos < len(sql) && depth > 0 {
				if sql[pos] == '(' {
					depth++
				} else if sql[pos] == ')' {
					depth--
				}
				pos++
			}

			// 现在 pos 指向右括号后的位置
			// 跳过空格
			for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
				pos++
			}

			// 检查是否有 AS 关键字
			if pos+2 < len(sql) && strings.ToUpper(sql[pos:pos+2]) == "AS" {
				pos += 2
				// 跳过空格
				for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
					pos++
				}
			}

			// 提取别名
			aliasStart := pos
			for pos < len(sql) && (isAlphaNumeric(rune(sql[pos])) || sql[pos] == '_') {
				pos++
			}

			if pos > aliasStart {
				alias := sql[aliasStart:pos]
				aliases = append(aliases, alias)
			}

			i = pos
		} else {
			i = pos
		}
	}

	return aliases
}

// removeFunctionCalls 移除SQL中的函数调用，避免函数参数中的关键字被误识别
// 例如：SUBSTRING(field FROM position) → SUBSTRING()
// 保留函数名和括号，但清空括号内的内容
func removeFunctionCalls(sql string) string {
	result := []rune(sql)
	depth := 0
	inFunction := false
	functionStart := -1

	for i := 0; i < len(result); i++ {
		if result[i] == '(' {
			if depth == 0 {
				// 检查左括号前是否是函数名（字母、数字、下划线）
				if i > 0 {
					j := i - 1
					for j >= 0 && (isAlphaNumeric(result[j]) || result[j] == '_') {
						j--
					}
					if j < i-1 {
						// 找到函数名，标记为函数调用
						inFunction = true
						functionStart = j + 1
					}
				}
			}
			depth++
		} else if result[i] == ')' {
			depth--
			if depth == 0 && inFunction {
				// 函数调用结束，清空括号内的内容
				for j := functionStart; j < i; j++ {
					if result[j] == '(' {
						break
					}
				}
				// 找到左括号位置
				leftParen := -1
				for j := functionStart; j < i; j++ {
					if result[j] == '(' {
						leftParen = j
						break
					}
				}
				if leftParen > 0 {
					// 清空括号内的内容（保留括号）
					for j := leftParen + 1; j < i; j++ {
						result[j] = ' '
					}
				}
				inFunction = false
			}
		}
	}

	return string(result)
}

// isAlphaNumeric 判断字符是否为字母或数字
func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// extractCTENames 提取CTE名称
func extractCTENames(sql string) map[string]bool {
	cteNames := make(map[string]bool)

	upperSQL := strings.ToUpper(sql)
	if !strings.HasPrefix(strings.TrimSpace(upperSQL), "WITH") {
		return cteNames
	}

	// 匹配 WITH name AS 或 WITH RECURSIVE name AS 或 , name AS 格式
	// WITH t1 AS (...), t2 AS (...)
	// WITH RECURSIVE numbers AS (...)
	re := regexp.MustCompile(`(?i)(?:WITH\s+(?:RECURSIVE\s+)?|,\s*)([a-zA-Z_][a-zA-Z0-9_]*)\s+AS\s*\(`)
	matches := re.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) > 1 {
			cteName := strings.ToLower(match[1])
			cteNames[cteName] = true
			// 同时存大写版本，因为表名比较时可能大小写不一致
			cteNames[strings.ToUpper(match[1])] = true
			cteNames[match[1]] = true
		}
	}

	return cteNames
}

// ExtractColumns 提取SQL中涉及的字段名
// 返回格式：字段名 或 字段名(别名)
func ExtractColumns(sql string) []string {
	var columns []string

	sql = RemoveSQLComments(sql)

	upperSQL := strings.ToUpper(sql)
	if strings.HasPrefix(strings.TrimSpace(upperSQL), "WITH") {
		lastSelectIdx := FindLastMainSelect(sql)
		if lastSelectIdx > 0 {
			sql = sql[lastSelectIdx:]
			upperSQL = strings.ToUpper(sql)
		}
	}

	selectIdx := strings.Index(upperSQL, "SELECT")
	fromIdx := strings.Index(upperSQL, "FROM")

	if selectIdx == -1 || fromIdx == -1 || selectIdx >= fromIdx {
		return []string{}
	}

	selectClause := sql[selectIdx+6 : fromIdx]
	selectClause = strings.TrimSpace(selectClause)

	upperClause := strings.ToUpper(selectClause)
	if upperClause == "*" || strings.HasPrefix(upperClause, "DISTINCT *") {
		return []string{"*"}
	}

	selectClause = strings.TrimPrefix(selectClause, "DISTINCT ")
	selectClause = strings.TrimPrefix(selectClause, "distinct ")

	fields := SplitByComma(selectClause)

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" || field == "*" {
			continue
		}

		columnName := extractColumnName(field)
		if columnName != "" && columnName != "*" {
			columns = append(columns, columnName)
		}
	}

	return columns
}

// extractColumnName 从字段表达式中提取字段名
func extractColumnName(field string) string {
	field = strings.TrimSpace(field)
	var alias string
	var columnPart string

	if idx := strings.Index(strings.ToUpper(field), " AS "); idx > 0 {
		alias = strings.TrimSpace(field[idx+4:])
		alias = strings.Trim(alias, "`'\"")
		columnPart = strings.TrimSpace(field[:idx])
	} else {
		parts := splitFieldAndAlias(field)
		if len(parts) == 2 {
			columnPart = parts[0]
			alias = parts[1]
		} else {
			columnPart = field
		}
	}

	upperField := strings.ToUpper(columnPart)

	// 先检查是否是函数调用（优先级高于CASE检测）
	if strings.Contains(columnPart, "(") {
		// 判断表达式类型
		exprType := getExpressionType(upperField)
		if alias != "" {
			return fmt.Sprintf("%s(%s)", exprType, alias)
		}
		return exprType
	}

	// 纯CASE表达式（不在函数内）
	if strings.HasPrefix(strings.TrimSpace(upperField), "CASE ") {
		if alias != "" {
			return fmt.Sprintf("CASE表达式(%s)", alias)
		}
		return ""
	}

	if idx := strings.LastIndex(columnPart, "."); idx > 0 {
		columnPart = columnPart[idx+1:]
	}

	columnPart = strings.Trim(columnPart, "` \t\n\r")

	if columnPart == "" {
		return ""
	}

	if alias != "" && alias != columnPart {
		return fmt.Sprintf("%s(%s)", columnPart, alias)
	}
	return columnPart
}

// getExpressionType 判断表达式类型
func getExpressionType(upperField string) string {
	// 聚合函数
	if strings.Contains(upperField, "COUNT(") ||
		strings.Contains(upperField, "SUM(") ||
		strings.Contains(upperField, "AVG(") ||
		strings.Contains(upperField, "MAX(") ||
		strings.Contains(upperField, "MIN(") {
		return "聚合表达式"
	}

	// 字符串函数
	if strings.Contains(upperField, "CONCAT(") ||
		strings.Contains(upperField, "SUBSTRING(") ||
		strings.Contains(upperField, "REPLACE(") ||
		strings.Contains(upperField, "TRIM(") {
		return "字符串表达式"
	}

	// 日期函数
	if strings.Contains(upperField, "DATE_FORMAT(") ||
		strings.Contains(upperField, "DATE(") ||
		strings.Contains(upperField, "NOW(") ||
		strings.Contains(upperField, "DATEDIFF(") {
		return "日期表达式"
	}

	// 数学函数
	if strings.Contains(upperField, "ROUND(") ||
		strings.Contains(upperField, "FLOOR(") ||
		strings.Contains(upperField, "CEIL(") ||
		strings.Contains(upperField, "ABS(") {
		return "数值表达式"
	}

	// 条件函数
	if strings.Contains(upperField, "IF(") ||
		strings.Contains(upperField, "IFNULL(") ||
		strings.Contains(upperField, "COALESCE(") ||
		strings.Contains(upperField, "NULLIF(") {
		return "条件表达式"
	}

	// 包含CASE的复杂表达式
	if strings.Contains(upperField, "CASE ") {
		return "CASE表达式"
	}

	return "函数表达式"
}

// splitFieldAndAlias 分割字段和空格别名
func splitFieldAndAlias(field string) []string {
	field = strings.TrimSpace(field)

	if strings.Contains(field, "(") {
		depth := 0
		for i, ch := range field {
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					rest := strings.TrimSpace(field[i+1:])
					if rest != "" && !strings.HasPrefix(strings.ToUpper(rest), "AS ") {
						return []string{strings.TrimSpace(field[:i+1]), rest}
					}
					return []string{field}
				}
			}
		}
		return []string{field}
	}

	lastSpace := strings.LastIndex(field, " ")
	if lastSpace > 0 {
		possibleAlias := strings.TrimSpace(field[lastSpace+1:])
		columnPart := strings.TrimSpace(field[:lastSpace])
		if IsValidAlias(possibleAlias) && columnPart != "" {
			return []string{columnPart, possibleAlias}
		}
	}

	return []string{field}
}

// extractFieldFromFunction 从函数调用中提取字段名
func extractFieldFromFunction(funcExpr string) string {
	start := strings.Index(funcExpr, "(")
	end := strings.LastIndex(funcExpr, ")")
	if start == -1 || end == -1 || start >= end {
		return ""
	}

	funcName := strings.TrimSpace(funcExpr[:start])
	innerContent := funcExpr[start+1 : end]

	parts := SplitByComma(innerContent)
	if len(parts) == 0 {
		return ""
	}

	firstArg := strings.TrimSpace(parts[0])

	for _, op := range []string{">", "<", "=", "!=", "<>", ">=", "<="} {
		if idx := strings.Index(firstArg, op); idx > 0 {
			firstArg = strings.TrimSpace(firstArg[:idx])
			break
		}
	}

	if strings.Contains(firstArg, "(") {
		return extractFieldFromFunction(firstArg)
	}

	if idx := strings.LastIndex(firstArg, "."); idx > 0 {
		firstArg = firstArg[idx+1:]
	}

	firstArg = strings.Trim(firstArg, "` \t\n\r")
	if firstArg == "" || firstArg == "*" {
		return funcName
	}

	return firstArg
}
