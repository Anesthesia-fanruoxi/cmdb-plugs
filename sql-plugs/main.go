package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sql-plugs/common"
	"sql-plugs/config"
	"sql-plugs/router"
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
	common.Logger.Info("SQL查询服务启动中...")

	// 3. 测试数据库连接
	common.Logger.Info("\n")
	if _, err := common.GetDB(); err != nil {
		common.Logger.Errorf("数据库连接失败: %v", err)
		os.Exit(1)
	}
	common.Logger.Info("\n")

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
		common.Logger.Infof("🔗 SQL查询接口:    http://localhost%s/api/sql/search", addr)
		common.Logger.Infof("🔗 表结构接口:    http://localhost%s/api/sql/structure", addr)
		common.Logger.Infof("🔗 元数据接口:    http://localhost%s/api/sql/metadata", addr)
		common.Logger.Infof("🔗 数据导出接口:  http://localhost%s/api/sql/export", addr)
		common.Logger.Infof("🔗 SQL分析接口:   http://localhost%s/api/sql/analyze", addr)
		common.Logger.Infof("🔗 SQL执行接口:   http://localhost%s/api/sql/execute", addr)
		common.Logger.Infof("🔗 SQL检查接口:   http://localhost%s/api/sql/check", addr)
		common.Logger.Infof("🔗 连接池状态:    http://localhost%s/api/pool/stats", addr)
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
	common.CloseDB()
	common.Logger.Info("服务器已关闭")
}
