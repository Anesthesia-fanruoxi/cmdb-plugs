package router

import (
	"es-plugs/api"
	"net/http"
)

// InitRouter 初始化路由
func InitRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// ES搜索接口
	mux.HandleFunc("/api/elfk/search", api.SearchAPI)

	// 获取索引映射接口
	mux.HandleFunc("/api/elfk/indices", api.GetIndexMappingHandler)

	// 滚动查询接口
	mux.HandleFunc("/api/elfk/scroll", api.ScrollHandler)

	// 上下文查询接口
	mux.HandleFunc("/api/elfk/context", api.ContextHandler)

	// 健康检查
	mux.HandleFunc("/health", healthCheck)

	return mux
}

// healthCheck 健康检查接口
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
