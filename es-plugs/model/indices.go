package model

// IndexMappingRequest 索引映射请求
type IndexMappingRequest struct {
	Index string `json:"index"` // 索引模式，如 "logs-*"
}

// IndexMappingResponse 索引映射响应
type IndexMappingResponse struct {
	Indices []string               `json:"indices"` // 匹配的所有索引列表
	Fields  map[string]interface{} `json:"fields"`  // 最新索引的字段映射（简化后）
}
