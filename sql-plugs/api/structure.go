package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sql-plugs/common"
	"sql-plugs/model"
	"strings"
	"time"
)

// StructureHandler 处理数据库结构查询
func StructureHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许POST请求
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	// 解析请求
	var req model.StructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 根据类型调用不同处理函数
	switch req.Type {
	case "db":
		// 获取数据库列表
		handleDatabaseList(w, r)
	case "tb":
		// 获取表列表或表结构
		handleTableInfo(w, r, req.Op)
	default:
		common.ErrorWithCode(w, http.StatusBadRequest, "不支持的查询类型: "+req.Type)
	}
}

// DatabaseListResponse 数据库列表响应（含所有库元数据）
type DatabaseListResponse struct {
	Databases []string                    `json:"databases"` // 数据库名列表
	Metadata  *model.AllDatabasesMetadata `json:"metadata"`  // 所有库的完整元数据
}

// handleDatabaseList 获取数据库列表（同时返回所有库的元数据）
func handleDatabaseList(w http.ResponseWriter, r *http.Request) {
	db, err := common.GetDB()
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "数据库连接失败: "+err.Error())
		return
	}

	// 查询所有数据库
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "查询数据库列表失败: "+err.Error())
		return
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		// 过滤系统数据库
		if dbName != "information_schema" && dbName != "mysql" && dbName != "performance_schema" && dbName != "sys" {
			databases = append(databases, dbName)
		}
	}

	// 同时获取所有库的完整元数据
	metadata, err := fetchAllDatabasesMetadata()
	if err != nil {
		common.Logger.Warnf("获取跨库元数据失败: %v", err)
		metadata = nil
	}

	response := DatabaseListResponse{
		Databases: databases,
		Metadata:  metadata,
	}

	common.Logger.Infof("查询数据库列表成功 - 数量: %d", len(databases))
	common.SuccessWithMessage(w, "查询成功", response)
}

// handleTableInfo 获取表列表或表结构
func handleTableInfo(w http.ResponseWriter, r *http.Request, op map[string]interface{}) {
	// 获取数据库名
	dbName, ok := op["dbName"].(string)
	if !ok || dbName == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "缺少数据库名称 dbName")
		return
	}

	// 检查是否指定了表名
	tbName, hasTbName := op["tbName"].(string)

	if !hasTbName || tbName == "" {
		// 没有表名，返回表列表
		handleTableList(w, dbName)
	} else {
		// 有表名，返回表结构
		handleTableStructure(w, dbName, tbName)
	}
}

// handleTableList 获取表列表
func handleTableList(w http.ResponseWriter, dbName string) {
	db, err := common.GetDB()
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "数据库连接失败: "+err.Error())
		return
	}

	// 查询指定数据库的所有表
	query := fmt.Sprintf("SHOW TABLES FROM `%s`", dbName)
	rows, err := db.Query(query)
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "查询表列表失败: "+err.Error())
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tbName string
		if err := rows.Scan(&tbName); err != nil {
			continue
		}
		tables = append(tables, tbName)
	}

	common.Logger.Infof("查询表列表成功 - 数据库: %s, 表数量: %d", dbName, len(tables))
	common.SuccessWithMessage(w, "查询成功", tables)
}

// handleTableStructure 获取表结构详情
func handleTableStructure(w http.ResponseWriter, dbName, tbName string) {
	db, err := common.GetDB()
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "数据库连接失败: "+err.Error())
		return
	}

	var result model.TableStructure
	result.Name = tbName

	// 1. 获取表字段信息
	columns, err := getTableColumns(db, dbName, tbName)
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取表字段失败: "+err.Error())
		return
	}
	result.Columns = columns

	// 2. 获取表索引信息
	indexes, err := getTableIndexes(db, dbName, tbName)
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取表索引失败: "+err.Error())
		return
	}
	result.Indexes = indexes

	// 3. 获取建表语句
	createSQL, err := getCreateTableSQL(db, dbName, tbName)
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取建表语句失败: "+err.Error())
		return
	}
	result.CreateSQL = createSQL

	// 4. 获取表元信息（引擎、字符集、注释、创建时间）
	if err := getTableMetadata(db, dbName, tbName, &result); err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取表元信息失败: "+err.Error())
		return
	}

	// 5. 获取预览数据（前20条记录）
	previewData, err := getTablePreviewData(db, dbName, tbName)
	if err != nil {
		common.Logger.Warnf("获取表预览数据失败: %v", err)
		// 预览数据失败不影响整体返回，设置为nil
		result.PreviewData = nil
	} else {
		result.PreviewData = previewData
	}

	common.Logger.Infof("查询表结构成功 - 数据库: %s, 表: %s, 字段数: %d, 索引数: %d, 预览数据行数: %d",
		dbName, tbName, len(result.Columns), len(result.Indexes),
		func() int {
			if result.PreviewData != nil {
				return result.PreviewData.Total
			} else {
				return 0
			}
		}())
	common.Success(w, result)
}

// convertToString 将interface{}转换为字符串，特别处理[]byte类型
func convertToString(val interface{}) string {
	if val == nil {
		return ""
	}
	if byteVal, ok := val.([]byte); ok {
		return string(byteVal)
	}
	return fmt.Sprintf("%v", val)
}

