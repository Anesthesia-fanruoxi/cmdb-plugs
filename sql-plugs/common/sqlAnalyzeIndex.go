package common

import (
	"regexp"
	"strings"
)

// IndexSuggestion 索引建议
type IndexSuggestion struct {
	Table     string   `json:"table"`      // 表名
	Columns   []string `json:"columns"`    // 建议索引的字段
	IndexType string   `json:"index_type"` // 索引类型: single/composite
	Priority  string   `json:"priority"`   // 优先级: high/medium/low
	Reason    string   `json:"reason"`     // 建议原因
	CreateSQL string   `json:"create_sql"` // 创建索引的SQL
}

// AnalyzeIndexSuggestions 分析SQL并给出索引建议
func AnalyzeIndexSuggestions(sql string, structure *SQLStructure) []IndexSuggestion {
	var suggestions []IndexSuggestion

	if structure == nil {
		return suggestions
	}

	// 收集CTE名称（这些不是真实表，不需要索引）
	cteNames := make(map[string]bool)
	if structure.CTEs != nil {
		for _, cte := range structure.CTEs {
			cteNames[strings.ToLower(cte.Name)] = true
		}
	}

	// 收集表别名映射
	tableAliasMap := buildTableAliasMap(structure)

	// 1. 分析WHERE条件字段
	whereFields := analyzeWhereFields(structure, tableAliasMap)

	// 2. 分析JOIN条件字段
	joinFields := analyzeJoinFields(structure, tableAliasMap)

	// 3. 分析ORDER BY字段
	orderByFields := analyzeOrderByFields(structure, tableAliasMap)

	// 4. 分析GROUP BY字段
	groupByFields := analyzeGroupByFields(structure, tableAliasMap)

	// 5. 如果有CTE，分析整个SQL中的真实表
	if len(cteNames) > 0 {
		realTableFields := analyzeRealTablesFromSQL(sql, cteNames)
		for table, fields := range realTableFields {
			if whereFields[table] == nil {
				whereFields[table] = fields
			} else {
				whereFields[table] = appendUniqueSlice(whereFields[table], fields)
			}
		}
	}

	// 按表分组生成建议
	tableFieldsMap := make(map[string]*tableIndexInfo)

	// 合并所有字段信息
	mergeFields(tableFieldsMap, whereFields, "where")
	mergeFields(tableFieldsMap, joinFields, "join")
	mergeFields(tableFieldsMap, orderByFields, "order")
	mergeFields(tableFieldsMap, groupByFields, "group")

	// 过滤掉CTE表名
	for cteName := range cteNames {
		delete(tableFieldsMap, cteName)
	}

	// 生成索引建议
	for table, info := range tableFieldsMap {
		suggestion := generateSuggestion(table, info)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions
}

// analyzeRealTablesFromSQL 从整个SQL中分析真实表的字段使用情况
func analyzeRealTablesFromSQL(sql string, cteNames map[string]bool) map[string][]string {
	result := make(map[string][]string)

	// 建立别名到真实表名的映射
	aliasToTable := make(map[string]string)

	// 1. 匹配 FROM/JOIN 后面的 database.table alias 或 database.table AS alias
	// 例如: from jxh_system.sys_credit_repayment_plan p
	//       left join jxh_system.sys_credit_order_detailed o
	tableRe := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*)\s+(?:AS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)`)
	tableMatches := tableRe.FindAllStringSubmatch(sql, -1)

	for _, m := range tableMatches {
		if len(m) >= 3 {
			fullTableName := m[1] // database.table
			alias := strings.ToLower(m[2])

			// 提取表名（去掉数据库前缀）
			parts := strings.Split(fullTableName, ".")
			tableName := parts[len(parts)-1]

			// 跳过CTE名称
			if cteNames[strings.ToLower(tableName)] {
				continue
			}

			aliasToTable[alias] = tableName
			aliasToTable[strings.ToLower(tableName)] = tableName
		}
	}

	// 2. 分析WHERE条件中的字段
	// 匹配整个WHERE子句（可能跨多行）
	whereRe := regexp.MustCompile(`(?is)WHERE\s+(.+?)(?:GROUP\s+BY|HAVING|ORDER\s+BY|LIMIT|UNION|\)\s*,|\)\s*$)`)
	whereMatches := whereRe.FindAllStringSubmatch(sql, -1)

	for _, wm := range whereMatches {
		if len(wm) >= 2 {
			wherePart := wm[1]
			// 提取 alias.column 格式的字段（在比较操作符前）
			fieldRe := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)\s*[=<>!]`)
			fieldMatches := fieldRe.FindAllStringSubmatch(wherePart, -1)

			for _, fm := range fieldMatches {
				if len(fm) >= 3 {
					alias := strings.ToLower(fm[1])
					column := fm[2]

					// 查找真实表名
					if tableName, ok := aliasToTable[alias]; ok {
						result[tableName] = appendUnique(result[tableName], column)
					}
				}
			}
		}
	}

	// 3. 分析JOIN ON条件中的字段
	joinRe := regexp.MustCompile(`(?i)\bON\s+([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)`)
	joinMatches := joinRe.FindAllStringSubmatch(sql, -1)

	for _, jm := range joinMatches {
		if len(jm) >= 5 {
			alias1 := strings.ToLower(jm[1])
			column1 := jm[2]
			alias2 := strings.ToLower(jm[3])
			column2 := jm[4]

			// 查找真实表名并添加字段
			if tableName, ok := aliasToTable[alias1]; ok {
				result[tableName] = appendUnique(result[tableName], column1)
			}
			if tableName, ok := aliasToTable[alias2]; ok {
				result[tableName] = appendUnique(result[tableName], column2)
			}
		}
	}

	return result
}

