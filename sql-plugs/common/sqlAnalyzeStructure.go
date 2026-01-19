package common

import (
	"regexp"
	"strings"
)

// SQLStructure SQL结构分析结果
type SQLStructure struct {
	// SELECT子句
	SelectClause *SelectClause `json:"select_clause,omitempty"`
	// FROM子句
	FromClause *FromClause `json:"from_clause,omitempty"`
	// WHERE子句
	WhereClause *WhereClause `json:"where_clause,omitempty"`
	// GROUP BY子句
	GroupByClause *GroupByClause `json:"group_by_clause,omitempty"`
	// HAVING子句
	HavingClause *HavingClause `json:"having_clause,omitempty"`
	// ORDER BY子句
	OrderByClause *OrderByClause `json:"order_by_clause,omitempty"`
	// LIMIT子句
	LimitClause *LimitClause `json:"limit_clause,omitempty"`
	// 子查询
	Subqueries []SubqueryInfo `json:"subqueries,omitempty"`
	// 窗口函数
	WindowFunctions []WindowFunction `json:"window_functions,omitempty"`
	// CTE
	CTEs []CTEInfo `json:"ctes,omitempty"`
}

// SelectClause SELECT子句信息
type SelectClause struct {
	Raw        string      `json:"raw"`
	Fields     []FieldInfo `json:"fields"`
	HasStar    bool        `json:"has_star"`
	Aggregates []string    `json:"aggregates,omitempty"`
}

// FieldInfo 字段信息
type FieldInfo struct {
	Expression   string `json:"expression"`              // 原始表达式
	Alias        string `json:"alias,omitempty"`         // 别名
	SourceTable  string `json:"source_table,omitempty"`  // 来源表
	FieldName    string `json:"field_name,omitempty"`    // 字段名（不含表前缀）
	FieldType    string `json:"field_type"`              // 字段类型：column/function/expression/star
	FunctionName string `json:"function_name,omitempty"` // 函数名（如果是函数）
	IsAggregated bool   `json:"is_aggregated"`           // 是否是聚合函数
	IsWindow     bool   `json:"is_window"`               // 是否是窗口函数
}

// FromClause FROM子句信息
type FromClause struct {
	Raw       string     `json:"raw"`
	MainTable TableInfo  `json:"main_table"`
	Joins     []JoinInfo `json:"joins,omitempty"`
}

// TableInfo 表信息
type TableInfo struct {
	Name       string `json:"name"`
	Alias      string `json:"alias,omitempty"`
	IsCTE      bool   `json:"is_cte"`             // 是否为 CTE（WITH 临时表）
	CTEName    string `json:"cte_name,omitempty"` // CTE 名称（如果是 CTE）
	IsSubquery bool   `json:"is_subquery"`        // 是否为子查询派生表
}

// JoinInfo JOIN信息
type JoinInfo struct {
	Type      string `json:"type"`
	Table     string `json:"table"`
	Alias     string `json:"alias,omitempty"`
	Condition string `json:"condition"`
}

// WhereClause WHERE子句信息
type WhereClause struct {
	Raw        string   `json:"raw"`
	Conditions []string `json:"conditions"`
	Fields     []string `json:"fields"`
}

// GroupByClause GROUP BY子句信息
type GroupByClause struct {
	Raw    string   `json:"raw"`
	Fields []string `json:"fields"`
}

// HavingClause HAVING子句信息
type HavingClause struct {
	Raw        string   `json:"raw"`
	Conditions []string `json:"conditions"`
}

// OrderByClause ORDER BY子句信息
type OrderByClause struct {
	Raw    string      `json:"raw"`
	Fields []OrderInfo `json:"fields"`
}

