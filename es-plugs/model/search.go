package model

// SearchRequest 简化的查询请求
type SearchRequest struct {
	Index     string `json:"index"`      // 索引名
	StartTime string `json:"start_time"` // 开始时间 "2025-03-10 00:00:00"
	EndTime   string `json:"end_time"`   // 结束时间 "2025-03-11 00:00:00"
	TimeField string `json:"time_field"` // 时间字段名，如 "timestamp", "create_time" 等
	Keyword   string `json:"keyword"`    // 关键词搜索
	Size      int    `json:"size"`       // 返回条数
	SortOrder string `json:"sort_order"` // 时间排序方向：asc（升序）或 desc（降序），默认为desc
}

// SearchHit ES查询命中记录
type SearchHit struct {
	Index  string                 `json:"_index"`
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

// SearchResponse 查询响应
type SearchResponse struct {
	QueryTime  int         `json:"query_time"`  // 查询耗时（毫秒）
	TimedOut   bool        `json:"timed_out"`   // 是否超时
	TotalHits  int         `json:"total_hits"`  // 匹配的总记录数
	ActualHits int         `json:"actual_hits"` // 实际返回的记录条数
	Hits       []SearchHit `json:"hits"`        // 命中记录列表
}
