package router

import (
	"net/http"

	"nacos-plugs/api"
)

// Setup 设置路由
func Setup() *http.ServeMux {
	mux := http.NewServeMux()

	// 配置相关路由
	mux.HandleFunc("/api/config/get", api.HandleGetConfig)
	mux.HandleFunc("/api/config/list", api.HandleListConfigs)
	mux.HandleFunc("/api/config/search", api.HandleSearchConfigs)

	return mux
}
