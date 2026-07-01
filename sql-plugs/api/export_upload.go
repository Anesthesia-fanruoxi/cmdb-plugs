package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sql-plugs/common"
	"time"
)

// SetupStep 导出步骤状态
type SetupStep struct {
	Status      string `json:"status"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Duration    int64  `json:"duration"`
	Description string `json:"description"`
}

// CallbackPayload 回调CMDB的完整数据结构
type CallbackPayload struct {
	ID           string                `json:"id"`
	Type         string                `json:"type"`
	TypeText     string                `json:"type_text"`
	Status       string                `json:"status"`
	StatusText   string                `json:"status_text"`
	Progress     int                   `json:"progress"`
	CurrentStep  string                `json:"current_step"`
	TotalCount   int                   `json:"total_count"`
	ResultKey    string                `json:"result_key"`
	ErrorMessage string                `json:"error_message"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
	Setup        map[string]*SetupStep `json:"setup"`
}

// ExportTaskState 导出任务状态跟踪器
type ExportTaskState struct {
	payload        *CallbackPayload
	callbackURL    string
	completedSteps int // 已完成步骤数
}

// NewExportTaskState 创建导出任务状态
func NewExportTaskState(taskID, callbackURL, dbName string) *ExportTaskState {
	setup := map[string]*SetupStep{
		"setup1": {Status: "waiting", Description: "初始化完成"},
		"setup2": {Status: "waiting", Description: "获取查询数据"},
		"setup3": {Status: "waiting", Description: "上传文件"},
		"setup4": {Status: "waiting", Description: "完成通知"},
	}
	return &ExportTaskState{
		payload: &CallbackPayload{
			ID:         taskID,
			Type:       "sql_export",
			TypeText:   "SQL导出",
			Status:     "running",
			StatusText: "执行中",
			CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
			Setup:      setup,
		},
		callbackURL: callbackURL,
	}
}

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// StartStep 标记步骤开始
func (s *ExportTaskState) StartStep(step string) {
	if st, ok := s.payload.Setup[step]; ok {
		st.Status = "running"
		st.StartTime = nowStr()
		s.payload.CurrentStep = step
	}
}

// CompleteStep 标记步骤完成，外层progress +25%
func (s *ExportTaskState) CompleteStep(step string) {
	if st, ok := s.payload.Setup[step]; ok {
		st.Status = "success"
		st.EndTime = nowStr()
		if st.StartTime != "" {
			if t1, e1 := time.Parse("2006-01-02 15:04:05", st.StartTime); e1 == nil {
				if t2, e2 := time.Parse("2006-01-02 15:04:05", st.EndTime); e2 == nil {
					st.Duration = t2.Sub(t1).Milliseconds()
				}
			}
		}
		s.completedSteps++
		s.payload.Progress = s.completedSteps * 25
	}
}

// SendCallback 发送步骤变化回调
func (s *ExportTaskState) SendCallback() {
	s.payload.UpdatedAt = nowStr()
	jsonData, err := json.Marshal(s.payload)
	if err != nil {
		common.Logger.Errorf("导出任务[%s] 序列化回调失败: %v", s.payload.ID, err)
		return
	}

	common.Logger.Infof("导出任务[%s] 发送回调 - status: %s, progress: %d%%, step: %s",
		s.payload.ID, s.payload.Status, s.payload.Progress, s.payload.CurrentStep)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(s.callbackURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		common.Logger.Errorf("导出任务[%s] 回调请求失败: %v", s.payload.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		common.Logger.Warnf("导出任务[%s] 回调返回HTTP %d: %s", s.payload.ID, resp.StatusCode, string(body))
		return
	}
	common.Logger.Infof("导出任务[%s] 回调成功", s.payload.ID)
}

// FailTask 标记任务失败并回调
func (s *ExportTaskState) FailTask(errMsg string) {
	s.payload.Status = "failed"
	s.payload.StatusText = "执行失败"
	s.payload.ErrorMessage = errMsg
	s.SendCallback()
}

// SucceedTask 标记任务成功并回调
func (s *ExportTaskState) SucceedTask(fileID string) {
	s.payload.Status = "success"
	s.payload.StatusText = "执行成功"
	s.payload.Progress = 100
	s.payload.ResultKey = fileID
	s.SendCallback()
}

// uploadToCMDB 上传文件到CMDB，返回file_id
func uploadToCMDB(uploadURL, filePath string) (string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", fileInfo.Name())
	if err != nil {
		return "", fmt.Errorf("创建form文件失败: %w", err)
	}
	io.Copy(part, file)

	writer.WriteField("category", "sql-export")
	writer.WriteField("resource", "sql-plugs")
	writer.WriteField("is_private", "true")
	writer.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", uploadURL, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("上传返回HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	fileID := extractFileID(respBody)
	if fileID == "" {
		return "", fmt.Errorf("无法从上传响应中提取file_id: %s", string(respBody))
	}
	return fileID, nil
}

// extractFileID 从上传响应中提取文件标识
func extractFileID(respBody []byte) string {
	var resp map[string]interface{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return ""
	}
	data, ok := resp["data"]
	if !ok || data == nil {
		return ""
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		if s, ok := data.(string); ok {
			return s
		}
		return ""
	}
	if fid, ok := dataMap["file_id"].(string); ok && fid != "" {
		return fid
	}
	if uuid, ok := dataMap["uuid"].(string); ok && uuid != "" {
		return uuid
	}
	if files, ok := dataMap["files"].([]interface{}); ok && len(files) > 0 {
		if f, ok := files[0].(string); ok && f != "" {
			return f
		}
	}
	return ""
}
