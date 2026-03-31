package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sql-plugs/common"
	"sql-plugs/model"
	"time"
)

func SQLSearchHandler(w http.ResponseWriter, r *http.Request) {
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

	var rawReq struct {
		QueryID json.RawMessage `json:"query_id"`
		Query   json.RawMessage `json:"query"`
		DB      string          `json:"dbName"`
	}
	if err := json.Unmarshal(body, &rawReq); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败: "+err.Error())
		return
	}

	var queryID string
	if rawReq.QueryID != nil {
		json.Unmarshal(rawReq.QueryID, &queryID)
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

	queries := common.SplitSQLStatements(queryStr)
	if len(queries) == 0 {
		common.ErrorWithCode(w, http.StatusBadRequest, "查询语句不能为空")
		return
	}

	common.Logger.Infof("SQL批量查询请求 - queryID: %s, 查询数量: %d", queryID, len(queries))

	startTime := time.Now()
	results := make([]model.QueryResult, 0, len(queries))
	dbName := rawReq.DB

	for i, query := range queries {
		normalizedSQL := common.NormalizeWhitespace(query)
		sqlType := common.GetSQLType(normalizedSQL)
		sqlCategory := common.GetSQLCategory(sqlType)

		if sqlCategory != "DQL" && sqlCategory != "OTHER" {
			errMsg := fmt.Sprintf("SQL语句[%d]不合法：此接口只允许执行查询语句（SELECT/SHOW/DESCRIBE/EXPLAIN），不支持%s操作", i+1, sqlCategory)
			common.Logger.Warnf("拒绝非DQL语句[%d]: 类型=%s, 分类=%s", i+1, sqlType, sqlCategory)
			results = append(results, model.QueryResult{
				Columns: []string{"error"},
				Rows:    [][]interface{}{{errMsg}},
				Total:   0,
				Took:    0,
				DBName:  dbName,
			})
			continue
		}

		features := common.AnalyzeSQLFeatures(normalizedSQL)
		riskLevel, _ := common.AssessQueryRisk(normalizedSQL, features)

		common.Logger.Infof("查询[%d] 类型=%s, 风险=%s, HasWhere=%v, HasJoin=%v",
			i+1, sqlType, riskLevel, features.HasWhere, features.HasJoin)

		result, err := executeSingleQuery(dbName, normalizedSQL, features, riskLevel, i+1, queryID)
		if err != nil {
			common.Logger.Errorf("查询[%d] 失败: %v", i+1, err)
			results = append(results, model.QueryResult{
				Columns: []string{"error"},
				Rows:    [][]interface{}{{err.Error()}},
				Total:   0,
				Took:    0,
				DBName:  "",
			})
			continue
		}

		results = append(results, *result)
		if dbName == "" {
			dbName = result.DBName
		}
	}

	took := time.Since(startTime).Milliseconds()

	response := model.SQLSearchResponse{
		Results: results,
		Total:   len(results),
		Took:    took,
		DBName:  dbName,
	}

	common.SuccessWithMessage(w, "查询成功", response)
	common.Logger.Infof("SQL批量查询成功 - queryID: %s, 总耗时: %dms, 查询数: %d", queryID, took, len(results))
}

func executeSingleQuery(dbName string, sql string, features *common.SQLFeatures, riskLevel string, queryIndex int, externalQueryID string) (*model.QueryResult, error) {
	userOriginalLimit := common.GetUserOriginalLimit(sql)
	hasUserLimit := userOriginalLimit > 0

	var processedSQL string
	var shouldCount bool

	if riskLevel == "high" {
		processedSQL = common.ProcessSQLLimit(sql)
		shouldCount = true
		common.Logger.Infof("查询[%d] 高风险 - 执行COUNT，强制LIMIT", queryIndex)
	} else {
		processedSQL = sql
		shouldCount = false
		common.Logger.Infof("查询[%d] %s风险 - 查询全部数据", queryIndex, riskLevel)
	}

	return executeSingleQueryWithContext(dbName, processedSQL, hasUserLimit, shouldCount, userOriginalLimit, externalQueryID)
}