// OrderInfo 排序信息
type OrderInfo struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// LimitClause LIMIT子句信息
type LimitClause struct {
	Raw    string `json:"raw"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// SubqueryInfo 子查询信息
type SubqueryInfo struct {
	Location string `json:"location"`
	Raw      string `json:"raw"`
	Type     string `json:"type"`
}

// WindowFunction 窗口函数信息
type WindowFunction struct {
	Function    string `json:"function"`
	PartitionBy string `json:"partition_by,omitempty"`
	OrderBy     string `json:"order_by,omitempty"`
	Raw         string `json:"raw"`
}

// CTEInfo CTE信息
type CTEInfo struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

// AnalyzeSQLStructure 分析SQL结构
func AnalyzeSQLStructure(sql string) *SQLStructure {
	cleanSQL := RemoveSQLComments(sql)
	structure := &SQLStructure{}

	// 分析CTE
	mainSQL := cleanSQL
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(cleanSQL)), "WITH ") {
		structure.CTEs = extractCTEs(cleanSQL)
		// 找到最后一个主SELECT（不在CTE内的）
		mainSQL = extractMainQuery(cleanSQL)
	}

	// 先分析FROM子句（需要用于推断字段来源表）
	structure.FromClause = extractFromClause(mainSQL)

	// 分析SELECT子句（传入FROM信息用于推断来源表）
	structure.SelectClause = extractSelectClauseWithFrom(mainSQL, structure.FromClause)

	// 分析WHERE子句
	structure.WhereClause = extractWhereClause(mainSQL)

	// 分析GROUP BY子句
	structure.GroupByClause = extractGroupByClause(mainSQL)

	// 分析HAVING子句
	structure.HavingClause = extractHavingClause(mainSQL)

	// 分析ORDER BY子句
	structure.OrderByClause = extractOrderByClause(mainSQL)

	// 分析LIMIT子句
	structure.LimitClause = extractLimitClause(mainSQL)

	// 分析子查询（使用原始SQL）
	structure.Subqueries = extractSubqueries(cleanSQL)

	// 分析窗口函数
	structure.WindowFunctions = extractWindowFunctions(cleanSQL)

	return structure
}

// extractMainQuery 从CTE查询中提取主查询
func extractMainQuery(sql string) string {
	// 找到最后一个不在括号内的SELECT
	upperSQL := strings.ToUpper(sql)
	depth := 0
	lastSelectIdx := -1

	for i := 0; i < len(sql)-6; i++ {
		if sql[i] == '(' {
			depth++
		} else if sql[i] == ')' {
			depth--
		} else if depth == 0 && upperSQL[i:i+6] == "SELECT" {
			// 确保是完整的关键字
			if i > 0 && isAlphaNum(sql[i-1]) {
				continue
			}
			if i+6 < len(sql) && isAlphaNum(sql[i+6]) {
				continue
			}
			lastSelectIdx = i
		}
	}

	if lastSelectIdx > 0 {
		return sql[lastSelectIdx:]
	}
	return sql
}

// extractSelectClauseWithFrom 提取SELECT子句（带FROM信息用于推断来源表）
func extractSelectClauseWithFrom(sql string, fromClause *FromClause) *SelectClause {
	clause := extractSelectClause(sql)
	if clause == nil || fromClause == nil {
		return clause
	}

	// 推断没有表前缀的字段的来源表
	defaultTable := ""
	if fromClause.MainTable.Name != "" {
		defaultTable = fromClause.MainTable.Name
	}

	for i := range clause.Fields {
		field := &clause.Fields[i]
		// 如果字段没有来源表，且不是函数/表达式，推断为主表
		if field.SourceTable == "" && field.FieldType == "column" && defaultTable != "" {
			field.SourceTable = defaultTable
		}
	}

	return clause
}

// extractSelectClause 提取SELECT子句
func extractSelectClause(sql string) *SelectClause {
	upperSQL := strings.ToUpper(sql)
	selectIdx := strings.Index(upperSQL, "SELECT")
	if selectIdx == -1 {
		return nil
	}

	// 找到FROM的位置
	fromIdx := findKeywordPosition(sql, "FROM", selectIdx+6)
	if fromIdx == -1 {
		fromIdx = len(sql)
	}

	selectPart := strings.TrimSpace(sql[selectIdx+6 : fromIdx])
	// 移除DISTINCT
	if strings.HasPrefix(strings.ToUpper(selectPart), "DISTINCT ") {
		selectPart = strings.TrimSpace(selectPart[9:])
	}

	clause := &SelectClause{
		Raw:     selectPart,
		HasStar: strings.Contains(selectPart, "*"),
	}

	// 解析字段
	fields := splitSelectFields(selectPart)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		field := parseFieldInfo(f)
		clause.Fields = append(clause.Fields, field)

		// 收集聚合函数
		if field.IsAggregated && field.FunctionName != "" {
			clause.Aggregates = append(clause.Aggregates, field.FunctionName)
		}
	}

	return clause
}

// parseFieldInfo 解析单个字段信息
func parseFieldInfo(f string) FieldInfo {
	field := FieldInfo{Expression: f}

	// 检查别名 (AS 或 空格分隔)
	upperF := strings.ToUpper(f)
	expression := f
	if asIdx := strings.LastIndex(upperF, " AS "); asIdx > 0 {
		expression = strings.TrimSpace(f[:asIdx])
		field.Alias = strings.Trim(strings.TrimSpace(f[asIdx+4:]), "`'\"")
	}
	field.Expression = expression

	// 判断字段类型
	upperExpr := strings.ToUpper(expression)

	// 检查是否是 *
	if strings.TrimSpace(expression) == "*" || strings.HasSuffix(expression, ".*") {
		field.FieldType = "star"
		if strings.Contains(expression, ".") {
			parts := strings.Split(expression, ".")
			field.SourceTable = strings.Trim(parts[0], "`")
		}
		return field
	}

	// 检查是否是窗口函数
	if strings.Contains(upperExpr, " OVER") || strings.Contains(upperExpr, " OVER(") {
		field.FieldType = "window"
		field.IsWindow = true
		// 提取函数名
		if parenIdx := strings.Index(expression, "("); parenIdx > 0 {
			field.FunctionName = strings.ToUpper(strings.TrimSpace(expression[:parenIdx]))
		}
		return field
	}

	// 检查是否是聚合函数
	aggregateFuncs := []string{"COUNT", "SUM", "AVG", "MAX", "MIN", "GROUP_CONCAT", "COUNT_DISTINCT"}
	for _, agg := range aggregateFuncs {
		if strings.HasPrefix(upperExpr, agg+"(") || strings.Contains(upperExpr, " "+agg+"(") {
			field.FieldType = "aggregate"
			field.IsAggregated = true
			field.FunctionName = agg
			return field
		}
	}

	// 检查是否是普通函数
	if strings.Contains(expression, "(") {
		field.FieldType = "function"
		// 提取函数名
		if parenIdx := strings.Index(expression, "("); parenIdx > 0 {
			funcName := strings.TrimSpace(expression[:parenIdx])
			// 移除可能的表前缀
			if dotIdx := strings.LastIndex(funcName, "."); dotIdx >= 0 {
				funcName = funcName[dotIdx+1:]
			}
			field.FunctionName = strings.ToUpper(funcName)
		}
		return field
	}

	// 检查是否是表达式（包含运算符）
	if strings.ContainsAny(expression, "+-*/%") ||
		strings.Contains(upperExpr, " CASE ") ||
		strings.Contains(upperExpr, "CASE WHEN") {
		field.FieldType = "expression"
		return field
	}

	// 普通字段
	field.FieldType = "column"
	cleanExpr := strings.Trim(expression, "`'\"")

	// 检查是否有表前缀 (table.column 或 schema.table.column)
	if strings.Contains(cleanExpr, ".") {
		parts := strings.Split(cleanExpr, ".")
		if len(parts) == 2 {
			field.SourceTable = parts[0]
			field.FieldName = parts[1]
		} else if len(parts) == 3 {
			field.SourceTable = parts[1] // schema.table.column -> table
			field.FieldName = parts[2]
		}
	} else {
		field.FieldName = cleanExpr
	}

	return field
}

