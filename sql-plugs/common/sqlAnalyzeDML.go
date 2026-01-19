package common

import (
	"fmt"
	"regexp"
	"strings"
)

// DMLAnalysis DML语句分析结果
type DMLAnalysis struct {
	TargetTable  string     `json:"target_table"`
	AffectedCols []string   `json:"affected_cols"`
	DataSource   string     `json:"data_source"`
	HasWhere     bool       `json:"has_where"`
	WherePreview string     `json:"where_preview"`
	EstimateRows string     `json:"estimate_rows"`
	RiskLevel    string     `json:"risk_level"`
	RiskReason   string     `json:"risk_reason"`
	InsertValues [][]string `json:"insert_values,omitempty"` // INSERT 的值（每行是一个数组）
}

// AnalyzeDML 分析DML语句
func AnalyzeDML(sql string, sqlType string) *DMLAnalysis {
	result := &DMLAnalysis{}
	cleanSQL := RemoveSQLComments(sql)
	upperSQL := strings.ToUpper(cleanSQL)

	switch sqlType {
	case "INSERT":
		analyzeInsert(cleanSQL, upperSQL, result)
	case "UPDATE":
		analyzeUpdate(cleanSQL, upperSQL, result)
	case "DELETE":
		analyzeDelete(cleanSQL, upperSQL, result)
	}

	return result
}

// analyzeInsert 分析INSERT语句
func analyzeInsert(sql, upperSQL string, result *DMLAnalysis) {
	// 提取目标表
	// INSERT INTO table_name 或 INSERT INTO db.table_name
	re := regexp.MustCompile(`(?i)INSERT\s+(?:INTO\s+)?(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.TargetTable = matches[2]
	}

	// 提取插入字段
	// INSERT INTO table (col1, col2) VALUES ...
	colRe := regexp.MustCompile(`(?i)INSERT\s+(?:INTO\s+)?[^\(]+\(([^\)]+)\)`)
	if matches := colRe.FindStringSubmatch(sql); len(matches) > 1 {
		cols := strings.Split(matches[1], ",")
		for _, col := range cols {
			col = strings.Trim(col, " `\t\n\r")
			if col != "" {
				result.AffectedCols = append(result.AffectedCols, col)
			}
		}
	}

	// 判断数据来源
	if strings.Contains(upperSQL, " VALUES") || strings.Contains(upperSQL, " VALUE") {
		result.DataSource = "VALUES"
		// 统计VALUES数量（统计 ), ( 的数量 + 1，注意有空格）
		valuesCount := strings.Count(sql, "), (") + strings.Count(sql, "),(") + 1
		result.EstimateRows = formatRowCount(valuesCount)

		// 解析 VALUES 中的数据
		result.InsertValues = parseInsertValues(sql)
	} else if strings.Contains(upperSQL, " SELECT") {
		result.DataSource = "SELECT"
		result.EstimateRows = "取决于SELECT结果"
	} else {
		result.DataSource = "未知"
	}

	// INSERT风险评估
	result.RiskLevel = "low"
	result.RiskReason = "INSERT操作，新增数据"
	result.HasWhere = false
}

