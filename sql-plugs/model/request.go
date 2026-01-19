package model

// SQLSearchRequest SQL查询请求
type SQLSearchRequest struct {
	DB    string `json:"dbName"` // 数据库名称（agent使用dbName）
	Query string `json:"query"`  // SQL查询语句（支持用分号分隔多个SQL）
}

// StructureRequest 数据库结构查询请求
type StructureRequest struct {
	Type string                 `json:"type"` // 查询类型：db=数据库列表, tb=表列表/表结构
	Op   map[string]interface{} `json:"op"`   // 操作参数
}
