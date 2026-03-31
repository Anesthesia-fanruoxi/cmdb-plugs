package router

import (
	"net/http"
	"sql-plugs/api"
	"sql-plugs/common"
)

// SetupRoutes 设置路由
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// SQL查询接口
	mux.HandleFunc("/api/sql/search", loggingMiddleware(api.SQLSearchHandler))

	// 单字段查询接口（不限制返回数量）
	mux.HandleFunc("/api/searchPhone", loggingMiddleware(api.SearchPhoneHandler))

	// 数据库结构查询接口
	mux.HandleFunc("/api/sql/structure", loggingMiddleware(api.StructureHandler))

	// 数据库元数据接口（Navicat式全量元数据）
	mux.HandleFunc("/api/sql/metadata", loggingMiddleware(api.MetadataHandler))

	// 数据导出接口（无任何查询限制和统计）
	mux.HandleFunc("/api/sql/export", loggingMiddleware(api.ExportHandler))

	// SQL分析接口（用于调优，返回COUNT SQL和实际执行SQL）
	mux.HandleFunc("/api/sql/analyze", loggingMiddleware(api.SQLAnalyzeHandler))

	// SQL执行接口（增删改，带安全限制）
	mux.HandleFunc("/api/sql/execute", loggingMiddleware(api.SQLExecuteHandler))

	// SQL检查接口（EXPLAIN + 影响行数计算）
	mux.HandleFunc("/api/sql/check", loggingMiddleware(api.SQLCheckHandler))

	// 查询取消接口
	mux.HandleFunc("/api/sql/cancel", loggingMiddleware(api.CancelQueryHandler))

	// 活跃查询列表接口
	mux.HandleFunc("/api/sql/active", loggingMiddleware(api.ActiveQueriesHandler))

	// 连接池状态接口
	mux.HandleFunc("/api/pool/stats", loggingMiddleware(api.PoolStatsHandler))

	// 健康检查接口
	mux.HandleFunc("/health", healthCheckHandler)
	return mux
}

// loggingMiddleware 日志中间件
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理 OPTIONS 预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		common.Logger.Infof("请求: %s %s - 来源: %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
	}
}

// healthCheckHandler 健康检查
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	common.Success(w, map[string]string{
		"status": "ok",
	})
}
