package config

import (
	"os"
	"strconv"

	"al-plugs/logger"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	// 服务端口
	Port string `yaml:"port"`
	// 阿里云配置
	Aliyun AliyunConfig `yaml:"aliyun"`
	// 告警配置
	Alert AlertConfig `yaml:"alert"`
}

// AliyunConfig 阿里云配置
type AliyunConfig struct {
	// 访问密钥ID
	AccessKeyID string `yaml:"access_key_id"`
	// 访问密钥Secret
	AccessKeySecret string `yaml:"access_key_secret"`
	// 区域ID
	RegionID string `yaml:"region_id"`
}

// AlertConfig 告警配置
type AlertConfig struct {
	// Webhook 通知地址
	WebhookURL string `yaml:"webhook_url"`
	// 余额阈值（元），低于此值触发告警
	BalanceThreshold float64 `yaml:"balance_threshold"`
	// 告警抑制周期（小时），默认 24 小时
	SuppressHours int `yaml:"suppress_hours"`
	// 项目名称
	Project string `yaml:"project"`
	// 检查频次（分钟），默认 60 分钟
	CheckIntervalMinutes int `yaml:"check_interval_minutes"`
}

// LoadConfig 加载配置
// 优先级：环境变量 > 配置文件 > 默认值
func LoadConfig() *Config {
	// 先从配置文件读取
	cfg := loadFromFile()

	// 环境变量覆盖配置文件
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}
	if accessKeyID := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID"); accessKeyID != "" {
		cfg.Aliyun.AccessKeyID = accessKeyID
	}
	if accessKeySecret := os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET"); accessKeySecret != "" {
		cfg.Aliyun.AccessKeySecret = accessKeySecret
	}
	if regionID := os.Getenv("ALIBABA_CLOUD_REGION_ID"); regionID != "" {
		cfg.Aliyun.RegionID = regionID
	}
	if webhookURL := os.Getenv("ALERT_WEBHOOK_URL"); webhookURL != "" {
		cfg.Alert.WebhookURL = webhookURL
	}
	if project := os.Getenv("ALERT_PROJECT"); project != "" {
		cfg.Alert.Project = project
	}
	if thresholdStr := os.Getenv("ALERT_BALANCE_THRESHOLD"); thresholdStr != "" {
		logger.Info("从环境变量读取余额阈值: %s", thresholdStr)
		if threshold, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			cfg.Alert.BalanceThreshold = threshold
			logger.Info("余额阈值解析成功: %.2f", threshold)
		} else {
			logger.Error("余额阈值解析失败: %v", err)
		}
	}
	if suppressHoursStr := os.Getenv("ALERT_SUPPRESS_HOURS"); suppressHoursStr != "" {
		if suppressHours, err := strconv.Atoi(suppressHoursStr); err == nil {
			cfg.Alert.SuppressHours = suppressHours
		}
	}
	if checkIntervalStr := os.Getenv("ALERT_CHECK_INTERVAL_MINUTES"); checkIntervalStr != "" {
		if checkInterval, err := strconv.Atoi(checkIntervalStr); err == nil {
			cfg.Alert.CheckIntervalMinutes = checkInterval
		}
	}

	// 设置默认值
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Aliyun.RegionID == "" {
		cfg.Aliyun.RegionID = "cn-hangzhou"
	}
	if cfg.Alert.SuppressHours == 0 {
		cfg.Alert.SuppressHours = 24
	}
	if cfg.Alert.CheckIntervalMinutes == 0 {
		cfg.Alert.CheckIntervalMinutes = 60
	}

	// 输出加载的配置信息（不输出敏感信息）
	logger.Info("配置加载完成:")
	logger.Info("  - 端口: %s", cfg.Port)
	logger.Info("  - 区域: %s", cfg.Aliyun.RegionID)
	logger.Info("  - 余额阈值: %.2f 元", cfg.Alert.BalanceThreshold)
	logger.Info("  - 检查频次: %d 分钟", cfg.Alert.CheckIntervalMinutes)
	logger.Info("  - 抑制周期: %d 小时", cfg.Alert.SuppressHours)
	logger.Info("  - 项目名称: %s", cfg.Alert.Project)

	return cfg
}

// min 返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// loadFromFile 从配置文件加载
func loadFromFile() *Config {
	cfg := &Config{}

	// 尝试读取 config/config.yaml
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		// 配置文件不存在，返回空配置
		logger.Warn("配置文件 config/config.yaml 不存在，将使用环境变量或默认值")
		return cfg
	}

	// 解析 YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		logger.Error("解析配置文件失败: %v，将使用环境变量或默认值", err)
		return &Config{}
	}

	logger.Info("已从 config/config.yaml 加载配置")
	return cfg
}
