package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 全局配置结构
type Config struct {
	Server     ServerConfig   `yaml:"server"`
	Nginx      NginxConfig    `yaml:"nginx"`
	Aliyun     AliyunConfig   `yaml:"aliyun"`
	Domains    []DomainOption `yaml:"domains"`
	NginxCmd   NginxCmdConfig `yaml:"nginx_cmd"`
	SSHTargets []SSHTarget    `yaml:"ssh_targets"`
	Log        LogConfig      `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// NginxConfig nginx相关配置
type NginxConfig struct {
	ConfDir string `yaml:"conf_dir"` // nginx配置文件目录
}

// AliyunConfig 阿里云配置
type AliyunConfig struct {
	AccessKeyID     string `yaml:"access_key_id"`     // AccessKey ID
	AccessKeySecret string `yaml:"access_key_secret"` // AccessKey Secret
	RecordType      string `yaml:"record_type"`       // DNS记录类型: CNAME 或 A
	RecordValue     string `yaml:"record_value"`      // DNS记录值（CNAME为域名，A为IP）
}

// DomainOption 域名选项
type DomainOption struct {
	Name string `yaml:"name"` // 域名，如 hzlsg.com
}

// NginxCmdConfig nginx命令配置
type NginxCmdConfig struct {
	Reload string `yaml:"reload"` // nginx重载命令，默认 nginx -s reload
}

// SSHTarget SSH目标服务器配置
type SSHTarget struct {
	Name     string `yaml:"name"`     // 服务器名称
	Host     string `yaml:"host"`     // 服务器地址
	Port     int    `yaml:"port"`     // SSH端口，默认22
	User     string `yaml:"user"`     // SSH用户名
	KeyPath  string `yaml:"key_path"` // SSH私钥路径
	Password string `yaml:"password"` // SSH密码
}

// LogConfig 日志配置
type LogConfig struct {
	Level string `yaml:"level"` // info 或 error
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// LoadConfig 加载配置
// 优先级：命令行参数 > 系统环境变量 > 配置文件 > 默认值
func LoadConfig(path string, cliPort int) error {
	// 1. 初始化默认配置
	GlobalConfig = &Config{
		Server: ServerConfig{
			Port: 8091,
		},
		Nginx: NginxConfig{
			ConfDir: "/etc/nginx/conf.d/cmdb-plugs/",
		},
		Aliyun:  AliyunConfig{},
		Domains: []DomainOption{},
		NginxCmd: NginxCmdConfig{
			Reload: "nginx -s reload",
		},
		SSHTargets: []SSHTarget{},
		Log: LogConfig{
			Level: "info",
		},
	}

	// 2. 读取配置文件（覆盖默认值）
	data, err := os.ReadFile(path)
	if err != nil {
		// 尝试从二进制文件所在目录查找
		if execPath, execErr := os.Executable(); execErr == nil {
			binDir := filepath.Dir(execPath)
			altPath := filepath.Join(binDir, path)
			data, err = os.ReadFile(altPath)
			if err == nil {
				path = altPath
			}
		}
	}
	if err == nil {
		if err := yaml.Unmarshal(data, GlobalConfig); err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}
		fmt.Printf("📄 配置文件已加载: %s\n", path)
	} else {
		fmt.Println("⚠️  未找到配置文件,使用默认配置")
	}

	// 3. 应用环境变量（覆盖配置文件）
	if loadFromEnv() {
		fmt.Println("✅ 环境变量已应用")
	}

	// 4. 应用命令行参数（最高优先级,覆盖所有）
	if cliPort > 0 {
		GlobalConfig.Server.Port = cliPort
	}

	return nil
}

// loadFromEnv 从系统环境变量加载配置
// 返回 true 表示检测到环境变量并使用了
func loadFromEnv() bool {
	envUsed := false

	// 服务器端口
	if portStr := os.Getenv("NGINX_PLUGS_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			GlobalConfig.Server.Port = port
			envUsed = true
			fmt.Printf("[ENV] NGINX_PLUGS_PORT = %d\n", port)
		}
	}

	// 模板文件固定为 confdir/proxy.conf.template
	if confDir := os.Getenv("NGINX_CONF_DIR"); confDir != "" {
		GlobalConfig.Nginx.ConfDir = confDir
		envUsed = true
		fmt.Printf("[ENV] NGINX_CONF_DIR = %s\n", confDir)
	}

	// 日志级别
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		GlobalConfig.Log.Level = logLevel
		envUsed = true
		fmt.Printf("[ENV] LOG_LEVEL = %s\n", logLevel)
	}

	// 阿里云AK/SK
	if akID := os.Getenv("ALIYUN_ACCESS_KEY_ID"); akID != "" {
		GlobalConfig.Aliyun.AccessKeyID = akID
		envUsed = true
		fmt.Printf("[ENV] ALIYUN_ACCESS_KEY_ID = %s\n", akID)
	}

	if akSecret := os.Getenv("ALIYUN_ACCESS_KEY_SECRET"); akSecret != "" {
		GlobalConfig.Aliyun.AccessKeySecret = akSecret
		envUsed = true
		fmt.Println("[ENV] ALIYUN_ACCESS_KEY_SECRET = ***")
	}

	if recordType := os.Getenv("ALIYUN_RECORD_TYPE"); recordType != "" {
		GlobalConfig.Aliyun.RecordType = recordType
		envUsed = true
		fmt.Printf("[ENV] ALIYUN_RECORD_TYPE = %s\n", recordType)
	}

	if recordValue := os.Getenv("ALIYUN_RECORD_VALUE"); recordValue != "" {
		GlobalConfig.Aliyun.RecordValue = recordValue
		envUsed = true
		fmt.Printf("[ENV] ALIYUN_RECORD_VALUE = %s\n", recordValue)
	}

	// nginx重载命令
	if reloadCmd := os.Getenv("NGINX_RELOAD_CMD"); reloadCmd != "" {
		GlobalConfig.NginxCmd.Reload = reloadCmd
		envUsed = true
		fmt.Printf("[ENV] NGINX_RELOAD_CMD = %s\n", reloadCmd)
	}

	// 域名列表（环境变量中用逗号分隔，如 NGINX_DOMAINS=hzlsg.com,xdysh.com）
	if domainsStr := os.Getenv("NGINX_DOMAINS"); domainsStr != "" {
		GlobalConfig.Domains = []DomainOption{}
		for _, d := range strings.Split(domainsStr, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				GlobalConfig.Domains = append(GlobalConfig.Domains, DomainOption{Name: d})
			}
		}
		envUsed = true
		fmt.Printf("[ENV] NGINX_DOMAINS = %v\n", domainsStr)
	}

	return envUsed
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
			ConfDir: "/etc/nginx/conf.d/cmdb-plugs/",
		}
	}
	return GlobalConfig.Nginx
}

// GetTemplatePath 获取模板文件路径（固定为 confdir/proxy.conf.template）
func GetTemplatePath() string {
	if GlobalConfig == nil {
		return "/etc/nginx/conf.d/cmdb-plugs/proxy.conf.template"
	}
	return filepath.Join(GlobalConfig.Nginx.ConfDir, "proxy.conf.template")
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

// GetAliyunConfig 获取阿里云配置
func GetAliyunConfig() AliyunConfig {
	if GlobalConfig == nil {
		return AliyunConfig{}
	}
	return GlobalConfig.Aliyun
}

// GetDomainOptions 获取域名选项列表
func GetDomainOptions() []DomainOption {
	if GlobalConfig == nil {
		return []DomainOption{}
	}
	return GlobalConfig.Domains
}

// GetNginxCmdConfig 获取nginx命令配置
func GetNginxCmdConfig() NginxCmdConfig {
	if GlobalConfig == nil {
		return NginxCmdConfig{Reload: "nginx -s reload"}
	}
	if GlobalConfig.NginxCmd.Reload == "" {
		return NginxCmdConfig{Reload: "nginx -s reload"}
	}
	return GlobalConfig.NginxCmd
}

// GetSSHTargets 获取SSH目标服务器列表
func GetSSHTargets() []SSHTarget {
	if GlobalConfig == nil {
		return []SSHTarget{}
	}
	return GlobalConfig.SSHTargets
}
