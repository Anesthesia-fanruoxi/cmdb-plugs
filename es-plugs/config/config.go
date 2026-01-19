package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
)

type Config struct {
	Elasticsearch ESConfig    `yaml:"elasticsearch"`
	Log           LogConfig   `yaml:"log"`
	Limit         LimitConfig `yaml:"limit"`
}

type ESConfig struct {
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Timeout  int    `yaml:"timeout"`
}

type LogConfig struct {
	Level string `yaml:"level"` // info 或 error
}

type LimitConfig struct {
	MaxSize int `yaml:"max_size"` // ES返回最大条数限制
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(path string) error {
	// 初始化默认配置
	GlobalConfig = &Config{
		Elasticsearch: ESConfig{
			Host:     "http://localhost:9200",
			Username: "elastic",
			Password: "",
			Timeout:  30,
		},
		Log: LogConfig{
			Level: "info",
		},
		Limit: LimitConfig{
			MaxSize: 1000, // 默认最大3000条
		},
	}

	// 尝试读取配置文件（可选）
	data, err := os.ReadFile(path)
	if err == nil {
		// 配置文件存在，解析它
		if err := yaml.Unmarshal(data, GlobalConfig); err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}
	}
	// 如果文件不存在，使用默认配置（不报错）

	// 环境变量覆盖配置（优先级最高）
	overrideWithEnv()

	return nil
}

// overrideWithEnv 使用环境变量覆盖配置
func overrideWithEnv() {
	// ES配置
	if host := os.Getenv("ES_HOST"); host != "" {
		GlobalConfig.Elasticsearch.Host = host
	}
	if username := os.Getenv("ES_USERNAME"); username != "" {
		GlobalConfig.Elasticsearch.Username = username
	}
	if password := os.Getenv("ES_PASSWORD"); password != "" {
		GlobalConfig.Elasticsearch.Password = password
	}

	// 日志配置
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		GlobalConfig.Log.Level = logLevel
	}

	// 限制配置
	if maxSizeStr := os.Getenv("LIMIT_MAX_SIZE"); maxSizeStr != "" {
		if maxSize, err := strconv.Atoi(maxSizeStr); err == nil && maxSize > 0 {
			// 强制最大不超过3000
			if maxSize > 3000 {
				maxSize = 3000
			}
			GlobalConfig.Limit.MaxSize = maxSize
		}
	}
}

// GetESConfig 获取ES配置
func GetESConfig() ESConfig {
	if GlobalConfig == nil {
		return ESConfig{}
	}
	return GlobalConfig.Elasticsearch
}

// GetLogConfig 获取日志配置
func GetLogConfig() LogConfig {
	if GlobalConfig == nil {
		return LogConfig{Level: "info"} // 默认 info
	}
	if GlobalConfig.Log.Level == "" {
		return LogConfig{Level: "info"} // 默认 info
	}
	return GlobalConfig.Log
}

// GetLimitConfig 获取限制配置
func GetLimitConfig() LimitConfig {
	if GlobalConfig == nil {
		return LimitConfig{MaxSize: 3000} // 默认3000
	}
	if GlobalConfig.Limit.MaxSize <= 0 {
		return LimitConfig{MaxSize: 3000} // 默认3000
	}
	// 强制最大不超过3000
	if GlobalConfig.Limit.MaxSize > 3000 {
		return LimitConfig{MaxSize: 3000}
	}
	return GlobalConfig.Limit
}

// ApplySizeLimit 应用size限制，确保不超过配置的最大值
func ApplySizeLimit(requestedSize int) int {
	limitConfig := GetLimitConfig()
	maxSize := limitConfig.MaxSize

	// 如果请求的size超过最大限制，返回最大限制
	if requestedSize > maxSize {
		return maxSize
	}
	return requestedSize
}
