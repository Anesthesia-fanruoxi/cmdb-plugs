package api

import (
	"bytes"
	"encoding/json"
	"es-plugs/common"
	"es-plugs/config"
	"es-plugs/model"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SearchAPI ES搜索接口
func SearchAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 读取请求体
	reqBodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("读取请求体失败: %v", err))
		common.Error(w, http.StatusBadRequest, "读取请求体失败")
		return
	}

	// 输出原始请求体
	common.Logger.Info(fmt.Sprintf("POST请求 - 原始请求体: %s", string(reqBodyBytes)))

	// 解析请求
	var req model.SearchRequest
	if err := json.Unmarshal(reqBodyBytes, &req); err != nil {
		common.Logger.Error(fmt.Sprintf("解析请求失败: %v", err))
		common.Error(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	// 验证参数
	if req.Index == "" {
		common.Error(w, http.StatusBadRequest, "索引名不能为空")
		return
	}

	// 从配置中读取size（配置本身已限制不超过3000）
	limitConfig := config.GetLimitConfig()
	req.Size = limitConfig.MaxSize

	// 设置时间字段
	timeField := req.TimeField
	if timeField == "" {
		timeField = "@timestamp"
	}

	// 构建DSL查询
	dsl, err := buildSearchDSL(req)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("构建DSL失败: %v", err))
		common.Error(w, http.StatusBadRequest, fmt.Sprintf("构建查询失败: %v", err))
		return
	}

	// 输出DSL
	dslJSON, _ := json.Marshal(dsl)
	common.Logger.Info(fmt.Sprintf("构建的DSL: %s", string(dslJSON)))

	// 执行查询
	result, err := executeSearch(req.Index, dsl)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("ES查询失败: %v", err))
		common.Error(w, http.StatusInternalServerError, fmt.Sprintf("查询失败: %v", err))
		return
	}

	common.Success(w, result)
}

// buildSearchDSL 构建ES查询DSL（使用高级语法解析器）
func buildSearchDSL(req model.SearchRequest) (map[string]interface{}, error) {
	// 设置时间字段，如果没有指定则使用默认值
	timeField := req.TimeField
	if timeField == "" {
		timeField = "@timestamp" // 默认时间字段
	}

	// 使用QueryBuilder构建查询
	qb := &QueryBuilder{
		Index:      req.Index,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		TimeField:  timeField,
		TimeFormat: "epoch_millis", // 使用毫秒时间戳格式
		Size:       req.Size,
	}

	// 解析关键词（支持复杂语法：AND、OR、NOT、字段查询等）
	if err := qb.ParseKeyword(req.Keyword); err != nil {
		return nil, fmt.Errorf("解析关键词失败: %v", err)
	}

	// 设置排序方向，默认为降序
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	// 验证排序方向参数
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// 构建完整的DSL
	dsl := map[string]interface{}{
		"size":             req.Size,
		"track_total_hits": true, // 获取准确的总数，不限制在10000
		"query":            qb.Query,
		"sort": []interface{}{
			map[string]interface{}{
				timeField: map[string]interface{}{
					"order": sortOrder,
				},
			},
		},
	}

	return dsl, nil
}

// executeSearch 执行ES搜索
func executeSearch(index string, dsl map[string]interface{}) (*model.SearchResponse, error) {
	esConfig := config.GetESConfig()

	// 构建请求URL
	url := fmt.Sprintf("%s/%s/_search", esConfig.Host, index)

	// 序列化DSL
	dslJSON, err := json.Marshal(dsl)
	if err != nil {
		return nil, fmt.Errorf("序列化DSL失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(dslJSON))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if esConfig.Username != "" && esConfig.Password != "" {
		req.SetBasicAuth(esConfig.Username, esConfig.Password)
	}

	// 发送请求
	client := &http.Client{
		Timeout: time.Duration(esConfig.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析ES响应
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("解析ES响应失败: %w", err)
	}

	// 构建结果
	result := &model.SearchResponse{
		QueryTime: int(rawResponse["took"].(float64)),
		TimedOut:  rawResponse["timed_out"].(bool),
		Hits:      make([]model.SearchHit, 0),
	}

	// 提取总数
	if hitsMap, ok := rawResponse["hits"].(map[string]interface{}); ok {
		if total, ok := hitsMap["total"].(map[string]interface{}); ok {
			result.TotalHits = int(total["value"].(float64))
		}

		// 提取命中记录
		if hits, ok := hitsMap["hits"].([]interface{}); ok {
			for _, hit := range hits {
				hitMap := hit.(map[string]interface{})
				searchHit := model.SearchHit{
					Index:  hitMap["_index"].(string),
					ID:     hitMap["_id"].(string),
					Source: hitMap["_source"].(map[string]interface{}),
				}
				if score, ok := hitMap["_score"]; ok && score != nil {
					if scoreVal, ok := score.(float64); ok {
						searchHit.Score = scoreVal
					}
				}
				result.Hits = append(result.Hits, searchHit)
			}
		}
	}

	// 设置实际返回条数
	result.ActualHits = len(result.Hits)

	return result, nil
}
