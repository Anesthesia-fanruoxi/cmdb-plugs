package common

import (
	"encoding/json"
	"net/http"
)

// Response 统一返回结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success 成功响应
func Success(w http.ResponseWriter, data interface{}) {
	response := Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
	writeJSON(w, http.StatusOK, response)
}

// Error 错误响应
func Error(w http.ResponseWriter, statusCode int, message string) {
	response := Response{
		Code:    statusCode,
		Message: message,
	}
	writeJSON(w, statusCode, response)
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		Logger.Error("响应JSON编码失败: " + err.Error())
	}
}
