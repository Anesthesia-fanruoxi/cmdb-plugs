package common

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SQLAnalyzeLogData 分析日志数据
type SQLAnalyzeLogData struct {
	DBName       string
	SQLType      string
	SQLCategory  string
	HasLimit     bool
	UserLimit    int
	FinalLimit   int
	WillAddLimit bool
	WillCount    bool
	HasFilter    bool
	Databases    []string
	Tables       []string
	Columns      []string
	OriginalSQL  string
	Features     *SQLFeatures
	DMLInfo      *DMLLogInfo
	DDLInfo      *DDLLogInfo
}

// DMLLogInfo DML日志信息
type DMLLogInfo struct {
	TargetTable  string
	AffectedCols []string
	DataSource   string
	HasWhere     bool
	WherePreview string
	EstimateRows string
	RiskLevel    string
	RiskReason   string
}

// DDLLogInfo DDL日志信息
type DDLLogInfo struct {
	Operation    string
	ObjectType   string
	ObjectName   string
	ColumnsDef   []string
	AlterActions []string
	RiskLevel    string
	RiskReason   string
}

// WriteAnalysisLog 将分析结果写入文件
func WriteAnalysisLog(data *SQLAnalyzeLogData, took int64) {
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		Logger.Errorf("创建logs目录失败: %v", err)
		return
	}

	logFile := filepath.Join(logsDir, "sql_analyze.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		Logger.Errorf("打开日志文件失败: %v", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var content string

	switch data.SQLCategory {
	case "DQL":
		content = buildDQLLog(data, timestamp, took)
	case "DML":
		content = buildDMLLog(data, timestamp, took)
	case "DDL":
		content = buildDDLLog(data, timestamp, took)
	default:
		content = buildDefaultLog(data, timestamp, took)
	}

	if _, err := f.WriteString(content); err != nil {
		Logger.Errorf("写入日志文件失败: %v", err)
	}
}

func buildDQLLog(data *SQLAnalyzeLogData, timestamp string, took int64) string {
	featuresStr := ""
	if data.Features != nil {
		featuresStr = fmt.Sprintf(`
  - WHERE条件: %v
  - JOIN: %v (类型: %s, 数量: %d)
  - GROUP BY: %v
  - HAVING: %v
  - ORDER BY: %v
  - DISTINCT: %v
  - 子查询: %v
  - UNION: %v
  - 聚合函数: %v
  - CTE(WITH): %v`,
			data.Features.HasWhere,
			data.Features.HasJoin, data.Features.JoinType, data.Features.JoinCount,
			data.Features.HasGroupBy, data.Features.HasHaving, data.Features.HasOrderBy,
			data.Features.HasDistinct, data.Features.HasSubquery, data.Features.HasUnion,
			data.Features.HasAggregate, data.Features.HasCTE)
	}

	return fmt.Sprintf(`
========================================
时间: %s
数据库: %s
SQL类型: %s
SQL分类: %s
是否有LIMIT: %v
用户LIMIT值: %d
最终LIMIT值: %d
是否自动添加LIMIT: %v
是否执行COUNT: %v
是否有过滤条件: %v
涉及数据库: %v
涉及表: %v
涉及字段: %v
SQL特性:%s
分析耗时: %dms

--- 原始SQL ---
%s
========================================

`, timestamp, data.DBName, data.SQLType, data.SQLCategory, data.HasLimit, data.UserLimit,
		data.FinalLimit, data.WillAddLimit, data.WillCount, data.HasFilter,
		data.Databases, data.Tables, data.Columns, featuresStr, took, data.OriginalSQL)
}

func buildDMLLog(data *SQLAnalyzeLogData, timestamp string, took int64) string {
	dmlStr := ""
	if data.DMLInfo != nil {
		dmlStr = fmt.Sprintf(`
DML分析:
  - 目标表: %s
  - 影响字段: %v
  - 数据来源: %s
  - 有WHERE条件: %v
  - WHERE预览: %s
  - 预估影响行数: %s
  - 风险等级: %s
  - 风险说明: %s`,
			data.DMLInfo.TargetTable, data.DMLInfo.AffectedCols, data.DMLInfo.DataSource,
			data.DMLInfo.HasWhere, data.DMLInfo.WherePreview, data.DMLInfo.EstimateRows,
			data.DMLInfo.RiskLevel, data.DMLInfo.RiskReason)
	}

	return fmt.Sprintf(`
========================================
时间: %s
数据库: %s
SQL类型: %s
SQL分类: %s
涉及数据库: %v
涉及表: %v
%s
分析耗时: %dms

--- 原始SQL ---
%s
========================================

`, timestamp, data.DBName, data.SQLType, data.SQLCategory,
		data.Databases, data.Tables, dmlStr, took, data.OriginalSQL)
}

func buildDDLLog(data *SQLAnalyzeLogData, timestamp string, took int64) string {
	ddlStr := ""
	if data.DDLInfo != nil {
		ddlStr = fmt.Sprintf(`
DDL分析:
  - 操作类型: %s
  - 对象类型: %s
  - 对象名称: %s
  - 字段定义: %v
  - ALTER操作: %v
  - 风险等级: %s
  - 风险说明: %s`,
			data.DDLInfo.Operation, data.DDLInfo.ObjectType, data.DDLInfo.ObjectName,
			data.DDLInfo.ColumnsDef, data.DDLInfo.AlterActions,
			data.DDLInfo.RiskLevel, data.DDLInfo.RiskReason)
	}

	return fmt.Sprintf(`
========================================
时间: %s
数据库: %s
SQL类型: %s
SQL分类: %s
涉及数据库: %v
涉及表: %v
%s
分析耗时: %dms

--- 原始SQL ---
%s
========================================

`, timestamp, data.DBName, data.SQLType, data.SQLCategory,
		data.Databases, data.Tables, ddlStr, took, data.OriginalSQL)
}

func buildDefaultLog(data *SQLAnalyzeLogData, timestamp string, took int64) string {
	return fmt.Sprintf(`
========================================
时间: %s
数据库: %s
SQL类型: %s
SQL分类: %s
分析耗时: %dms

--- 原始SQL ---
%s
========================================

`, timestamp, data.DBName, data.SQLType, data.SQLCategory, took, data.OriginalSQL)
}
