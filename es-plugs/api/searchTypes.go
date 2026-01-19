package api

// QueryBuilder ES查询构建器
type QueryBuilder struct {
	Index      string                 // 索引名称
	StartTime  string                 // 开始时间
	EndTime    string                 // 结束时间
	TimeField  string                 // 时间字段
	TimeFormat string                 // 时间格式，如 "iso8601"、"epoch_millis"
	Size       int                    // 返回结果数量
	Query      map[string]interface{} // 查询条件
}

// Token 表示查询语句中的一个标记
type Token struct {
	Type      string // field, operator, value, logic, group
	Value     string
	SubTokens []Token // 用于 group 类型，存储括号内的 tokens
}
