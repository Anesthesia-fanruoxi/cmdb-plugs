package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sql-plugs/common"
	"sql-plugs/model"
	"time"
)

// SQLAnalyzeRequest SQL分析请求
type SQLAnalyzeRequest struct {
	DBName string `json:"dbName"`
	Query  string `json:"query"`
}

// MultiSQLAnalyzeResponse 多SQL分析响应
type MultiSQLAnalyzeResponse struct {
	Valid      bool                  `json:"valid"`
	Message    string                `json:"message,omitempty"`
	TotalCount int                   `json:"total_count"`
	Results    []*SQLAnalyzeResponse `json:"results"`
}

// SQLAnalyzeResponse SQL分析响应
type SQLAnalyzeResponse struct {
	OriginalSQL      string                   `json:"original_sql"`
	NormalizedSQL    string                   `json:"normalized_sql"`
	CountSQL         string                   `json:"count_sql"`
	ExecuteSQL       string                   `json:"execute_sql"`
	HasLimit         bool                     `json:"has_limit"`
	UserLimit        int                      `json:"user_limit"`
	FinalLimit       int                      `json:"final_limit"`
	WillAddLimit     bool                     `json:"will_add_limit"`
	WillCount        bool                     `json:"will_count"`
	SQLType          string                   `json:"sql_type"`
	SQLCategory      string                   `json:"sql_category"`
	RiskLevel        string                   `json:"risk_level"`
	RiskReason       string                   `json:"risk_reason"`
	DBName           string                   `json:"db_name"`
	HasFilter        bool                     `json:"has_filter"`
	Databases        []string                 `json:"databases"`
	Tables           []string                 `json:"tables"`
	Columns          []string                 `json:"columns"`
	Features         *common.SQLFeatures      `json:"features"`
	Structure        *common.SQLStructure     `json:"structure,omitempty"`
	IndexSuggestions []common.IndexSuggestion `json:"index_suggestions,omitempty"`
	DMLInfo          *model.DMLAnalysis       `json:"dml_info,omitempty"`
	DDLInfo          *model.DDLAnalysis       `json:"ddl_info,omitempty"`
}

// SQLAnalyzeHandler 处理SQL分析请求
func SQLAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "读取请求体失败: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req SQLAnalyzeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	if req.Query == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "查询语句不能为空")
		return
	}

	sqls := common.SplitMultipleSQL(req.Query)
	if len(sqls) == 0 {
		common.ErrorWithCode(w, http.StatusBadRequest, "未检测到有效的SQL语句")
		return
	}

	valid, message := common.ValidateSQLMix(sqls)

	var results []*SQLAnalyzeResponse
	for _, sql := range sqls {
		result := analyzeSingleSQL(req.DBName, sql)
		results = append(results, result)
	}

	response := &MultiSQLAnalyzeResponse{
		Valid:      valid,
		Message:    message,
		TotalCount: len(results),
		Results:    results,
	}

	common.Success(w, response)
}

// analyzeSingleSQL 分析单条SQL
func analyzeSingleSQL(dbName string, sql string) *SQLAnalyzeResponse {
	startTime := time.Now()
	result := analyzeSQL(dbName, sql)
	took := time.Since(startTime).Milliseconds()

	result.Databases = common.ExtractDatabases(sql)
	result.Tables = common.ExtractTables(sql)
	result.HasFilter = common.HasFilterConditions(sql)

	switch result.SQLCategory {
	case "DQL":
		result.Columns = common.ExtractColumns(sql)
		result.Structure = common.AnalyzeSQLStructure(sql)
		result.IndexSuggestions = common.AnalyzeIndexSuggestions(sql, result.Structure)
	case "DML":
		dmlInfo := common.AnalyzeDML(sql, result.SQLType)
		result.DMLInfo = &model.DMLAnalysis{
			TargetTable: dmlInfo.TargetTable, AffectedCols: dmlInfo.AffectedCols,
			DataSource: dmlInfo.DataSource, HasWhere: dmlInfo.HasWhere,
			WherePreview: dmlInfo.WherePreview, EstimateRows: dmlInfo.EstimateRows,
			RiskLevel: dmlInfo.RiskLevel, RiskReason: dmlInfo.RiskReason,
		}
	case "DDL":
		ddlInfo := common.AnalyzeDDL(sql, result.SQLType)
		result.DDLInfo = &model.DDLAnalysis{
			Operation: ddlInfo.Operation, ObjectType: ddlInfo.ObjectType,
			ObjectName: ddlInfo.ObjectName, ColumnsDef: ddlInfo.ColumnsDef,
			AlterActions: ddlInfo.AlterActions, RiskLevel: ddlInfo.RiskLevel,
			RiskReason: ddlInfo.RiskReason,
		}
		if ddlInfo.Details != nil {
			result.DDLInfo.Details = copyDDLDetails(ddlInfo.Details)
		}
	}

	writeAnalysisLog(result, took)
	return result
}

