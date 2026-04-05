package middleware

import (
	"time"

	"al-plugs/logger"

	"github.com/gin-gonic/gin"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 请求路径
		path := c.Request.URL.Path
		// 请求方法
		method := c.Request.Method
		// 客户端IP
		clientIP := c.ClientIP()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		// 执行时间
		latency := endTime.Sub(startTime)
		// 状态码
		statusCode := c.Writer.Status()

		// 记录日志
		logger.Infof("请求处理完成",
			"方法: %s, 路径: %s, 状态码: %d, 耗时: %v, IP: %s",
			method, path, statusCode, latency, clientIP)
	}
}