// getTableColumns 获取表字段信息
func getTableColumns(db *sql.DB, dbName, tbName string) ([]model.TableColumn, error) {
	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`.`%s`", dbName, tbName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []model.TableColumn
	for rows.Next() {
		var col model.TableColumn
		var collation, privileges interface{}
		var defaultVal interface{}

		err := rows.Scan(
			&col.Field,
			&col.Type,
			&collation,
			&col.Null,
			&col.Key,
			&defaultVal,
			&col.Extra,
			&privileges,
			&col.Comment,
		)
		if err != nil {
			continue
		}

		// 处理默认值（可能为 NULL）
		if defaultVal != nil {
			defStr := convertToString(defaultVal)
			col.Default = &defStr
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// getTableIndexes 获取表索引信息
func getTableIndexes(db *sql.DB, dbName, tbName string) ([]model.TableIndex, error) {
	query := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", dbName, tbName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列信息，动态扫描
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	common.Logger.Infof("SHOW INDEX 返回字段数: %d, 字段名: %v", len(columns), columns)

	// 用map来聚合同一索引的多个字段
	indexMap := make(map[string]*model.TableIndex)

	for rows.Next() {
		// 创建动态数量的接收变量
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			common.Logger.Errorf("扫描索引行失败: %v", err)
			continue
		}

		// 按照SHOW INDEX的标准字段顺序解析
		// Table, Non_unique, Key_name, Seq_in_index, Column_name, Collation, Cardinality, Sub_part, Packed, Null, Index_type, Comment, Index_comment, [Visible], [Expression]
		if len(values) < 13 {
			common.Logger.Errorf("索引字段数量不足: %d", len(values))
			continue
		}

		// 从values中提取字段（索引: 0=Table, 1=Non_unique, 2=Key_name, 3=Seq_in_index, 4=Column_name, 10=Index_type, 12=Index_comment）
		idxName := convertToString(values[2])
		idxType := convertToString(values[10])
		colName := convertToString(values[4])
		isUnique := convertToString(values[1]) == "0"
		idxComment := ""
		if len(values) > 12 {
			idxComment = convertToString(values[12])
		}

		common.Logger.Infof("解析索引: name=%s, type=%s, column=%s, unique=%v", idxName, idxType, colName, isUnique)

		if idx, exists := indexMap[idxName]; exists {
			// 索引已存在，添加字段
			idx.Columns = append(idx.Columns, colName)
		} else {
			// 新索引
			indexMap[idxName] = &model.TableIndex{
				Name:    idxName,
				Type:    idxType,
				Columns: []string{colName},
				Unique:  isUnique,
				Comment: idxComment,
			}
		}
	}

	// 转map为slice
	indexes := make([]model.TableIndex, 0)
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	common.Logger.Infof("获取表索引完成 - 数据库: %s, 表: %s, 索引数: %d", dbName, tbName, len(indexes))

	return indexes, nil
}

// getCreateTableSQL 获取建表语句
func getCreateTableSQL(db *sql.DB, dbName, tbName string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", dbName, tbName)
	var table, createSQL string
	err := db.QueryRow(query).Scan(&table, &createSQL)
	if err != nil {
		return "", err
	}
	return createSQL, nil
}

// getTableMetadata 获取表元信息（引擎、字符集、注释、创建时间）
func getTableMetadata(db *sql.DB, dbName, tbName string, result *model.TableStructure) error {
	query := `
		SELECT 
			ENGINE,
			TABLE_COLLATION,
			TABLE_COMMENT,
			CREATE_TIME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	`

	var createTime interface{}
	err := db.QueryRow(query, dbName, tbName).Scan(
		&result.Engine,
		&result.Collation,
		&result.Comment,
		&createTime,
	)
	if err != nil {
		return err
	}

	// 处理创建时间
	if createTime != nil {
		result.CreateTime = fmt.Sprintf("%v", createTime)
		// 去掉MySQL时间的小数部分
		if idx := strings.Index(result.CreateTime, "."); idx != -1 {
			result.CreateTime = result.CreateTime[:idx]
		}
	}

	return nil
}

// getTablePreviewData 获取表预览数据（前20条记录）
func getTablePreviewData(db *sql.DB, dbName, tbName string) (*model.QueryResult, error) {
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT 20", dbName, tbName)

	startTime := time.Now()
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 读取数据行
	var dataRows [][]interface{}
	for rows.Next() {
		// 使用 sql.RawBytes 接收原始数据，避免类型转换
		values := make([]sql.RawBytes, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		// 直接使用原始字节数据，保持数据库原始格式
		row := make([]interface{}, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = nil
			} else {
				// 直接转换为字符串，保持数据库原始格式
				row[i] = string(val)
			}
		}
		dataRows = append(dataRows, row)
	}

	took := time.Since(startTime).Milliseconds()

	return &model.QueryResult{
		Columns: columns,
		Rows:    dataRows,
		Total:   len(dataRows),
		Took:    took,
		DBName:  dbName,
	}, nil
}
