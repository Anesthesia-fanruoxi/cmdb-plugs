package model

import "time"

// ExecuteResponse 执行响应(异步接收)
type ExecuteResponse struct {
	Code int         `json:"code"` // 状态码 0:成功 其他:失败
	Msg  string      `json:"msg"`  // 提示信息
	Data interface{} `json:"data"` // 返回数据
}

// TaskResult 回调数据
type TaskResult struct {
	JobID      int64     `json:"job_id"`      // 任务ID
	TaskName   string    `json:"task_name"`   // 任务名称
	TaskKey    string    `json:"task_key"`    // 任务标识符
	ExecStatus int       `json:"exec_status"` // 执行状态 0:执行中 1:成功 2:失败
	Result     string    `json:"result"`      // 执行结果 success/failed/timeout/error
	ErrorMsg   string    `json:"error_msg"`   // 错误信息
	ExecLog    string    `json:"exec_log"`    // 执行日志
	StartTime  time.Time `json:"start_time"`  // 开始时间
	EndTime    time.Time `json:"end_time"`    // 结束时间
	Duration   int       `json:"duration"`    // 执行时长(秒)
}
