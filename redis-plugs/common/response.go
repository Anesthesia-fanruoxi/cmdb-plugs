package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"redis-plugs/models"
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, models.APIResponse{
		Code:    code,
		Message: message,
	})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// ServerError 服务器错误
func ServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
