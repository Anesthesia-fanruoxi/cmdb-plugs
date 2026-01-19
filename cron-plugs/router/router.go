package router

import "net/http"

// Setup 设置路由
func Setup() {
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/api/task/execute", ExecuteHandler)
}
