package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"file-plugs/config"

	"github.com/mozillazg/go-pinyin"
)

// pinyinArgs 拼音转换参数
var pinyinArgs = pinyin.NewArgs()

// FileInfo 文件信息
type FileInfo struct {
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	ModTime       string    `json:"mod_time"`
	modTimeParsed time.Time `json:"-"` // 内部使用，用于排序
	IsDir         bool      `json:"is_dir"`
}

// sequenceMatch 顺序模糊匹配
// 检查 pattern 是否按顺序出现在 text 中（不区分大小写）
func sequenceMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	textRunes := []rune(text)
	patternRunes := []rune(pattern)

	j := 0
	for i := 0; i < len(textRunes) && j < len(patternRunes); i++ {
		if textRunes[i] == patternRunes[j] {
			j++
		}
	}

	return j == len(patternRunes)
}

// getPinyinInitials 获取拼音首字母
func getPinyinInitials(text string) string {
	result := strings.Builder{}

	for _, r := range text {
		// 如果是 ASCII 字符，直接添加
		if r < 128 {
			result.WriteRune(r)
		} else {
			// 中文转拼音首字母
			py := pinyin.LazyPinyin(string(r), pinyinArgs)
			if len(py) > 0 && len(py[0]) > 0 {
				result.WriteByte(py[0][0])
			}
		}
	}

	return result.String()
}

// matchWithPinyin 支持拼音的顺序模糊匹配
func matchWithPinyin(filename, search string) bool {
	// 1. 直接匹配原文件名
	if sequenceMatch(filename, search) {
		return true
	}

	// 2. 匹配拼音首字母
	initials := getPinyinInitials(filename)
	if sequenceMatch(initials, search) {
		return true
	}

	return false
}

// ListResult 列表结果
type ListResult struct {
	Key        string     `json:"key"`
	Path       string     `json:"path"`
	Files      []FileInfo `json:"files"`
	Count      int        `json:"count"`       // 当前页文件数量
	Total      int        `json:"total"`       // 总文件数量
	Page       int        `json:"page"`        // 当前页码
	TotalPages int        `json:"total_pages"` // 总页数
}

// ListHandler 文件列表处理器
func ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Code:    -1,
			Message: "仅支持 GET 方法",
		})
		return
	}

	// 从参数获取 key
	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    -1,
			Message: "缺少参数 key",
		})
		return
	}

	// 获取子目录路径（可选）
	subDir := r.URL.Query().Get("path")

	// 获取搜索关键词（可选）
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	// 获取分页参数
	page := 1
	pageSize := 20 // 固定每页20条
	if p := r.URL.Query().Get("page"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}

	// 获取排序参数：name(默认)、time(按时间倒序)、size(按大小倒序)
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "name"
	}

	// 获取路径配置
	pathCfg := config.GetPathConfig(key)
	if pathCfg == nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    -1,
			Message: "无效的路径标识: " + key,
		})
		return
	}

	// 构建完整路径
	targetPath := pathCfg.Path
	if subDir != "" {
		// 安全检查：防止路径穿越
		cleanSub := filepath.Clean(subDir)
		if cleanSub == ".." || strings.HasPrefix(cleanSub, ".."+string(filepath.Separator)) {
			writeJSON(w, http.StatusBadRequest, Response{
				Code:    -1,
				Message: "非法的路径",
			})
			return
		}
		targetPath = filepath.Join(pathCfg.Path, cleanSub)

		// 二次校验：确保在根目录下
		cleanTarget := filepath.Clean(targetPath)
		cleanBase := filepath.Clean(pathCfg.Path)
		if !strings.HasPrefix(cleanTarget, cleanBase) {
			writeJSON(w, http.StatusBadRequest, Response{
				Code:    -1,
				Message: "非法的路径",
			})
			return
		}
	}

	// 读取目录
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, Response{
				Code:    0,
				Message: "success",
				Data: ListResult{
					Key:        key,
					Path:       subDir,
					Files:      []FileInfo{},
					Count:      0,
					Total:      0,
					Page:       page,
					TotalPages: 0,
				},
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Code:    -1,
			Message: "读取目录失败: " + err.Error(),
		})
		return
	}

	// 构建完整文件列表
	allFiles := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 如果有搜索关键词，进行顺序模糊匹配过滤（支持拼音）
		if search != "" {
			if !matchWithPinyin(entry.Name(), search) {
				continue
			}
		}

		allFiles = append(allFiles, FileInfo{
			Name:          entry.Name(),
			Size:          info.Size(),
			ModTime:       info.ModTime().Format("2006-01-02 15:04:05"),
			modTimeParsed: info.ModTime(),
			IsDir:         entry.IsDir(),
		})
	}

	// 排序：目录优先，然后按指定方式排序
	sort.Slice(allFiles, func(i, j int) bool {
		// 目录优先
		if allFiles[i].IsDir != allFiles[j].IsDir {
			return allFiles[i].IsDir
		}
		// 根据排序参数排序
		switch sortBy {
		case "time":
			// 按时间倒序（最新的在前）
			return allFiles[i].modTimeParsed.After(allFiles[j].modTimeParsed)
		case "size":
			// 按大小倒序（最大的在前）
			return allFiles[i].Size > allFiles[j].Size
		default:
			// 按名称正序
			return allFiles[i].Name < allFiles[j].Name
		}
	})

	// 计算分页信息
	total := len(allFiles)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// 校正页码
	if page > totalPages {
		page = totalPages
	}

	// 计算切片范围
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 获取当前页数据
	pageFiles := allFiles[start:end]

	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: ListResult{
			Key:        key,
			Path:       subDir,
			Files:      pageFiles,
			Count:      len(pageFiles),
			Total:      total,
			Page:       page,
			TotalPages: totalPages,
		},
	})
}

// ListKeysHandler 列出所有可用的 key
func ListKeysHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Code:    -1,
			Message: "仅支持 GET 方法",
		})
		return
	}

	keys := make([]map[string]interface{}, 0)
	for _, p := range config.Cfg.Storage.Paths {
		// 统计文件数量
		count := 0
		if entries, err := os.ReadDir(p.Path); err == nil {
			count = len(entries)
		}

		keys = append(keys, map[string]interface{}{
			"key":        p.Key,
			"path":       p.Path,
			"auto_unzip": p.AutoUnzip,
			"file_count": count,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"keys":  keys,
			"count": len(keys),
		},
	})
}
