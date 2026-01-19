package model

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	JobID       int64  `json:"job_id"`       // 任务ID
	TaskName    string `json:"task_name"`    // 任务名称
	TaskKey     string `json:"task_key"`     // 任务标识,作为脚本名
	CallbackURL string `json:"callback_url"` // 回调地址
}
