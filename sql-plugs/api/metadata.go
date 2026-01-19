package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sql-plugs/common"
	"sql-plugs/model"
	"strings"
	"sync"
	"time"
)

// 元数据缓存
var (
	metadataCache     = make(map[string]*cachedMetadata)
	allMetadataCache  *cachedAllMetadata // 跨库缓存
	metadataCacheLock sync.RWMutex
	cacheTTL          = 10 * time.Minute // 缓存过期时间
)

type cachedMetadata struct {
	data      *model.DatabaseMetadata
	cachedAt  time.Time
	fetching  bool       // 是否正在获取中（防止并发重复拉取）
	fetchLock sync.Mutex // 获取锁
}

type cachedAllMetadata struct {
	data     *model.AllDatabasesMetadata
	cachedAt time.Time
}

// MetadataHandler 处理元数据查询请求
// POST /api/sql/metadata
// 请求体: {"dbName": "xxx", "refresh": false}
// dbName为空时返回所有库的元数据
func MetadataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	var req model.MetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// dbName为空时，跨库查询所有数据库
	if req.DBName == "" {
		handleAllDatabasesMetadata(w, req.Refresh)
		return
	}

	// 验证数据库名（放宽到64字符，MySQL标准）
	if !isValidDBName(req.DBName) {
		common.ErrorWithCode(w, http.StatusBadRequest, "无效的数据库名称")
		return
	}

	// 尝试从缓存获取
	if !req.Refresh {
		if cached := getFromCache(req.DBName); cached != nil {
			cached.FromCache = true
			common.Logger.Infof("元数据缓存命中 - 库: %s, 缓存时间: %s", req.DBName, cached.CachedAt)
			common.Success(w, cached)
			return
		}
	}

	// 采集元数据
	metadata, err := fetchDatabaseMetadata(req.DBName)
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取元数据失败: "+err.Error())
		return
	}

	// 存入缓存
	saveToCache(req.DBName, metadata)

	common.Logger.Infof("元数据采集完成 - 库: %s, 表: %d, 视图: %d, 耗时: %dms",
		req.DBName, len(metadata.Tables), len(metadata.Views), metadata.Took)
	common.Success(w, metadata)
}

// handleAllDatabasesMetadata 处理跨库元数据查询
func handleAllDatabasesMetadata(w http.ResponseWriter, refresh bool) {
	// 尝试从缓存获取
	if !refresh {
		metadataCacheLock.RLock()
		if allMetadataCache != nil && time.Since(allMetadataCache.cachedAt) < cacheTTL {
			result := *allMetadataCache.data
			result.FromCache = true
			metadataCacheLock.RUnlock()
			common.Logger.Infof("跨库元数据缓存命中 - 缓存时间: %s", result.CachedAt)
			common.Success(w, result)
			return
		}
		metadataCacheLock.RUnlock()
	}

	// 采集所有库元数据
	metadata, err := fetchAllDatabasesMetadata()
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取跨库元数据失败: "+err.Error())
		return
	}

	// 存入缓存
	metadataCacheLock.Lock()
	allMetadataCache = &cachedAllMetadata{
		data:     metadata,
		cachedAt: time.Now(),
	}
	metadataCacheLock.Unlock()

	common.Logger.Infof("跨库元数据采集完成 - 库数量: %d, 耗时: %dms", metadata.Total, metadata.Took)
	common.Success(w, metadata)
}

// isValidDBName 验证数据库名是否合法（放宽到64字符）
func isValidDBName(dbName string) bool {
	if len(dbName) == 0 || len(dbName) > 64 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, dbName)
	return matched
}

// getFromCache 从缓存获取元数据
func getFromCache(dbName string) *model.DatabaseMetadata {
	metadataCacheLock.RLock()
	defer metadataCacheLock.RUnlock()

	cached, exists := metadataCache[dbName]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Since(cached.cachedAt) > cacheTTL {
		return nil
	}

	// 返回副本
	result := *cached.data
	return &result
}

// saveToCache 保存元数据到缓存
func saveToCache(dbName string, metadata *model.DatabaseMetadata) {
	metadataCacheLock.Lock()
	defer metadataCacheLock.Unlock()

	metadataCache[dbName] = &cachedMetadata{
		data:     metadata,
		cachedAt: time.Now(),
	}
}

