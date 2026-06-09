package main

import (
	"fmt"
	"net/http"
	"nginx-plugs/common"
	"nginx-plugs/config"
	"nginx-plugs/router"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. 加载配置
	if err := config.LoadConfig("config/config.yml"); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	common.InitLogger()
	common.Logger.Info("Nginx配置生成服务启动中...")

	// 3. 初始化模板（不存在则警告，不退出）
	common.InitTemplate()

	// 4. 设置路由
	mux := router.SetupRoutes()

	// 5. 获取服务器配置
	serverConfig := config.GetServerConfig()
	addr := fmt.Sprintf(":%d", serverConfig.Port)

	// 6. 创建HTTP服务器
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 7. 启动服务器
	go func() {
		common.Logger.Infof("🚀 服务器启动成功，监听端口: %d", serverConfig.Port)
		common.Logger.Infof("🔗 生成配置接口:  http://localhost%s/api/nginx/add", addr)
		common.Logger.Infof("🔗 预览配置接口:  http://localhost%s/api/nginx/preview", addr)
		common.Logger.Infof("🔗 删除配置接口:  http://localhost%s/api/nginx/delete", addr)
		common.Logger.Infof("🔗 配置列表接口:  http://localhost%s/api/nginx/list", addr)
		common.Logger.Infof("🔗 健康检查接口:  http://localhost%s/health", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			common.Logger.Errorf("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 8. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	common.Logger.Info("正在关闭服务器...")
	common.Logger.Info("服务器已关闭")
}