// extractFromClause 提取FROM子句
func extractFromClause(sql string) *FromClause {
	fromIdx := findKeywordPosition(sql, "FROM", 0)
	if fromIdx == -1 {
		return nil
	}

	// 找到下一个关键字的位置
	endIdx := len(sql)
	for _, kw := range []string{"WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION"} {
		if idx := findKeywordPosition(sql, kw, fromIdx+4); idx > 0 && idx < endIdx {
			endIdx = idx
		}
	}

	fromPart := strings.TrimSpace(sql[fromIdx+4 : endIdx])
	clause := &FromClause{Raw: fromPart}

	// 解析JOIN
	joins := extractJoins(fromPart)
	if len(joins) > 0 {
		clause.Joins = joins
		// 主表是第一个JOIN之前的部分
		firstJoinIdx := strings.Index(strings.ToUpper(fromPart), " JOIN")
		if firstJoinIdx == -1 {
			firstJoinIdx = len(fromPart)
		}
		// 找LEFT/RIGHT/INNER等关键字
		for _, prefix := range []string{" LEFT", " RIGHT", " INNER", " OUTER", " CROSS", " FULL"} {
			if idx := strings.Index(strings.ToUpper(fromPart), prefix); idx > 0 && idx < firstJoinIdx {
				firstJoinIdx = idx
			}
		}
		mainTablePart := strings.TrimSpace(fromPart[:firstJoinIdx])
		clause.MainTable = parseTableInfo(mainTablePart)
	} else {
		clause.MainTable = parseTableInfo(fromPart)
	}

	return clause
}

