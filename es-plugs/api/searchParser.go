package api

// ParseKeyword 解析关键词并构建查询
func (qb *QueryBuilder) ParseKeyword(keyword string) error {
	if qb.TimeField == "" {
		qb.TimeField = "@timestamp"
	}

	startTime, endTime := qb.formatTimeValues()

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

	// 添加时间范围
	qb.Query = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				query,
				map[string]interface{}{
					"range": map[string]interface{}{
						qb.TimeField: map[string]interface{}{
							"gte": startTime,
							"lte": endTime,
						},
					},
				},
			},
		},
	}
	return nil
}
