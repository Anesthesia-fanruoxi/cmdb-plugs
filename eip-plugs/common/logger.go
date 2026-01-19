package common

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger 定制化日志记录器
type Logger struct {
	logger *log.Logger
}

var logger *Logger

func init() {
	logger = &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

// GetLogger 获取日志实例
func GetLogger() *Logger {
	return logger
}

// formatMessage 格式化日志消息
func (l *Logger) formatMessage(level, message string) string {
	return fmt.Sprintf("[%s] [%s] %s", time.Now().Format("2006-01-02 15:04:05"), level, message)
}

// Info 信息日志
func (l *Logger) Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Println(l.formatMessage("INFO", msg))
}

// Error 错误日志
func (l *Logger) Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Println(l.formatMessage("ERROR", msg))
}

// Warn 警告日志
func (l *Logger) Warn(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Println(l.formatMessage("WARN", msg))
}

// Debug 调试日志
func (l *Logger) Debug(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Println(l.formatMessage("DEBUG", msg))
}
