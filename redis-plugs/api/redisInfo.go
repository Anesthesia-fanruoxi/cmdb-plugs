package api

import (
	"github.com/gin-gonic/gin"
	"redis-plugs/common"
)

// RedisInfo 获取 Redis 状态信息
func RedisInfo(c *gin.Context) {
	info, err := common.GetRedisInfo(c.Request.Context())
	if err != nil {
		common.ServerError(c, err.Error())
		return
	}
	common.Success(c, info)
}
