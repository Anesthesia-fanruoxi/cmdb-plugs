package common

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"file-plugs/config"

	"github.com/nwaples/rardecode"
)

// SaveFile 保存上传的文件
func SaveFile(file multipart.File, header *multipart.FileHeader, pathCfg *config.PathConfig, subDir string) (string, error) {
	// 构建目标目录
	targetDir := pathCfg.Path
	if subDir != "" {
		// 安全检查：防止路径穿越
		cleanSub := filepath.Clean(subDir)
		if cleanSub == ".." || strings.HasPrefix(cleanSub, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("非法的子目录路径")
		}
		targetDir = filepath.Join(pathCfg.Path, cleanSub)

		// 二次校验
		cleanTarget := filepath.Clean(targetDir)
		cleanBase := filepath.Clean(pathCfg.Path)
		if !strings.HasPrefix(cleanTarget, cleanBase) {
			return "", fmt.Errorf("非法的子目录路径")
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 清理文件名，防止路径穿越
	filename := filepath.Base(header.Filename)
	if filename == "." || filename == ".." || filename == "" {
		return "", fmt.Errorf("无效的文件名")
	}

	// 构建目标路径
	destPath := filepath.Join(targetDir, filename)

	// 安全检查：确保最终路径在配置目录下
	cleanDest := filepath.Clean(destPath)
	cleanBase := filepath.Clean(pathCfg.Path)
	if !strings.HasPrefix(cleanDest, cleanBase+string(filepath.Separator)) && cleanDest != cleanBase {
		return "", fmt.Errorf("非法的文件路径")
	}

	// 创建目标文件（同名直接覆盖）
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return destPath, nil
}

// SaveFileWithPath 保存文件（保留相对路径结构）
func SaveFileWithPath(file multipart.File, filePath string, pathCfg *config.PathConfig, subDir string) (string, error) {
	// 构建目标目录
	targetDir := pathCfg.Path
	if subDir != "" {
		cleanSub := filepath.Clean(subDir)
		if cleanSub == ".." || strings.HasPrefix(cleanSub, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("非法的子目录路径")
		}
		targetDir = filepath.Join(pathCfg.Path, cleanSub)
	}

	// 清理文件路径，防止路径穿越
	cleanPath := filepath.Clean(filePath)
	if strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("非法的文件路径")
	}

	// 拒绝隐藏文件或目录（以 . 开头）
	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("不允许上传隐藏文件或目录: %s", part)
		}
	}

	// 构建完整目标路径
	destPath := filepath.Join(targetDir, cleanPath)

	// 安全检查：确保最终路径在配置目录下
	cleanDest := filepath.Clean(destPath)
	cleanBase := filepath.Clean(pathCfg.Path)
	if !strings.HasPrefix(cleanDest, cleanBase+string(filepath.Separator)) && cleanDest != cleanBase {
		return "", fmt.Errorf("非法的文件路径")
	}

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建目标文件（同名直接覆盖）
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return destPath, nil
}

// ValidateFileType 校验文件类型
func ValidateFileType(filename string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(filename))
	for _, t := range allowedTypes {
		if strings.ToLower(t) == ext {
			return true
		}
	}
	return false
}

// IsZipFile 判断是否为 ZIP 文件
func IsZipFile(filename string) bool {
	return strings.ToLower(filepath.Ext(filename)) == ".zip"
}

// IsRarFile 判断是否为 RAR 文件
func IsRarFile(filename string) bool {
	return strings.ToLower(filepath.Ext(filename)) == ".rar"
}

// IsArchiveFile 判断是否为支持的压缩文件（ZIP 或 RAR）
func IsArchiveFile(filename string) bool {
	return IsZipFile(filename) || IsRarFile(filename)
}

// UnzipResult 解压结果
type UnzipResult struct {
	Files []string // 解压后的文件路径列表
	Count int      // 文件数量
}

