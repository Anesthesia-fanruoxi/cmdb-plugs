package common

import (
	"fmt"
	"regexp"
	"strings"
)

// DDLAnalysis DDL语句分析结果
type DDLAnalysis struct {
	Operation    string      `json:"operation"`
	ObjectType   string      `json:"object_type"`
	ObjectName   string      `json:"object_name"`
	ColumnsDef   []string    `json:"columns_def"`
	AlterActions []string    `json:"alter_actions"`
	RiskLevel    string      `json:"risk_level"`
	RiskReason   string      `json:"risk_reason"`
	Details      *DDLDetails `json:"details,omitempty"`
}

// DDLDetails DDL详细信息
type DDLDetails struct {
	// CREATE TABLE 详情
	ColumnCount  int            `json:"column_count,omitempty"`
	Columns      []ColumnDetail `json:"columns,omitempty"`
	TableComment string         `json:"table_comment,omitempty"`
	Engine       string         `json:"engine,omitempty"`
	Charset      string         `json:"charset,omitempty"`
	Collation    string         `json:"collation,omitempty"`
	PrimaryKey   string         `json:"primary_key,omitempty"`
	HasIndex     bool           `json:"has_index"`
	IndexCount   int            `json:"index_count,omitempty"`
	Indexes      []IndexDetail  `json:"indexes,omitempty"`
	ForeignKeys  []string       `json:"foreign_keys,omitempty"`
	// ALTER TABLE 详情
	AddColumns    []ColumnDetail `json:"add_columns,omitempty"`
	ModifyColumns []ColumnDetail `json:"modify_columns,omitempty"`
	DropColumns   []string       `json:"drop_columns,omitempty"`
	AddIndexes    []IndexDetail  `json:"add_indexes,omitempty"`
	DropIndexes   []string       `json:"drop_indexes,omitempty"`
	RenameInfo    string         `json:"rename_info,omitempty"`
	ChangeComment string         `json:"change_comment,omitempty"`
}

// IndexDetail 索引详情
type IndexDetail struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"` // PRIMARY, UNIQUE, INDEX, FULLTEXT, SPATIAL
	Columns   []string `json:"columns"`
	ColumnStr string   `json:"column_str"` // 列名字符串
}

// ColumnDetail 字段详情
type ColumnDetail struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
	Default  string `json:"default"`
	Comment  string `json:"comment"`
	Extra    string `json:"extra"`
}

// AnalyzeDDL 分析DDL语句
func AnalyzeDDL(sql string, sqlType string) *DDLAnalysis {
	result := &DDLAnalysis{
		Operation: sqlType,
	}
	cleanSQL := RemoveSQLComments(sql)
	upperSQL := strings.ToUpper(cleanSQL)

	switch sqlType {
	case "CREATE":
		analyzeCreate(cleanSQL, upperSQL, result)
	case "ALTER":
		analyzeAlter(cleanSQL, upperSQL, result)
	case "DROP":
		analyzeDrop(cleanSQL, upperSQL, result)
	case "TRUNCATE":
		analyzeTruncate(cleanSQL, upperSQL, result)
	case "RENAME":
		analyzeRename(cleanSQL, upperSQL, result)
	}

	return result
}

// analyzeCreate 分析CREATE语句
func analyzeCreate(sql, upperSQL string, result *DDLAnalysis) {
	// 判断对象类型
	if strings.Contains(upperSQL, "CREATE TABLE") || strings.Contains(upperSQL, "CREATE TEMPORARY TABLE") {
		result.ObjectType = "TABLE"
		analyzeCreateTable(sql, upperSQL, result)
	} else if strings.Contains(upperSQL, "CREATE INDEX") || strings.Contains(upperSQL, "CREATE UNIQUE INDEX") {
		result.ObjectType = "INDEX"
		analyzeCreateIndex(sql, upperSQL, result)
	} else if strings.Contains(upperSQL, "CREATE VIEW") {
		result.ObjectType = "VIEW"
		analyzeCreateView(sql, upperSQL, result)
	} else if strings.Contains(upperSQL, "CREATE DATABASE") || strings.Contains(upperSQL, "CREATE SCHEMA") {
		result.ObjectType = "DATABASE"
		analyzeCreateDatabase(sql, upperSQL, result)
	} else {
		result.ObjectType = "OTHER"
	}

	result.RiskLevel = "low"
	result.RiskReason = "CREATE操作，创建新对象"
}

