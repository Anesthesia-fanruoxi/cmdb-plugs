package api

import (
	"database/sql"
	"fmt"
	"sql-plugs/common"
	"sql-plugs/config"
	"sql-plugs/model"
	"time"
)

// executeCountWithTimeout 执行带超时的 COUNT 查询
func executeCountWithTimeout(db *sql.DB, countQuery string, timeout time.Duration) int {
	resultChan := make(chan int, 1)
	errorChan := make(chan error, 1)

	go func() {
		var count int
		err := db.QueryRow(countQuery).Scan(&count)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- count
		}
	}()

	select {
	case count := <-resultChan:
		return count
	case err := <-errorChan:
		common.Logger.Warnf("COUNT 查询失败: %v", err)
		return -1
	case <-time.After(timeout):
		common.Logger.Warnf("COUNT 查询超时（>%v）", timeout)
		return -2
	}
}

// executeSingleQueryWithContext 执行单个SQL查询（带上下文信息）
// shouldCount: 是否需要执行COUNT查询（仅高风险查询需要）
func executeSingleQueryWithContext(dbName string, query string, hasUserLimit bool, shouldCount bool, userOriginalLimit int) (*model.QueryResult, error) {
	db, err := common.GetDB()
	if err != nil {
		return nil, err
	}

	dbConfig := config.GetDatabaseConfig()
	actualDBName := dbConfig.Database

	if dbName != "" {
		if !common.IsValidDatabaseName(dbName, 12) {
			return nil, fmt.Errorf("无效的数据库名称: %s", dbName)
		}
		_, err = db.Exec("USE `" + dbName + "`")
		if err != nil {
			return nil, fmt.Errorf("切换数据库失败: %w", err)
		}
		common.Logger.Infof("已切换到数据库: %s", dbName)
		actualDBName = dbName
	}

	connID, _ := GetConnectionID(db)
	queryID, ctx, cancel := GetQueryManager().RegisterQuery(connID, query, actualDBName)
	defer GetQueryManager().UnregisterQuery(queryID)
	defer cancel()

	startTime := time.Now()
	var totalCount int

	// 只有高风险查询才执行COUNT
	if shouldCount {
		countQuery := common.BuildCountSQL(query)
		_, _ = db.Exec("SET NAMES utf8mb4")
		countStartTime := time.Now()
		totalCount = executeCountWithTimeout(db, countQuery, 10*time.Second)
		countTook := time.Since(countStartTime).Milliseconds()

		if totalCount == -2 {
			common.Logger.Warnf("COUNT 查询超时（>10秒）- 设置total为-1")
			totalCount = -1
		} else if totalCount < 0 {
			common.Logger.Warnf("COUNT 查询失败")
			totalCount = 0
		} else {
			common.Logger.Infof("COUNT 查询成功 - 总记录数: %d, 耗时: %dms", totalCount, countTook)
		}
	}

	_, _ = db.Exec("SET NAMES utf8mb4")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("查询已取消")
		}
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}

	// 获取列类型信息，用于处理BIT类型
	columnTypes, _ := rows.ColumnTypes()

	// 读取全部数据，但只保留前100条返回给客户端
	dataRows := make([][]interface{}, 0)
	totalRowCount := 0

	for rows.Next() {
		totalRowCount++

		// 只保留前 DefaultLimit(100) 条数据
		if len(dataRows) >= common.DefaultLimit {
			continue // 继续遍历以获取真实总数，但不再保存数据
		}

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
				// 处理BIT类型：转换为0/1
				if columnTypes != nil && i < len(columnTypes) {
					typeName := columnTypes[i].DatabaseTypeName()
					if typeName == "BIT" {
						if len(val) > 0 && val[0] == 1 {
							row[i] = "1"
						} else {
							row[i] = "0"
						}
						continue
					}
				}
				row[i] = string(val)
			}
		}
		dataRows = append(dataRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历数据失败: %w", err)
	}

	took := time.Since(startTime).Milliseconds()

	// 设置total值
	if shouldCount && totalCount > 0 {
		// 高风险查询：使用COUNT结果
		common.Logger.Infof("查询完成 - 返回行数: %d, 真实总数(COUNT): %d", len(dataRows), totalCount)
	} else {
		// 其他查询：使用遍历得到的真实总数
		totalCount = totalRowCount
		common.Logger.Infof("查询完成 - 返回行数: %d, 真实总数(遍历): %d", len(dataRows), totalCount)
	}

	return &model.QueryResult{
		QueryID: queryID,
		Columns: columns,
		Rows:    dataRows,
		Total:   totalCount,
		Took:    took,
		DBName:  actualDBName,
	}, nil
}
