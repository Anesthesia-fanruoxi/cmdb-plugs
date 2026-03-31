package main

import (
	"fmt"
	"sql-plugs/common"
)

func main() {
	// 测试用例：表名包含关键字
	testCases := []struct {
		name string
		sql  string
	}{
		{
			name: "表名包含limit",
			sql:  "SELECT * FROM user_limit WHERE id = 1",
		},
		{
			name: "表名包含where",
			sql:  "SELECT * FROM table_where_data WHERE status = 'active'",
		},
		{
			name: "表名包含order",
			sql:  "SELECT * FROM order_by WHERE id = 1",
		},
		{
			name: "表名包含group",
			sql:  "SELECT * FROM group_by_user WHERE id = 1",
		},
		{
			name: "正常SQL带LIMIT",
			sql:  "SELECT * FROM users LIMIT 100",
		},
		{
			name: "表名包含limit且SQL也有LIMIT",
			sql:  "SELECT * FROM user_limit WHERE id = 1 LIMIT 50",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n========== %s ==========\n", tc.name)
		fmt.Printf("SQL: %s\n", tc.sql)

		// 测试特征分析
		features := common.AnalyzeSQLFeatures(tc.sql)
		fmt.Printf("HasWhere: %v\n", features.HasWhere)
		fmt.Printf("HasGroupBy: %v\n", features.HasGroupBy)
		fmt.Printf("HasOrderBy: %v\n", features.HasOrderBy)

		// 测试LIMIT检测
		userLimit := common.GetUserOriginalLimit(tc.sql)
		fmt.Printf("UserLimit: %d\n", userLimit)

		// 测试表名提取
		tables := common.ExtractTables(tc.sql)
		fmt.Printf("Tables: %v\n", tables)

		// 测试过滤条件检测
		hasFilter := common.HasFilterConditions(tc.sql)
		fmt.Printf("HasFilter: %v\n", hasFilter)
	}
}