// analyzeCreateTable 分析CREATE TABLE
func analyzeCreateTable(sql, upperSQL string, result *DDLAnalysis) {
	// 提取表名
	re := regexp.MustCompile(`(?i)CREATE\s+(?:TEMPORARY\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.ObjectName = matches[2]
	}

	result.Details = &DDLDetails{}

	// 提取表选项（在最后一个括号之后）
	lastParen := strings.LastIndex(sql, ")")
	if lastParen > 0 && lastParen < len(sql)-1 {
		tableOptions := sql[lastParen+1:]
		parseTableOptions(tableOptions, result.Details)
	}

	// 提取字段定义
	start := strings.Index(sql, "(")
	end := strings.LastIndex(sql, ")")
	if start > 0 && end > start {
		colDefs := sql[start+1 : end]
		cols := SplitByComma(colDefs)

		for _, col := range cols {
			col = strings.TrimSpace(col)
			if col == "" {
				continue
			}

			upperCol := strings.ToUpper(col)

			// 检查是否是约束定义
			if strings.HasPrefix(upperCol, "PRIMARY KEY") {
				result.Details.PrimaryKey = extractKeyColumns(col)
				// 添加主键索引
				idx := parseIndexDefinition(col, "PRIMARY")
				result.Details.Indexes = append(result.Details.Indexes, idx)
				result.ColumnsDef = append(result.ColumnsDef, "🔑 "+col)
				continue
			}
			if strings.HasPrefix(upperCol, "UNIQUE KEY") || strings.HasPrefix(upperCol, "UNIQUE INDEX") || strings.HasPrefix(upperCol, "UNIQUE ") {
				idx := parseIndexDefinition(col, "UNIQUE")
				result.Details.Indexes = append(result.Details.Indexes, idx)
				result.ColumnsDef = append(result.ColumnsDef, "📇 "+col)
				continue
			}
			if strings.HasPrefix(upperCol, "FULLTEXT") {
				idx := parseIndexDefinition(col, "FULLTEXT")
				result.Details.Indexes = append(result.Details.Indexes, idx)
				result.ColumnsDef = append(result.ColumnsDef, "📇 "+col)
				continue
			}
			if strings.HasPrefix(upperCol, "SPATIAL") {
				idx := parseIndexDefinition(col, "SPATIAL")
				result.Details.Indexes = append(result.Details.Indexes, idx)
				result.ColumnsDef = append(result.ColumnsDef, "📇 "+col)
				continue
			}
			if strings.HasPrefix(upperCol, "INDEX") || strings.HasPrefix(upperCol, "KEY") {
				idx := parseIndexDefinition(col, "INDEX")
				result.Details.Indexes = append(result.Details.Indexes, idx)
				result.ColumnsDef = append(result.ColumnsDef, "📇 "+col)
				continue
			}
			if strings.HasPrefix(upperCol, "FOREIGN KEY") || strings.HasPrefix(upperCol, "CONSTRAINT") {
				result.Details.ForeignKeys = append(result.Details.ForeignKeys, col)
				result.ColumnsDef = append(result.ColumnsDef, "🔗 "+col)
				continue
			}

			// 解析字段详情
			colDetail := parseColumnDefinition(col)
			result.Details.Columns = append(result.Details.Columns, colDetail)

			// 检查字段级别的PRIMARY KEY
			if strings.Contains(upperCol, "PRIMARY KEY") && result.Details.PrimaryKey == "" {
				result.Details.PrimaryKey = colDetail.Name
				// 添加主键索引
				idx := IndexDetail{
					Name:      "PRIMARY",
					Type:      "PRIMARY",
					Columns:   []string{colDetail.Name},
					ColumnStr: colDetail.Name,
				}
				result.Details.Indexes = append(result.Details.Indexes, idx)
			}

			// 检查字段级别的UNIQUE
			if strings.Contains(upperCol, " UNIQUE") && !strings.Contains(upperCol, "PRIMARY") {
				idx := IndexDetail{
					Name:      colDetail.Name,
					Type:      "UNIQUE",
					Columns:   []string{colDetail.Name},
					ColumnStr: colDetail.Name,
				}
				result.Details.Indexes = append(result.Details.Indexes, idx)
			}

			// 简化显示
			displayCol := col
			if len(displayCol) > 80 {
				displayCol = displayCol[:80] + "..."
			}
			result.ColumnsDef = append(result.ColumnsDef, displayCol)
		}

		// 设置索引统计
		result.Details.IndexCount = len(result.Details.Indexes)
		result.Details.HasIndex = result.Details.IndexCount > 0

		result.Details.ColumnCount = len(result.Details.Columns)
	}
}

// parseTableOptions 解析表选项
func parseTableOptions(options string, details *DDLDetails) {
	upperOptions := strings.ToUpper(options)

	// ENGINE
	if re := regexp.MustCompile(`(?i)ENGINE\s*=\s*(\w+)`); true {
		if matches := re.FindStringSubmatch(options); len(matches) > 1 {
			details.Engine = matches[1]
		}
	}

	// CHARSET
	if re := regexp.MustCompile(`(?i)(?:DEFAULT\s+)?(?:CHARACTER\s+SET|CHARSET)\s*=?\s*(\w+)`); true {
		if matches := re.FindStringSubmatch(options); len(matches) > 1 {
			details.Charset = matches[1]
		}
	}

	// COLLATE
	if re := regexp.MustCompile(`(?i)COLLATE\s*=?\s*(\w+)`); true {
		if matches := re.FindStringSubmatch(options); len(matches) > 1 {
			details.Collation = matches[1]
		}
	}

	// COMMENT
	if re := regexp.MustCompile(`(?i)COMMENT\s*=?\s*'([^']*)'`); true {
		if matches := re.FindStringSubmatch(options); len(matches) > 1 {
			details.TableComment = matches[1]
		}
	}

	_ = upperOptions // 避免未使用警告
}

