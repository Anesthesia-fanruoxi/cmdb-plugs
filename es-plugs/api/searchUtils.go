package api

import (
	"fmt"
	"strings"
	"time"
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

// formatTimeValues 根据时间格式处理时间值
func (qb *QueryBuilder) formatTimeValues() (string, string) {
	if qb.TimeFormat == "" {
		qb.TimeFormat = "iso8601"
	}

	shanghaiLoc, _ := time.LoadLocation("Asia/Shanghai")

	switch qb.TimeFormat {
	case "iso8601":
		startTime := ""
		endTime := ""

		if qb.StartTime != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", qb.StartTime, shanghaiLoc)
			if err == nil {
				utcTime := t.UTC()
				startTime = utcTime.Format(time.RFC3339)
			} else {
				startTime = qb.StartTime
			}
		}

		if qb.EndTime != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", qb.EndTime, shanghaiLoc)
			if err == nil {
				utcTime := t.UTC()
				endTime = utcTime.Format(time.RFC3339)
			} else {
				endTime = qb.EndTime
			}
		}

		return startTime, endTime

	case "epoch_millis":
		startTime := ""
		endTime := ""

		if qb.StartTime != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", qb.StartTime, shanghaiLoc)
			if err == nil {
				startTime = fmt.Sprintf("%d", t.UnixNano()/1000000)
			} else {
				startTime = qb.StartTime
			}
		}

		if qb.EndTime != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", qb.EndTime, shanghaiLoc)
			if err == nil {
				endTime = fmt.Sprintf("%d", t.UnixNano()/1000000)
			} else {
				endTime = qb.EndTime
			}
		}

		return startTime, endTime

	case "epoch_second":
		startTime := ""
		endTime := ""

		if qb.StartTime != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", qb.StartTime, shanghaiLoc)
			if err == nil {
				startTime = fmt.Sprintf("%d", t.Unix())
			} else {
				startTime = qb.StartTime
			}
		}

		if qb.EndTime != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04:05", qb.EndTime, shanghaiLoc)
			if err == nil {
				endTime = fmt.Sprintf("%d", t.Unix())
			} else {
				endTime = qb.EndTime
			}
		}

		return startTime, endTime

	default:
		return qb.StartTime, qb.EndTime
	}
}

// buildTimeRangeQuery 构建时间范围查询
func (qb *QueryBuilder) buildTimeRangeQuery() {
	if qb.TimeField == "" {
		qb.TimeField = "@timestamp"
	}

	startTime, endTime := qb.formatTimeValues()

	qb.Query = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
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
}