// fetchDatabaseMetadata 采集数据库全量元数据
func fetchDatabaseMetadata(dbName string) (*model.DatabaseMetadata, error) {
	startTime := time.Now()

	db, err := common.GetDB()
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	metadata := &model.DatabaseMetadata{
		DBName:   dbName,
		Warnings: make([]string, 0),
	}

	// 1. 获取表列表（含基础信息）
	tables, err := fetchTables(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取表列表失败: "+err.Error())
	} else {
		metadata.Tables = tables
	}

	// 2. 获取视图列表
	views, err := fetchViews(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取视图列表失败: "+err.Error())
	} else {
		metadata.Views = views
	}

	// 3. 获取所有表的字段信息
	columnMap, err := fetchAllColumns(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取字段信息失败: "+err.Error())
	}

	// 4. 获取所有约束（主键/外键）
	pkMap, fkMap, err := fetchAllConstraints(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取约束信息失败: "+err.Error())
	}

	// 5. 获取所有索引
	indexMap, err := fetchAllIndexes(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取索引信息失败: "+err.Error())
	}

	// 6. 将字段/约束/索引关联到表
	for i := range metadata.Tables {
		tableName := metadata.Tables[i].Name
		if cols, ok := columnMap[tableName]; ok {
			metadata.Tables[i].Columns = cols
		}
		if pk, ok := pkMap[tableName]; ok {
			metadata.Tables[i].PrimaryKey = pk
		}
		if fks, ok := fkMap[tableName]; ok {
			metadata.Tables[i].ForeignKeys = fks
		}
		if idxs, ok := indexMap[tableName]; ok {
			metadata.Tables[i].Indexes = idxs
		}
	}

	// 7. 获取触发器
	triggers, err := fetchTriggers(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取触发器失败: "+err.Error())
	} else {
		metadata.Triggers = triggers
	}

	// 8. 获取存储过程/函数
	routines, err := fetchRoutines(db, dbName)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取存储过程/函数失败: "+err.Error())
	} else {
		metadata.Routines = routines
	}

	// 9. 获取UDF（可能需要权限）
	udfs, err := fetchUDFs(db)
	if err != nil {
		metadata.Warnings = append(metadata.Warnings, "获取UDF失败（可能权限不足）: "+err.Error())
	} else {
		metadata.UDFs = udfs
	}

	metadata.Took = time.Since(startTime).Milliseconds()
	metadata.CachedAt = time.Now().Format("2006-01-02 15:04:05")
	metadata.FromCache = false

	return metadata, nil
}

// fetchTables 获取表列表及基础信息
func fetchTables(db *sql.DB, dbName string) ([]model.TableMeta, error) {
	query := `
		SELECT 
			TABLE_NAME,
			IFNULL(TABLE_COMMENT, ''),
			IFNULL(ENGINE, ''),
			IFNULL(TABLE_COLLATION, ''),
			IFNULL(TABLE_ROWS, 0),
			IFNULL(DATA_LENGTH, 0),
			IFNULL(CREATE_TIME, ''),
			IFNULL(UPDATE_TIME, '')
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []model.TableMeta
	for rows.Next() {
		var t model.TableMeta
		var createTime, updateTime interface{}
		err := rows.Scan(&t.Name, &t.Comment, &t.Engine, &t.Collation,
			&t.RowCount, &t.DataLength, &createTime, &updateTime)
		if err != nil {
			continue
		}
		t.CreateTime = formatTime(createTime)
		t.UpdateTime = formatTime(updateTime)
		t.Columns = make([]model.ColumnMeta, 0)
		t.ForeignKeys = make([]model.ForeignKeyMeta, 0)
		t.Indexes = make([]model.IndexMeta, 0)
		tables = append(tables, t)
	}

	return tables, nil
}

// fetchViews 获取视图列表
func fetchViews(db *sql.DB, dbName string) ([]model.ViewMeta, error) {
	query := `
		SELECT 
			TABLE_NAME,
			IFNULL(VIEW_DEFINITION, ''),
			IFNULL(DEFINER, ''),
			CASE WHEN IS_UPDATABLE = 'YES' THEN 1 ELSE 0 END
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []model.ViewMeta
	for rows.Next() {
		var v model.ViewMeta
		var updatable int
		err := rows.Scan(&v.Name, &v.Definition, &v.Definer, &updatable)
		if err != nil {
			continue
		}
		v.Updatable = updatable == 1
		views = append(views, v)
	}

	return views, nil
}

