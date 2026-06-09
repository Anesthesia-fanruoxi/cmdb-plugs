package common

import (
	"fmt"
	"log"
	"nginx-plugs/config"
	"os"
	"strings"
)

// CustomLogger 自定义日志结构
type CustomLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	level       string // info 或 error
}

// Logger 全局日志实例
var Logger *CustomLogger

// InitLogger 初始化日志（输出到控制台）
func InitLogger() {
	logConfig := config.GetLogConfig()
	level := strings.ToLower(logConfig.Level)

	if level != "info" && level != "error" {
		level = "info"
	}

	Logger = &CustomLogger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger:  log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		level:       level,
	}

	Logger.infoLogger.Output(2, fmt.Sprintf("日志级别: %s", level))
}

// Info 记录信息日志
func (l *CustomLogger) Info(msg string) {
	if l.level == "error" {
		return
	}
	l.infoLogger.Output(2, msg)
}

// Infof 记录格式化信息日志
func (l *CustomLogger) Infof(format string, args ...interface{}) {
	if l.level == "error" {
		return
	}
	l.infoLogger.Output(2, fmt.Sprintf(format, args...))
}

// Warn 记录警告日志
func (l *CustomLogger) Warn(msg string) {
	l.warnLogger.Output(2, msg)
}

// Warnf 记录格式化警告日志
func (l *CustomLogger) Warnf(format string, args ...interface{}) {
	l.warnLogger.Output(2, fmt.Sprintf(format, args...))
}

// Error 记录错误日志
func (l *CustomLogger) Error(msg string) {
	l.errorLogger.Output(2, msg)
}

// Errorf 记录格式化错误日志
func (l *CustomLogger) Errorf(format string, args ...interface{}) {
	l.errorLogger.Output(2, fmt.Sprintf(format, args...))
}