// parseColumnDefinition 解析字段定义
func parseColumnDefinition(colDef string) ColumnDetail {
	detail := ColumnDetail{}

	// 移除多余空格
	colDef = regexp.MustCompile(`\s+`).ReplaceAllString(colDef, " ")
	parts := strings.SplitN(colDef, " ", 2)

	if len(parts) >= 1 {
		detail.Name = strings.Trim(parts[0], "`")
	}

	if len(parts) >= 2 {
		rest := parts[1]
		upperRest := strings.ToUpper(rest)

		// 提取类型（第一个词或带括号的）
		typeRe := regexp.MustCompile(`^(\w+(?:\([^)]+\))?)`)
		if matches := typeRe.FindStringSubmatch(rest); len(matches) > 1 {
			detail.Type = matches[1]
		}

		// NULL/NOT NULL
		if strings.Contains(upperRest, "NOT NULL") {
			detail.Nullable = "NOT NULL"
		} else if strings.Contains(upperRest, " NULL") {
			detail.Nullable = "NULL"
		}

		// DEFAULT
		if re := regexp.MustCompile(`(?i)DEFAULT\s+('([^']*)'|(\w+))`); true {
			if matches := re.FindStringSubmatch(rest); len(matches) > 1 {
				if matches[2] != "" {
					detail.Default = "'" + matches[2] + "'"
				} else {
					detail.Default = matches[3]
				}
			}
		}

		// COMMENT
		if re := regexp.MustCompile(`(?i)COMMENT\s+'([^']*)'`); true {
			if matches := re.FindStringSubmatch(rest); len(matches) > 1 {
				detail.Comment = matches[1]
			}
		}

		// AUTO_INCREMENT, PRIMARY KEY等
		var extras []string
		if strings.Contains(upperRest, "AUTO_INCREMENT") {
			extras = append(extras, "AUTO_INCREMENT")
		}
		if strings.Contains(upperRest, "PRIMARY KEY") {
			extras = append(extras, "PRIMARY KEY")
		}
		if strings.Contains(upperRest, "UNIQUE") {
			extras = append(extras, "UNIQUE")
		}
		if strings.Contains(upperRest, "UNSIGNED") {
			extras = append(extras, "UNSIGNED")
		}
		if len(extras) > 0 {
			detail.Extra = strings.Join(extras, ", ")
		}
	}

	return detail
}

