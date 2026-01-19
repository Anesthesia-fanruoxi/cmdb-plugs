package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"redis-plugs/common"
)

// RedisTree 获取 Key 树
func RedisTree(c *gin.Context) {
	key := c.Query("key")

	client, err := common.GetClient()
	if err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	var tree interface{}

	// 判断是模糊搜索还是精确前缀展开
	if strings.Contains(key, "*") {
		// 模糊搜索模式
		tree, err = common.SearchKeyTree(c.Request.Context(), client, key, ":", 10000)
	} else {
		// 精确前缀展开模式（懒加载）
		pattern := "*"
		prefix := ""
		if key != "" {
			prefix = key
			pattern = key + ":*"
		}
		tree, err = common.GetKeyTree(c.Request.Context(), client, pattern, ":", prefix, 10000)
	}

	if err != nil {
		common.ServerError(c, "扫描 Key 失败: "+err.Error())
		return
	}

	common.Success(c, tree)
}
