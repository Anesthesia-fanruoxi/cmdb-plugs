package api

import (
	"strings"
)

// processEscapedChars 处理字符串中的转义字符
func processEscapedChars(input string) string {
	result := strings.ReplaceAll(input, "\\\"", "\"")
	result = strings.ReplaceAll(result, "\\'", "'")
	result = strings.ReplaceAll(result, "\\:", ":")
	result = strings.ReplaceAll(result, "\\=", "=")
	result = strings.ReplaceAll(result, "\\,", ",")
	return result
}

// buildTimeRange 构建时间范围条件（左闭右开：gte + lt，0表示不限制该边界）
func (qb *QueryBuilder) buildTimeRange() map[string]interface{} {
	timeRange := map[string]interface{}{}
	if qb.StartTime > 0 {
		timeRange["gte"] = qb.StartTime
	}
	if qb.EndTime > 0 {
		timeRange["lt"] = qb.EndTime
	}
	return timeRange
}

// buildTimeRangeQuery 构建时间范围查询
func (qb *QueryBuilder) buildTimeRangeQuery() {
	if qb.TimeField == "" {
		qb.TimeField = "@timestamp"
	}

	timeRange := qb.buildTimeRange()

	// 未传时间范围时退化为 match_all
	if len(timeRange) == 0 {
		qb.Query = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
		return
	}

	qb.Query = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"range": map[string]interface{}{
						qb.TimeField: timeRange,
					},
				},
			},
		},
	}
}