// SaveAndUnzip 保存并解压 ZIP 文件（扁平化，去掉目录结构）
func SaveAndUnzip(file multipart.File, header *multipart.FileHeader, pathCfg *config.PathConfig, subDir string) (*UnzipResult, error) {
	// 构建目标目录
	targetDir := pathCfg.Path
	if subDir != "" {
		cleanSub := filepath.Clean(subDir)
		if cleanSub == ".." || strings.HasPrefix(cleanSub, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("非法的子目录路径")
		}
		targetDir = filepath.Join(pathCfg.Path, cleanSub)

		cleanTarget := filepath.Clean(targetDir)
		cleanBase := filepath.Clean(pathCfg.Path)
		if !strings.HasPrefix(cleanTarget, cleanBase) {
			return nil, fmt.Errorf("非法的子目录路径")
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 读取上传的文件内容到内存
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 创建 ZIP reader
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, fmt.Errorf("解析 ZIP 文件失败: %w", err)
	}

	result := &UnzipResult{
		Files: make([]string, 0),
	}

	cleanBase := filepath.Clean(pathCfg.Path)

	for _, f := range zipReader.File {
		// 跳过目录
		if f.FileInfo().IsDir() {
			continue
		}

		// 获取文件名（去掉目录结构）
		filename := filepath.Base(f.Name)
		if filename == "." || filename == ".." || filename == "" {
			continue
		}

		// 跳过隐藏文件（如 macOS 的 .DS_Store）
		if strings.HasPrefix(filename, ".") {
			continue
		}

		// 校验文件类型
		if !ValidateFileType(filename, pathCfg.AllowedTypes) {
			continue // 跳过不允许的类型
		}

		// 构建目标路径
		destPath := filepath.Join(targetDir, filename)

		// 安全检查
		cleanDest := filepath.Clean(destPath)
		if !strings.HasPrefix(cleanDest, cleanBase+string(filepath.Separator)) && cleanDest != cleanBase {
			continue
		}

		// 解压文件（同名直接覆盖）
		if err := extractFile(f, destPath); err != nil {
			return nil, fmt.Errorf("解压文件 %s 失败: %w", filename, err)
		}

		result.Files = append(result.Files, destPath)
		result.Count++
	}

	return result, nil
}

// extractFile 解压单个文件
func extractFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, rc)
	return err
}

// SaveAndUnzipRar 保存并解压 RAR 文件（扁平化，去掉目录结构）
func SaveAndUnzipRar(file multipart.File, header *multipart.FileHeader, pathCfg *config.PathConfig, subDir string) (*UnzipResult, error) {
	// 构建目标目录
	targetDir := pathCfg.Path
	if subDir != "" {
		cleanSub := filepath.Clean(subDir)
		if cleanSub == ".." || strings.HasPrefix(cleanSub, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("非法的子目录路径")
		}
		targetDir = filepath.Join(pathCfg.Path, cleanSub)

		cleanTarget := filepath.Clean(targetDir)
		cleanBase := filepath.Clean(pathCfg.Path)
		if !strings.HasPrefix(cleanTarget, cleanBase) {
			return nil, fmt.Errorf("非法的子目录路径")
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 读取上传的文件内容到内存
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 创建 RAR reader
	rarReader, err := rardecode.NewReader(bytes.NewReader(buf.Bytes()), "")
	if err != nil {
		return nil, fmt.Errorf("解析 RAR 文件失败: %w", err)
	}

	result := &UnzipResult{
		Files: make([]string, 0),
	}

	cleanBase := filepath.Clean(pathCfg.Path)

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取 RAR 条目失败: %w", err)
		}

		// 跳过目录
		if header.IsDir {
			continue
		}

		// 获取文件名（去掉目录结构）
		filename := filepath.Base(header.Name)
		if filename == "." || filename == ".." || filename == "" {
			continue
		}

		// 跳过隐藏文件
		if strings.HasPrefix(filename, ".") {
			continue
		}

		// 校验文件类型
		if !ValidateFileType(filename, pathCfg.AllowedTypes) {
			continue
		}

		// 构建目标路径
		destPath := filepath.Join(targetDir, filename)

		// 安全检查
		cleanDest := filepath.Clean(destPath)
		if !strings.HasPrefix(cleanDest, cleanBase+string(filepath.Separator)) && cleanDest != cleanBase {
			continue
		}

		// 解压文件
		dst, err := os.Create(destPath)
		if err != nil {
			return nil, fmt.Errorf("创建文件 %s 失败: %w", filename, err)
		}

		_, err = io.Copy(dst, rarReader)
		dst.Close()
		if err != nil {
			return nil, fmt.Errorf("解压文件 %s 失败: %w", filename, err)
		}

		result.Files = append(result.Files, destPath)
		result.Count++
	}

	return result, nil
}
