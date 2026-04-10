package router

import (
	"io"

	"al-plugs/api"
	"al-plugs/config"
	"al-plugs/middleware"
	"al-plugs/model"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(cfg *config.Config) *gin.Engine {
	// 禁用 Gin 的默认日志输出
	gin.DefaultWriter = io.Discard

	r := gin.New()

	// 使用 Recovery 中间件（不输出日志）
	r.Use(gin.Recovery())

	// 使用自定义请求日志中间件
	r.Use(middleware.Logger())

	// 创建API实例
	accountAPI := api.NewAccountAPI(cfg)
	resourceAPI := api.NewResourceAPI(cfg)

	// API路由组
	apiGroup := r.Group("/api/")
	{
		// 账户相关接口
		account := apiGroup.Group("/account")
		{
			account.GET("/balance", accountAPI.QueryBalance)
		}

		// 资源相关接口
		resource := apiGroup.Group("/resource")
		{
			resource.GET("/ecs/expiry", resourceAPI.QueryEcsExpiry)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, model.NewSuccessResponse(gin.H{
			"status": "ok",
		}))
	})

	return r
}
