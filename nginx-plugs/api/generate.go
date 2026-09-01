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

// AddHandler 全流程添加：DNS解析 + nginx配置 + nginx重载
// GET /api/nginx/add?server_name=testok.hzbxhd.com
func AddHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 检查模板是否就绪
	if !common.TemplateMgr.IsReady() {
		common.ErrorWithCode(w, 500, "未找到模板文件，无法添加解析")
		return
	}

	// 获取server_name参数（完整域名，如 testok.hzbxhd.com）
	serverName := r.URL.Query().Get("server_name")
	if serverName == "" {
		common.ErrorWithCode(w, 400, "server_name 不能为空")
		return
	}

	// 拆分为子域名和主域名: testok.hzbxhd.com -> (testok, hzbxhd.com)
	subDomain, domain := splitDomain(serverName)
	if subDomain == "" || domain == "" {
		common.ErrorWithCode(w, 400, "server_name 格式不正确，需为完整域名如 testok.hzbxhd.com")
		return
	}

	// 步骤1：添加DNS记录
	aliyunConf := config.GetAliyunConfig()
	var dnsRecordId string
	recordType := aliyunConf.RecordType
	if recordType == "" {
		recordType = "CNAME" // 默认CNAME
	}
	if aliyunConf.AccessKeyID != "" && aliyunConf.AccessKeySecret != "" && aliyunConf.RecordValue != "" {
		dnsResp, err := common.AddDomainRecord(domain, subDomain, recordType, aliyunConf.RecordValue)
		if err != nil {
			common.ErrorWithCode(w, 500, "添加DNS记录失败: "+err.Error())
			return
		}
		dnsRecordId = dnsResp.RecordId

		// 持久化 RecordId，便于删除时直接按 Id 删除
		if err := common.SaveDNSRecord(common.DNSRecordEntry{
			ServerName: serverName,
			SubDomain:  subDomain,
			Domain:     domain,
			RecordId:   dnsRecordId,
			RecordType: recordType,
		}); err != nil {
			common.Logger.Warnf("保存DNS记录映射失败: %v", err)
		}
	} else {
		common.Logger.Warn("阿里云配置不完整，跳过DNS记录添加")
	}

	// 步骤2：生成nginx配置并写入
	data := &templateData{
		GenerateTime: time.Now().Format("2006-01-02 15:04:05"),
		ServerName:   serverName,
	}

	content, err := common.TemplateMgr.Render(data)
	if err != nil {
		common.ErrorWithCode(w, 500, "生成配置失败: "+err.Error())
		return
	}

	outputFile := strings.ReplaceAll(serverName, ".", "_") + ".conf"
	outputPath, err := common.WriteConfFile(outputFile, content)
	if err != nil {
		common.ErrorWithCode(w, 500, "写入配置文件失败: "+err.Error())
		return
	}

	// 步骤3：SSH远程重载nginx（异步执行，接口先返回，避免HTTP连接超时）
	go func() {
		results := common.ReloadAllNginx()
		for _, r := range results {
			if r.Status == "success" {
				common.Logger.Infof("[ASYNC-RELOAD] %s(%s) 重载成功", r.Name, r.Host)
			} else {
				common.Logger.Errorf("[ASYNC-RELOAD] %s(%s) 重载失败: %s", r.Name, r.Host, r.Error)
			}
		}
	}()

	common.Success(w, model.AddResponse{
		ServerName:    serverName,
		SubDomain:     subDomain,
		Domain:        domain,
		DNSRecordId:   dnsRecordId,
		OutputFile:    outputPath,
		ReloadResults: []model.SSHReloadResult{},
	})
}

// DeleteHandler 全流程删除：DNS记录 + nginx配置 + nginx重载
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

	// 步骤1：删除DNS记录（按本地持久化的 RecordId 直接删除）
	dnsDeleted := false
	aliyunConf := config.GetAliyunConfig()
	if aliyunConf.AccessKeyID != "" && aliyunConf.AccessKeySecret != "" {
		if entry, ok := common.GetDNSRecord(serverName); ok && entry.RecordId != "" {
			if err := common.DeleteDomainRecordById(entry.RecordId, serverName); err != nil {
				common.Logger.Warnf("删除DNS记录失败: %v", err)
			} else {
				dnsDeleted = true
				_ = common.DeleteDNSRecord(serverName)
			}
		} else {
			common.Logger.Warnf("本地未找到 %s 的DNS记录映射，跳过DNS删除", serverName)
		}
	} else {
		common.Logger.Warn("阿里云配置不完整，跳过DNS记录删除")
	}

	// 步骤2：删除nginx配置文件
	filename := strings.ReplaceAll(serverName, ".", "_") + ".conf"
	if err := common.DeleteConfFile(filename); err != nil {
		common.ErrorWithCode(w, 500, err.Error())
		return
	}

	// 步骤3：SSH远程重载nginx（异步执行，接口先返回）
	go func() {
		results := common.ReloadAllNginx()
		for _, r := range results {
			if r.Status == "success" {
				common.Logger.Infof("[ASYNC-RELOAD] %s(%s) 重载成功", r.Name, r.Host)
			} else {
				common.Logger.Errorf("[ASYNC-RELOAD] %s(%s) 重载失败: %s", r.Name, r.Host, r.Error)
			}
		}
	}()

	common.Success(w, model.DeleteResponse{
		ServerName:    serverName,
		Filename:      filename,
		DNSDeleted:    dnsDeleted,
		ReloadResults: []model.SSHReloadResult{},
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

// OptionsHandler 返回下拉选项（可选的主域名列表）
func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	domains := config.GetDomainOptions()
	var items []model.DomainItem
	for _, d := range domains {
		items = append(items, model.DomainItem{Name: d.Name, Owner: d.Owner})
	}

	common.Success(w, model.OptionsResponse{
		Domains: items,
	})
}

// splitDomain 将完整域名拆分为子域名和主域名
// 例如 "api.hzlsg.com" -> ("api", "hzlsg.com")
// 通过配置的域名列表进行匹配
func splitDomain(serverName string) (subDomain string, domain string) {
	domainOptions := config.GetDomainOptions()
	for _, d := range domainOptions {
		if strings.HasSuffix(serverName, "."+d.Name) {
			subDomain = strings.TrimSuffix(serverName, "."+d.Name)
			domain = d.Name
			return
		}
	}

	// 如果配置中没有匹配的主域名，按第一个"."拆分
	parts := strings.SplitN(serverName, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", serverName
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
