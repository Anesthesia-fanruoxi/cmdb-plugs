package router

import (
	"eip-plugs/api"
	"net/http"
)

// SetupRoutes 设置路由
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// IP查询接口
	mux.HandleFunc("/api/ip", api.GetIPHandler)

	return mux
}
