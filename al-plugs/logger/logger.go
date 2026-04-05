package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	logger      *log.Logger
	logLevel    = INFO
	logLevelMap = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
)

func init() {
	// 初始化日志器，输出到标准输出
	logger = log.New(os.Stdout, "", 0)
}

// SetLogLevel 设置日志级别
func SetLogLevel(level LogLevel) {
	logLevel = level
}

// getCallerInfo 获取调用者信息
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// formatLog 格式化日志
func formatLog(level LogLevel, skip int, format string, args ...interface{}) string {
	now := time.Now().Format("2006/01/02 15:04:05")
	caller := getCallerInfo(skip)
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s %s: %s", logLevelMap[level], now, caller, message)
}

// Debug 调试日志
func Debug(format string, args ...interface{}) {
	if logLevel <= DEBUG {
		logger.Println(formatLog(DEBUG, 3, format, args...))
	}
}

// Info 信息日志
func Info(format string, args ...interface{}) {
	if logLevel <= INFO {
		logger.Println(formatLog(INFO, 3, format, args...))
	}
}

// Warn 警告日志
func Warn(format string, args ...interface{}) {
	if logLevel <= WARN {
		logger.Println(formatLog(WARN, 3, format, args...))
	}
}

// Error 错误日志
func Error(format string, args ...interface{}) {
	if logLevel <= ERROR {
		logger.Println(formatLog(ERROR, 3, format, args...))
	}
}

// Infof 信息日志（带详细信息）
func Infof(message string, details string, args ...interface{}) {
	if logLevel <= INFO {
		detailMsg := fmt.Sprintf(details, args...)
		fullMsg := fmt.Sprintf("%s - %s", message, detailMsg)
		logger.Println(formatLog(INFO, 3, fullMsg))
	}
}
