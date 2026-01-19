package common

import (
	"database/sql"
	"fmt"
	"sql-plugs/config"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbConnection  *sql.DB
	dbMutex       sync.RWMutex
	dbInitialized bool
)

// GetDB 获取数据库连接
func GetDB() (*sql.DB, error) {
	dbMutex.RLock()
	if dbInitialized && dbConnection != nil {
		dbMutex.RUnlock()
		return dbConnection, nil
	}
	dbMutex.RUnlock()

	// 连接不存在，创建新连接
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// 双重检查
	if dbInitialized && dbConnection != nil {
		return dbConnection, nil
	}

	// 从配置中获取数据库信息
	dbConfig := config.GetDatabaseConfig()

	// 打印详细的数据库配置信息
	Logger.Info("========== 数据库连接配置 ==========")
	Logger.Infof("地址(MYSQL_ADDR):    %s", dbConfig.Addr)
	Logger.Infof("端口(MYSQL_PORT):    %d", dbConfig.Port)
	Logger.Infof("数据库(MYSQL_DB):    %s", dbConfig.Database)
	Logger.Infof("用户(MYSQL_USER):    %s", dbConfig.User)
	Logger.Infof("密码(MYSQL_PASSWORD): ******")
	Logger.Infof("字符集:              %s", dbConfig.Charset)
	Logger.Infof("最大连接数:          %d", dbConfig.Pool.MaxOpenConns)
	Logger.Infof("最大空闲数:          %d", dbConfig.Pool.MaxIdleConns)
	Logger.Info("===================================")

	// 构建MySQL DSN连接字符串
	dsn := buildMySQLDSN(dbConfig)

	// 打开MySQL数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 设置字符集（确保中文不乱码）
	charsetCommands := []string{
		"SET NAMES utf8mb4",
		"SET CHARACTER SET utf8mb4",
		"SET character_set_client = utf8mb4",
		"SET character_set_connection = utf8mb4",
		"SET character_set_results = utf8mb4",
	}
	for _, cmd := range charsetCommands {
		_, err = db.Exec(cmd)
		if err != nil {
			Logger.Warnf("执行%s失败: %v", cmd, err)
		}
	}

	// 验证字符集设置
	var charset, collation string
	err = db.QueryRow("SELECT @@character_set_client, @@collation_connection").Scan(&charset, &collation)
	if err == nil {
		Logger.Infof("✓ 连接字符集: %s, 排序规则: %s", charset, collation)
	} else {
		Logger.Warnf("无法验证字符集: %v", err)
	}

	// 设置连接池参数
	poolConfig := dbConfig.Pool
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(poolConfig.ConnMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(poolConfig.ConnMaxIdleTime) * time.Second)

	dbConnection = db
	dbInitialized = true
	Logger.Infof("✓ MySQL连接成功: %s:%d/%s", dbConfig.Addr, dbConfig.Port, dbConfig.Database)

	return db, nil
}

// GetDBStats 获取数据库连接池状态
func GetDBStats() (sql.DBStats, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	if !dbInitialized || dbConnection == nil {
		return sql.DBStats{}, fmt.Errorf("数据库连接未初始化")
	}

	return dbConnection.Stats(), nil
}

// CloseDB 关闭数据库连接
func CloseDB() {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if dbConnection != nil {
		if err := dbConnection.Close(); err != nil {
			Logger.Errorf("MySQL连接关闭失败: %v", err)
		} else {
			Logger.Info("MySQL连接已关闭")
		}
		dbConnection = nil
		dbInitialized = false
	}
}

// buildMySQLDSN 构建MySQL DSN连接字符串
// 格式: user:password@tcp(addr:port)/database?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=false&loc=Local
func buildMySQLDSN(cfg config.MySQLConfig) string {
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	// 添加完整的UTF-8字符集参数，防止中文乱码
	// parseTime=false 禁止自动解析时间类型，保持数据库原始格式
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&collation=utf8mb4_general_ci&parseTime=false&loc=Local",
		cfg.User, cfg.Password, cfg.Addr, cfg.Port, cfg.Database, charset)
}
