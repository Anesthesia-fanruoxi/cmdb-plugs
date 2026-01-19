package common

import (
	"strings"
)

// SplitByComma 按逗号分割，但忽略括号内和字符串内的逗号
func SplitByComma(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0
	inString := false
	stringChar := rune(0)

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// 处理字符串
		if ch == '\'' || ch == '"' || ch == '`' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				// 检查是否是转义
				if i > 0 && runes[i-1] != '\\' {
					inString = false
				}
			}
			current.WriteRune(ch)
		} else if !inString {
			// 不在字符串内，处理括号和逗号
			if ch == '(' {
				depth++
				current.WriteRune(ch)
			} else if ch == ')' {
				depth--
				current.WriteRune(ch)
			} else if ch == ',' && depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		} else {
			// 在字符串内，直接添加
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// FindLastMainSelect 查找最后一个主 SELECT（不在括号内的）
func FindLastMainSelect(sql string) int {
	upperSQL := strings.ToUpper(sql)
	depth := 0
	lastIdx := -1

	for i := 0; i < len(sql)-6; i++ {
		if sql[i] == '(' {
			depth++
		} else if sql[i] == ')' {
			depth--
		} else if depth == 0 && upperSQL[i:i+6] == "SELECT" {
			lastIdx = i
		}
	}

	return lastIdx
}

// IsValidAlias 检查是否是有效的别名（非SQL关键字）
func IsValidAlias(s string) bool {
	if s == "" {
		return false
	}
	upperS := strings.ToUpper(s)
	keywords := []string{
		"FROM", "WHERE", "JOIN", "ON", "AND", "OR", "AS", "LEFT", "RIGHT",
		"INNER", "OUTER", "GROUP", "ORDER", "BY", "HAVING", "LIMIT", "OFFSET",
		"UNION", "SELECT", "INTO", "VALUES", "SET", "NULL", "NOT", "IN", "LIKE",
		"BETWEEN", "EXISTS", "CASE", "WHEN", "THEN", "ELSE", "END", "IS",
	}
	for _, kw := range keywords {
		if upperS == kw {
			return false
		}
	}
	return true
}

// IsValidTableName 检查是否是有效的表名（过滤关键字和CTE名称）
func IsValidTableName(name string) bool {
	upperName := strings.ToUpper(name)
	keywords := []string{
		"SELECT", "WHERE", "ON", "AND", "OR", "AS", "FROM", "JOIN",
		"LEFT", "RIGHT", "INNER", "OUTER", "FULL", "CROSS",
	}

	for _, keyword := range keywords {
		if upperName == keyword {
			return false
		}
	}

	return len(name) > 0
}
