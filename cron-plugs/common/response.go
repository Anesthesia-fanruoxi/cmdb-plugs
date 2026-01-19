package common

import (
	"cron-plugs/model"
	"encoding/json"
	"net/http"
)

// ResponseJSON 返回标准JSON响应
func ResponseJSON(w http.ResponseWriter, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.ExecuteResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}