// extractKeyColumns 提取键的列名
func extractKeyColumns(keyDef string) string {
	re := regexp.MustCompile(`\(([^)]+)\)`)
	if matches := re.FindStringSubmatch(keyDef); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseIndexDefinition 解析索引定义
func parseIndexDefinition(indexDef string, indexType string) IndexDetail {
	detail := IndexDetail{
		Type: indexType,
	}

	// 提取索引名称
	// PRIMARY KEY (`id`) - 无名称
	// UNIQUE KEY `idx_name` (`col1`, `col2`)
	// INDEX `idx_name` (`col1`)
	// KEY `idx_name` (`col1`)
	nameRe := regexp.MustCompile("(?i)(?:UNIQUE\\s+)?(?:KEY|INDEX)\\s+`?(\\w+)`?\\s*\\(")
	if matches := nameRe.FindStringSubmatch(indexDef); len(matches) > 1 {
		detail.Name = matches[1]
	} else if indexType == "PRIMARY" {
		detail.Name = "PRIMARY"
	}

	// 提取列名
	colsStr := extractKeyColumns(indexDef)
	if colsStr != "" {
		detail.ColumnStr = colsStr
		// 分割列名
		cols := strings.Split(colsStr, ",")
		for _, col := range cols {
			col = strings.Trim(col, " `\t\n\r")
			// 移除排序方向和长度
			col = regexp.MustCompile(`\s*\(\d+\)$`).ReplaceAllString(col, "")
			col = regexp.MustCompile(`\s+(ASC|DESC)$`).ReplaceAllString(col, "")
			if col != "" {
				detail.Columns = append(detail.Columns, col)
			}
		}
	}

	return detail
}

// describeColumnChanges 描述字段修改的详细内容
func describeColumnChanges(detail ColumnDetail, action string) string {
	var parts []string

	if detail.Type != "" {
		parts = append(parts, fmt.Sprintf("类型=%s", detail.Type))
	}
	if detail.Nullable != "" {
		if detail.Nullable == "NOT NULL" {
			parts = append(parts, "非空约束")
		} else {
			parts = append(parts, "允许NULL")
		}
	}
	if detail.Default != "" {
		parts = append(parts, fmt.Sprintf("默认值=%s", detail.Default))
	}
	if detail.Comment != "" {
		parts = append(parts, fmt.Sprintf("注释='%s'", detail.Comment))
	}
	if detail.Extra != "" {
		parts = append(parts, detail.Extra)
	}

	if len(parts) == 0 {
		return "修改字段定义"
	}
	return strings.Join(parts, ", ")
}

// analyzeCreateIndex 分析CREATE INDEX
func analyzeCreateIndex(sql, upperSQL string, result *DDLAnalysis) {
	re := regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 1 {
		result.ObjectName = matches[1]
	}

	// 提取ON表名
	onRe := regexp.MustCompile(`(?i)ON\s+(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := onRe.FindStringSubmatch(sql); len(matches) > 2 {
		result.AlterActions = append(result.AlterActions, "目标表: "+matches[2])
	}
}

// analyzeCreateView 分析CREATE VIEW
func analyzeCreateView(sql, upperSQL string, result *DDLAnalysis) {
	re := regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?VIEW\s+(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.ObjectName = matches[2]
	}
}

// analyzeCreateDatabase 分析CREATE DATABASE
func analyzeCreateDatabase(sql, upperSQL string, result *DDLAnalysis) {
	re := regexp.MustCompile(`(?i)CREATE\s+(?:DATABASE|SCHEMA)\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 1 {
		result.ObjectName = matches[1]
	}
}

// analyzeAlter 分析ALTER语句
func analyzeAlter(sql, upperSQL string, result *DDLAnalysis) {
	// 判断对象类型
	if strings.Contains(upperSQL, "ALTER TABLE") {
		result.ObjectType = "TABLE"
		analyzeAlterTable(sql, upperSQL, result)
	} else if strings.Contains(upperSQL, "ALTER INDEX") {
		result.ObjectType = "INDEX"
	} else if strings.Contains(upperSQL, "ALTER DATABASE") {
		result.ObjectType = "DATABASE"
	} else {
		result.ObjectType = "OTHER"
	}

	result.RiskLevel = "medium"
	result.RiskReason = "ALTER操作，修改表结构"
}

// analyzeAlterTable 分析ALTER TABLE
func analyzeAlterTable(sql, upperSQL string, result *DDLAnalysis) {
	// 提取表名
	re := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.ObjectName = matches[2]
	}

	result.Details = &DDLDetails{}

	// ADD COLUMN
	addColRe := regexp.MustCompile(`(?i)ADD\s+(?:COLUMN\s+)?` + "`?" + `(\w+)` + "`?" + `\s+(.+?)(?:,|$|ADD|DROP|MODIFY|CHANGE|RENAME)`)
	if matches := addColRe.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 2 {
				colDef := m[1] + " " + strings.TrimSpace(m[2])
				detail := parseColumnDefinition(colDef)
				detail.Name = m[1]
				result.Details.AddColumns = append(result.Details.AddColumns, detail)
				// 详细描述新增字段
				changes := describeColumnChanges(detail, "ADD")
				result.AlterActions = append(result.AlterActions, fmt.Sprintf("ADD COLUMN [%s]: %s", m[1], changes))
			}
		}
	}

	// DROP COLUMN
	dropColRe := regexp.MustCompile(`(?i)DROP\s+(?:COLUMN\s+)?` + "`?" + `(\w+)` + "`?")
	if matches := dropColRe.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 1 && !strings.EqualFold(m[1], "INDEX") && !strings.EqualFold(m[1], "KEY") && !strings.EqualFold(m[1], "PRIMARY") && !strings.EqualFold(m[1], "FOREIGN") {
				result.Details.DropColumns = append(result.Details.DropColumns, m[1])
				result.AlterActions = append(result.AlterActions, fmt.Sprintf("DROP COLUMN: %s ⚠️", m[1]))
			}
		}
		if len(result.Details.DropColumns) > 0 {
			result.RiskLevel = "high"
			result.RiskReason = fmt.Sprintf("⚠️ 删除 %d 个字段，数据将丢失！", len(result.Details.DropColumns))
		}
	}

	// MODIFY COLUMN
	modifyRe := regexp.MustCompile(`(?i)MODIFY\s+(?:COLUMN\s+)?` + "`?" + `(\w+)` + "`?" + `\s+(.+?)(?:,|$|ADD|DROP|MODIFY|CHANGE|RENAME)`)
	if matches := modifyRe.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 2 {
				colDef := m[1] + " " + strings.TrimSpace(m[2])
				detail := parseColumnDefinition(colDef)
				detail.Name = m[1]
				result.Details.ModifyColumns = append(result.Details.ModifyColumns, detail)
				// 详细描述修改内容
				changes := describeColumnChanges(detail, "MODIFY")
				result.AlterActions = append(result.AlterActions, fmt.Sprintf("MODIFY COLUMN [%s]: %s", m[1], changes))
			}
		}
	}

	// CHANGE COLUMN
	changeRe := regexp.MustCompile(`(?i)CHANGE\s+(?:COLUMN\s+)?` + "`?" + `(\w+)` + "`?" + `\s+` + "`?" + `(\w+)` + "`?" + `\s+(.+?)(?:,|$|ADD|DROP|MODIFY|CHANGE|RENAME)`)
	if matches := changeRe.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 3 {
				colDef := m[2] + " " + strings.TrimSpace(m[3])
				detail := parseColumnDefinition(colDef)
				detail.Name = m[2]
				result.Details.ModifyColumns = append(result.Details.ModifyColumns, detail)
				// 详细描述修改内容
				changes := describeColumnChanges(detail, "CHANGE")
				if m[1] != m[2] {
					result.AlterActions = append(result.AlterActions, fmt.Sprintf("CHANGE COLUMN [%s→%s]: 重命名字段, %s", m[1], m[2], changes))
				} else {
					result.AlterActions = append(result.AlterActions, fmt.Sprintf("CHANGE COLUMN [%s]: %s", m[1], changes))
				}
			}
		}
	}

	// ADD INDEX
	addIdxRe := regexp.MustCompile(`(?i)ADD\s+(UNIQUE\s+)?(?:INDEX|KEY)\s+` + "`?" + `(\w+)` + "`?" + `\s*\(([^)]+)\)`)
	if matches := addIdxRe.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 3 {
				idxType := "INDEX"
				if strings.TrimSpace(m[1]) != "" {
					idxType = "UNIQUE"
				}
				idx := IndexDetail{
					Name:      m[2],
					Type:      idxType,
					ColumnStr: m[3],
				}
				cols := strings.Split(m[3], ",")
				for _, col := range cols {
					col = strings.Trim(col, " `\t\n\r")
					if col != "" {
						idx.Columns = append(idx.Columns, col)
					}
				}
				result.Details.AddIndexes = append(result.Details.AddIndexes, idx)
				result.AlterActions = append(result.AlterActions, fmt.Sprintf("ADD %s INDEX: %s(%s)", idxType, m[2], m[3]))
			}
		}
	}

	// DROP INDEX
	dropIdxRe := regexp.MustCompile(`(?i)DROP\s+(?:INDEX|KEY)\s+` + "`?" + `(\w+)` + "`?")
	if matches := dropIdxRe.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 1 {
				result.Details.DropIndexes = append(result.Details.DropIndexes, m[1])
				result.AlterActions = append(result.AlterActions, fmt.Sprintf("DROP INDEX: %s", m[1]))
			}
		}
	}

	// RENAME
	renameRe := regexp.MustCompile(`(?i)RENAME\s+(?:TO|AS)\s+` + "`?" + `(\w+)` + "`?")
	if matches := renameRe.FindStringSubmatch(sql); len(matches) > 1 {
		result.Details.RenameInfo = fmt.Sprintf("%s → %s", result.ObjectName, matches[1])
		result.AlterActions = append(result.AlterActions, fmt.Sprintf("RENAME: %s → %s", result.ObjectName, matches[1]))
	}

	// COMMENT
	commentRe := regexp.MustCompile(`(?i)COMMENT\s*=?\s*'([^']*)'`)
	if matches := commentRe.FindStringSubmatch(sql); len(matches) > 1 {
		result.Details.ChangeComment = matches[1]
		result.AlterActions = append(result.AlterActions, fmt.Sprintf("COMMENT: '%s'", matches[1]))
	}

	// 如果没有检测到具体操作，使用通用检测
	if len(result.AlterActions) == 0 {
		if strings.Contains(upperSQL, " ADD ") {
			result.AlterActions = append(result.AlterActions, "ADD（添加）")
		}
		if strings.Contains(upperSQL, " DROP ") {
			result.AlterActions = append(result.AlterActions, "DROP（删除）")
			result.RiskLevel = "high"
			result.RiskReason = "⚠️ 包含删除操作，请谨慎执行！"
		}
		if strings.Contains(upperSQL, " MODIFY ") || strings.Contains(upperSQL, " CHANGE ") {
			result.AlterActions = append(result.AlterActions, "MODIFY（修改）")
		}
	}
}

