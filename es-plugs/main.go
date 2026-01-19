package main

import (
	"es-plugs/common"
	"es-plugs/config"
	"es-plugs/router"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// 加载配置（配置文件可选，优先使用环境变量）
	if err := config.LoadConfig("config/config.yml"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志（需要在配置加载后，以便读取日志级别）
	common.InitLogger()

	common.Logger.Info("配置加载完成 (配置文件可选，环境变量优先)")

	// 输出配置信息（敏感信息脱敏）
	esConfig := config.GetESConfig()
	password := "***"
	if esConfig.Password != "" {
		password = "***"
	}
	common.Logger.Info(fmt.Sprintf("ES配置: host=%s, username=%s, password=%s, timeout=%d",
		esConfig.Host, esConfig.Username, password, esConfig.Timeout))

	// 输出限制配置
	limitConfig := config.GetLimitConfig()
	common.Logger.Info(fmt.Sprintf("限制配置: max_size=%d (最大返回条数，硬限制3000)", limitConfig.MaxSize))

	// 初始化路由
	r := router.InitRouter()

	// 启动服务
	port := ":8081"
	common.Logger.Info(fmt.Sprintf("服务启动在端口 %s", port))
	if err := http.ListenAndServe(port, r); err != nil {
		common.Logger.Error(fmt.Sprintf("服务启动失败: %v", err))
		log.Fatal(err)
	}
}