// extractJoins 提取JOIN信息
func extractJoins(fromPart string) []JoinInfo {
	var joins []JoinInfo
	upperFrom := strings.ToUpper(fromPart)

	// 使用简单方法解析JOIN（Go正则不支持前瞻断言）
	if strings.Contains(upperFrom, " JOIN ") {
		// 按JOIN分割
		joinRe := regexp.MustCompile(`(?i)(LEFT\s+OUTER\s+|RIGHT\s+OUTER\s+|FULL\s+OUTER\s+|LEFT\s+|RIGHT\s+|INNER\s+|CROSS\s+|FULL\s+)?JOIN\s+`)
		parts := joinRe.Split(fromPart, -1)
		typeMatches := joinRe.FindAllStringSubmatch(fromPart, -1)

		for i := 1; i < len(parts); i++ {
			joinType := "INNER"
			if i-1 < len(typeMatches) && len(typeMatches[i-1]) > 1 {
				jt := strings.TrimSpace(strings.ToUpper(typeMatches[i-1][1]))
				if jt != "" {
					joinType = strings.TrimSuffix(jt, " ")
					joinType = strings.Replace(joinType, " OUTER", "", 1)
				}
			}

			join := JoinInfo{Type: joinType}
			part := parts[i]

			// 提取表名和别名
			onIdx := strings.Index(strings.ToUpper(part), " ON ")
			if onIdx > 0 {
				tablePart := strings.TrimSpace(part[:onIdx])
				// 移除反引号
				tablePart = strings.ReplaceAll(tablePart, "`", "")
				words := strings.Fields(tablePart)
				if len(words) > 0 {
					join.Table = words[0]
					if len(words) > 1 && strings.ToUpper(words[1]) != "AS" {
						join.Alias = words[1]
					} else if len(words) > 2 {
						join.Alias = words[2]
					}
				}

				// 提取ON条件（到下一个关键字为止）
				condPart := part[onIdx+4:]
				// 找到下一个关键字
				endIdx := len(condPart)
				for _, kw := range []string{" LEFT ", " RIGHT ", " INNER ", " CROSS ", " FULL ", " JOIN ", " WHERE ", " GROUP ", " ORDER ", " HAVING ", " LIMIT "} {
					if idx := strings.Index(strings.ToUpper(condPart), kw); idx > 0 && idx < endIdx {
						endIdx = idx
					}
				}
				join.Condition = strings.TrimSpace(condPart[:endIdx])
			} else {
				// 没有ON的情况（如CROSS JOIN）
				words := strings.Fields(strings.ReplaceAll(part, "`", ""))
				if len(words) > 0 {
					join.Table = words[0]
					if len(words) > 1 {
						join.Alias = words[1]
					}
				}
			}

			if join.Table != "" {
				joins = append(joins, join)
			}
		}
	}

	_ = upperFrom

	return joins
}

