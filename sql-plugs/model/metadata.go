package model

// MetadataRequest 元数据查询请求
type MetadataRequest struct {
	DBName  string `json:"dbName"`  // 可选：数据库名（为空则查询所有库）
	Refresh bool   `json:"refresh"` // 可选：是否强制刷新缓存
}

// AllDatabasesMetadata 所有数据库元数据（跨库查询）
type AllDatabasesMetadata struct {
	Databases []DatabaseMetadata `json:"databases"`  // 各数据库元数据
	Total     int                `json:"total"`      // 数据库总数
	CachedAt  string             `json:"cached_at"`  // 缓存时间
	FromCache bool               `json:"from_cache"` // 是否来自缓存
	Took      int64              `json:"took"`       // 采集耗时（毫秒）
	Warnings  []string           `json:"warnings"`   // 警告信息
}

// DatabaseMetadata 数据库元数据（Navicat式全量快照）
type DatabaseMetadata struct {
	DBName    string        `json:"db_name"`    // 数据库名
	Tables    []TableMeta   `json:"tables"`     // 表列表
	Views     []ViewMeta    `json:"views"`      // 视图列表
	Triggers  []TriggerMeta `json:"triggers"`   // 触发器列表
	Routines  []RoutineMeta `json:"routines"`   // 存储过程/函数列表
	UDFs      []UDFMeta     `json:"udfs"`       // 用户自定义函数列表
	CachedAt  string        `json:"cached_at"`  // 缓存时间
	FromCache bool          `json:"from_cache"` // 是否来自缓存
	Took      int64         `json:"took"`       // 采集耗时（毫秒）
	Warnings  []string      `json:"warnings"`   // 警告信息（权限不足等）
}

// TableMeta 表元数据
type TableMeta struct {
	Name        string           `json:"name"`         // 表名
	Comment     string           `json:"comment"`      // 表注释
	Engine      string           `json:"engine"`       // 存储引擎
	Collation   string           `json:"collation"`    // 字符集排序规则
	RowCount    int64            `json:"row_count"`    // 预估行数
	DataLength  int64            `json:"data_length"`  // 数据大小（字节）
	CreateTime  string           `json:"create_time"`  // 创建时间
	UpdateTime  string           `json:"update_time"`  // 更新时间
	Columns     []ColumnMeta     `json:"columns"`      // 字段列表
	PrimaryKey  *PrimaryKeyMeta  `json:"primary_key"`  // 主键
	ForeignKeys []ForeignKeyMeta `json:"foreign_keys"` // 外键列表
	Indexes     []IndexMeta      `json:"indexes"`      // 索引列表
}

// ViewMeta 视图元数据
type ViewMeta struct {
	Name       string `json:"name"`       // 视图名
	Definition string `json:"definition"` // 视图定义SQL
	Definer    string `json:"definer"`    // 定义者
	Updatable  bool   `json:"updatable"`  // 是否可更新
	Comment    string `json:"comment"`    // 注释
}

// ColumnMeta 字段元数据
type ColumnMeta struct {
	Name         string  `json:"name"`           // 字段名
	OrdinalPos   int     `json:"ordinal_pos"`    // 字段顺序位置
	DataType     string  `json:"data_type"`      // 数据类型（如 varchar, int）
	ColumnType   string  `json:"column_type"`    // 完整类型（如 varchar(255), int unsigned）
	Nullable     bool    `json:"nullable"`       // 是否允许NULL
	DefaultValue *string `json:"default_value"`  // 默认值
	IsPrimaryKey bool    `json:"is_primary_key"` // 是否主键
	IsAutoIncr   bool    `json:"is_auto_incr"`   // 是否自增
	CharMaxLen   *int64  `json:"char_max_len"`   // 字符最大长度
	NumPrecision *int64  `json:"num_precision"`  // 数值精度
	NumScale     *int64  `json:"num_scale"`      // 数值小数位
	CharacterSet string  `json:"character_set"`  // 字符集
	Collation    string  `json:"collation"`      // 排序规则
	ColumnKey    string  `json:"column_key"`     // 索引类型（PRI/UNI/MUL）
	Extra        string  `json:"extra"`          // 额外信息（auto_increment等）
	Comment      string  `json:"comment"`        // 字段注释
}

// PrimaryKeyMeta 主键元数据
type PrimaryKeyMeta struct {
	Name    string   `json:"name"`    // 主键名（MySQL通常为PRIMARY）
	Columns []string `json:"columns"` // 主键字段列表（支持联合主键）
}

// ForeignKeyMeta 外键元数据
type ForeignKeyMeta struct {
	Name       string   `json:"name"`        // 外键名
	Columns    []string `json:"columns"`     // 外键字段列表
	RefTable   string   `json:"ref_table"`   // 引用表名
	RefColumns []string `json:"ref_columns"` // 引用字段列表
	OnUpdate   string   `json:"on_update"`   // 更新规则
	OnDelete   string   `json:"on_delete"`   // 删除规则
}

// IndexMeta 索引元数据
type IndexMeta struct {
	Name      string   `json:"name"`       // 索引名
	Columns   []string `json:"columns"`    // 索引字段列表（按顺序）
	IndexType string   `json:"index_type"` // 索引类型（BTREE/HASH/FULLTEXT等）
	IsUnique  bool     `json:"is_unique"`  // 是否唯一索引
	IsPrimary bool     `json:"is_primary"` // 是否主键索引
	Comment   string   `json:"comment"`    // 索引注释
}

// TriggerMeta 触发器元数据
type TriggerMeta struct {
	Name      string `json:"name"`      // 触发器名
	Table     string `json:"table"`     // 关联表
	Event     string `json:"event"`     // 事件（INSERT/UPDATE/DELETE）
	Timing    string `json:"timing"`    // 时机（BEFORE/AFTER）
	Statement string `json:"statement"` // 触发器语句
	Definer   string `json:"definer"`   // 定义者
	Created   string `json:"created"`   // 创建时间
}

// RoutineMeta 存储过程/函数元数据
type RoutineMeta struct {
	Name       string          `json:"name"`       // 名称
	Type       string          `json:"type"`       // 类型（PROCEDURE/FUNCTION）
	Definer    string          `json:"definer"`    // 定义者
	DataType   string          `json:"data_type"`  // 返回类型（仅FUNCTION）
	Parameters []ParameterMeta `json:"parameters"` // 参数列表
	Definition string          `json:"definition"` // 定义SQL
	Created    string          `json:"created"`    // 创建时间
	Modified   string          `json:"modified"`   // 修改时间
	Comment    string          `json:"comment"`    // 注释
}

// ParameterMeta 参数元数据
type ParameterMeta struct {
	Name       string `json:"name"`        // 参数名
	Mode       string `json:"mode"`        // 模式（IN/OUT/INOUT）
	DataType   string `json:"data_type"`   // 数据类型
	OrdinalPos int    `json:"ordinal_pos"` // 参数顺序
}

// UDFMeta 用户自定义函数元数据
type UDFMeta struct {
	Name       string `json:"name"`        // 函数名
	ReturnType string `json:"return_type"` // 返回类型
	Type       string `json:"type"`        // 类型（function/aggregate）
	Library    string `json:"library"`     // 共享库路径
}
