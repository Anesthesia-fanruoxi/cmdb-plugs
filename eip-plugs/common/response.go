package common

import (
	"encoding/json"
	"net/http"
)

// Response 统一响应体结构
type Response struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`    // 数据
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	response := Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
	WriteJSON(w, http.StatusOK, response)
}

// Error 错误响应
func Error(w http.ResponseWriter, code int, message string) {
	response := Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}
	WriteJSON(w, http.StatusOK, response)
}

// WriteJSON 写入JSON响应
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
