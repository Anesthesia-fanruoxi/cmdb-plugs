package api

import (
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

// SearchPhoneHandler 处理字段查询请求（不限制返回数量）
// 支持单个或多个字段查询
func SearchPhoneHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许POST请求
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	// 读取原始请求体字节
	body, err := io.ReadAll(r.Body)
	if err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "读取请求体失败: "+err.Error())
		return
	}
	defer r.Body.Close()

	// 解析JSON
	var rawReq struct {
		Query json.RawMessage `json:"query"`
		DB    string          `json:"dbName"`
	}
	if err := json.Unmarshal(body, &rawReq); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	// 将RawMessage转为string
	var queryStr string
	if err := json.Unmarshal(rawReq.Query, &queryStr); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "query字段解析失败: "+err.Error())
		return
	}

	// 构造请求对象
	var req model.SQLSearchRequest
	req.Query = queryStr
	req.DB = rawReq.DB

	// 验证必填参数
	if req.Query == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "查询语句不能为空")
		return
	}

	// 验证SQL安全性
	if err := validateFieldQuery(req.Query); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, err.Error())
		return
	}

	// 记录请求日志
	common.Logger.Infof("字段查询请求 - SQL: %s", req.Query)

	// 执行查询（不对SQL进行任何处理）
	startTime := time.Now()
	result, err := executeRawQuery(req.DB, req.Query)
	if err != nil {
		common.Logger.Errorf("查询失败: %v", err)
		common.ErrorWithCode(w, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	took := time.Since(startTime).Milliseconds()

	// 构造响应
	response := model.SQLSearchResponse{
		Results: []model.QueryResult{*result},
		Total:   1,
		Took:    took,
		DBName:  result.DBName,
	}

	// 返回成功响应
	common.SuccessWithMessage(w, "查询成功", response)
	common.Logger.Infof("字段查询成功 - 总耗时: %dms, 返回行数: %d", took, result.Total)
}

// validateFieldQuery 验证SQL查询安全性（支持多字段）
func validateFieldQuery(query string) error {
	query = strings.TrimSpace(query)
	queryUpper := strings.ToUpper(query)

	// 只允许SELECT语句
	if !strings.HasPrefix(queryUpper, "SELECT") {
		return fmt.Errorf("只允许SELECT查询语句")
	}

	// 移除多行注释和单行注释
	query = removeCommentsForValidation(query)

	// 提取SELECT和FROM之间的字段部分
	// 使用正则表达式匹配 SELECT ... FROM
	re := regexp.MustCompile(`(?i)^\s*SELECT\s+(.*?)\s+FROM\s+`)
	matches := re.FindStringSubmatch(query)
	if matches == nil || len(matches) < 2 {
		return fmt.Errorf("无法解析SELECT语句")
	}

	fieldsStr := strings.TrimSpace(matches[1])

	// 检查是否为 SELECT *
	if fieldsStr == "*" {
		return fmt.Errorf("不允许使用 SELECT *，必须明确指定字段")
	}

	return nil
}

// removeCommentsForValidation 移除SQL注释用于验证
func removeCommentsForValidation(query string) string {
	// 移除多行注释 /* ... */
	re := regexp.MustCompile(`/\*.*?\*/`)
	query = re.ReplaceAllString(query, "")

	// 移除单行注释 --
	lines := strings.Split(query, "\n")
	var cleanLines []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, " ")
}

// executeRawQuery 执行原始SQL查询（不做任何处理）
func executeRawQuery(dbName string, query string) (*model.QueryResult, error) {
	// 获取数据库连接
	db, err := common.GetDB()
	if err != nil {
		return nil, err
	}

	// 获取数据库配置
	dbConfig := config.GetDatabaseConfig()

	// 如果指定了dbName，切换到该数据库
	if dbName != "" {
		// 验证数据库名（只允许字母、数字、下划线，防止SQL注入）
		if !common.IsValidDatabaseName(dbName, 64) {
			return nil, fmt.Errorf("无效的数据库名称: %s", dbName)
		}
		_, err = db.Exec("USE `" + dbName + "`")
		if err != nil {
			return nil, fmt.Errorf("切换数据库失败: %w", err)
		}
		common.Logger.Infof("已切换到数据库: %s", dbName)
		dbConfig.Database = dbName
	}

	// 执行查询
	startTime := time.Now()

	// 强制设置字符集
	_, _ = db.Exec("SET NAMES utf8mb4")

	// 直接执行原始SQL（不做任何LIMIT处理）
	rows, err := db.Query(query)
	if err != nil {
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
	for rows.Next() {
		// 使用 sql.RawBytes 接收原始数据
		values := make([]sql.RawBytes, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取数据失败: %w", err)
		}

		// 转换数据类型
		row := make([]interface{}, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = nil
			} else {
				row[i] = string(val)
			}
		}
		dataRows = append(dataRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历数据失败: %w", err)
	}

	took := time.Since(startTime).Milliseconds()

	// total等于实际返回的行数
	totalCount := len(dataRows)

	return &model.QueryResult{
		Columns: columns,
		Rows:    dataRows,
		Total:   totalCount,
		Took:    took,
		DBName:  dbConfig.Database,
	}, nil
}
