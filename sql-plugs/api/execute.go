package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sql-plugs/common"
	"sql-plugs/config"
	"strings"
	"time"
)

// SQLExecuteRequest SQL执行请求
type SQLExecuteRequest struct {
	DBName string `json:"dbName"`
	Query  string `json:"query"`
}

// SQLExecuteResponse SQL执行响应
type SQLExecuteResponse struct {
	Success      bool   `json:"success"`
	AffectedRows int64  `json:"affected_rows"`
	LastInsertID int64  `json:"last_insert_id,omitempty"`
	Message      string `json:"message"`
	SQLType      string `json:"sql_type"`
	Took         int64  `json:"took"`
	DBName       string `json:"db_name"`
}

// SQLBatchExecuteResponse 批量SQL执行响应
type SQLBatchExecuteResponse struct {
	Results []SQLExecuteResponse `json:"results"`
	Total   int                  `json:"total"`
	Took    int64                `json:"took"`
	DBName  string               `json:"db_name"`
}

// SQLExecuteHandler 处理SQL执行请求（增删改）
func SQLExecuteHandler(w http.ResponseWriter, r *http.Request) {
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

	var req SQLExecuteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	if req.Query == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "SQL语句不能为空")
		return
	}

	// 按分号分割多个SQL语句
	queries := common.SplitSQLStatements(req.Query)
	if len(queries) == 0 {
		common.ErrorWithCode(w, http.StatusBadRequest, "SQL语句不能为空")
		return
	}

	common.Logger.Infof("SQL批量执行请求 - 语句数量: %d", len(queries))

	startTime := time.Now()
	results := make([]SQLExecuteResponse, 0, len(queries))
	dbName := req.DBName

	for i, query := range queries {
		// 规范化SQL
		normalizedSQL := common.NormalizeWhitespace(query)

		// 安全检查
		if err := validateExecuteSQL(normalizedSQL); err != nil {
			common.Logger.Warnf("SQL语句[%d]验证失败: %v", i+1, err)
			results = append(results, SQLExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("SQL语句[%d]验证失败: %s", i+1, err.Error()),
				Took:    0,
				DBName:  dbName,
			})
			continue
		}

		// 执行SQL
		execStart := time.Now()
		result, err := executeSQL(dbName, normalizedSQL)
		execTook := time.Since(execStart).Milliseconds()

		if err != nil {
			common.Logger.Errorf("SQL语句[%d]执行失败: %v", i+1, err)
			results = append(results, SQLExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("SQL语句[%d]执行失败: %s", i+1, err.Error()),
				Took:    execTook,
				DBName:  dbName,
			})
			continue
		}

		result.Took = execTook
		results = append(results, *result)
		if dbName == "" {
			dbName = result.DBName
		}
	}

	took := time.Since(startTime).Milliseconds()

	// 统一返回批量结果格式
	response := SQLBatchExecuteResponse{
		Results: results,
		Total:   len(results),
		Took:    took,
		DBName:  dbName,
	}

	common.Success(w, response)
	common.Logger.Infof("SQL执行完成 - 总耗时: %dms, 语句数: %d", took, len(results))
}

// validateExecuteSQL 验证SQL安全性
func validateExecuteSQL(query string) error {
	cleanSQL := common.TrimSQL(query)
	if cleanSQL == "" {
		return fmt.Errorf("SQL语句为空或只包含注释")
	}

	// 使用 common.GetSQLType 获取SQL类型
	sqlType := common.GetSQLType(cleanSQL)
	upperSQL := strings.ToUpper(common.NormalizeWhitespace(cleanSQL))

	// 1. 禁止DROP TABLE/DATABASE
	if sqlType == "DROP" {
		return fmt.Errorf("安全限制：禁止执行DROP语句（删除表/库）")
	}

	// 2. 禁止TRUNCATE
	if sqlType == "TRUNCATE" {
		return fmt.Errorf("安全限制：禁止执行TRUNCATE语句（清空表）")
	}

	// 3. UPDATE/DELETE必须带WHERE
	if sqlType == "UPDATE" || sqlType == "DELETE" {
		if !hasWhereClause(upperSQL) {
			return fmt.Errorf("安全限制：%s语句必须包含WHERE条件", sqlType)
		}
	}

	// 4. CREATE只允许创建表，禁止创建库
	if sqlType == "CREATE" {
		if strings.Contains(upperSQL, "CREATE DATABASE") ||
			strings.Contains(upperSQL, "CREATE SCHEMA") {
			return fmt.Errorf("安全限制：禁止创建数据库")
		}
	}

	// 5. 只允许INSERT/UPDATE/DELETE/CREATE/ALTER
	allowedTypes := map[string]bool{
		"INSERT": true,
		"UPDATE": true,
		"DELETE": true,
		"CREATE": true,
		"ALTER":  true,
	}
	if !allowedTypes[sqlType] {
		return fmt.Errorf("此接口只允许执行INSERT/UPDATE/DELETE/CREATE/ALTER语句，不支持: %s", sqlType)
	}

	return nil
}

// hasWhereClause 检查SQL是否包含WHERE子句
func hasWhereClause(upperSQL string) bool {
	whereRegex := regexp.MustCompile(`\bWHERE\b`)
	return whereRegex.MatchString(upperSQL)
}

// executeSQL 执行SQL语句
func executeSQL(dbName string, query string) (*SQLExecuteResponse, error) {
	db, err := common.GetDB()
	if err != nil {
		return nil, err
	}

	dbConfig := config.GetDatabaseConfig()

	// 切换数据库
	if dbName != "" {
		if !common.IsValidDatabaseName(dbName, 64) {
			return nil, fmt.Errorf("无效的数据库名称: %s", dbName)
		}
		_, err = db.Exec("USE `" + dbName + "`")
		if err != nil {
			return nil, fmt.Errorf("切换数据库失败: %w", err)
		}
		dbConfig.Database = dbName
	}

	// 规范化SQL（统一空白字符）
	normalizedQuery := common.NormalizeWhitespace(query)

	// 执行SQL
	result, err := db.Exec(normalizedQuery)
	if err != nil {
		return nil, err
	}

	affectedRows, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	sqlType := common.GetSQLType(query)

	response := &SQLExecuteResponse{
		Success:      true,
		AffectedRows: affectedRows,
		SQLType:      sqlType,
		DBName:       dbConfig.Database,
	}

	if sqlType == "INSERT" && lastInsertID > 0 {
		response.LastInsertID = lastInsertID
	}

	// 生成消息
	switch sqlType {
	case "INSERT":
		response.Message = fmt.Sprintf("插入成功，影响 %d 行", affectedRows)
	case "UPDATE":
		response.Message = fmt.Sprintf("更新成功，影响 %d 行", affectedRows)
	case "DELETE":
		response.Message = fmt.Sprintf("删除成功，影响 %d 行", affectedRows)
	case "CREATE":
		response.Message = "创建成功"
	case "ALTER":
		response.Message = "修改成功"
	default:
		response.Message = fmt.Sprintf("执行成功，影响 %d 行", affectedRows)
	}

	common.Logger.Infof("SQL执行成功 - 类型: %s, 影响行数: %d, 数据库: %s",
		sqlType, affectedRows, dbConfig.Database)

	return response, nil
}
