package api

import (
	"bufio"
	"net/http"
	"nginx-plugs/common"
	"nginx-plugs/config"
	"nginx-plugs/model"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// templateData 模板渲染数据结构
type templateData struct {
	GenerateTime string
	ServerName   string
}

// GenerateHandler 生成nginx配置文件并写入配置目录
func GenerateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 检查模板是否就绪
	if !common.TemplateMgr.IsReady() {
		common.ErrorWithCode(w, 500, "未找到模板文件，无法生成配置")
		return
	}

	serverName := r.URL.Query().Get("server_name")
	if serverName == "" {
		common.ErrorWithCode(w, 400, "server_name 不能为空")
		return
	}

	data := &templateData{
		GenerateTime: time.Now().Format("2006-01-02 15:04:05"),
		ServerName:   serverName,
	}

	// 渲染模板
	content, err := common.TemplateMgr.Render(data)
	if err != nil {
		common.ErrorWithCode(w, 500, "生成配置失败: "+err.Error())
		return
	}

	// 确定输出文件名
	outputFile := strings.ReplaceAll(serverName, ".", "_") + ".conf"

	// 覆盖写入配置文件
	outputPath, err := common.WriteConfFile(outputFile, content)
	if err != nil {
		common.ErrorWithCode(w, 500, "写入配置文件失败: "+err.Error())
		return
	}

	common.Success(w, model.GenerateResponse{
		ServerName: serverName,
		OutputFile: outputPath,
		Content:    content,
	})
}

// PreviewHandler 读取已生成的配置文件内容
func PreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	serverName := r.URL.Query().Get("server_name")
	if serverName == "" {
		common.ErrorWithCode(w, 400, "server_name 不能为空")
		return
	}

	filename := strings.ReplaceAll(serverName, ".", "_") + ".conf"
	confDir := config.GetNginxConfig().ConfDir
	filePath := filepath.Join(confDir, filename)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			common.ErrorWithCode(w, 404, "配置文件不存在: "+serverName)
			return
		}
		common.ErrorWithCode(w, 500, "读取配置文件失败: "+err.Error())
		return
	}

	common.Success(w, model.GenerateResponse{
		ServerName: serverName,
		OutputFile: filePath,
		Content:    string(content),
	})
}

// DeleteHandler 删除指定的nginx配置文件
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	serverName := r.URL.Query().Get("server_name")
	if serverName == "" {
		common.ErrorWithCode(w, 400, "server_name 不能为空")
		return
	}

	filename := strings.ReplaceAll(serverName, ".", "_") + ".conf"

	if err := common.DeleteConfFile(filename); err != nil {
		common.ErrorWithCode(w, 500, err.Error())
		return
	}

	common.Success(w, model.DeleteResponse{
		ServerName: serverName,
		Filename:   filename,
	})
}

// ListHandler 列出配置目录中已存在的 .conf 文件
func ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	nginxConf := config.GetNginxConfig()
	confDir := nginxConf.ConfDir

	// 读取目录
	entries, err := os.ReadDir(confDir)
	if err != nil {
		if os.IsNotExist(err) {
			common.Success(w, model.ListResponse{
				Files: []model.FileInfo{},
				Total: 0,
			})
			return
		}
		common.ErrorWithCode(w, 500, "读取配置目录失败: "+err.Error())
		return
	}

	var files []model.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".conf") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(confDir, entry.Name())
		domain := extractDomain(filePath)

		files = append(files, model.FileInfo{
			Domain:    domain,
			Path:      filePath,
			CreatedAt: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	common.Success(w, model.ListResponse{
		Files: files,
		Total: len(files),
	})
}

// extractDomain 从 nginx 配置文件中提取 server_name
func extractDomain(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "server_name") {
			// 格式: server_name xxx.com;
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strings.TrimSuffix(parts[1], ";")
			}
		}
	}
	return ""
}
