package model

// ContextRequest 上下文查询请求
type ContextRequest struct {
	Index     string      `json:"index"`      // 索引名
	DocID     string      `json:"doc_id"`     // 中心文档ID
	Before    int         `json:"before"`     // 获取前面多少条
	After     int         `json:"after"`      // 获取后面多少条
	SortField string      `json:"sort_field"` // 排序字段
	Source    interface{} `json:"_source"`    // 字段过滤（可选）
}

// ContextResponse 上下文查询响应
type ContextResponse struct {
	Before      []map[string]interface{} `json:"before"`       // 中心文档之前的记录
	Center      map[string]interface{}   `json:"center"`       // 中心文档
	After       []map[string]interface{} `json:"after"`        // 中心文档之后的记录
	Total       int                      `json:"total"`        // 总记录数
	BeforeTotal int                      `json:"before_total"` // 前面记录数
	AfterTotal  int                      `json:"after_total"`  // 后面记录数
	Took        int                      `json:"took"`         // 查询耗时（毫秒）
}
