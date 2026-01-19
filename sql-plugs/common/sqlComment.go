package common

import (
	"strings"
)

// RemoveSQLComments 移除SQL注释，但保留字符串内的注释符号
func RemoveSQLComments(sql string) string {
	// 先移除多行注释 /* */
	sql = RemoveMultiLineComments(sql)

	// 再移除单行注释 -- 和 #
	lines := strings.Split(sql, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = RemoveLineComment(line)
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, " ")
}

// RemoveMultiLineComments 移除多行注释 /* */，但跳过字符串内的
func RemoveMultiLineComments(query string) string {
	var result strings.Builder
	inComment := false
	inString := false
	stringChar := rune(0)
	runes := []rune(query)

	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if !inComment {
			// 处理字符串边界
			if !inString && (ch == '\'' || ch == '"') {
				inString = true
				stringChar = ch
				result.WriteRune(ch)
				continue
			}
			if inString && ch == stringChar {
				// 检查是否是转义的引号 '' 或 ""
				if i+1 < len(runes) && runes[i+1] == stringChar {
					result.WriteRune(ch)
					i++
					result.WriteRune(runes[i])
					continue
				}
				inString = false
				result.WriteRune(ch)
				continue
			}

			// 不在字符串内，检测多行注释开始
			if !inString && i+1 < len(runes) && ch == '/' && runes[i+1] == '*' {
				inComment = true
				i++
				result.WriteRune(' ') // 用空格替代注释
				continue
			}
			result.WriteRune(ch)
		} else {
			// 在注释内，检测多行注释结束
			if i+1 < len(runes) && ch == '*' && runes[i+1] == '/' {
				inComment = false
				i++
				continue
			}
		}
	}

	return result.String()
}

// RemoveLineComment 移除单行注释 -- 和 #，但跳过字符串内的
func RemoveLineComment(line string) string {
	inString := false
	stringChar := rune(0)
	runes := []rune(line)

	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// 处理字符串边界
		if !inString && (ch == '\'' || ch == '"') {
			inString = true
			stringChar = ch
			continue
		}
		if inString && ch == stringChar {
			// 检查是否是转义的引号
			if i+1 < len(runes) && runes[i+1] == stringChar {
				i++
				continue
			}
			inString = false
			continue
		}

		// 不在字符串内，检查注释
		if !inString {
			// 检查 --
			if ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
				return string(runes[:i])
			}
			// 检查 #
			if ch == '#' {
				return string(runes[:i])
			}
		}
	}

	return line
}

// SplitSQLStatements 按分号分割SQL语句，并过滤注释
// 支持：
// - 多个SQL用分号分隔
// - 单行注释：-- 和 # (MySQL特有)
// - 多行注释：/* */
// - 字符串内的分号和注释符号不会被误处理
func SplitSQLStatements(query string) []string {
	// 先移除多行注释
	query = RemoveMultiLineComments(query)
	parts := strings.Split(query, ";")

	var result []string
	for _, part := range parts {
		lines := strings.Split(part, "\n")
		var cleanLines []string
		for _, line := range lines {
			line = RemoveLineComment(line)
			line = strings.TrimSpace(line)
			if line != "" {
				cleanLines = append(cleanLines, line)
			}
		}

		cleanSQL := strings.Join(cleanLines, " ")
		cleanSQL = strings.TrimSpace(cleanSQL)
		if cleanSQL != "" {
			result = append(result, cleanSQL)
		}
	}

	return result
}

// AssessQueryRisk 评估查询风险等级
// 返回: (风险等级, 原因)
// 风险等级: "low"(低风险), "medium"(中风险), "high"(高风险)
func AssessQueryRisk(sql string, features *SQLFeatures) (string, string) {
	// 1. 用户指定了LIMIT - 低风险
	if GetUserOriginalLimit(sql) > 0 {
		return "low", "用户指定了LIMIT"
	}

	// 2. 有WHERE条件 - 低风险
	if features.HasWhere {
		return "low", "有WHERE过滤条件"
	}

	// 3. 聚合函数（无GROUP BY）- 低风险（如 SELECT COUNT(*) FROM table）
	if features.HasAggregate && !features.HasGroupBy {
		return "low", "聚合函数查询，结果只有1行"
	}

	// 4. 有HAVING - 低风险
	if features.HasHaving {
		return "low", "有HAVING过滤条件"
	}

	// 5. 有JOIN - 中风险
	if features.HasJoin {
		return "medium", "有JOIN关联，结果可能较大"
	}

	// 6. 有GROUP BY - 中风险
	if features.HasGroupBy {
		return "medium", "有GROUP BY聚合"
	}

	// 7. DISTINCT - 中风险
	if features.HasDistinct {
		return "medium", "有DISTINCT去重"
	}

	// 8. 无任何过滤条件（SELECT * FROM table）- 高风险
	return "high", "无过滤条件，全表查询"
}
