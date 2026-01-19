package common

import (
	"bytes"
	"context"
	"cron-plugs/model"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ExecuteLocalScript 执行本地脚本并回调
func ExecuteLocalScript(req model.ExecuteRequest, callbackURL string) {
	startTime := time.Now()
	result := model.TaskResult{
		JobID:     req.JobID,
		TaskName:  req.TaskName,
		TaskKey:   req.TaskKey,
		StartTime: startTime,
	}

	// 使用task_key作为脚本名
	scriptName := req.TaskKey + ".sh"

	// 验证脚本路径
	scriptPath, err := ValidateScriptPath(scriptName)
	if err != nil {
		result.ExecStatus = 2
		result.ErrorMsg = fmt.Sprintf("脚本验证失败: %v", err)
		result.Result = "failed"
		result.EndTime = time.Now()
		result.Duration = int(result.EndTime.Sub(startTime).Seconds())
		SendCallback(callbackURL, result)
		return
	}

	// 执行脚本
	executeScript(scriptPath, &result)

	result.EndTime = time.Now()
	result.Duration = int(result.EndTime.Sub(startTime).Seconds())
	log.Printf("任务执行完成: JobID=%d, 脚本=%s, 状态=%d, 耗时=%ds",
		req.JobID, scriptName, result.ExecStatus, result.Duration)

	// 回调
	SendCallback(callbackURL, result)
}

// ValidateScriptPath 验证脚本路径
func ValidateScriptPath(scriptName string) (string, error) {
	// 防止路径穿越
	if strings.Contains(scriptName, "..") || strings.ContainsAny(scriptName, "/\\") {
		return "", fmt.Errorf("非法脚本名称: %s", scriptName)
	}

	// 构建完整路径
	scriptPath := filepath.Join(model.ScriptDir, scriptName)

	// 检查文件是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("脚本不存在: /data/script/%s", scriptName)
	}

	return scriptPath, nil
}

// executeScript 执行脚本
func executeScript(scriptPath string, result *model.TaskResult) {
	// 不设置超时限制，允许长时间运行的任务（如备份、打包等）
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("开始执行: JobID=%d, 脚本=%s", result.JobID, scriptPath)
	err := cmd.Run()

	// 合并输出作为执行日志
	result.ExecLog = stdout.String()
	if stderr.Len() > 0 {
		if len(result.ExecLog) > 0 {
			result.ExecLog += "\n"
		}
		result.ExecLog += stderr.String()
	}

	// 处理执行结果
	if err != nil {
		result.ExecStatus = 2
		if exitError, ok := err.(*exec.ExitError); ok {
			result.Result = "failed"
			result.ErrorMsg = fmt.Sprintf("退出码: %d", exitError.ExitCode())
		} else {
			result.Result = "error"
			result.ErrorMsg = err.Error()
		}
	} else {
		result.ExecStatus = 1
		result.Result = "success"
	}
}
