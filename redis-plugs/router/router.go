package router

import (
	"github.com/gin-gonic/gin"
	"redis-plugs/api"
	"redis-plugs/common"
)

// Setup 初始化路由
func Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(common.LoggerMiddleware())
	r.Use(common.CorsMiddleware())
	r.Use(gin.Recovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API 路由
	registerAPI(r)

	return r
}

// registerAPI 注册 API 路由
func registerAPI(r *gin.Engine) {
	g := r.Group("/api")
	{
		g.GET("/info", api.RedisInfo)
		g.GET("/tree", api.RedisTree)
		g.GET("/key", api.RedisKey)
		g.DELETE("/delete", api.RedisKeyDelete)
	}
}