// fetchAllColumns 获取所有表的字段信息
func fetchAllColumns(db *sql.DB, dbName string) (map[string][]model.ColumnMeta, error) {
	query := `
		SELECT 
			TABLE_NAME,
			COLUMN_NAME,
			ORDINAL_POSITION,
			DATA_TYPE,
			COLUMN_TYPE,
			CASE WHEN IS_NULLABLE = 'YES' THEN 1 ELSE 0 END,
			COLUMN_DEFAULT,
			CASE WHEN COLUMN_KEY = 'PRI' THEN 1 ELSE 0 END,
			CASE WHEN EXTRA LIKE '%auto_increment%' THEN 1 ELSE 0 END,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			IFNULL(COLUMN_KEY, ''),
			IFNULL(EXTRA, ''),
			IFNULL(COLUMN_COMMENT, '')
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, ORDINAL_POSITION
	`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]model.ColumnMeta)
	for rows.Next() {
		var tableName string
		var c model.ColumnMeta
		var nullable, isPK, isAutoIncr int
		var defaultVal, charMaxLen, numPrecision, numScale interface{}

		err := rows.Scan(&tableName, &c.Name, &c.OrdinalPos, &c.DataType, &c.ColumnType,
			&nullable, &defaultVal, &isPK, &isAutoIncr,
			&charMaxLen, &numPrecision, &numScale,
			&c.CharacterSet, &c.Collation, &c.ColumnKey, &c.Extra, &c.Comment)
		if err != nil {
			continue
		}

		c.Nullable = nullable == 1
		c.IsPrimaryKey = isPK == 1
		c.IsAutoIncr = isAutoIncr == 1

		if defaultVal != nil {
			s := formatValue(defaultVal)
			c.DefaultValue = &s
		}
		if charMaxLen != nil {
			if v, ok := toInt64(charMaxLen); ok {
				c.CharMaxLen = &v
			}
		}
		if numPrecision != nil {
			if v, ok := toInt64(numPrecision); ok {
				c.NumPrecision = &v
			}
		}
		if numScale != nil {
			if v, ok := toInt64(numScale); ok {
				c.NumScale = &v
			}
		}

		result[tableName] = append(result[tableName], c)
	}

	return result, nil
}

// fetchAllConstraints 获取所有表的主键和外键约束
func fetchAllConstraints(db *sql.DB, dbName string) (map[string]*model.PrimaryKeyMeta, map[string][]model.ForeignKeyMeta, error) {
	pkMap := make(map[string]*model.PrimaryKeyMeta)
	fkMap := make(map[string][]model.ForeignKeyMeta)

	// 获取主键
	pkQuery := `
		SELECT 
			tc.TABLE_NAME,
			tc.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.ORDINAL_POSITION
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.KEY_COLUMN_USAGE kcu 
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME 
			AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
		WHERE tc.TABLE_SCHEMA = ? AND tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		ORDER BY tc.TABLE_NAME, kcu.ORDINAL_POSITION
	`

	rows, err := db.Query(pkQuery, dbName)
	if err != nil {
		return pkMap, fkMap, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, constraintName, columnName string
		var ordinal int
		if err := rows.Scan(&tableName, &constraintName, &columnName, &ordinal); err != nil {
			continue
		}
		if _, exists := pkMap[tableName]; !exists {
			pkMap[tableName] = &model.PrimaryKeyMeta{
				Name:    constraintName,
				Columns: make([]string, 0),
			}
		}
		pkMap[tableName].Columns = append(pkMap[tableName].Columns, columnName)
	}

	// 获取外键
	fkQuery := `
		SELECT 
			tc.TABLE_NAME,
			tc.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			rc.UPDATE_RULE,
			rc.DELETE_RULE,
			kcu.ORDINAL_POSITION
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.KEY_COLUMN_USAGE kcu 
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME 
			AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
		JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
			ON tc.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
			AND tc.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE tc.TABLE_SCHEMA = ? AND tc.CONSTRAINT_TYPE = 'FOREIGN KEY'
		ORDER BY tc.TABLE_NAME, tc.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
	`

	fkRows, err := db.Query(fkQuery, dbName)
	if err != nil {
		return pkMap, fkMap, err
	}
	defer fkRows.Close()

	// 用于聚合同一外键的多列
	fkTemp := make(map[string]map[string]*model.ForeignKeyMeta)
	for fkRows.Next() {
		var tableName, constraintName, colName, refTable, refCol, onUpdate, onDelete string
		var ordinal int
		if err := fkRows.Scan(&tableName, &constraintName, &colName, &refTable, &refCol, &onUpdate, &onDelete, &ordinal); err != nil {
			continue
		}
		if _, ok := fkTemp[tableName]; !ok {
			fkTemp[tableName] = make(map[string]*model.ForeignKeyMeta)
		}
		if _, ok := fkTemp[tableName][constraintName]; !ok {
			fkTemp[tableName][constraintName] = &model.ForeignKeyMeta{
				Name:       constraintName,
				Columns:    make([]string, 0),
				RefTable:   refTable,
				RefColumns: make([]string, 0),
				OnUpdate:   onUpdate,
				OnDelete:   onDelete,
			}
		}
		fkTemp[tableName][constraintName].Columns = append(fkTemp[tableName][constraintName].Columns, colName)
		fkTemp[tableName][constraintName].RefColumns = append(fkTemp[tableName][constraintName].RefColumns, refCol)
	}

	// 转换为slice
	for tableName, constraints := range fkTemp {
		for _, fk := range constraints {
			fkMap[tableName] = append(fkMap[tableName], *fk)
		}
	}

	return pkMap, fkMap, nil
}

