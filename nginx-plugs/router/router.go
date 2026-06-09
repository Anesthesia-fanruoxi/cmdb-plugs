package router

import (
	"net/http"
	"nginx-plugs/api"
	"nginx-plugs/common"
)

// SetupRoutes 设置路由
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// 生成nginx配置（写入文件）
	mux.HandleFunc("/api/nginx/add", loggingMiddleware(api.GenerateHandler))

	// 预览nginx配置（不写入文件，只返回内容）
	mux.HandleFunc("/api/nginx/preview", loggingMiddleware(api.PreviewHandler))

	// 删除nginx配置文件
	mux.HandleFunc("/api/nginx/delete", loggingMiddleware(api.DeleteHandler))

	// 列出已有的配置文件
	mux.HandleFunc("/api/nginx/list", loggingMiddleware(api.ListHandler))

	// 健康检查
	mux.HandleFunc("/health", healthCheckHandler)

	return mux
}

// loggingMiddleware 日志和CORS中间件
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
	templateReady := common.TemplateMgr.IsReady()
	status := "ok"
	if !templateReady {
		status = "degraded"
	}
	common.Success(w, map[string]interface{}{
		"status":         status,
		"template_ready": templateReady,
	})
}
