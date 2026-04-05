package common

import (
	"strconv"
	"strings"
	"time"

	"al-plugs/config"
	"al-plugs/logger"

	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

// BalanceScheduler 余额检查调度器
type BalanceScheduler struct {
	cfg          *config.Config
	alertManager *AlertManager
	ticker       *time.Ticker
	stopChan     chan struct{}
}

// NewBalanceScheduler 创建余额检查调度器
func NewBalanceScheduler(cfg *config.Config) *BalanceScheduler {
	return &BalanceScheduler{
		cfg:          cfg,
		alertManager: NewAlertManager(cfg.Alert.SuppressHours),
		stopChan:     make(chan struct{}),
	}
}

// Start 启动定时任务
func (bs *BalanceScheduler) Start() {
	// 使用配置的检查频次
	interval := time.Duration(bs.cfg.Alert.CheckIntervalMinutes) * time.Minute
	bs.ticker = time.NewTicker(interval)

	logger.Info("余额检查定时任务已启动，每 %d 分钟执行一次", bs.cfg.Alert.CheckIntervalMinutes)

	// 启动时立即执行一次
	go bs.checkBalance()

	// 定时执行
	go func() {
		for {
			select {
			case <-bs.ticker.C:
				bs.checkBalance()
			case <-bs.stopChan:
				logger.Info("余额检查定时任务已停止")
				return
			}
		}
	}()
}

// Stop 停止定时任务
func (bs *BalanceScheduler) Stop() {
	if bs.ticker != nil {
		bs.ticker.Stop()
	}
	close(bs.stopChan)
}

// checkBalance 检查余额
func (bs *BalanceScheduler) checkBalance() {
	logger.Info("开始执行定时余额检查...")

	// 创建客户端
	client, err := CreateBssClient(bs.cfg)
	if err != nil {
		logger.Error("创建客户端失败: %v", err)
		return
	}

	// 调用API查询余额
	runtime := &util.RuntimeOptions{}
	resp, err := client.QueryAccountBalanceWithOptions(runtime)
	if err != nil {
		logger.Error("查询账户余额失败: %v", err)
		return
	}

	// 解析余额
	if resp.Body == nil || resp.Body.Data == nil {
		logger.Error("余额数据为空")
		return
	}

	availableAmountStr := tea.StringValue(resp.Body.Data.AvailableAmount)
	// 保存原始格式用于显示
	originalAmountStr := availableAmountStr
	// 移除千位分隔符逗号用于数值比较
	availableAmount, err := strconv.ParseFloat(removeCommas(availableAmountStr), 64)
	if err != nil {
		logger.Error("解析余额失败: %v", err)
		return
	}

	logger.Info("当前账户余额: %.2f 元，阈值: %.2f 元", availableAmount, bs.cfg.Alert.BalanceThreshold)

	// 判断是否低于阈值
	if availableAmount < bs.cfg.Alert.BalanceThreshold {
		logger.Warn("余额低于阈值！当前余额: %.2f 元", availableAmount)

		// 判断是否应该发送告警（检查抑制周期）
		if bs.alertManager.ShouldAlert() {
			logger.Info("距离上次告警已超过抑制周期，准备发送告警通知")

			// 发送 Webhook 通知
			if bs.cfg.Alert.WebhookURL != "" {
				err := SendWebhook(bs.cfg.Alert.WebhookURL, originalAmountStr, bs.cfg.Alert.BalanceThreshold, bs.cfg.Alert.Project)
				if err != nil {
					logger.Error("发送 Webhook 通知失败: %v", err)
				} else {
					// 记录告警时间
					bs.alertManager.RecordAlert()
					logger.Info("告警通知已发送，下次告警时间: %s",
						time.Now().Add(time.Duration(bs.cfg.Alert.SuppressHours)*time.Hour).Format("2006-01-02 15:04:05"))
				}
			} else {
				logger.Warn("Webhook URL 未配置，跳过告警通知")
			}
		} else {
			lastAlertTime := bs.alertManager.GetLastAlertTime()
			nextAlertTime := lastAlertTime.Add(time.Duration(bs.cfg.Alert.SuppressHours) * time.Hour)
			logger.Info("在告警抑制周期内，跳过发送。上次告警: %s，下次可告警: %s",
				lastAlertTime.Format("2006-01-02 15:04:05"),
				nextAlertTime.Format("2006-01-02 15:04:05"))
		}
	} else {
		logger.Info("余额充足，无需告警")
	}
}

// removeCommas 移除字符串中的逗号（千位分隔符）
func removeCommas(s string) string {
	return strings.ReplaceAll(s, ",", "")
}