// parseTableInfo 解析表信息
func parseTableInfo(tablePart string) TableInfo {
	tablePart = strings.TrimSpace(tablePart)
	info := TableInfo{}

	// 移除反引号
	tablePart = strings.ReplaceAll(tablePart, "`", "")

	parts := strings.Fields(tablePart)
	if len(parts) >= 1 {
		info.Name = parts[0]
	}
	if len(parts) >= 2 {
		if strings.ToUpper(parts[1]) == "AS" && len(parts) >= 3 {
			info.Alias = parts[2]
		} else if strings.ToUpper(parts[1]) != "AS" {
			info.Alias = parts[1]
		}
	}

	return info
}

// extractWhereClause 提取WHERE子句
func extractWhereClause(sql string) *WhereClause {
	whereIdx := findKeywordPosition(sql, "WHERE", 0)
	if whereIdx == -1 {
		return nil
	}

	// 找到下一个关键字的位置
	endIdx := len(sql)
	for _, kw := range []string{"GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION"} {
		if idx := findKeywordPosition(sql, kw, whereIdx+5); idx > 0 && idx < endIdx {
			endIdx = idx
		}
	}

	wherePart := strings.TrimSpace(sql[whereIdx+5 : endIdx])
	clause := &WhereClause{Raw: wherePart}

	// 提取条件（按AND/OR分割）
	conditions := splitConditions(wherePart)
	clause.Conditions = conditions

	// 提取涉及的字段
	clause.Fields = extractFieldsFromConditions(wherePart)

	return clause
}

// extractGroupByClause 提取GROUP BY子句
func extractGroupByClause(sql string) *GroupByClause {
	groupIdx := findKeywordPosition(sql, "GROUP BY", 0)
	if groupIdx == -1 {
		return nil
	}

	endIdx := len(sql)
	for _, kw := range []string{"HAVING", "ORDER BY", "LIMIT", "UNION"} {
		if idx := findKeywordPosition(sql, kw, groupIdx+8); idx > 0 && idx < endIdx {
			endIdx = idx
		}
	}

	groupPart := strings.TrimSpace(sql[groupIdx+8 : endIdx])
	clause := &GroupByClause{Raw: groupPart}

	// 解析字段
	fields := strings.Split(groupPart, ",")
	for _, f := range fields {
		f = strings.TrimSpace(strings.Trim(f, "`"))
		if f != "" {
			clause.Fields = append(clause.Fields, f)
		}
	}

	return clause
}

// extractHavingClause 提取HAVING子句
func extractHavingClause(sql string) *HavingClause {
	havingIdx := findKeywordPosition(sql, "HAVING", 0)
	if havingIdx == -1 {
		return nil
	}

	endIdx := len(sql)
	for _, kw := range []string{"ORDER BY", "LIMIT", "UNION"} {
		if idx := findKeywordPosition(sql, kw, havingIdx+6); idx > 0 && idx < endIdx {
			endIdx = idx
		}
	}

	havingPart := strings.TrimSpace(sql[havingIdx+6 : endIdx])
	clause := &HavingClause{Raw: havingPart}
	clause.Conditions = splitConditions(havingPart)

	return clause
}

