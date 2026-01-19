package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
)

type Config struct {
	Server    ServerConfig   `yaml:"server"`
	Databases DatabaseConfig `yaml:"databases"`
	Log       LogConfig      `yaml:"log"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string     `yaml:"host"`     // 数据库主机
	Port     int        `yaml:"port"`     // 数据库端口
	User     string     `yaml:"user"`     // 用户名
	Password string     `yaml:"password"` // 密码
	Database string     `yaml:"database"` // 数据库名
	Charset  string     `yaml:"charset"`  // 字符集（默认utf8mb4）
	Pool     PoolConfig `yaml:"pool"`     // 连接池配置
}

type PoolConfig struct {
	MaxOpenConns    int `yaml:"max_open_conns"`     // 最大打开连接数
	MaxIdleConns    int `yaml:"max_idle_conns"`     // 最大空闲连接数
	ConnMaxLifetime int `yaml:"conn_max_lifetime"`  // 连接最大生命周期（秒）
	ConnMaxIdleTime int `yaml:"conn_max_idle_time"` // 连接最大空闲时间（秒）
}

type LogConfig struct {
	Level string `yaml:"level"` // info 或 error
}

// MySQLConfig 简化的MySQL配置（用于返回）
type MySQLConfig struct {
	Addr     string
	Port     int
	User     string
	Password string
	Database string
	Charset  string
	Pool     PoolConfig
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(path string) error {
	// 初始化默认配置
	GlobalConfig = &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Databases: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "",
			Database: "test",
			Charset:  "utf8mb4",
			Pool: PoolConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    10,
				ConnMaxLifetime: 3600,
				ConnMaxIdleTime: 600,
			},
		},
		Log: LogConfig{
			Level: "info",
		},
	}

	// 尝试读取配置文件（可选）
	data, err := os.ReadFile(path)
	if err == nil {
		// 配置文件存在，解析它
		if err := yaml.Unmarshal(data, GlobalConfig); err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}
		fmt.Printf("📄 配置文件已加载: %s\n", path)
	} else {
		fmt.Println("⚠️  未找到配置文件，使用默认配置")
	}
	// 如果文件不存在，使用默认配置（不报错）

	// 环境变量覆盖配置（优先级最高）
	fmt.Println() // 空行分隔
	overrideWithEnv()

	return nil
}

// overrideWithEnv 使用环境变量覆盖配置（优先级最高）
// 支持环境变量：MYSQL_ADDR, MYSQL_PORT, MYSQL_DB, MYSQL_USER, MYSQL_PASSWORD, LOG_LEVEL
func overrideWithEnv() {
	envUsed := false

	// MySQL配置 - 使用新的环境变量名
	if addr := os.Getenv("MYSQL_ADDR"); addr != "" {
		GlobalConfig.Databases.Host = addr
		envUsed = true
		fmt.Printf("[ENV] MYSQL_ADDR = %s\n", addr)
	}
	if portStr := os.Getenv("MYSQL_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			GlobalConfig.Databases.Port = port
			envUsed = true
			fmt.Printf("[ENV] MYSQL_PORT = %d\n", port)
		}
	}
	if user := os.Getenv("MYSQL_USER"); user != "" {
		GlobalConfig.Databases.User = user
		envUsed = true
		fmt.Printf("[ENV] MYSQL_USER = %s\n", user)
	}
	if password := os.Getenv("MYSQL_PASSWORD"); password != "" {
		GlobalConfig.Databases.Password = password
		envUsed = true
		fmt.Printf("[ENV] MYSQL_PASSWORD = ******\n")
	}
	if database := os.Getenv("MYSQL_DB"); database != "" {
		GlobalConfig.Databases.Database = database
		envUsed = true
		fmt.Printf("[ENV] MYSQL_DB = %s\n", database)
	}

	// 日志配置
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
		return ServerConfig{Port: 8090}
	}
	return GlobalConfig.Server
}

// GetDatabaseConfig 获取数据库配置（返回简化的MySQLConfig）
func GetDatabaseConfig() MySQLConfig {
	if GlobalConfig == nil {
		return MySQLConfig{
			Addr:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "",
			Database: "test",
			Charset:  "utf8mb4",
			Pool: PoolConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    10,
				ConnMaxLifetime: 3600,
				ConnMaxIdleTime: 600,
			},
		}
	}
	// 将DatabaseConfig转换为MySQLConfig
	return MySQLConfig{
		Addr:     GlobalConfig.Databases.Host,
		Port:     GlobalConfig.Databases.Port,
		User:     GlobalConfig.Databases.User,
		Password: GlobalConfig.Databases.Password,
		Database: GlobalConfig.Databases.Database,
		Charset:  GlobalConfig.Databases.Charset,
		Pool:     GlobalConfig.Databases.Pool,
	}
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