// fetchAllIndexes 获取所有表的索引信息
func fetchAllIndexes(db *sql.DB, dbName string) (map[string][]model.IndexMeta, error) {
	query := `
		SELECT 
			TABLE_NAME,
			INDEX_NAME,
			COLUMN_NAME,
			SEQ_IN_INDEX,
			CASE WHEN NON_UNIQUE = 0 THEN 1 ELSE 0 END,
			INDEX_TYPE,
			IFNULL(INDEX_COMMENT, '')
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX
	`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 用于聚合同一索引的多列
	idxTemp := make(map[string]map[string]*model.IndexMeta)
	for rows.Next() {
		var tableName, indexName, colName, indexType, comment string
		var seqInIndex, isUnique int
		if err := rows.Scan(&tableName, &indexName, &colName, &seqInIndex, &isUnique, &indexType, &comment); err != nil {
			continue
		}
		if _, ok := idxTemp[tableName]; !ok {
			idxTemp[tableName] = make(map[string]*model.IndexMeta)
		}
		if _, ok := idxTemp[tableName][indexName]; !ok {
			idxTemp[tableName][indexName] = &model.IndexMeta{
				Name:      indexName,
				Columns:   make([]string, 0),
				IndexType: indexType,
				IsUnique:  isUnique == 1,
				IsPrimary: indexName == "PRIMARY",
				Comment:   comment,
			}
		}
		idxTemp[tableName][indexName].Columns = append(idxTemp[tableName][indexName].Columns, colName)
	}

	// 转换为slice
	result := make(map[string][]model.IndexMeta)
	for tableName, indexes := range idxTemp {
		for _, idx := range indexes {
			result[tableName] = append(result[tableName], *idx)
		}
	}

	return result, nil
}

// fetchTriggers 获取触发器列表
func fetchTriggers(db *sql.DB, dbName string) ([]model.TriggerMeta, error) {
	query := `
		SELECT 
			TRIGGER_NAME,
			EVENT_OBJECT_TABLE,
			EVENT_MANIPULATION,
			ACTION_TIMING,
			ACTION_STATEMENT,
			IFNULL(DEFINER, ''),
			IFNULL(CREATED, '')
		FROM information_schema.TRIGGERS
		WHERE TRIGGER_SCHEMA = ?
		ORDER BY TRIGGER_NAME
	`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []model.TriggerMeta
	for rows.Next() {
		var t model.TriggerMeta
		var created interface{}
		err := rows.Scan(&t.Name, &t.Table, &t.Event, &t.Timing, &t.Statement, &t.Definer, &created)
		if err != nil {
			continue
		}
		t.Created = formatTime(created)
		triggers = append(triggers, t)
	}

	return triggers, nil
}