// tableIndexInfo 表的索引信息
type tableIndexInfo struct {
	whereFields   []string
	joinFields    []string
	orderByFields []string
	groupByFields []string
}

// buildTableAliasMap 构建表别名映射
func buildTableAliasMap(structure *SQLStructure) map[string]string {
	aliasMap := make(map[string]string) // alias -> tableName

	if structure.FromClause != nil {
		if structure.FromClause.MainTable.Name != "" {
			tableName := cleanTableName(structure.FromClause.MainTable.Name)
			if structure.FromClause.MainTable.Alias != "" {
				aliasMap[structure.FromClause.MainTable.Alias] = tableName
			}
			aliasMap[tableName] = tableName
		}

		for _, join := range structure.FromClause.Joins {
			tableName := cleanTableName(join.Table)
			if join.Alias != "" {
				aliasMap[join.Alias] = tableName
			}
			aliasMap[tableName] = tableName
		}
	}

	return aliasMap
}

// cleanTableName 清理表名（移除数据库前缀和反引号）
func cleanTableName(name string) string {
	name = strings.Trim(name, "`")
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

// analyzeWhereFields 分析WHERE条件中的字段
func analyzeWhereFields(structure *SQLStructure, aliasMap map[string]string) map[string][]string {
	result := make(map[string][]string)

	if structure.WhereClause == nil {
		return result
	}

	// 获取主表名（用于没有表前缀的字段）
	mainTable := ""
	if structure.FromClause != nil && structure.FromClause.MainTable.Name != "" {
		mainTable = cleanTableName(structure.FromClause.MainTable.Name)
	}

	// 从WHERE子句中提取字段
	for _, field := range structure.WhereClause.Fields {
		table, col := parseFieldWithTable(field, aliasMap)
		// 如果没有表名，使用主表
		if table == "" && mainTable != "" {
			table = mainTable
		}
		if table != "" && col != "" {
			result[table] = appendUnique(result[table], col)
		}
	}

	return result
}

// analyzeJoinFields 分析JOIN条件中的字段
func analyzeJoinFields(structure *SQLStructure, aliasMap map[string]string) map[string][]string {
	result := make(map[string][]string)

	if structure.FromClause == nil {
		return result
	}

	for _, join := range structure.FromClause.Joins {
		if join.Condition == "" {
			continue
		}

		// 从ON条件中提取字段
		fields := extractFieldsFromCondition(join.Condition)
		for _, field := range fields {
			table, col := parseFieldWithTable(field, aliasMap)
			if table != "" && col != "" {
				result[table] = appendUnique(result[table], col)
			}
		}
	}

	return result
}

// analyzeOrderByFields 分析ORDER BY字段
func analyzeOrderByFields(structure *SQLStructure, aliasMap map[string]string) map[string][]string {
	result := make(map[string][]string)

	if structure.OrderByClause == nil {
		return result
	}

	for _, order := range structure.OrderByClause.Fields {
		table, col := parseFieldWithTable(order.Field, aliasMap)
		if table != "" && col != "" {
			result[table] = appendUnique(result[table], col)
		} else if col == "" && order.Field != "" {
			// 没有表前缀的字段，尝试关联到主表
			if structure.FromClause != nil && structure.FromClause.MainTable.Name != "" {
				mainTable := cleanTableName(structure.FromClause.MainTable.Name)
				result[mainTable] = appendUnique(result[mainTable], strings.Trim(order.Field, "`"))
			}
		}
	}

	return result
}

// analyzeGroupByFields 分析GROUP BY字段
func analyzeGroupByFields(structure *SQLStructure, aliasMap map[string]string) map[string][]string {
	result := make(map[string][]string)

	if structure.GroupByClause == nil {
		return result
	}

	for _, field := range structure.GroupByClause.Fields {
		table, col := parseFieldWithTable(field, aliasMap)
		if table != "" && col != "" {
			result[table] = appendUnique(result[table], col)
		} else if col == "" && field != "" {
			// 没有表前缀的字段
			if structure.FromClause != nil && structure.FromClause.MainTable.Name != "" {
				mainTable := cleanTableName(structure.FromClause.MainTable.Name)
				result[mainTable] = appendUnique(result[mainTable], strings.Trim(field, "`"))
			}
		}
	}

	return result
}

// parseFieldWithTable 解析字段，返回表名和字段名
func parseFieldWithTable(field string, aliasMap map[string]string) (string, string) {
	field = strings.Trim(field, "`")

	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		if len(parts) >= 2 {
			tableOrAlias := parts[len(parts)-2]
			colName := parts[len(parts)-1]

			// 查找真实表名
			if realTable, ok := aliasMap[tableOrAlias]; ok {
				return realTable, colName
			}
			return tableOrAlias, colName
		}
	}

	return "", field
}

