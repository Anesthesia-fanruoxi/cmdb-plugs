package api

import (
	"encoding/json"
	"es-plugs/common"
	"es-plugs/config"
	"es-plugs/model"
	"fmt"
	"io"
	"net/http"
	"sort"
)

// GetIndexMappingHandler 获取索引列表和最新索引的映射
func GetIndexMappingHandler(w http.ResponseWriter, r *http.Request) {
	common.Logger.Info("GET请求 - 获取索引映射")

	// 从 URL query 参数获取索引模式
	indexPattern := r.URL.Query().Get("index")

	// 验证参数
	if indexPattern == "" {
		common.Error(w, http.StatusBadRequest, "索引模式不能为空")
		return
	}

	common.Logger.Info(fmt.Sprintf("开始获取索引映射 - 索引模式: %s", indexPattern))

	// 步骤1: 获取索引列表
	indices, err := getIndicesList(indexPattern)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("获取索引列表失败: %v", err))
		common.Error(w, http.StatusInternalServerError, fmt.Sprintf("获取索引列表失败: %v", err))
		return
	}

	if len(indices) == 0 {
		common.Error(w, http.StatusNotFound, "未找到匹配的索引")
		return
	}

	// 对索引列表排序（确保获取最新的）
	sort.Strings(indices)
	totalCount := len(indices)

	// 只保留最新的10个索引，避免数据量过大
	if len(indices) > 10 {
		indices = indices[len(indices)-10:]
		common.Logger.Info(fmt.Sprintf("找到 %d 个索引，仅返回最新的 10 个", totalCount))
	}

	latestIndex := indices[len(indices)-1]
	common.Logger.Info(fmt.Sprintf("最新索引: %s", latestIndex))

	// 步骤2: 获取最新索引的映射
	mappings, err := getIndexMappings(latestIndex)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("获取索引映射失败: %v", err))
		common.Error(w, http.StatusInternalServerError, fmt.Sprintf("获取索引映射失败: %v", err))
		return
	}

	// 步骤3: 简化字段结构
	simplifiedFields := simplifyMappings(mappings)

	// 构建响应
	result := model.IndexMappingResponse{
		Indices: indices,
		Fields:  simplifiedFields,
	}

	common.Logger.Info(fmt.Sprintf("成功获取索引映射，返回 %d 个索引（总共 %d 个）", len(indices), totalCount))

	common.Success(w, result)
}

// getIndicesList 获取索引列表
func getIndicesList(index string) ([]string, error) {
	cfg := config.GetESConfig()
	esURL := fmt.Sprintf("%s/_cat/indices/%s?format=json&h=index", cfg.Host, index)

	// 创建请求
	req, err := http.NewRequest("GET", esURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置认证
	if cfg.Username != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES返回错误: %s", string(body))
	}

	// 解析响应
	var indexList []map[string]interface{}
	if err := json.Unmarshal(body, &indexList); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取索引名称
	var indices []string
	for _, item := range indexList {
		if indexName, ok := item["index"].(string); ok {
			indices = append(indices, indexName)
		}
	}

	return indices, nil
}

// getIndexMappings 获取指定索引的映射
func getIndexMappings(indexName string) (map[string]interface{}, error) {
	cfg := config.GetESConfig()
	esURL := fmt.Sprintf("%s/%s/_mapping", cfg.Host, indexName)

	// 创建请求
	req, err := http.NewRequest("GET", esURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置认证
	if cfg.Username != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求ES失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ES返回错误: %s", string(body))
	}

	// 解析响应
	var mappings map[string]interface{}
	if err := json.Unmarshal(body, &mappings); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return mappings, nil
}

// simplifyMappings 简化映射结构，只保留字段名和类型
func simplifyMappings(mappings map[string]interface{}) map[string]interface{} {
	// ES返回的格式通常是: {"index_name": {"mappings": {"properties": {...}}}}
	// 我们需要提取 properties 并简化

	// 遍历每个索引
	for _, indexMapping := range mappings {
		if indexMap, ok := indexMapping.(map[string]interface{}); ok {
			// 查找 mappings
			if mappingsObj, ok := indexMap["mappings"].(map[string]interface{}); ok {
				// 查找 properties
				if properties, ok := mappingsObj["properties"].(map[string]interface{}); ok {
					// 简化 properties
					simplified := make(map[string]interface{})
					simplifyProperties(properties, simplified)

					return map[string]interface{}{
						"properties": simplified,
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"properties": map[string]interface{}{},
	}
}

// simplifyProperties 递归简化字段属性，只保留类型信息
func simplifyProperties(properties map[string]interface{}, result map[string]interface{}) {
	for fieldName, fieldInfo := range properties {
		if fieldMap, ok := fieldInfo.(map[string]interface{}); ok {
			// 提取字段类型
			if fieldType, ok := fieldMap["type"].(string); ok {
				result[fieldName] = map[string]interface{}{
					"type": fieldType,
				}
			} else if nestedProperties, ok := fieldMap["properties"].(map[string]interface{}); ok {
				// 如果是嵌套对象，递归处理
				nestedResult := make(map[string]interface{})
				simplifyProperties(nestedProperties, nestedResult)
				result[fieldName] = map[string]interface{}{
					"type":       "object",
					"properties": nestedResult,
				}
			}
		}
	}
}
