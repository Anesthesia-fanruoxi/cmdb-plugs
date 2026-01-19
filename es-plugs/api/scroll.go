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
)

// ScrollHandler 滚动查询处理器
func ScrollHandler(w http.ResponseWriter, r *http.Request) {
	// 解析请求
	var req model.ScrollRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("读取请求体失败: %v", err))
		common.Error(w, http.StatusBadRequest, "读取请求失败")
		return
	}
	defer r.Body.Close()

	common.Logger.Info(fmt.Sprintf("POST请求 - 滚动查询，原始请求体: %s", string(body)))

	if err := json.Unmarshal(body, &req); err != nil {
		common.Logger.Error(fmt.Sprintf("解析请求失败: %v", err))
		common.Error(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	// 验证 action 参数
	if req.Action == "" {
		common.Error(w, http.StatusBadRequest, "缺少必要参数: action (init/continue/clear)")
		return
	}

	common.Logger.Info(fmt.Sprintf("滚动查询操作: %s", req.Action))

	var result interface{}

	switch req.Action {
	case "init":
		// 初始化滚动查询
		result, err = initScroll(req)
	case "continue":
		// 继续滚动查询
		result, err = continueScroll(req)
	case "clear":
		// 清除滚动上下文
		result, err = clearScroll(req)
	default:
		common.Error(w, http.StatusBadRequest, fmt.Sprintf("不支持的操作类型: %s", req.Action))
		return
	}

	if err != nil {
		common.Logger.Error(fmt.Sprintf("滚动查询失败: %v", err))
		common.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	common.Logger.Info("滚动查询成功")

	common.Success(w, result)
}

// initScroll 初始化滚动查询
func initScroll(req model.ScrollRequest) (*model.ScrollResponse, error) {
	// 验证参数
	if req.Index == "" {
		return nil, fmt.Errorf("缺少必要参数: index")
	}

	// 设置默认值
	if req.Size <= 0 {
		req.Size = 1000
	}

	// 应用最大值限制
	req.Size = config.ApplySizeLimit(req.Size)

	if req.ScrollTime == "" {
		req.ScrollTime = "1m"
	}

	// 构建查询条件
	var query map[string]interface{}

	// 优先使用时间范围+关键词方式（类似普通查询）
	if req.StartTime != "" || req.EndTime != "" || req.Keyword != "" {
		// 设置时间字段默认值
		timeField := req.TimeField
		if timeField == "" {
			timeField = "@timestamp"
		}

		// 使用 QueryBuilder 构建查询
		qb := &QueryBuilder{
			Index:      req.Index,
			StartTime:  req.StartTime,
			EndTime:    req.EndTime,
			TimeField:  timeField,
			TimeFormat: "epoch_millis",
			Size:       req.Size,
		}

		// 解析关键词
		if req.Keyword != "" {
			if err := qb.ParseKeyword(req.Keyword); err != nil {
				return nil, fmt.Errorf("解析关键词失败: %v", err)
			}
		}

		query = qb.Query
	} else if req.Query != nil {
		// 否则使用传入的自定义查询
		query = req.Query
	} else {
		// 如果都没有，使用 match_all
		query = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// 构建查询体
	queryBody := map[string]interface{}{
		"query":            query,
		"size":             req.Size,
		"track_total_hits": true, // 获取准确的总数
	}

	// 添加排序
	if req.Sort != nil && len(req.Sort) > 0 {
		queryBody["sort"] = req.Sort
	} else {
		// 默认按 _doc 排序，最高效
		queryBody["sort"] = []map[string]interface{}{
			{"_doc": map[string]interface{}{"order": "asc"}},
		}
	}

	// 添加字段过滤
	if req.Source != nil {
		queryBody["_source"] = req.Source
	}

	// 输出构建的DSL
	dslJSON, _ := json.Marshal(queryBody)
	common.Logger.Info(fmt.Sprintf("滚动查询初始化 - 索引: %s, 时间范围: %s ~ %s, 关键词: %s, 每批: %d",
		req.Index, req.StartTime, req.EndTime, req.Keyword, req.Size))
	common.Logger.Info(fmt.Sprintf("构建的DSL: %s", string(dslJSON)))

	// 发送请求到 ES
	cfg := config.GetESConfig()
	esURL := fmt.Sprintf("%s/%s/_search?scroll=%s", cfg.Host, req.Index, req.ScrollTime)

	jsonData := dslJSON
	httpReq, err := http.NewRequest("POST", esURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.Username != "" && cfg.Password != "" {
		httpReq.SetBasicAuth(cfg.Username, cfg.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES返回错误: %s", string(respBody))
	}

	// 解析响应
	var esResp map[string]interface{}
	if err := json.Unmarshal(respBody, &esResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取 scroll_id
	scrollID, ok := esResp["_scroll_id"].(string)
	if !ok {
		return nil, fmt.Errorf("未找到滚动ID")
	}

	// 提取结果
	result := &model.ScrollResponse{
		ScrollID:  scrollID,
		QueryTime: int(esResp["took"].(float64)),
	}

	// 提取总数和记录
	if hits, ok := esResp["hits"].(map[string]interface{}); ok {
		// 提取总数
		if total, ok := hits["total"].(map[string]interface{}); ok {
			result.TotalHits = int(total["value"].(float64))
		}

		// 提取命中记录
		if hitsArray, ok := hits["hits"].([]interface{}); ok {
			for _, hit := range hitsArray {
				if hitMap, ok := hit.(map[string]interface{}); ok {
					result.Hits = append(result.Hits, hitMap)
				}
			}
		}
	}

	// 设置实际返回条数
	result.ActualHits = len(result.Hits)

	return result, nil
}

// continueScroll 继续滚动查询
func continueScroll(req model.ScrollRequest) (*model.ScrollResponse, error) {
	// 验证参数
	if req.ScrollID == "" {
		return nil, fmt.Errorf("缺少必要参数: scroll_id")
	}

	// 设置默认值
	if req.ScrollTime == "" {
		req.ScrollTime = "1m"
	}

	// 构建请求体
	queryBody := map[string]interface{}{
		"scroll":    req.ScrollTime,
		"scroll_id": req.ScrollID,
	}

	// 发送请求到 ES
	cfg := config.GetESConfig()
	esURL := fmt.Sprintf("%s/_search/scroll", cfg.Host)

	jsonData, _ := json.Marshal(queryBody)
	httpReq, err := http.NewRequest("POST", esURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.Username != "" && cfg.Password != "" {
		httpReq.SetBasicAuth(cfg.Username, cfg.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES返回错误: %s", string(respBody))
	}

	// 解析响应
	var esResp map[string]interface{}
	if err := json.Unmarshal(respBody, &esResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取 scroll_id
	scrollID, ok := esResp["_scroll_id"].(string)
	if !ok {
		return nil, fmt.Errorf("未找到滚动ID")
	}

	// 提取结果
	result := &model.ScrollResponse{
		ScrollID:  scrollID,
		QueryTime: int(esResp["took"].(float64)),
	}

	// 提取总数和记录
	if hits, ok := esResp["hits"].(map[string]interface{}); ok {
		// 提取总数
		if total, ok := hits["total"].(map[string]interface{}); ok {
			result.TotalHits = int(total["value"].(float64))
		}

		// 提取命中记录
		if hitsArray, ok := hits["hits"].([]interface{}); ok {
			for _, hit := range hitsArray {
				if hitMap, ok := hit.(map[string]interface{}); ok {
					result.Hits = append(result.Hits, hitMap)
				}
			}
		}
	}

	// 设置实际返回条数
	result.ActualHits = len(result.Hits)

	// 如果没有更多结果，自动清除滚动上下文
	if len(result.Hits) == 0 {
		clearScroll(model.ScrollRequest{ScrollID: scrollID})
		result.ScrollID = "" // 清空表示结束
	}

	return result, nil
}

// clearScroll 清除滚动上下文
func clearScroll(req model.ScrollRequest) (*model.ScrollResponse, error) {
	// 验证参数
	if req.ScrollID == "" {
		return nil, fmt.Errorf("缺少必要参数: scroll_id")
	}

	// 构建请求体
	queryBody := map[string]interface{}{
		"scroll_id": []string{req.ScrollID},
	}

	// 发送请求到 ES
	cfg := config.GetESConfig()
	esURL := fmt.Sprintf("%s/_search/scroll", cfg.Host)

	jsonData, _ := json.Marshal(queryBody)
	httpReq, err := http.NewRequest("DELETE", esURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.Username != "" && cfg.Password != "" {
		httpReq.SetBasicAuth(cfg.Username, cfg.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ES返回错误: %s", string(respBody))
	}

	return &model.ScrollResponse{
		ScrollID: req.ScrollID,
		Cleared:  true,
	}, nil
}