// analyzeUpdate 分析UPDATE语句
func analyzeUpdate(sql, upperSQL string, result *DMLAnalysis) {
	// 提取目标表
	re := regexp.MustCompile(`(?i)UPDATE\s+(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.TargetTable = matches[2]
	}

	// 提取SET字段
	setRe := regexp.MustCompile(`(?i)SET\s+(.+?)(?:\s+WHERE|$)`)
	if matches := setRe.FindStringSubmatch(sql); len(matches) > 1 {
		setPart := matches[1]
		// 解析 col1=val1, col2=val2
		assignments := SplitByComma(setPart)
		for _, assign := range assignments {
			if idx := strings.Index(assign, "="); idx > 0 {
				col := strings.Trim(assign[:idx], " `\t\n\r")
				// 移除表前缀
				if dotIdx := strings.LastIndex(col, "."); dotIdx > 0 {
					col = col[dotIdx+1:]
				}
				result.AffectedCols = append(result.AffectedCols, col)
			}
		}
	}

	result.DataSource = "SET"

	// 检查WHERE条件
	result.HasWhere = strings.Contains(upperSQL, " WHERE ")
	if result.HasWhere {
		whereRe := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+ORDER|\s+LIMIT|$)`)
		if matches := whereRe.FindStringSubmatch(sql); len(matches) > 1 {
			preview := matches[1]
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			result.WherePreview = preview
		}
		result.RiskLevel = "medium"
		result.RiskReason = "UPDATE有WHERE条件，请确认条件正确"
		result.EstimateRows = "取决于WHERE条件"
	} else {
		result.RiskLevel = "high"
		result.RiskReason = "⚠️ UPDATE无WHERE条件，将更新全表数据！"
		result.EstimateRows = "全表所有行"
	}
}

// analyzeDelete 分析DELETE语句
func analyzeDelete(sql, upperSQL string, result *DMLAnalysis) {
	// 提取目标表
	re := regexp.MustCompile(`(?i)DELETE\s+FROM\s+(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.TargetTable = matches[2]
	}

	result.DataSource = "DELETE"
	result.AffectedCols = []string{"*（整行删除）"}

	// 检查WHERE条件
	result.HasWhere = strings.Contains(upperSQL, " WHERE ")
	if result.HasWhere {
		whereRe := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+ORDER|\s+LIMIT|$)`)
		if matches := whereRe.FindStringSubmatch(sql); len(matches) > 1 {
			preview := matches[1]
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			result.WherePreview = preview
		}
		result.RiskLevel = "medium"
		result.RiskReason = "DELETE有WHERE条件，请确认条件正确"
		result.EstimateRows = "取决于WHERE条件"
	} else {
		result.RiskLevel = "high"
		result.RiskReason = "⚠️ DELETE无WHERE条件，将删除全表数据！"
		result.EstimateRows = "全表所有行"
	}

	// 检查LIMIT
	if strings.Contains(upperSQL, " LIMIT ") {
		result.RiskLevel = "medium"
		result.RiskReason = "DELETE有LIMIT限制"
	}
}

func formatRowCount(count int) string {
	if count == 1 {
		return "1行"
	}
	return fmt.Sprintf("%d行", count)
}

// parseInsertValues 解析 INSERT VALUES 中的数据
func parseInsertValues(sql string) [][]string {
	var result [][]string

	// 找到 VALUES 关键字位置
	upperSQL := strings.ToUpper(sql)
	valuesIdx := strings.Index(upperSQL, " VALUES")
	if valuesIdx == -1 {
		valuesIdx = strings.Index(upperSQL, " VALUE")
	}
	if valuesIdx == -1 {
		return result
	}

	// 提取 VALUES 后面的内容
	valuesPart := sql[valuesIdx+7:] // 跳过 " VALUES"
	valuesPart = strings.TrimSpace(valuesPart)

	// 解析每组值：(val1, val2, ...), (val1, val2, ...)
	depth := 0
	inString := false
	stringChar := rune(0)
	var currentRow strings.Builder

	runes := []rune(valuesPart)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// 处理字符串
		if ch == '\'' || ch == '"' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				if i > 0 && runes[i-1] != '\\' {
					inString = false
				}
			}
			currentRow.WriteRune(ch)
			continue
		}

		if inString {
			currentRow.WriteRune(ch)
			continue
		}

		// 处理括号
		if ch == '(' {
			depth++
			if depth == 1 {
				currentRow.Reset() // 开始新的一行
			} else {
				currentRow.WriteRune(ch)
			}
		} else if ch == ')' {
			depth--
			if depth == 0 {
				// 一行结束，解析这一行的值
				rowStr := currentRow.String()
				values := parseRowValues(rowStr)
				if len(values) > 0 {
					result = append(result, values)
				}
				currentRow.Reset()
			} else {
				currentRow.WriteRune(ch)
			}
		} else if depth > 0 {
			currentRow.WriteRune(ch)
		}
	}

	return result
}

// parseRowValues 解析一行的值
func parseRowValues(rowStr string) []string {
	var values []string
	var current strings.Builder
	inString := false
	stringChar := rune(0)
	depth := 0

	runes := []rune(rowStr)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// 处理字符串
		if ch == '\'' || ch == '"' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				if i > 0 && runes[i-1] != '\\' {
					inString = false
				}
			}
			current.WriteRune(ch)
			continue
		}

		if inString {
			current.WriteRune(ch)
			continue
		}

		// 处理括号（函数调用等）
		if ch == '(' {
			depth++
			current.WriteRune(ch)
		} else if ch == ')' {
			depth--
			current.WriteRune(ch)
		} else if ch == ',' && depth == 0 {
			// 遇到逗号，保存当前值
			val := strings.TrimSpace(current.String())
			values = append(values, val)
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	// 保存最后一个值
	if current.Len() > 0 {
		val := strings.TrimSpace(current.String())
		values = append(values, val)
	}

	return values
}