// extractOrderByClause 提取ORDER BY子句
func extractOrderByClause(sql string) *OrderByClause {
	orderIdx := findKeywordPosition(sql, "ORDER BY", 0)
	if orderIdx == -1 {
		return nil
	}

	endIdx := len(sql)
	for _, kw := range []string{"LIMIT", "UNION"} {
		if idx := findKeywordPosition(sql, kw, orderIdx+8); idx > 0 && idx < endIdx {
			endIdx = idx
		}
	}

	orderPart := strings.TrimSpace(sql[orderIdx+8 : endIdx])
	clause := &OrderByClause{Raw: orderPart}

	// 智能解析排序字段（考虑括号和 CASE）
	fields := splitOrderByFields(orderPart)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		order := OrderInfo{Direction: "ASC"}
		upperF := strings.ToUpper(f)
		if strings.HasSuffix(upperF, " DESC") {
			order.Direction = "DESC"
			order.Field = strings.TrimSpace(f[:len(f)-5])
		} else if strings.HasSuffix(upperF, " ASC") {
			order.Field = strings.TrimSpace(f[:len(f)-4])
		} else {
			order.Field = f
		}
		order.Field = strings.Trim(order.Field, "`")
		clause.Fields = append(clause.Fields, order)
	}

	return clause
}

