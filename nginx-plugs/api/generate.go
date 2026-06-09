package api

import (
	"encoding/json"
	"net/http"
	"nginx-plugs/common"
	"nginx-plugs/config"
	"nginx-plugs/model"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// templateData 模板渲染数据结构（比请求模型多了GenerateTime字段）
type templateData struct {
	GenerateTime string
	ServerName   string
	Listen       int

	UpstreamName   string
	UpstreamServer string

	Location     string
	ProxyPass    string
	HostHeader   string
	RealIPHeader string

	SSL           bool
	SSLCert       string
	SSLKey        string
	ExtraConfig   string
	ClientMaxBody string
}

// GenerateHandler 生成nginx配置文件并写入配置目录
func GenerateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 检查模板是否就绪
	if !common.TemplateMgr.IsReady() {
		common.ErrorWithCode(w, 500, "未找到模板文件，无法生成配置")
		return
	}

	// 解析请求
	var req model.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorWithCode(w, 400, "请求参数解析失败: "+err.Error())
		return
	}

	// 校验必填字段
	if req.ServerName == "" {
		common.ErrorWithCode(w, 400, "server_name 不能为空")
		return
	}

	// 构建模板数据
	data := buildTemplateData(&req)

	// 渲染模板
	content, err := common.TemplateMgr.Render(data)
	if err != nil {
		common.ErrorWithCode(w, 500, "生成配置失败: "+err.Error())
		return
	}

	// 确定输出文件名
	outputFile := req.OutputFile
	if outputFile == "" {
		outputFile = strings.ReplaceAll(req.ServerName, ".", "_") + ".conf"
	}

	// 追加写入配置文件
	outputPath, err := common.AppendConfFile(outputFile, content)
	if err != nil {
		common.ErrorWithCode(w, 500, "写入配置文件失败: "+err.Error())
		return
	}

	common.Success(w, model.GenerateResponse{
		ServerName: req.ServerName,
		OutputFile: outputPath,
		Content:    content,
		ConfDir:    config.GetNginxConfig().ConfDir,
	})
}

// PreviewHandler 预览生成的配置内容（不写入文件）
func PreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	if !common.TemplateMgr.IsReady() {
		common.ErrorWithCode(w, 500, "未找到模板文件，无法预览配置")
		return
	}

	var req model.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorWithCode(w, 400, "请求参数解析失败: "+err.Error())
		return
	}

	if req.ServerName == "" {
		common.ErrorWithCode(w, 400, "server_name 不能为空")
		return
	}

	data := buildTemplateData(&req)

	content, err := common.TemplateMgr.Render(data)
	if err != nil {
		common.ErrorWithCode(w, 500, "预览配置失败: "+err.Error())
		return
	}

	common.Success(w, model.GenerateResponse{
		ServerName: req.ServerName,
		Content:    content,
		ConfDir:    config.GetNginxConfig().ConfDir,
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
				ConfDir: confDir,
				Files:   []model.FileInfo{},
				Total:   0,
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
		files = append(files, model.FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(confDir, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	common.Success(w, model.ListResponse{
		ConfDir: confDir,
		Files:   files,
		Total:   len(files),
	})
}

// buildTemplateData 从请求构建模板渲染数据
func buildTemplateData(req *model.GenerateRequest) *templateData {
	return &templateData{
		GenerateTime:   time.Now().Format("2006-01-02 15:04:05"),
		ServerName:     req.ServerName,
		Listen:         req.Listen,
		UpstreamName:   req.UpstreamName,
		UpstreamServer: req.UpstreamServer,
		Location:       req.Location,
		ProxyPass:      req.ProxyPass,
		HostHeader:     req.HostHeader,
		RealIPHeader:   req.RealIPHeader,
		SSL:            req.SSL,
		SSLCert:        req.SSLCert,
		SSLKey:         req.SSLKey,
		ExtraConfig:    req.ExtraConfig,
		ClientMaxBody:  req.ClientMaxBody,
	}
}
