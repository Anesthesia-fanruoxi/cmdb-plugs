package config

import (
	"encoding/json"
	"os"
	"strconv"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
}

// RedisConfig Redis 默认连接配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// ScanConfig 扫描配置
type ScanConfig struct {
	DefaultCount     int64  `json:"defaultCount"`
	MaxCount         int64  `json:"maxCount"`
	DefaultSeparator string `json:"defaultSeparator"`
}

// AppConfig 应用配置
type AppConfig struct {
	Server ServerConfig `json:"server"`
	Redis  RedisConfig  `json:"redis"`
	Scan   ScanConfig   `json:"scan"`
}

// 默认配置
var defaultConfig = AppConfig{
	Server: ServerConfig{
		Port: ":8080",
	},
	Redis: RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	},
	Scan: ScanConfig{
		DefaultCount:     1000,
		MaxCount:         10000,
		DefaultSeparator: ":",
	},
}

// 全局配置
var (
	DefaultConfig     = defaultConfig.Server
	DefaultRedis      = defaultConfig.Redis
	DefaultScanConfig = defaultConfig.Scan
)

// 配置文件路径
var configPaths = []string{
	"config.json",
	"./config/config.json",
}

func init() {
	Load()
}

// Load 加载配置：环境变量 > 配置文件 > 默认值
func Load() {
	cfg := defaultConfig

	// 1. 先尝试读取配置文件
	loadFromFile(&cfg)

	// 2. 环境变量覆盖（优先级最高）
	loadFromEnv(&cfg)

	// 更新全局配置
	DefaultConfig = cfg.Server
	DefaultRedis = cfg.Redis
	DefaultScanConfig = cfg.Scan
}

// loadFromFile 从配置文件加载
func loadFromFile(cfg *AppConfig) {
	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(data, cfg); err == nil {
			return
		}
	}
}

// loadFromEnv 从环境变量加载
func loadFromEnv(cfg *AppConfig) {
	// 服务器配置
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}

	// Redis 配置
	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}

	// 扫描配置
	if v := os.Getenv("SCAN_MAX_COUNT"); v != "" {
		if count, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Scan.MaxCount = count
		}
	}
	if v := os.Getenv("SCAN_SEPARATOR"); v != "" {
		cfg.Scan.DefaultSeparator = v
	}
}
