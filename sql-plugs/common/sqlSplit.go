package common

import (
	"strings"
)

// SplitMultipleSQL 拆分多个 SQL 语句
// 按分号分隔,但需要考虑:
// 1. 字符串中的分号不算分隔符
// 2. 注释中的分号不算分隔符
// 3. 函数/存储过程中的分号不算分隔符
func SplitMultipleSQL(sql string) []string {
	// 先去除所有注释，避免注释中的分号干扰拆分
	sql = RemoveSQLComments(sql)

	var sqls []string
	var current strings.Builder
	inString := false
	stringChar := rune(0)

	runes := []rune(sql)
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
		}

		// 处理分号
		if ch == ';' && !inString {
			// 遇到分号,保存当前 SQL
			sqlText := strings.TrimSpace(current.String())
			if sqlText != "" && sqlText != ";" {
				sqls = append(sqls, sqlText)
			}
			current.Reset()
			continue
		}

		current.WriteRune(ch)
	}

	// 保存最后一个 SQL
	sqlText := strings.TrimSpace(current.String())
	if sqlText != "" && sqlText != ";" {
		sqls = append(sqls, sqlText)
	}

	return sqls
}

// ValidateSQLMix 验证 SQL 混合是否合法
// 规则:DQL 不能与 DDL/DML 混合
func ValidateSQLMix(sqls []string) (bool, string) {
	if len(sqls) <= 1 {
		return true, ""
	}

	hasDQL := false
	hasDDLOrDML := false

	for _, sql := range sqls {
		sqlType := GetSQLType(sql)
		category := GetSQLCategory(sqlType)

		if category == "DQL" {
			hasDQL = true
		} else if category == "DDL" || category == "DML" {
			hasDDLOrDML = true
		}
	}

	if hasDQL && hasDDLOrDML {
		return false, "DQL 查询不能与 DDL/DML 语句混合执行"
	}

	return true, ""
}
