package api

// ParseKeyword 解析关键词并构建查询
func (qb *QueryBuilder) ParseKeyword(keyword string) error {
	if qb.TimeField == "" {
		qb.TimeField = "@timestamp"
	}

	// 如果关键词为空，直接返回时间范围查询
	if keyword == "" {
		qb.buildTimeRangeQuery()
		return nil
	}

	// 分割关键词
	tokens, err := tokenize(keyword)
	if err != nil {
		return err
	}

	// 构建查询
	query, err := buildQuery(tokens)
	if err != nil {
		return err
	}

	// 添加时间范围（左闭右开：gte + lt）
	must := []interface{}{query}
	if timeRange := qb.buildTimeRange(); len(timeRange) > 0 {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				qb.TimeField: timeRange,
			},
		})
	}

	qb.Query = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": must,
		},
	}
	return nil
}