// fetchRoutines 获取存储过程/函数列表
func fetchRoutines(db *sql.DB, dbName string) ([]model.RoutineMeta, error) {
	query := `
		SELECT 
			ROUTINE_NAME,
			ROUTINE_TYPE,
			IFNULL(DEFINER, ''),
			IFNULL(DTD_IDENTIFIER, ''),
			IFNULL(ROUTINE_DEFINITION, ''),
			IFNULL(CREATED, ''),
			IFNULL(LAST_ALTERED, ''),
			IFNULL(ROUTINE_COMMENT, '')
		FROM information_schema.ROUTINES
		WHERE ROUTINE_SCHEMA = ?
		ORDER BY ROUTINE_TYPE, ROUTINE_NAME
	`

	rows, err := db.Query(query, dbName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	routineMap := make(map[string]*model.RoutineMeta)
	var routineOrder []string

	for rows.Next() {
		var r model.RoutineMeta
		var created, modified interface{}
		err := rows.Scan(&r.Name, &r.Type, &r.Definer, &r.DataType, &r.Definition, &created, &modified, &r.Comment)
		if err != nil {
			continue
		}
		r.Created = formatTime(created)
		r.Modified = formatTime(modified)
		r.Parameters = make([]model.ParameterMeta, 0)

		key := r.Type + ":" + r.Name
		routineMap[key] = &r
		routineOrder = append(routineOrder, key)
	}

	// 获取参数信息
	paramQuery := `
		SELECT 
			SPECIFIC_NAME,
			ROUTINE_TYPE,
			IFNULL(PARAMETER_NAME, ''),
			IFNULL(PARAMETER_MODE, ''),
			IFNULL(DTD_IDENTIFIER, ''),
			ORDINAL_POSITION
		FROM information_schema.PARAMETERS
		WHERE SPECIFIC_SCHEMA = ? AND ORDINAL_POSITION > 0
		ORDER BY SPECIFIC_NAME, ORDINAL_POSITION
	`

	paramRows, err := db.Query(paramQuery, dbName)
	if err == nil {
		defer paramRows.Close()
		for paramRows.Next() {
			var routineName, routineType, paramName, paramMode, dataType string
			var ordinal int
			if err := paramRows.Scan(&routineName, &routineType, &paramName, &paramMode, &dataType, &ordinal); err != nil {
				continue
			}
			key := routineType + ":" + routineName
			if r, ok := routineMap[key]; ok {
				r.Parameters = append(r.Parameters, model.ParameterMeta{
					Name:       paramName,
					Mode:       paramMode,
					DataType:   dataType,
					OrdinalPos: ordinal,
				})
			}
		}
	}

	// 按原始顺序返回
	var routines []model.RoutineMeta
	for _, key := range routineOrder {
		if r, ok := routineMap[key]; ok {
			routines = append(routines, *r)
		}
	}

	return routines, nil
}

// fetchUDFs 获取用户自定义函数（需要mysql.func权限）
func fetchUDFs(db *sql.DB) ([]model.UDFMeta, error) {
	query := `SELECT name, ret, type, dl FROM mysql.func ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		// 权限不足时返回空列表
		if strings.Contains(err.Error(), "denied") || strings.Contains(err.Error(), "SELECT command denied") {
			return []model.UDFMeta{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var udfs []model.UDFMeta
	for rows.Next() {
		var u model.UDFMeta
		var retType int
		err := rows.Scan(&u.Name, &retType, &u.Type, &u.Library)
		if err != nil {
			continue
		}
		// MySQL mysql.func ret: 0=string, 1=real, 2=int
		switch retType {
		case 0:
			u.ReturnType = "STRING"
		case 1:
			u.ReturnType = "REAL"
		case 2:
			u.ReturnType = "INTEGER"
		default:
			u.ReturnType = "UNKNOWN"
		}
		udfs = append(udfs, u)
	}

	return udfs, nil
}

// 辅助函数
func formatTime(val interface{}) string {
	if val == nil {
		return ""
	}
	s := fmt.Sprintf("%v", val)
	// 去掉时间的小数部分
	if idx := strings.Index(s, "."); idx != -1 {
		s = s[:idx]
	}
	return s
}

func formatValue(val interface{}) string {
	if val == nil {
		return ""
	}
	if b, ok := val.([]byte); ok {
		return string(b)
	}
	return fmt.Sprintf("%v", val)
}

func toInt64(val interface{}) (int64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case []byte:
		var i int64
		fmt.Sscanf(string(v), "%d", &i)
		return i, true
	}
	return 0, false
}

// fetchAllDatabasesMetadata 跨库采集所有数据库元数据
func fetchAllDatabasesMetadata() (*model.AllDatabasesMetadata, error) {
	startTime := time.Now()

	db, err := common.GetDB()
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	result := &model.AllDatabasesMetadata{
		Databases: make([]model.DatabaseMetadata, 0),
		Warnings:  make([]string, 0),
	}

	// 1. 获取所有非系统数据库
	dbNames, err := fetchDatabaseList(db)
	if err != nil {
		return nil, fmt.Errorf("获取数据库列表失败: %w", err)
	}

	// 2. 一次性获取所有库的表信息（跨库查询）
	allTables, err := fetchAllTablesAcrossDBs(db, dbNames)
	if err != nil {
		result.Warnings = append(result.Warnings, "获取表列表失败: "+err.Error())
	}

	// 3. 一次性获取所有库的字段信息
	allColumns, err := fetchAllColumnsAcrossDBs(db, dbNames)
	if err != nil {
		result.Warnings = append(result.Warnings, "获取字段信息失败: "+err.Error())
	}

	// 4. 一次性获取所有库的索引信息
	allIndexes, err := fetchAllIndexesAcrossDBs(db, dbNames)
	if err != nil {
		result.Warnings = append(result.Warnings, "获取索引信息失败: "+err.Error())
	}

	// 5. 一次性获取所有库的约束信息
	allPKs, allFKs, err := fetchAllConstraintsAcrossDBs(db, dbNames)
	if err != nil {
		result.Warnings = append(result.Warnings, "获取约束信息失败: "+err.Error())
	}

	// 6. 组装各库的元数据
	for _, dbName := range dbNames {
		dbMeta := model.DatabaseMetadata{
			DBName:   dbName,
			Tables:   make([]model.TableMeta, 0),
			Views:    make([]model.ViewMeta, 0),
			Warnings: make([]string, 0),
		}

		// 关联表
		if tables, ok := allTables[dbName]; ok {
			for i := range tables {
				tableName := tables[i].Name
				// 关联字段
				if cols, ok := allColumns[dbName+"."+tableName]; ok {
					tables[i].Columns = cols
				}
				// 关联索引
				if idxs, ok := allIndexes[dbName+"."+tableName]; ok {
					tables[i].Indexes = idxs
				}
				// 关联主键
				if pk, ok := allPKs[dbName+"."+tableName]; ok {
					tables[i].PrimaryKey = pk
				}
				// 关联外键
				if fks, ok := allFKs[dbName+"."+tableName]; ok {
					tables[i].ForeignKeys = fks
				}
			}
			dbMeta.Tables = tables
		}

		result.Databases = append(result.Databases, dbMeta)
	}

	result.Total = len(result.Databases)
	result.CachedAt = time.Now().Format("2006-01-02 15:04:05")
	result.FromCache = false
	result.Took = time.Since(startTime).Milliseconds()

	return result, nil
}

// fetchDatabaseList 获取非系统数据库列表
func fetchDatabaseList(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		if !systemDBs[dbName] {
			databases = append(databases, dbName)
		}
	}
	return databases, nil
}

// fetchAllTablesAcrossDBs 跨库获取所有表
func fetchAllTablesAcrossDBs(db *sql.DB, dbNames []string) (map[string][]model.TableMeta, error) {
	query := `
		SELECT 
			TABLE_SCHEMA,
			TABLE_NAME,
			IFNULL(TABLE_COMMENT, ''),
			IFNULL(ENGINE, ''),
			IFNULL(TABLE_COLLATION, ''),
			IFNULL(TABLE_ROWS, 0),
			IFNULL(DATA_LENGTH, 0),
			IFNULL(CREATE_TIME, ''),
			IFNULL(UPDATE_TIME, '')
		FROM information_schema.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE'
		  AND TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]model.TableMeta)
	for rows.Next() {
		var dbName string
		var t model.TableMeta
		var createTime, updateTime interface{}
		err := rows.Scan(&dbName, &t.Name, &t.Comment, &t.Engine, &t.Collation,
			&t.RowCount, &t.DataLength, &createTime, &updateTime)
		if err != nil {
			continue
		}
		t.CreateTime = formatTime(createTime)
		t.UpdateTime = formatTime(updateTime)
		t.Columns = make([]model.ColumnMeta, 0)
		t.ForeignKeys = make([]model.ForeignKeyMeta, 0)
		t.Indexes = make([]model.IndexMeta, 0)
		result[dbName] = append(result[dbName], t)
	}

	return result, nil
}

