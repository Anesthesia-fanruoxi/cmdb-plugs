package common

import (
	"bytes"
	"cron-plugs/model"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// SendCallback 发送回调
func SendCallback(callbackURL string, data model.TaskResult) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("回调数据序列化失败: %v", err)
		return
	}

	// 创建请求
	req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("回调请求创建失败: %s, 错误: %v", callbackURL, err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CronPlugs-Agent/1.0")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("回调失败: %s, 错误: %v", callbackURL, err)
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("回调成功: JobID=%d, URL=%s, 响应=%s", data.JobID, callbackURL, string(body))
	} else {
		log.Printf("回调失败: JobID=%d, URL=%s, 状态码=%d, 响应=%s",
			data.JobID, callbackURL, resp.StatusCode, string(body))
	}
}
