package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sql-plugs/common"
	"sync"
	"time"
)

type QueryInfo struct {
	QueryID      string             `json:"query_id"`
	ConnectionID int64              `json:"connection_id"`
	StartTime    time.Time          `json:"start_time"`
	SQL          string             `json:"sql"`
	DBName       string             `json:"db_name"`
	Status       string             `json:"status"`
	Cancel       context.CancelFunc `json:"-"`
}

type QueryManager struct {
	mu      sync.RWMutex
	queries map[string]*QueryInfo
	counter int64
}

var queryManager = &QueryManager{
	queries: make(map[string]*QueryInfo),
}

func (qm *QueryManager) RegisterQuery(connID int64, sqlText, dbName string) (string, context.Context, context.CancelFunc) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.counter++
	queryID := fmt.Sprintf("q_%d_%d", time.Now().UnixMilli(), qm.counter)
	ctx, cancel := context.WithCancel(context.Background())

	displaySQL := sqlText
	if len(displaySQL) > 200 {
		displaySQL = displaySQL[:200] + "..."
	}

	qm.queries[queryID] = &QueryInfo{
		QueryID:      queryID,
		ConnectionID: connID,
		StartTime:    time.Now(),
		SQL:          displaySQL,
		DBName:       dbName,
		Status:       "running",
		Cancel:       cancel,
	}

	common.Logger.Infof("注册查询: %s, ConnectionID: %d", queryID, connID)
	return queryID, ctx, cancel
}

func (qm *QueryManager) RegisterWithID(queryID string, sqlText, dbName string, ctx context.Context, cancel context.CancelFunc) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	displaySQL := sqlText
	if len(displaySQL) > 200 {
		displaySQL = displaySQL[:200] + "..."
	}

	qm.queries[queryID] = &QueryInfo{
		QueryID:      queryID,
		ConnectionID: 0,
		StartTime:    time.Now(),
		SQL:          displaySQL,
		DBName:       dbName,
		Status:       "running",
		Cancel:       cancel,
	}

	common.Logger.Infof("注册查询(外部ID): %s", queryID)
}

func (qm *QueryManager) CancelQuery(queryID string) (bool, string) {
	qm.mu.Lock()
	info, exists := qm.queries[queryID]
	if !exists {
		qm.mu.Unlock()
		return false, "查询不存在或已完成"
	}

	if info.Status != "running" {
		qm.mu.Unlock()
		return false, "查询已经" + info.Status
	}

	connID := info.ConnectionID
	info.Status = "cancelled"
	info.Cancel()
	qm.mu.Unlock()

	if connID > 0 {
		if err := killMySQLQuery(connID); err != nil {
			common.Logger.Warnf("KILL QUERY %d 失败: %v", connID, err)
			return true, fmt.Sprintf("Go层已取消，但MySQL终止失败: %v", err)
		}
		common.Logger.Infof("成功终止查询: %s, ConnectionID: %d", queryID, connID)
	}

	return true, "查询已终止"
}

func (qm *QueryManager) UnregisterQuery(queryID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if info, exists := qm.queries[queryID]; exists {
		if info.Status == "running" {
			info.Status = "completed"
		}
		delete(qm.queries, queryID)
		common.Logger.Infof("注销查询: %s", queryID)
	}
}

func (qm *QueryManager) GetActiveQueries() []*QueryInfo {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := make([]*QueryInfo, 0)
	for _, info := range qm.queries {
		if info.Status == "running" {
			infoCopy := *info
			result = append(result, &infoCopy)
		}
	}
	return result
}

func killMySQLQuery(connectionID int64) error {
	db, err := common.GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("KILL QUERY %d", connectionID))
	return err
}

func GetConnectionID(db *sql.DB) (int64, error) {
	var connID int64
	err := db.QueryRow("SELECT CONNECTION_ID()").Scan(&connID)
	return connID, err
}

func CancelQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许POST请求")
		return
	}

	var req struct {
		QueryID string `json:"query_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorWithCode(w, http.StatusBadRequest, "请求参数解析失败")
		return
	}

	if req.QueryID == "" {
		common.ErrorWithCode(w, http.StatusBadRequest, "query_id不能为空")
		return
	}

	cancelled, message := queryManager.CancelQuery(req.QueryID)
	common.Success(w, map[string]interface{}{
		"cancelled": cancelled,
		"query_id":  req.QueryID,
		"message":   message,
	})
}

func ActiveQueriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许GET请求")
		return
	}

	queries := queryManager.GetActiveQueries()

	type QueryWithDuration struct {
		*QueryInfo
		Duration string `json:"duration"`
	}

	result := make([]QueryWithDuration, 0, len(queries))
	for _, q := range queries {
		duration := time.Since(q.StartTime)
		result = append(result, QueryWithDuration{
			QueryInfo: q,
			Duration:  fmt.Sprintf("%.1fs", duration.Seconds()),
		})
	}

	common.Success(w, map[string]interface{}{
		"count":   len(result),
		"queries": result,
	})
}

func GetQueryManager() *QueryManager {
	return queryManager
}