// analyzeSQL 分析SQL基本信息
func analyzeSQL(dbName string, originalSQL string) *SQLAnalyzeResponse {
	normalizedSQL := common.NormalizeWhitespace(common.TrimSQL(originalSQL))

	result := &SQLAnalyzeResponse{
		OriginalSQL:   originalSQL,
		NormalizedSQL: normalizedSQL,
		DBName:        dbName,
	}

	result.SQLType = common.GetSQLType(originalSQL)
	result.SQLCategory = common.GetSQLCategory(result.SQLType)
	result.Features = common.AnalyzeSQLFeatures(originalSQL)

	userLimit := common.GetUserOriginalLimit(originalSQL)
	result.HasLimit = userLimit > 0
	result.UserLimit = userLimit
	result.HasFilter = common.HasFilterConditions(originalSQL)

	if result.SQLCategory == "DQL" || result.SQLCategory == "OTHER" {
		result.RiskLevel, result.RiskReason = common.AssessQueryRisk(originalSQL, result.Features)
	}

	if result.SQLType != "SELECT" && result.SQLType != "WITH" {
		result.ExecuteSQL = originalSQL
		result.CountSQL = "-- 非SELECT/WITH查询，不执行COUNT"
	} else if result.HasLimit {
		result.ExecuteSQL = common.ProcessSQLLimit(originalSQL)
		result.FinalLimit = userLimit
		if userLimit > common.MaxLimit {
			result.FinalLimit = common.MaxLimit
		}
		result.CountSQL = "-- 用户指定LIMIT，不执行COUNT，total=用户LIMIT值"
	} else if result.HasFilter {
		result.ExecuteSQL = originalSQL
		result.CountSQL = "-- 有过滤条件，不执行COUNT，直接返回全部数据"
	} else {
		result.WillCount = true
		result.WillAddLimit = true
		result.ExecuteSQL = common.ProcessSQLLimit(originalSQL)
		result.FinalLimit = common.DefaultLimit
		result.CountSQL = common.BuildCountSQL(originalSQL)
	}

	return result
}

// writeAnalysisLog 写入分析日志
func writeAnalysisLog(result *SQLAnalyzeResponse, took int64) {
	data := &common.SQLAnalyzeLogData{
		DBName:       result.DBName,
		SQLType:      result.SQLType,
		SQLCategory:  result.SQLCategory,
		HasLimit:     result.HasLimit,
		UserLimit:    result.UserLimit,
		FinalLimit:   result.FinalLimit,
		WillAddLimit: result.WillAddLimit,
		WillCount:    result.WillCount,
		HasFilter:    result.HasFilter,
		Databases:    result.Databases,
		Tables:       result.Tables,
		Columns:      result.Columns,
		OriginalSQL:  result.OriginalSQL,
		Features:     result.Features,
	}

	if result.DMLInfo != nil {
		data.DMLInfo = &common.DMLLogInfo{
			TargetTable:  result.DMLInfo.TargetTable,
			AffectedCols: result.DMLInfo.AffectedCols,
			DataSource:   result.DMLInfo.DataSource,
			HasWhere:     result.DMLInfo.HasWhere,
			WherePreview: result.DMLInfo.WherePreview,
			EstimateRows: result.DMLInfo.EstimateRows,
			RiskLevel:    result.DMLInfo.RiskLevel,
			RiskReason:   result.DMLInfo.RiskReason,
		}
	}

	if result.DDLInfo != nil {
		data.DDLInfo = &common.DDLLogInfo{
			Operation:    result.DDLInfo.Operation,
			ObjectType:   result.DDLInfo.ObjectType,
			ObjectName:   result.DDLInfo.ObjectName,
			ColumnsDef:   result.DDLInfo.ColumnsDef,
			AlterActions: result.DDLInfo.AlterActions,
			RiskLevel:    result.DDLInfo.RiskLevel,
			RiskReason:   result.DDLInfo.RiskReason,
		}
	}

	common.WriteAnalysisLog(data, took)
}

// copyDDLDetails 复制DDL详情
func copyDDLDetails(src *common.DDLDetails) *model.DDLDetails {
	dst := &model.DDLDetails{
		ColumnCount: src.ColumnCount, TableComment: src.TableComment,
		Engine: src.Engine, Charset: src.Charset, Collation: src.Collation,
		PrimaryKey: src.PrimaryKey, HasIndex: src.HasIndex, IndexCount: src.IndexCount,
		ForeignKeys: src.ForeignKeys, DropColumns: src.DropColumns,
		DropIndexes: src.DropIndexes, RenameInfo: src.RenameInfo,
		ChangeComment: src.ChangeComment,
	}
	for _, idx := range src.Indexes {
		dst.Indexes = append(dst.Indexes, model.IndexDetail{
			Name: idx.Name, Type: idx.Type, Columns: idx.Columns, ColumnStr: idx.ColumnStr,
		})
	}
	for _, idx := range src.AddIndexes {
		dst.AddIndexes = append(dst.AddIndexes, model.IndexDetail{
			Name: idx.Name, Type: idx.Type, Columns: idx.Columns, ColumnStr: idx.ColumnStr,
		})
	}
	for _, col := range src.Columns {
		dst.Columns = append(dst.Columns, model.ColumnDetail{
			Name: col.Name, Type: col.Type, Nullable: col.Nullable,
			Default: col.Default, Comment: col.Comment, Extra: col.Extra,
		})
	}
	for _, col := range src.AddColumns {
		dst.AddColumns = append(dst.AddColumns, model.ColumnDetail{
			Name: col.Name, Type: col.Type, Nullable: col.Nullable,
			Default: col.Default, Comment: col.Comment, Extra: col.Extra,
		})
	}
	for _, col := range src.ModifyColumns {
		dst.ModifyColumns = append(dst.ModifyColumns, model.ColumnDetail{
			Name: col.Name, Type: col.Type, Nullable: col.Nullable,
			Default: col.Default, Comment: col.Comment, Extra: col.Extra,
		})
	}
	return dst
}