// splitOrderByFields 智能拆分 ORDER BY 字段（考虑括号和 CASE）
func splitOrderByFields(orderPart string) []string {
	var fields []string
	var current strings.Builder
	depth := 0
	caseDepth := 0
	upperPart := strings.ToUpper(orderPart)

	for i := 0; i < len(orderPart); i++ {
		ch := orderPart[i]

		if ch == '(' {
			depth++
			current.WriteByte(ch)
		} else if ch == ')' {
			depth--
			current.WriteByte(ch)
		} else if depth == 0 {
			// 检查 CASE 关键字
			if i+4 <= len(upperPart) && upperPart[i:i+4] == "CASE" {
				// 确保是完整的关键字
				if (i == 0 || !isAlphaNum(orderPart[i-1])) &&
					(i+4 >= len(orderPart) || !isAlphaNum(orderPart[i+4])) {
					caseDepth++
					current.WriteString(orderPart[i : i+4])
					i += 3
					continue
				}
			}
			// 检查 END 关键字
			if caseDepth > 0 && i+3 <= len(upperPart) && upperPart[i:i+3] == "END" {
				// 确保是完整的关键字
				if (i == 0 || !isAlphaNum(orderPart[i-1])) &&
					(i+3 >= len(orderPart) || !isAlphaNum(orderPart[i+3])) {
					caseDepth--
					current.WriteString(orderPart[i : i+3])
					i += 2
					continue
				}
			}
			// 逗号分隔（只在不在 CASE 内部时）
			if ch == ',' && caseDepth == 0 {
				field := strings.TrimSpace(current.String())
				if field != "" {
					fields = append(fields, field)
				}
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		} else {
			current.WriteByte(ch)
		}
	}

	// 添加最后一个字段
	field := strings.TrimSpace(current.String())
	if field != "" {
		fields = append(fields, field)
	}

	return fields
}

// extractLimitClause 提取LIMIT子句
func extractLimitClause(sql string) *LimitClause {
	limitIdx := findKeywordPosition(sql, "LIMIT", 0)
	if limitIdx == -1 {
		return nil
	}

	limitPart := strings.TrimSpace(sql[limitIdx+5:])
	// 移除后面的分号
	limitPart = strings.TrimSuffix(limitPart, ";")
	limitPart = strings.TrimSpace(limitPart)

	clause := &LimitClause{Raw: limitPart}

	// 解析LIMIT值
	re := regexp.MustCompile(`(?i)(\d+)(?:\s*,\s*(\d+)|\s+OFFSET\s+(\d+))?`)
	if matches := re.FindStringSubmatch(limitPart); len(matches) > 1 {
		if matches[2] != "" {
			// LIMIT offset, count 格式
			clause.Offset = parseInt(matches[1])
			clause.Limit = parseInt(matches[2])
		} else if matches[3] != "" {
			// LIMIT count OFFSET offset 格式
			clause.Limit = parseInt(matches[1])
			clause.Offset = parseInt(matches[3])
		} else {
			clause.Limit = parseInt(matches[1])
		}
	}

	return clause
}

// extractSubqueries 提取子查询
func extractSubqueries(sql string) []SubqueryInfo {
	var subqueries []SubqueryInfo
	upperSQL := strings.ToUpper(sql)

	// 查找括号内的SELECT
	depth := 0
	start := -1
	for i := 0; i < len(sql); i++ {
		if sql[i] == '(' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if sql[i] == ')' {
			depth--
			if depth == 0 && start >= 0 {
				inner := strings.TrimSpace(sql[start+1 : i])
				if strings.HasPrefix(strings.ToUpper(inner), "SELECT") {
					// 确定子查询位置
					location := "未知"
					beforePart := strings.ToUpper(sql[:start])
					if strings.Contains(beforePart, " IN ") || strings.Contains(beforePart, " IN(") {
						location = "IN子查询"
					} else if strings.Contains(beforePart, " EXISTS ") || strings.Contains(beforePart, " EXISTS(") {
						location = "EXISTS子查询"
					} else if strings.Contains(beforePart, " FROM ") {
						location = "FROM子查询(派生表)"
					} else if strings.Contains(beforePart, "SELECT") {
						location = "SELECT子查询(标量)"
					}

					// 简化显示
					displaySQL := inner
					if len(displaySQL) > 100 {
						displaySQL = displaySQL[:100] + "..."
					}

					subqueries = append(subqueries, SubqueryInfo{
						Location: location,
						Raw:      displaySQL,
						Type:     "SELECT",
					})
				}
				start = -1
			}
		}
	}

	_ = upperSQL
	return subqueries
}

// extractWindowFunctions 提取窗口函数
func extractWindowFunctions(sql string) []WindowFunction {
	var windows []WindowFunction

	// 匹配窗口函数: func() OVER (...)
	re := regexp.MustCompile(`(?i)(\w+)\s*\([^)]*\)\s+OVER\s*\(([^)]*)\)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	for _, m := range matches {
		if len(m) >= 3 {
			wf := WindowFunction{
				Function: strings.ToUpper(m[1]),
				Raw:      m[0],
			}

			overPart := m[2]
			upperOver := strings.ToUpper(overPart)

			// 提取PARTITION BY
			if pbIdx := strings.Index(upperOver, "PARTITION BY"); pbIdx >= 0 {
				rest := overPart[pbIdx+12:]
				if obIdx := strings.Index(strings.ToUpper(rest), "ORDER BY"); obIdx > 0 {
					wf.PartitionBy = strings.TrimSpace(rest[:obIdx])
				} else {
					wf.PartitionBy = strings.TrimSpace(rest)
				}
			}

			// 提取ORDER BY
			if obIdx := strings.Index(upperOver, "ORDER BY"); obIdx >= 0 {
				wf.OrderBy = strings.TrimSpace(overPart[obIdx+8:])
			}

			windows = append(windows, wf)
		}
	}

	return windows
}

// extractCTEs 提取CTE
func extractCTEs(sql string) []CTEInfo {
	var ctes []CTEInfo

	// 匹配 WITH name AS 和 ,name AS 格式
	re := regexp.MustCompile(`(?i)(?:WITH\s+|,\s*)(\w+)\s+AS\s*\(`)
	matches := re.FindAllStringSubmatch(sql, -1)

	for _, m := range matches {
		if len(m) >= 2 {
			ctes = append(ctes, CTEInfo{
				Name:  m[1],
				Query: "(查询内容)",
			})
		}
	}

	return ctes
}

// 辅助函数

// findKeywordPosition 查找关键字位置（不在括号内）
func findKeywordPosition(sql string, keyword string, startPos int) int {
	upperSQL := strings.ToUpper(sql)
	upperKeyword := strings.ToUpper(keyword)
	depth := 0

	for i := startPos; i < len(sql)-len(keyword); i++ {
		if sql[i] == '(' {
			depth++
		} else if sql[i] == ')' {
			depth--
		} else if depth == 0 {
			if upperSQL[i:i+len(keyword)] == upperKeyword {
				// 确保是完整的关键字
				if i > 0 && isAlphaNum(sql[i-1]) {
					continue
				}
				if i+len(keyword) < len(sql) && isAlphaNum(sql[i+len(keyword)]) {
					continue
				}
				return i
			}
		}
	}
	return -1
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// splitSelectFields 分割SELECT字段
func splitSelectFields(selectPart string) []string {
	var fields []string
	var current strings.Builder
	depth := 0

	for _, ch := range selectPart {
		if ch == '(' {
			depth++
			current.WriteRune(ch)
		} else if ch == ')' {
			depth--
			current.WriteRune(ch)
		} else if ch == ',' && depth == 0 {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// splitConditions 分割条件（智能处理括号和 BETWEEN）
func splitConditions(condPart string) []string {
	var result []string

	// 第一步：按顶层 AND/OR 分割
	parts := splitByTopLevelOperator(condPart)

	// 第二步：递归处理每个部分，移除外层括号
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 移除外层括号并递归处理
		if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
			// 检查是否是完整的外层括号
			depth := 0
			isOuterParen := true
			for i, ch := range part {
				if ch == '(' {
					depth++
				} else if ch == ')' {
					depth--
					if depth == 0 && i < len(part)-1 {
						isOuterParen = false
						break
					}
				}
			}
			if isOuterParen {
				// 移除外层括号，递归处理内部
				inner := strings.TrimSpace(part[1 : len(part)-1])
				innerParts := splitConditions(inner)
				result = append(result, innerParts...)
				continue
			}
		}

		result = append(result, part)
	}

	return result
}

// splitByTopLevelOperator 按顶层 AND/OR 分割（考虑括号深度和 BETWEEN）
func splitByTopLevelOperator(condPart string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	inBetween := false
	upperPart := strings.ToUpper(condPart)

	for i := 0; i < len(condPart); i++ {
		ch := condPart[i]

		if ch == '(' {
			depth++
			current.WriteByte(ch)
		} else if ch == ')' {
			depth--
			current.WriteByte(ch)
		} else if depth == 0 {
			// 检查是否是 BETWEEN ... AND
			if i+7 <= len(upperPart) && upperPart[i:i+7] == "BETWEEN" {
				inBetween = true
				current.WriteString(condPart[i : i+7])
				i += 6
			} else if inBetween && i+4 <= len(upperPart) && upperPart[i:i+4] == " AND" {
				// BETWEEN 中的 AND，不分割
				inBetween = false
				current.WriteString(condPart[i : i+4])
				i += 3
			} else if !inBetween && i+4 <= len(upperPart) && upperPart[i:i+4] == " AND" {
				// 普通 AND，分割
				part := strings.TrimSpace(current.String())
				if part != "" {
					parts = append(parts, part)
				}
				current.Reset()
				i += 3 // 跳过 " AND"
			} else if i+3 <= len(upperPart) && upperPart[i:i+3] == " OR" {
				// OR，分割
				part := strings.TrimSpace(current.String())
				if part != "" {
					parts = append(parts, part)
				}
				current.Reset()
				i += 2 // 跳过 " OR"
			} else {
				current.WriteByte(ch)
			}
		} else {
			current.WriteByte(ch)
		}
	}

	// 添加最后一个部分
	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}

	return parts
}

// extractFieldsFromConditions 从条件中提取字段
func extractFieldsFromConditions(condPart string) []string {
	var fields []string
	seen := make(map[string]bool)

	// 匹配字段名（简单匹配）
	re := regexp.MustCompile(`(?:^|[^a-zA-Z_])([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)(?:\s*[=<>!]|\s+(?:IN|LIKE|BETWEEN|IS))`)
	matches := re.FindAllStringSubmatch(condPart, -1)

	for _, m := range matches {
		if len(m) >= 2 {
			field := m[1]
			upperField := strings.ToUpper(field)
			// 排除关键字
			if upperField != "AND" && upperField != "OR" && upperField != "NOT" && upperField != "NULL" {
				if !seen[field] {
					seen[field] = true
					fields = append(fields, field)
				}
			}
		}
	}

	return fields
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
