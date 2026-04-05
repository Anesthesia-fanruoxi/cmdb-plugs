package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"al-plugs/logger"
)

// WebhookPayload Webhook 通知负载（通用格式，用于日志）
type WebhookPayload struct {
	AlertType        string    `json:"alert_type"`        // 告警类型
	AlertTime        string    `json:"alert_time"`        // 告警时间
	AvailableAmount  string    `json:"available_amount"`  // 可用余额
	BalanceThreshold float64   `json:"balance_threshold"` // 余额阈值
	Message          string    `json:"message"`           // 告警消息
	Timestamp        time.Time `json:"timestamp"`         // 时间戳
}

// FeishuTextMessage 飞书文本消息格式
type FeishuTextMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// FeishuCardMessage 飞书卡片消息格式
type FeishuCardMessage struct {
	MsgType string `json:"msg_type"`
	Card    struct {
		Header struct {
			Title struct {
				Tag     string `json:"tag"`
				Content string `json:"content"`
			} `json:"title"`
			Template string `json:"template"`
		} `json:"header"`
		Elements []map[string]interface{} `json:"elements"`
	} `json:"card"`
}

// SendWebhook 发送 Webhook 通知（飞书格式）
func SendWebhook(webhookURL string, availableAmount string, threshold float64, project string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL 未配置")
	}

	alertTime := time.Now().Format("2006-01-02 15:04:05")

	// 构建飞书卡片消息
	message := FeishuCardMessage{
		MsgType: "interactive",
	}

	// 设置卡片头部（红色警告样式）
	message.Card.Header.Title.Tag = "plain_text"
	message.Card.Header.Title.Content = fmt.Sprintf("⚠️ %s - 余额告警", project)
	message.Card.Header.Template = "red"

	// 设置卡片内容
	message.Card.Elements = []map[string]interface{}{
		{
			"tag": "div",
			"text": map[string]string{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**项目名称：** %s", project),
			},
		},
		{
			"tag": "div",
			"text": map[string]string{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**告警时间：** %s", alertTime),
			},
		},
		{
			"tag": "div",
			"text": map[string]string{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**当前余额：** %s 元", availableAmount),
			},
		},
		{
			"tag": "div",
			"text": map[string]string{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**告警阈值：** %.2f 元", threshold),
			},
		},
		{
			"tag": "hr",
		},
		{
			"tag": "div",
			"text": map[string]string{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**告警消息：** 账户余额已低于设定阈值，请及时充值！"),
			},
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化 webhook 数据失败: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	logger.Info("正在发送 Webhook 到: %s", webhookURL)
	logger.Info("Webhook 数据: %s", string(jsonData))

	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送 webhook 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error("Webhook 响应状态码异常: %d, 响应内容: %s", resp.StatusCode, string(body))
		return fmt.Errorf("webhook 响应状态码异常: %d", resp.StatusCode)
	}

	logger.Info("Webhook 通知发送成功，状态码: %d, 响应内容: %s", resp.StatusCode, string(body))
	return nil
}
