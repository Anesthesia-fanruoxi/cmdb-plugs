package router

import (
	"cron-plugs/common"
	"cron-plugs/model"
	"encoding/json"
	"log"
	"net/http"
)

// HealthHandler 健康检查
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ExecuteHandler 执行脚本接口
func ExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	var req model.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ResponseJSON(w, 400, "请求参数解析失败: "+err.Error(), nil)
		return
	}

	// 参数校验
	if req.JobID == 0 {
		common.ResponseJSON(w, 400, "job_id不能为空", nil)
		return
	}
	if req.TaskKey == "" {
		common.ResponseJSON(w, 400, "task_key不能为空", nil)
		return
	}
	if req.CallbackURL == "" {
		common.ResponseJSON(w, 400, "callback_url不能为空", nil)
		return
	}

	// 异步执行
	go common.ExecuteLocalScript(req, req.CallbackURL)

	// 立即返回响应
	common.ResponseJSON(w, 0, "任务已接收,正在执行", map[string]interface{}{
		"job_id":    req.JobID,
		"task_name": req.TaskName,
		"task_key":  req.TaskKey,
	})

	log.Printf("任务已接收: JobID=%d, 任务名=%s, 任务标识=%s", req.JobID, req.TaskName, req.TaskKey)
}
