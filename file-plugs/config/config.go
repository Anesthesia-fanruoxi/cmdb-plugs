package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type StorageConfig struct {
	MaxFileSize int64        `yaml:"max_file_size"`
	Paths       []PathConfig `yaml:"paths"`
}

type PathConfig struct {
	Key          string   `yaml:"key"`
	Path         string   `yaml:"path"`
	MaxFileSize  int64    `yaml:"max_file_size"`
	AllowedTypes []string `yaml:"allowed_types"`
	AutoUnzip    bool     `yaml:"auto_unzip"` // 自动解压 ZIP 文件
}

var Cfg *Config

// Load 加载配置文件
func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 校验配置
	if err := cfg.validate(); err != nil {
		return err
	}

	Cfg = &cfg
	return nil
}

// validate 校验配置合法性
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Server.Port)
	}

	if len(c.Storage.Paths) == 0 {
		return fmt.Errorf("至少需要配置一个上传路径")
	}

	keys := make(map[string]bool)
	for _, p := range c.Storage.Paths {
		// 检查 key 唯一性
		if keys[p.Key] {
			return fmt.Errorf("重复的路径标识: %s", p.Key)
		}
		keys[p.Key] = true

		// 检查 key 非空
		if p.Key == "" {
			return fmt.Errorf("路径标识不能为空")
		}

		// 检查路径非空
		if p.Path == "" {
			return fmt.Errorf("路径 [%s] 的 path 不能为空", p.Key)
		}

		// 核心安全检查：禁止根目录
		cleanPath := filepath.Clean(p.Path)
		if cleanPath == "/" || cleanPath == "\\" || cleanPath == "." {
			return fmt.Errorf("路径 [%s] 不能设置为根目录或当前目录: %s", p.Key, p.Path)
		}
	}

	return nil
}

// GetPathConfig 根据 key 获取路径配置
func GetPathConfig(key string) *PathConfig {
	if Cfg == nil {
		return nil
	}
	for i := range Cfg.Storage.Paths {
		if Cfg.Storage.Paths[i].Key == key {
			return &Cfg.Storage.Paths[i]
		}
	}
	return nil
}

// GetMaxFileSize 获取指定路径的最大文件大小
func (p *PathConfig) GetMaxFileSize(globalMax int64) int64 {
	if p.MaxFileSize > 0 {
		return p.MaxFileSize
	}
	return globalMax
}
