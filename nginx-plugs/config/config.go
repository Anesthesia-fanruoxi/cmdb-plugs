package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 全局配置结构
type Config struct {
	Server ServerConfig `yaml:"server"`
	Nginx  NginxConfig  `yaml:"nginx"`
	Log    LogConfig    `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// NginxConfig nginx相关配置
type NginxConfig struct {
	ConfDir      string `yaml:"conf_dir"`      // nginx配置文件目录，默认 /etc/nginx/conf.d/
	TemplateFile string `yaml:"template_file"` // 模板文件路径，默认 templates/proxy.conf.template
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `yaml:"level"` // info 或 error
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(path string) error {
	// 初始化默认配置
	GlobalConfig = &Config{
		Server: ServerConfig{
			Port: 8091,
		},
		Nginx: NginxConfig{
			ConfDir:      "/etc/nginx/conf.d/cmdb-plugs/",
			TemplateFile: "templates/proxy.conf.template",
		},
		Log: LogConfig{
			Level: "info",
		},
	}

	// 尝试读取配置文件（可选）
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, GlobalConfig); err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}
		fmt.Printf("📄 配置文件已加载: %s\n", path)
	} else {
		fmt.Println("⚠️  未找到配置文件，使用默认配置")
	}

	// 环境变量覆盖配置（优先级最高）
	fmt.Println()
	overrideWithEnv()

	return nil
}

// overrideWithEnv 使用环境变量覆盖配置
func overrideWithEnv() {
	envUsed := false

	if portStr := os.Getenv("NGINX_PLUGS_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			GlobalConfig.Server.Port = port
			envUsed = true
			fmt.Printf("[ENV] NGINX_PLUGS_PORT = %d\n", port)
		}
	}

	if confDir := os.Getenv("NGINX_CONF_DIR"); confDir != "" {
		GlobalConfig.Nginx.ConfDir = confDir
		envUsed = true
		fmt.Printf("[ENV] NGINX_CONF_DIR = %s\n", confDir)
	}

	if templateFile := os.Getenv("NGINX_TEMPLATE_FILE"); templateFile != "" {
		GlobalConfig.Nginx.TemplateFile = templateFile
		envUsed = true
		fmt.Printf("[ENV] NGINX_TEMPLATE_FILE = %s\n", templateFile)
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		GlobalConfig.Log.Level = logLevel
		envUsed = true
		fmt.Printf("[ENV] LOG_LEVEL = %s\n", logLevel)
	}

	if envUsed {
		fmt.Println("\n✅ 环境变量配置已加载（优先级最高）")
	} else {
		fmt.Println("🔵 未检测到环境变量，使用配置文件/默认配置")
	}
}

// GetServerConfig 获取服务器配置
func GetServerConfig() ServerConfig {
	if GlobalConfig == nil {
		return ServerConfig{Port: 8091}
	}
	return GlobalConfig.Server
}

// GetNginxConfig 获取nginx配置
func GetNginxConfig() NginxConfig {
	if GlobalConfig == nil {
		return NginxConfig{
			ConfDir:      "/etc/nginx/conf.d/cmdb-plugs/",
			TemplateFile: "templates/proxy.conf.template",
		}
	}
	return GlobalConfig.Nginx
}

// GetLogConfig 获取日志配置
func GetLogConfig() LogConfig {
	if GlobalConfig == nil {
		return LogConfig{Level: "info"}
	}
	if GlobalConfig.Log.Level == "" {
		return LogConfig{Level: "info"}
	}
	return GlobalConfig.Log
}
