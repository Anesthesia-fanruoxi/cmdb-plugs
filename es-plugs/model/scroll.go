package model

// ScrollRequest 滚动查询请求
type ScrollRequest struct {
	Action     string                 `json:"action"`      // 操作类型: init, continue, clear
	Index      string                 `json:"index"`       // 索引名（init时需要）
	StartTime  string                 `json:"start_time"`  // 开始时间（init时可选）
	EndTime    string                 `json:"end_time"`    // 结束时间（init时可选）
	TimeField  string                 `json:"time_field"`  // 时间字段名（init时可选）
	Keyword    string                 `json:"keyword"`     // 关键词搜索（init时可选）
	Query      map[string]interface{} `json:"query"`       // 自定义查询条件（init时可选，优先级低于时间+关键词）
	Size       int                    `json:"size"`        // 每次返回的记录数（init时可选，默认100）
	ScrollTime string                 `json:"scroll_time"` // 滚动上下文保持时间（可选，默认1m）
	ScrollID   string                 `json:"scroll_id"`   // 滚动ID（continue/clear时需要）
	Sort       []interface{}          `json:"sort"`        // 排序（init时可选）
	Source     interface{}            `json:"_source"`     // 字段过滤（init时可选）
}

// ScrollResponse 滚动查询响应
type ScrollResponse struct {
	ScrollID   string                   `json:"scroll_id"`   // 滚动ID
	QueryTime  int                      `json:"query_time"`  // 查询耗时（毫秒）
	TotalHits  int                      `json:"total_hits"`  // 总记录数
	ActualHits int                      `json:"actual_hits"` // 实际返回的记录条数
	Hits       []map[string]interface{} `json:"hits"`        // 命中记录
	Cleared    bool                     `json:"cleared"`     // 是否已清除（clear操作时返回）
}
