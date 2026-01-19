package api

import (
	"github.com/gin-gonic/gin"
	"redis-plugs/common"
)

// RedisKey 获取 Key 详情
func RedisKey(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		common.BadRequest(c, "key 参数不能为空")
		return
	}

	client, err := common.GetClient()
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	info, err := common.GetKeyInfo(c.Request.Context(), client, key)
	if err != nil {
		common.ServerError(c, err.Error())
		return
	}

	common.Success(c, info)
}

// RedisKeyDelete 删除 Key
func RedisKeyDelete(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		common.BadRequest(c, "key 参数不能为空")
		return
	}

	client, err := common.GetClient()
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	if err := common.DeleteKey(c.Request.Context(), client, key); err != nil {
		common.ServerError(c, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "删除成功"})
}