// analyzeDrop 分析DROP语句
func analyzeDrop(sql, upperSQL string, result *DDLAnalysis) {
	// 判断对象类型
	if strings.Contains(upperSQL, "DROP TABLE") {
		result.ObjectType = "TABLE"
		re := regexp.MustCompile(`(?i)DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
		if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
			result.ObjectName = matches[2]
		}
		result.RiskLevel = "high"
		result.RiskReason = "⚠️ DROP TABLE将删除整个表及所有数据！"
	} else if strings.Contains(upperSQL, "DROP INDEX") {
		result.ObjectType = "INDEX"
		result.RiskLevel = "medium"
		result.RiskReason = "删除索引，可能影响查询性能"
	} else if strings.Contains(upperSQL, "DROP VIEW") {
		result.ObjectType = "VIEW"
		result.RiskLevel = "medium"
		result.RiskReason = "删除视图"
	} else if strings.Contains(upperSQL, "DROP DATABASE") {
		result.ObjectType = "DATABASE"
		re := regexp.MustCompile(`(?i)DROP\s+DATABASE\s+(?:IF\s+EXISTS\s+)?(?:` + "`" + `)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
		if matches := re.FindStringSubmatch(sql); len(matches) > 1 {
			result.ObjectName = matches[1]
		}
		result.RiskLevel = "high"
		result.RiskReason = "⚠️ DROP DATABASE将删除整个数据库！"
	}
}

// analyzeTruncate 分析TRUNCATE语句
func analyzeTruncate(sql, upperSQL string, result *DDLAnalysis) {
	result.ObjectType = "TABLE"

	re := regexp.MustCompile(`(?i)TRUNCATE\s+(?:TABLE\s+)?(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.ObjectName = matches[2]
	}

	result.RiskLevel = "high"
	result.RiskReason = "⚠️ TRUNCATE将清空表中所有数据！"
}

// analyzeRename 分析RENAME语句
func analyzeRename(sql, upperSQL string, result *DDLAnalysis) {
	result.ObjectType = "TABLE"

	re := regexp.MustCompile(`(?i)RENAME\s+TABLE\s+(?:` + "`" + `)?(?:([a-zA-Z_][a-zA-Z0-9_]*)\.)?([a-zA-Z_][a-zA-Z0-9_]*)(?:` + "`" + `)?`)
	if matches := re.FindStringSubmatch(sql); len(matches) > 2 {
		result.ObjectName = matches[2]
	}

	result.RiskLevel = "medium"
	result.RiskReason = "重命名表"
}
