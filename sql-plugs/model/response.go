package model

// SQLSearchResponse SQL查询响应（批量查询）
type SQLSearchResponse struct {
	Results []QueryResult `json:"results"` // 查询结果列表
	Total   int           `json:"total"`   // 查询数量
	Took    int64         `json:"took"`    // 总耗时（毫秒）
	DBName  string        `json:"db_name"` // 数据库名
}

// QueryResult 单个查询结果
type QueryResult struct {
	QueryID string          `json:"query_id,omitempty"` // 查询ID（用于取消查询）
	Columns []string        `json:"columns"`            // 列名
	Rows    [][]interface{} `json:"rows"`               // 数据行
	Total   int             `json:"total"`              // 该查询的真实记录数
	Took    int64           `json:"took"`               // 该查询的耗时（毫秒）
	DBName  string          `json:"db_name"`            // 数据库名
}

// TableColumn 表字段信息
type TableColumn struct {
	Field   string  `json:"field"`   // 字段名
	Type    string  `json:"type"`    // 字段类型
	Null    string  `json:"null"`    // 是否允许NULL
	Key     string  `json:"key"`     // 索引类型
	Default *string `json:"default"` // 默认值（可为null）
	Extra   string  `json:"extra"`   // 额外信息
	Comment string  `json:"comment"` // 字段注释
}

// TableIndex 表索引信息
type TableIndex struct {
	Name    string   `json:"name"`    // 索引名
	Type    string   `json:"type"`    // 索引类型
	Columns []string `json:"columns"` // 索引字段
	Unique  bool     `json:"unique"`  // 是否唯一索引
	Comment string   `json:"comment"` // 索引注释
}

// TableStructure 表结构详细信息
type TableStructure struct {
	Name        string        `json:"name"`         // 表名
	Comment     string        `json:"comment"`      // 表注释
	Columns     []TableColumn `json:"columns"`      // 字段列表
	Indexes     []TableIndex  `json:"indexes"`      // 索引列表
	CreateSQL   string        `json:"create_sql"`   // 建表语句
	Engine      string        `json:"engine"`       // 存储引擎
	Collation   string        `json:"collation"`    // 字符集排序规则
	CreateTime  string        `json:"create_time"`  // 创建时间
	PreviewData *QueryResult  `json:"preview_data"` // 预览数据（前20条）
}
