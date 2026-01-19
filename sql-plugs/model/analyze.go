package model

// DMLAnalysis DML语句分析
type DMLAnalysis struct {
	TargetTable  string   `json:"target_table"`
	AffectedCols []string `json:"affected_cols"`
	DataSource   string   `json:"data_source"`
	HasWhere     bool     `json:"has_where"`
	WherePreview string   `json:"where_preview"`
	EstimateRows string   `json:"estimate_rows"`
	RiskLevel    string   `json:"risk_level"`
	RiskReason   string   `json:"risk_reason"`
}

// DDLAnalysis DDL语句分析
type DDLAnalysis struct {
	Operation    string      `json:"operation"`
	ObjectType   string      `json:"object_type"`
	ObjectName   string      `json:"object_name"`
	ColumnsDef   []string    `json:"columns_def"`
	AlterActions []string    `json:"alter_actions"`
	RiskLevel    string      `json:"risk_level"`
	RiskReason   string      `json:"risk_reason"`
	Details      *DDLDetails `json:"details,omitempty"`
}

// DDLDetails DDL详细信息
type DDLDetails struct {
	ColumnCount   int            `json:"column_count,omitempty"`
	Columns       []ColumnDetail `json:"columns,omitempty"`
	TableComment  string         `json:"table_comment,omitempty"`
	Engine        string         `json:"engine,omitempty"`
	Charset       string         `json:"charset,omitempty"`
	Collation     string         `json:"collation,omitempty"`
	PrimaryKey    string         `json:"primary_key,omitempty"`
	HasIndex      bool           `json:"has_index"`
	IndexCount    int            `json:"index_count,omitempty"`
	Indexes       []IndexDetail  `json:"indexes,omitempty"`
	ForeignKeys   []string       `json:"foreign_keys,omitempty"`
	AddColumns    []ColumnDetail `json:"add_columns,omitempty"`
	ModifyColumns []ColumnDetail `json:"modify_columns,omitempty"`
	DropColumns   []string       `json:"drop_columns,omitempty"`
	AddIndexes    []IndexDetail  `json:"add_indexes,omitempty"`
	DropIndexes   []string       `json:"drop_indexes,omitempty"`
	RenameInfo    string         `json:"rename_info,omitempty"`
	ChangeComment string         `json:"change_comment,omitempty"`
}

// IndexDetail 索引详情
type IndexDetail struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Columns   []string `json:"columns"`
	ColumnStr string   `json:"column_str"`
}

// ColumnDetail 字段详情
type ColumnDetail struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
	Default  string `json:"default"`
	Comment  string `json:"comment"`
	Extra    string `json:"extra"`
}