// fetchAllColumnsAcrossDBs 跨库获取所有字段
func fetchAllColumnsAcrossDBs(db *sql.DB, dbNames []string) (map[string][]model.ColumnMeta, error) {
	query := `
		SELECT 
			TABLE_SCHEMA,
			TABLE_NAME,
			COLUMN_NAME,
			ORDINAL_POSITION,
			DATA_TYPE,
			COLUMN_TYPE,
			CASE WHEN IS_NULLABLE = 'YES' THEN 1 ELSE 0 END,
			COLUMN_DEFAULT,
			CASE WHEN COLUMN_KEY = 'PRI' THEN 1 ELSE 0 END,
			CASE WHEN EXTRA LIKE '%auto_increment%' THEN 1 ELSE 0 END,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			IFNULL(CHARACTER_SET_NAME, ''),
			IFNULL(COLLATION_NAME, ''),
			IFNULL(COLUMN_KEY, ''),
			IFNULL(EXTRA, ''),
			IFNULL(COLUMN_COMMENT, '')
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]model.ColumnMeta)
	for rows.Next() {
		var dbName, tableName string
		var c model.ColumnMeta
		var nullable, isPK, isAutoIncr int
		var defaultVal, charMaxLen, numPrecision, numScale interface{}

		err := rows.Scan(&dbName, &tableName, &c.Name, &c.OrdinalPos, &c.DataType, &c.ColumnType,
			&nullable, &defaultVal, &isPK, &isAutoIncr,
			&charMaxLen, &numPrecision, &numScale,
			&c.CharacterSet, &c.Collation, &c.ColumnKey, &c.Extra, &c.Comment)
		if err != nil {
			continue
		}

		c.Nullable = nullable == 1
		c.IsPrimaryKey = isPK == 1
		c.IsAutoIncr = isAutoIncr == 1

		if defaultVal != nil {
			s := formatValue(defaultVal)
			c.DefaultValue = &s
		}
		if charMaxLen != nil {
			if v, ok := toInt64(charMaxLen); ok {
				c.CharMaxLen = &v
			}
		}
		if numPrecision != nil {
			if v, ok := toInt64(numPrecision); ok {
				c.NumPrecision = &v
			}
		}
		if numScale != nil {
			if v, ok := toInt64(numScale); ok {
				c.NumScale = &v
			}
		}

		key := dbName + "." + tableName
		result[key] = append(result[key], c)
	}

	return result, nil
}

// fetchAllIndexesAcrossDBs 跨库获取所有索引
func fetchAllIndexesAcrossDBs(db *sql.DB, dbNames []string) (map[string][]model.IndexMeta, error) {
	query := `
		SELECT 
			TABLE_SCHEMA,
			TABLE_NAME,
			INDEX_NAME,
			COLUMN_NAME,
			SEQ_IN_INDEX,
			CASE WHEN NON_UNIQUE = 0 THEN 1 ELSE 0 END,
			INDEX_TYPE,
			IFNULL(INDEX_COMMENT, '')
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 用于聚合同一索引的多列
	idxTemp := make(map[string]map[string]*model.IndexMeta)
	for rows.Next() {
		var dbName, tableName, indexName, colName, indexType, comment string
		var seqInIndex, isUnique int
		if err := rows.Scan(&dbName, &tableName, &indexName, &colName, &seqInIndex, &isUnique, &indexType, &comment); err != nil {
			continue
		}
		key := dbName + "." + tableName
		if _, ok := idxTemp[key]; !ok {
			idxTemp[key] = make(map[string]*model.IndexMeta)
		}
		if _, ok := idxTemp[key][indexName]; !ok {
			idxTemp[key][indexName] = &model.IndexMeta{
				Name:      indexName,
				Columns:   make([]string, 0),
				IndexType: indexType,
				IsUnique:  isUnique == 1,
				IsPrimary: indexName == "PRIMARY",
				Comment:   comment,
			}
		}
		idxTemp[key][indexName].Columns = append(idxTemp[key][indexName].Columns, colName)
	}

	// 转换为slice
	result := make(map[string][]model.IndexMeta)
	for key, indexes := range idxTemp {
		for _, idx := range indexes {
			result[key] = append(result[key], *idx)
		}
	}

	return result, nil
}

