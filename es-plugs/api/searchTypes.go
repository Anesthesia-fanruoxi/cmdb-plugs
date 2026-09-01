package api

// QueryBuilder ES查询构建器
type QueryBuilder struct {
	Index     string                 // 索引名称
	StartTime int64                  // 开始时间（毫秒时间戳，含，0表示不限制）
	EndTime   int64                  // 结束时间（毫秒时间戳，不含，0表示不限制）
	TimeField string                 // 时间字段
	Size      int                    // 返回结果数量
	Query     map[string]interface{} // 查询条件
}

// Token 表示查询语句中的一个标记
type Token struct {
	Type      string // field, operator, value, logic, group
	Value     string
	SubTokens []Token // 用于 group 类型，存储括号内的 tokens
}
