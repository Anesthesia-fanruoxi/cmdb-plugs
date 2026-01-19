package api

import (
	"net/http"
	"sql-plugs/common"
)

// PoolStatsHandler 获取连接池状态
func PoolStatsHandler(w http.ResponseWriter, r *http.Request) {
	// 只允许GET请求
	if r.Method != http.MethodGet {
		common.ErrorWithCode(w, http.StatusMethodNotAllowed, "只允许GET请求")
		return
	}

	// 获取数据库连接池状态
	stats, err := common.GetDBStats()
	if err != nil {
		common.ErrorWithCode(w, http.StatusInternalServerError, "获取连接池状态失败: "+err.Error())
		return
	}

	// 格式化响应数据
	response := map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,          // 最大打开连接数
		"open_connections":     stats.OpenConnections,             // 当前打开的连接数
		"in_use":               stats.InUse,                       // 正在使用的连接数
		"idle":                 stats.Idle,                        // 空闲连接数
		"wait_count":           stats.WaitCount,                   // 等待连接的总次数
		"wait_duration_ms":     stats.WaitDuration.Milliseconds(), // 等待连接的总时长（毫秒）
		"max_idle_closed":      stats.MaxIdleClosed,               // 因超过最大空闲连接数而关闭的连接数
		"max_idle_time_closed": stats.MaxIdleTimeClosed,           // 因超过最大空闲时间而关闭的连接数
		"max_lifetime_closed":  stats.MaxLifetimeClosed,           // 因超过最大生命周期而关闭的连接数
	}

	common.Success(w, response)
	common.Logger.Info("连接池状态查询成功")
}