// extractFieldsFromCondition 从条件表达式中提取字段
func extractFieldsFromCondition(condition string) []string {
	var fields []string

	// 匹配 table.column 或 column 格式
	re := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)(?:\s*[=<>!]|\s+(?:IN|LIKE|BETWEEN|IS))`)
	matches := re.FindAllStringSubmatch(condition, -1)

	for _, m := range matches {
		if len(m) >= 2 {
			field := m[1]
			upperField := strings.ToUpper(field)
			// 排除关键字
			if upperField != "AND" && upperField != "OR" && upperField != "NOT" && upperField != "NULL" {
				fields = append(fields, field)
			}
		}
	}

	return fields
}

// mergeFields 合并字段到表索引信息
func mergeFields(tableMap map[string]*tableIndexInfo, fields map[string][]string, fieldType string) {
	for table, cols := range fields {
		if _, ok := tableMap[table]; !ok {
			tableMap[table] = &tableIndexInfo{}
		}

		switch fieldType {
		case "where":
			tableMap[table].whereFields = append(tableMap[table].whereFields, cols...)
		case "join":
			tableMap[table].joinFields = append(tableMap[table].joinFields, cols...)
		case "order":
			tableMap[table].orderByFields = append(tableMap[table].orderByFields, cols...)
		case "group":
			tableMap[table].groupByFields = append(tableMap[table].groupByFields, cols...)
		}
	}
}

// generateSuggestion 生成索引建议
func generateSuggestion(table string, info *tableIndexInfo) *IndexSuggestion {
	var columns []string
	var reasons []string
	priority := "low"

	// JOIN字段优先级最高
	if len(info.joinFields) > 0 {
		columns = appendUniqueSlice(columns, info.joinFields)
		reasons = append(reasons, "JOIN连接条件")
		priority = "high"
	}

	// WHERE字段优先级高
	if len(info.whereFields) > 0 {
		columns = appendUniqueSlice(columns, info.whereFields)
		reasons = append(reasons, "WHERE过滤条件")
		if priority != "high" {
			priority = "high"
		}
	}

	// GROUP BY字段
	if len(info.groupByFields) > 0 {
		columns = appendUniqueSlice(columns, info.groupByFields)
		reasons = append(reasons, "GROUP BY分组")
		if priority == "low" {
			priority = "medium"
		}
	}

	// ORDER BY字段
	if len(info.orderByFields) > 0 {
		columns = appendUniqueSlice(columns, info.orderByFields)
		reasons = append(reasons, "ORDER BY排序")
		if priority == "low" {
			priority = "medium"
		}
	}

	if len(columns) == 0 {
		return nil
	}

	// 去重
	columns = uniqueStrings(columns)

	// 确定索引类型
	indexType := "single"
	if len(columns) > 1 {
		indexType = "composite"
	}

	// 生成创建索引的SQL
	indexName := "idx_" + table + "_" + strings.Join(columns, "_")
	if len(indexName) > 64 {
		indexName = indexName[:64]
	}
	createSQL := "CREATE INDEX " + indexName + " ON " + table + " (" + strings.Join(columns, ", ") + ");"

	return &IndexSuggestion{
		Table:     table,
		Columns:   columns,
		IndexType: indexType,
		Priority:  priority,
		Reason:    strings.Join(reasons, ", "),
		CreateSQL: createSQL,
	}
}

// appendUnique 追加不重复的元素
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// appendUniqueSlice 追加不重复的切片
func appendUniqueSlice(slice []string, items []string) []string {
	for _, item := range items {
		slice = appendUnique(slice, item)
	}
	return slice
}

// uniqueStrings 去重
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
