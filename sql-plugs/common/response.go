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

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(w http.ResponseWriter, message string, data interface{}) {
	response := Response{
		Code:    200,
		Message: message,
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
	// 调试日志：记录错误响应内容
	Logger.Infof("返回错误响应 - HTTP状态码: %d, Code: %d, Message: %s", statusCode, response.Code, response.Message)
	writeJSON(w, statusCode, response)
}

// ErrorWithCode 错误响应（HTTP 200 + 业务错误码）
// 用于避免代理层拦截非 200 状态码
// 返回格式：{"code": 200, "message": "error", "data": "具体错误信息"}
func ErrorWithCode(w http.ResponseWriter, code int, message string) {
	response := Response{
		Code:    200, // HTTP层面永远返回200
		Message: "error",
		Data:    message, // 错误信息直接放data
	}
	Logger.Infof("返回业务错误 - HTTP状态码: 200, 业务Code: %d, Message: %s", code, message)
	writeJSON(w, http.StatusOK, response)
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	// 序列化为JSON并写入响应
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		Logger.Error("响应JSON编码失败: " + err.Error())
		return
	}

	// 写入响应
	if _, err := w.Write(jsonBytes); err != nil {
		Logger.Error("写入响应失败: " + err.Error())
	}
}
