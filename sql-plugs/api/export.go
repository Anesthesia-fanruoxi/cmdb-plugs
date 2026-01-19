package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sql-plugs/common"
	"sql-plugs/config"
	"sql-plugs/model"
	"strings"
	"time"
)

const (
	// QueryTimeout 查询超时时间（10分钟）
	QueryTimeout = 10 * time.Minute
)

// ExportHandler 处理数据导出请求
// 不做任何查询限制，不执行COUNT统计，直接返回所有数据
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	// 解析请求
	body, err := io.ReadAll(r.Body)
	if err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "读取请求体失败: "+err.Error())
		return
	}
	defer r.Body.Close()

	var rawReq struct {
		Query json.RawMessage `json:"query"`
		DB    string          `json:"dbName"`
	}
	if err := json.Unmarshal(body, &rawReq); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	var queryStr string
	if err := json.Unmarshal(rawReq.Query, &queryStr); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "query字段解析失败: "+err.Error())
		return
	}

	if queryStr == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "查询语句不能为空")
		return
	}

	// 验证SQL安全性
	if err := validateExportQuery(queryStr); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, err.Error())
		return
	}

	common.Logger.Infof("数据导出请求 - 数据库: %s, SQL: %s", rawReq.DB, queryStr)

	// 执行导出
	startTime := time.Now()
	result, err := executeExportQuery(rawReq.DB, queryStr)
	if err != nil {
		common.Logger.Errorf("导出查询失败: %v", err)
		common.ErrorWithCode(w, http.StatusInternalServerError, "导出查询失败: "+err.Error())
		return
	}

	took := time.Since(startTime).Milliseconds()

	// 返回响应（格式与search接口一致）
	response := model.SQLSearchResponse{
		Results: []model.QueryResult{*result},
		Total:   1,
		Took:    took,
		DBName:  result.DBName,
	}

	common.SuccessWithMessage(w, "导出成功", response)
	common.Logger.Infof("数据导出成功 - 总耗时: %dms, 返回行数: %d", took, result.Total)
}

// validateExportQuery 验证SQL安全性
func validateExportQuery(query string) error {
	// 移除注释
	re := regexp.MustCompile(`/\*.*?\*/`)
	query = re.ReplaceAllString(query, "")
	lines := strings.Split(query, "\n")
	var cleanLines []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		if line = strings.TrimSpace(line); line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	cleanQuery := strings.ToUpper(strings.Join(cleanLines, " "))

	// 允许的查询类型
	allowedPrefixes := []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"}
	isAllowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanQuery, prefix) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return fmt.Errorf("导出接口只允许查询语句（SELECT/WITH/SHOW/DESCRIBE/EXPLAIN）")
	}

	// 禁止的操作
	dangerousPrefixes := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "REPLACE", "GRANT", "REVOKE"}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(cleanQuery, prefix) {
			return fmt.Errorf("导出接口不允许执行 %s 操作", prefix)
		}
	}

	return nil
}

// executeExportQuery 执行导出查询
func executeExportQuery(dbName string, query string) (*model.QueryResult, error) {
	db, err := common.GetDB()
	if err != nil {
		return nil, err
	}

	dbConfig := config.GetDatabaseConfig()

	// 切换数据库
	if dbName != "" {
		if len(dbName) == 0 || len(dbName) > 64 {
			return nil, fmt.Errorf("无效的数据库名称")
		}
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, dbName)
		if !matched {
			return nil, fmt.Errorf("无效的数据库名称")
		}
		if _, err = db.Exec("USE `" + dbName + "`"); err != nil {
			return nil, fmt.Errorf("切换数据库失败: %w", err)
		}
		common.Logger.Infof("已切换到数据库: %s", dbName)
		dbConfig.Database = dbName
	}

	startTime := time.Now()
	db.Exec("SET NAMES utf8mb4")

	// 带超时的查询
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	common.Logger.Infof("开始执行导出查询（超时: %v）", QueryTimeout)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("查询超时（超过%v），请优化SQL", QueryTimeout)
		}
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}

	// 读取数据
	dataRows := make([][]interface{}, 0)
	rowCount := 0

	for rows.Next() {
		values := make([]sql.RawBytes, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取数据失败: %w", err)
		}

		row := make([]interface{}, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = nil
			} else {
				row[i] = string(val)
			}
		}
		dataRows = append(dataRows, row)
		rowCount++

		if rowCount%10000 == 0 {
			common.Logger.Infof("导出进度: 已读取 %d 行", rowCount)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历数据失败: %w", err)
	}

	took := time.Since(startTime).Milliseconds()
	common.Logger.Infof("导出完成 - 返回: %d 行, 耗时: %dms", len(dataRows), took)

	return &model.QueryResult{
		Columns: columns,
		Rows:    dataRows,
		Total:   len(dataRows),
		Took:    took,
		DBName:  dbConfig.Database,
	}, nil
}
