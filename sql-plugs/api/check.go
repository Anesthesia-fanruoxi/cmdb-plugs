package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sql-plugs/common"
	"sql-plugs/config"
	"time"
)

// SQLCheckRequest SQL检查请求
type SQLCheckRequest struct {
	DBName string `json:"dbName"`
	Sql    string `json:"sql"`
}

// SQLCheckResult 单条SQL检查结果
type SQLCheckResult struct {
	Success       bool   `json:"success"`
	SQLType       string `json:"sql_type"`
	Valid         bool   `json:"valid"`
	AffectedCount int64  `json:"affected_count,omitempty"` // DELETE/UPDATE的影响行数
	Message       string `json:"message"`
	Error         string `json:"error,omitempty"`
	Took          int64  `json:"took"`
}

// SQLCheckResponse SQL检查响应
type SQLCheckResponse struct {
	Results []SQLCheckResult `json:"results"`
	Total   int              `json:"total"`
	Took    int64            `json:"took"`
	DBName  string           `json:"db_name"`
}

// SQLCheckHandler 处理SQL检查请求
func SQLCheckHandler(w http.ResponseWriter, r *http.Request) {
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

	var req SQLCheckRequest
	if err := json.Unmarshal(body, &req); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	if req.Sql == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "SQL语句不能为空")
		return
	}

	// 按分号分割多个SQL语句
	queries := common.SplitSQLStatements(req.Sql)
	if len(queries) == 0 {
		common.ErrorWithCode(w, http.StatusBadRequest, "SQL语句不能为空")
		return
	}

	common.Logger.Infof("SQL检查请求 - 语句数量: %d", len(queries))

	startTime := time.Now()
	results := make([]SQLCheckResult, 0, len(queries))
	dbName := req.DBName

	for i, query := range queries {
		normalizedSQL := common.NormalizeWhitespace(query)

		execStart := time.Now()
		result := checkSingleSQL(dbName, normalizedSQL, i+1)
		result.Took = time.Since(execStart).Milliseconds()

		results = append(results, result)
		if dbName == "" && result.Success {
			dbName = req.DBName
		}
	}

	took := time.Since(startTime).Milliseconds()

	response := SQLCheckResponse{
		Results: results,
		Total:   len(results),
		Took:    took,
		DBName:  dbName,
	}

	common.Success(w, response)
	common.Logger.Infof("SQL检查完成 - 总耗时: %dms, 语句数: %d", took, len(results))
}

// checkSingleSQL 检查单条SQL
func checkSingleSQL(dbName string, sql string, index int) SQLCheckResult {
	result := SQLCheckResult{
		Success: false,
	}

	// 1. 获取SQL类型
	sqlType := common.GetSQLType(sql)
	result.SQLType = sqlType

	// 2. 执行EXPLAIN验证SQL有效性
	valid, errMsg := explainSQL(dbName, sql, sqlType)
	result.Valid = valid

	if !valid {
		result.Error = errMsg
		result.Message = fmt.Sprintf("SQL语句[%d]无效", index)
		common.Logger.Warnf("SQL语句[%d]验证失败: %s", index, errMsg)
		return result
	}

	// 3. 对于DELETE/UPDATE，计算影响行数
	if sqlType == "DELETE" || sqlType == "UPDATE" {
		count, err := calculateAffectedCount(dbName, sql, sqlType)
		if err != nil {
			result.Error = err.Error()
			result.Message = fmt.Sprintf("SQL语句[%d]计算影响行数失败", index)
			common.Logger.Warnf("SQL语句[%d]计算影响行数失败: %v", index, err)
			return result
		}
		result.AffectedCount = count
		result.Message = fmt.Sprintf("SQL语句[%d]有效，预计影响 %d 行", index, count)
	} else {
		result.Message = fmt.Sprintf("SQL语句[%d]有效", index)
	}

	result.Success = true
	common.Logger.Infof("SQL语句[%d]验证成功 - 类型: %s", index, sqlType)

	return result
}

// explainSQL 使用EXPLAIN验证SQL有效性
func explainSQL(dbName string, sql string, sqlType string) (bool, string) {
	db, err := common.GetDB()
	if err != nil {
		return false, "数据库连接失败: " + err.Error()
	}

	// 切换数据库
	if dbName != "" {
		if !common.IsValidDatabaseName(dbName, 64) {
			return false, "无效的数据库名称"
		}
		_, err = db.Exec("USE `" + dbName + "`")
		if err != nil {
			return false, "切换数据库失败: " + err.Error()
		}
	}

	// DDL语句（CREATE/ALTER/DROP）不支持EXPLAIN，使用语法检查
	sqlCategory := common.GetSQLCategory(sqlType)
	if sqlCategory == "DDL" {
		// 对于DDL，只做基本的语法检查（尝试PREPARE）
		stmt, err := db.Prepare(sql)
		if err != nil {
			return false, err.Error()
		}
		defer stmt.Close()
		return true, ""
	}

	// DML语句（INSERT/UPDATE/DELETE）和DQL语句（SELECT）使用EXPLAIN
	explainSQL := "EXPLAIN " + sql

	// 执行EXPLAIN
	rows, err := db.Query(explainSQL)
	if err != nil {
		return false, err.Error()
	}
	defer rows.Close()

	return true, ""
}

// calculateAffectedCount 计算DELETE/UPDATE影响的行数
func calculateAffectedCount(dbName string, sql string, sqlType string) (int64, error) {
	db, err := common.GetDB()
	if err != nil {
		return 0, err
	}

	dbConfig := config.GetDatabaseConfig()

	// 切换数据库
	if dbName != "" {
		if !common.IsValidDatabaseName(dbName, 64) {
			return 0, fmt.Errorf("无效的数据库名称")
		}
		_, err = db.Exec("USE `" + dbName + "`")
		if err != nil {
			return 0, fmt.Errorf("切换数据库失败: %w", err)
		}
		dbConfig.Database = dbName
	}

	// 将DELETE/UPDATE转换为SELECT COUNT(*)
	countSQL := convertToCountSQL(sql, sqlType)
	if countSQL == "" {
		return 0, fmt.Errorf("无法转换为COUNT查询")
	}

	common.Logger.Infof("计算影响行数 - 原SQL类型: %s, COUNT查询: %s", sqlType, countSQL)

	// 执行COUNT查询
	var count int64
	err = db.QueryRow(countSQL).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("执行COUNT查询失败: %w", err)
	}

	return count, nil
}

// convertToCountSQL 将DELETE/UPDATE转换为SELECT COUNT(*)
func convertToCountSQL(sql string, sqlType string) string {
	if sqlType == "DELETE" {
		// DELETE FROM table WHERE ... -> SELECT COUNT(*) FROM table WHERE ...
		re := regexp.MustCompile(`(?i)DELETE\s+FROM\s+(.+)`)
		matches := re.FindStringSubmatch(sql)
		if len(matches) > 1 {
			return "SELECT COUNT(*) FROM " + matches[1]
		}
	} else if sqlType == "UPDATE" {
		// UPDATE table SET ... WHERE ... -> SELECT COUNT(*) FROM table WHERE ...
		// 需要提取表名和WHERE条件
		re := regexp.MustCompile(`(?i)UPDATE\s+([^\s]+)\s+SET\s+.+?(WHERE\s+.+)?$`)
		matches := re.FindStringSubmatch(sql)
		if len(matches) > 1 {
			tableName := matches[1]
			whereClause := ""
			if len(matches) > 2 {
				whereClause = matches[2]
			}
			return "SELECT COUNT(*) FROM " + tableName + " " + whereClause
		}
	}

	return ""
}
