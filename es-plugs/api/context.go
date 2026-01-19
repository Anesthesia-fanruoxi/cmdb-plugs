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

// ContextHandler 上下文查询处理器
func ContextHandler(w http.ResponseWriter, r *http.Request) {
	common.Logger.Info("POST请求 - 上下文查询")

	startTime := time.Now()

	// 解析请求
	var req model.ContextRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("读取请求体失败: %v", err))
		common.Error(w, http.StatusBadRequest, "读取请求失败")
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		common.Logger.Error(fmt.Sprintf("解析请求失败: %v", err))
		common.Error(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	// 验证参数
	if req.Index == "" {
		common.Error(w, http.StatusBadRequest, "缺少必要参数: index")
		return
	}
	if req.DocID == "" {
		common.Error(w, http.StatusBadRequest, "缺少必要参数: doc_id")
		return
	}

	// 设置默认排序字段
	if req.SortField == "" {
		req.SortField = "@timestamp"
	}

	common.Logger.Info(fmt.Sprintf("上下文查询 - 索引: %s, 文档ID: %s, before: %d, after: %d, 排序字段: %s",
		req.Index, req.DocID, req.Before, req.After, req.SortField))

	// 执行上下文查询
	result, err := getDocumentContext(req)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("上下文查询失败: %v", err))
		common.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 计算总耗时
	result.Took = int(time.Since(startTime).Milliseconds())

	common.Logger.Info(fmt.Sprintf("上下文查询成功，总记录数: %d", result.Total))

	common.Success(w, result)
}

// getDocumentContext 获取文档上下文
func getDocumentContext(req model.ContextRequest) (*model.ContextResponse, error) {
	cfg := config.GetESConfig()

	result := &model.ContextResponse{
		Before: make([]map[string]interface{}, 0),
		After:  make([]map[string]interface{}, 0),
	}

	// 1. 获取中心文档
	centerDoc, err := getCenterDocument(cfg, req.Index, req.DocID)
	if err != nil {
		return nil, err
	}

	result.Center = centerDoc

	// 2. 从中心文档获取排序字段的值
	source, ok := centerDoc["_source"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("中心文档格式错误")
	}

	sortValue, exists := source[req.SortField]
	if !exists {
		return nil, fmt.Errorf("中心文档不包含排序字段: %s", req.SortField)
	}

	// 规范化排序值
	normalizedSortValue := normalizeSortValue(sortValue)

	// 3. 获取中心文档之前的记录
	if req.Before > 0 {
		beforeDocs, err := getBeforeDocuments(cfg, req, normalizedSortValue)
		if err != nil {
			common.Logger.Error(fmt.Sprintf("获取before文档失败: %v", err))
		} else {
			result.Before = beforeDocs
			result.BeforeTotal = len(beforeDocs)
		}
	}

	// 4. 获取中心文档之后的记录
	if req.After > 0 {
		afterDocs, err := getAfterDocuments(cfg, req, normalizedSortValue)
		if err != nil {
			common.Logger.Error(fmt.Sprintf("获取after文档失败: %v", err))
		} else {
			result.After = afterDocs
			result.AfterTotal = len(afterDocs)
		}
	}

	// 5. 计算总数
	result.Total = result.BeforeTotal + 1 + result.AfterTotal

	return result, nil
}

// getCenterDocument 获取中心文档
func getCenterDocument(cfg config.ESConfig, index, docID string) (map[string]interface{}, error) {
	esURL := fmt.Sprintf("%s/%s/_doc/%s", cfg.Host, index, docID)

	req, err := http.NewRequest("GET", esURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	if cfg.Username != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("中心文档不存在，ID: %s", docID)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES返回错误: %s", string(respBody))
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(respBody, &doc); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return doc, nil
}

// getBeforeDocuments 获取中心文档之前的记录
func getBeforeDocuments(cfg config.ESConfig, req model.ContextRequest, sortValue interface{}) ([]map[string]interface{}, error) {
	// 构建查询：排序字段值小于中心文档
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							req.SortField: map[string]interface{}{
								"lt": sortValue,
							},
						},
					},
				},
			},
		},
		"size": req.Before,
		"sort": []map[string]interface{}{
			{
				req.SortField: map[string]interface{}{
					"order": "desc", // 降序
				},
			},
		},
	}

	// 添加字段过滤
	if req.Source != nil {
		query["_source"] = req.Source
	}

	// 发送请求
	esURL := fmt.Sprintf("%s/%s/_search", cfg.Host, req.Index)
	jsonData, _ := json.Marshal(query)

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

	var esResp map[string]interface{}
	if err := json.Unmarshal(respBody, &esResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取hits
	hits, ok := esResp["hits"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	hitsArray, ok := hits["hits"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	// 结果需要反转（因为是降序查询，但要升序返回）
	docs := make([]map[string]interface{}, 0, len(hitsArray))
	for i := len(hitsArray) - 1; i >= 0; i-- {
		if hit, ok := hitsArray[i].(map[string]interface{}); ok {
			docs = append(docs, hit)
		}
	}

	return docs, nil
}

// getAfterDocuments 获取中心文档之后的记录
func getAfterDocuments(cfg config.ESConfig, req model.ContextRequest, sortValue interface{}) ([]map[string]interface{}, error) {
	// 构建查询：排序字段值大于中心文档
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							req.SortField: map[string]interface{}{
								"gt": sortValue,
							},
						},
					},
				},
			},
		},
		"size": req.After,
		"sort": []map[string]interface{}{
			{
				req.SortField: map[string]interface{}{
					"order": "asc", // 升序
				},
			},
		},
	}

	// 添加字段过滤
	if req.Source != nil {
		query["_source"] = req.Source
	}

	// 发送请求
	esURL := fmt.Sprintf("%s/%s/_search", cfg.Host, req.Index)
	jsonData, _ := json.Marshal(query)

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

	var esResp map[string]interface{}
	if err := json.Unmarshal(respBody, &esResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取hits
	hits, ok := esResp["hits"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	hitsArray, ok := hits["hits"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	docs := make([]map[string]interface{}, 0, len(hitsArray))
	for _, hit := range hitsArray {
		if hitMap, ok := hit.(map[string]interface{}); ok {
			docs = append(docs, hitMap)
		}
	}

	return docs, nil
}

// normalizeSortValue 规范化排序值
func normalizeSortValue(value interface{}) interface{} {
	switch v := value.(type) {
	case float64:
		// 转换为整数（如果是时间戳）
		return int64(v)
	case float32:
		return int64(v)
	default:
		return v
	}
}
