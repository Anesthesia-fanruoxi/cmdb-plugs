package api

import (
	"fmt"
	"regexp"
	"strings"
)

// validateExportQuery 验证导出SQL安全性（仅允许DQL）
func validateExportQuery(query string) error {
	re := regexp.MustCompile(`/\*.*?\*/`)
	query = re.ReplaceAllString(query, "")
	lines := strings.Split(query, "\n")
	var cleanLines []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		if line = strings.TrimSpace(line); line != "" {
			cleanLines = append(cleanLines, line)
		}
	}
	cleanQuery := strings.ToUpper(strings.Join(cleanLines, " "))

	allowedPrefixes := []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"}
	isAllowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanQuery, prefix) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return fmt.Errorf("导出接口只允许查询语句（SELECT/WITH/SHOW/DESCRIBE/EXPLAIN）")
	}

	dangerousPrefixes := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "REPLACE", "GRANT", "REVOKE"}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(cleanQuery, prefix) {
			return fmt.Errorf("导出接口不允许执行 %s 操作", prefix)
		}
	}

	return nil
}
