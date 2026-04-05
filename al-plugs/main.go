package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"al-plugs/common"
	"al-plugs/config"
	"al-plugs/logger"
	"al-plugs/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// 设置 Gin 为 Release 模式，去掉调试日志
	gin.SetMode(gin.ReleaseMode)

	// 加载配置
	logger.Info("开始加载配置...")
	cfg := config.LoadConfig()

	// 启动余额检查定时任务
	scheduler := common.NewBalanceScheduler(cfg)
	scheduler.Start()

	// 设置路由
	r := router.SetupRouter(cfg)

	// 启动服务
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("=== 服务信息 ===")
	logger.Info("服务启动在端口 %s", cfg.Port)
	logger.Info("余额阈值: %.2f 元", cfg.Alert.BalanceThreshold)
	logger.Info("检查频次: %d 分钟", cfg.Alert.CheckIntervalMinutes)
	logger.Info("告警抑制周期: %d 小时", cfg.Alert.SuppressHours)
	if cfg.Alert.WebhookURL != "" {
		logger.Info("Webhook 已配置")
	} else {
		logger.Warn("Webhook 未配置，不会发送告警通知")
	}
	logger.Info("================")

	// 优雅关闭
	go func() {
		if err := r.Run(addr); err != nil {
			logger.Error("服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")
	scheduler.Stop()
	logger.Info("服务已关闭")
}
