package api

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"file-plugs/common"
	"file-plugs/config"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// UploadResult 上传成功返回数据
type UploadResult struct {
	Key    string   `json:"key"`
	Files  []string `json:"files,omitempty"`
	Count  int      `json:"count"`
	Size   int64    `json:"size"`
	Errors []string `json:"errors,omitempty"`
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// UploadHandler 文件上传处理器
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Code:    -1,
			Message: "仅支持 POST 方法",
		})
		return
	}

	// 从参数获取 key（支持 query 参数或 form 参数）
	key := r.URL.Query().Get("key")
	if key == "" {
		key = r.FormValue("key")
	}
	if key == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    -1,
			Message: "缺少参数 key",
		})
		return
	}

	// 获取子目录路径（可选）
	subDir := r.URL.Query().Get("path")
	if subDir == "" {
		subDir = r.FormValue("path")
	}

	// 获取路径配置
	pathCfg := config.GetPathConfig(key)
	if pathCfg == nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    -1,
			Message: "无效的上传路径标识: " + key,
		})
		return
	}

	// 获取最大文件大小
	maxSize := pathCfg.GetMaxFileSize(config.Cfg.Storage.MaxFileSize)
	if maxSize <= 0 {
		maxSize = 10 << 20 // 默认 10MB
	}

	// 限制请求体大小
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// 解析 multipart form
	if err := r.ParseMultipartForm(maxSize); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    -1,
			Message: "文件过大或解析失败",
		})
		return
	}

	// 收集所有上传的文件（只取 file 字段保证顺序）
	var allFiles []*multipart.FileHeader
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if files, ok := r.MultipartForm.File["file"]; ok {
			allFiles = files
		} else {
			// 兜底：如果没有 file 字段，遍历所有
			for _, files := range r.MultipartForm.File {
				allFiles = append(allFiles, files...)
			}
		}
	}

	if len(allFiles) == 0 {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    -1,
			Message: "未找到上传文件",
		})
		return
	}

	// 获取路径数组（支持多种字段名）
	var pathList []string
	if r.MultipartForm != nil && r.MultipartForm.Value != nil {
		// 优先级：filePath > paths > paths[]
		if paths, ok := r.MultipartForm.Value["filePath"]; ok {
			pathList = paths
		} else if paths, ok := r.MultipartForm.Value["paths"]; ok {
			pathList = paths
		} else if paths, ok := r.MultipartForm.Value["paths[]"]; ok {
			pathList = paths
		}
	}

	// 批量处理文件
	var savedFiles []string
	var totalSize int64
	var errors []string

	for i, header := range allFiles {
		file, err := header.Open()
		if err != nil {
			errors = append(errors, header.Filename+": 打开失败")
			continue
		}

		// 构建完整文件路径
		var filePath string
		if i < len(pathList) && pathList[i] != "" {
			// 拼接目录路径 + 文件名
			dirPath := strings.TrimRight(pathList[i], "/\\")
			filePath = filepath.ToSlash(filepath.Join(dirPath, header.Filename))
		} else {
			filePath = header.Filename
		}

		// 判断是否为压缩文件且开启自动解压
		if pathCfg.AutoUnzip && common.IsArchiveFile(filePath) {
			var result *common.UnzipResult
			var err error

			if common.IsZipFile(filePath) {
				result, err = common.SaveAndUnzip(file, header, pathCfg, subDir)
			} else if common.IsRarFile(filePath) {
				result, err = common.SaveAndUnzipRar(file, header, pathCfg, subDir)
			}

			file.Close()
			if err != nil {
				errors = append(errors, filePath+": "+err.Error())
				continue
			}
			if result != nil {
				savedFiles = append(savedFiles, result.Files...)
				totalSize += header.Size
			}
			continue
		}

		// 校验文件类型
		if !common.ValidateFileType(filePath, pathCfg.AllowedTypes) {
			file.Close()
			errors = append(errors, filePath+": 不支持的文件类型")
			continue
		}

		// 保存文件（保留目录结构）
		savedPath, err := common.SaveFileWithPath(file, filePath, pathCfg, subDir)
		file.Close()
		if err != nil {
			errors = append(errors, filePath+": "+err.Error())
			continue
		}

		savedFiles = append(savedFiles, savedPath)
		totalSize += header.Size
	}

	// 返回结果
	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: UploadResult{
			Key:    key,
			Files:  savedFiles,
			Count:  len(savedFiles),
			Size:   totalSize,
			Errors: errors,
		},
	})
}