// fetchAllConstraintsAcrossDBs 跨库获取所有约束（主键/外键）
func fetchAllConstraintsAcrossDBs(db *sql.DB, dbNames []string) (map[string]*model.PrimaryKeyMeta, map[string][]model.ForeignKeyMeta, error) {
	pkMap := make(map[string]*model.PrimaryKeyMeta)
	fkMap := make(map[string][]model.ForeignKeyMeta)

	// 获取主键
	pkQuery := `
		SELECT 
			tc.TABLE_SCHEMA,
			tc.TABLE_NAME,
			tc.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.ORDINAL_POSITION
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.KEY_COLUMN_USAGE kcu 
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME 
			AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
		WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		  AND tc.TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY tc.TABLE_SCHEMA, tc.TABLE_NAME, kcu.ORDINAL_POSITION
	`

	rows, err := db.Query(pkQuery)
	if err != nil {
		return pkMap, fkMap, err
	}
	defer rows.Close()

	for rows.Next() {
		var dbName, tableName, constraintName, columnName string
		var ordinal int
		if err := rows.Scan(&dbName, &tableName, &constraintName, &columnName, &ordinal); err != nil {
			continue
		}
		key := dbName + "." + tableName
		if _, exists := pkMap[key]; !exists {
			pkMap[key] = &model.PrimaryKeyMeta{
				Name:    constraintName,
				Columns: make([]string, 0),
			}
		}
		pkMap[key].Columns = append(pkMap[key].Columns, columnName)
	}

	// 获取外键
	fkQuery := `
		SELECT 
			tc.TABLE_SCHEMA,
			tc.TABLE_NAME,
			tc.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME,
			rc.UPDATE_RULE,
			rc.DELETE_RULE,
			kcu.ORDINAL_POSITION
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.KEY_COLUMN_USAGE kcu 
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME 
			AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
		JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
			ON tc.CONSTRAINT_NAME = rc.CONSTRAINT_NAME
			AND tc.TABLE_SCHEMA = rc.CONSTRAINT_SCHEMA
		WHERE tc.CONSTRAINT_TYPE = 'FOREIGN KEY'
		  AND tc.TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY tc.TABLE_SCHEMA, tc.TABLE_NAME, tc.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
	`

	fkRows, err := db.Query(fkQuery)
	if err != nil {
		return pkMap, fkMap, err
	}
	defer fkRows.Close()

	// 用于聚合同一外键的多列
	fkTemp := make(map[string]map[string]*model.ForeignKeyMeta)
	for fkRows.Next() {
		var dbName, tableName, constraintName, colName, refTable, refCol, onUpdate, onDelete string
		var ordinal int
		if err := fkRows.Scan(&dbName, &tableName, &constraintName, &colName, &refTable, &refCol, &onUpdate, &onDelete, &ordinal); err != nil {
			continue
		}
		key := dbName + "." + tableName
		if _, ok := fkTemp[key]; !ok {
			fkTemp[key] = make(map[string]*model.ForeignKeyMeta)
		}
		if _, ok := fkTemp[key][constraintName]; !ok {
			fkTemp[key][constraintName] = &model.ForeignKeyMeta{
				Name:       constraintName,
				Columns:    make([]string, 0),
				RefTable:   refTable,
				RefColumns: make([]string, 0),
				OnUpdate:   onUpdate,
				OnDelete:   onDelete,
			}
		}
		fkTemp[key][constraintName].Columns = append(fkTemp[key][constraintName].Columns, colName)
		fkTemp[key][constraintName].RefColumns = append(fkTemp[key][constraintName].RefColumns, refCol)
	}

	// 转换为slice
	for key, constraints := range fkTemp {
		for _, fk := range constraints {
			fkMap[key] = append(fkMap[key], *fk)
		}
	}

	return pkMap, fkMap, nil
}
