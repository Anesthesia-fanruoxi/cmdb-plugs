package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type NacosConfig struct {
	Host        string `json:"host" yaml:"host"`
	Port        int    `json:"port" yaml:"port"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Username    string `json:"username" yaml:"username"`
	Password    string `json:"password" yaml:"password"`
	ContextPath string `json:"contextPath" yaml:"contextPath"`
}

func DefaultConfig() *NacosConfig {
	return &NacosConfig{
		Host:        "127.0.0.1",
		Port:        8848,
		Namespace:   "public",
		ContextPath: "/nacos",
	}
}

func LoadConfig(configPath string) *NacosConfig {
	cfg := DefaultConfig()

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			yamlCfg := &NacosConfig{}
			if err := yaml.Unmarshal(data, yamlCfg); err == nil {
				if yamlCfg.Host != "" {
					cfg.Host = yamlCfg.Host
				}
				if yamlCfg.Port != 0 {
					cfg.Port = yamlCfg.Port
				}
				if yamlCfg.Namespace != "" {
					cfg.Namespace = yamlCfg.Namespace
				}
				if yamlCfg.Username != "" {
					cfg.Username = yamlCfg.Username
				}
				if yamlCfg.Password != "" {
					cfg.Password = yamlCfg.Password
				}
				if yamlCfg.ContextPath != "" {
					cfg.ContextPath = yamlCfg.ContextPath
				}
			}
		}
	}

	if host := os.Getenv("NACOS_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("NACOS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}
	if namespace := os.Getenv("NACOS_NAMESPACE"); namespace != "" {
		cfg.Namespace = namespace
	}
	if username := os.Getenv("NACOS_USERNAME"); username != "" {
		cfg.Username = username
	}
	if password := os.Getenv("NACOS_PASSWORD"); password != "" {
		cfg.Password = password
	}
	if contextPath := os.Getenv("NACOS_CONTEXT_PATH"); contextPath != "" {
		cfg.ContextPath = contextPath
	}

	return cfg
}

func (c *NacosConfig) GetServerAddress() string {
	return fmt.Sprintf("http://%s:%d%s", c.Host, c.Port, c.ContextPath)
}
