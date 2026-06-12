package common

import (
	"bytes"
	"fmt"
	"nginx-plugs/config"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateManager 模板管理器
type TemplateManager struct {
	tmpl          *template.Template
	templateFile  string
	templateReady bool
}

// TemplateMgr 全局模板管理器实例
var TemplateMgr *TemplateManager

// InitTemplate 初始化模板，在启动时调用
func InitTemplate() {
	confDir := config.GetNginxConfig().ConfDir
	templatePath := config.GetTemplatePath()

	TemplateMgr = &TemplateManager{
		templateFile:  templatePath,
		templateReady: false,
	}

	// 1. 确保 confdir 存在
	if err := os.MkdirAll(confDir, 0755); err != nil {
		Logger.Errorf("创建配置目录失败: %v", err)
		fmt.Printf("❌ 创建配置目录失败: %s, 错误: %v\n", confDir, err)
		return
	}
	fmt.Printf("✅ 配置目录已就绪: %s\n", confDir)

	// 2. 检查模板文件是否存在
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// 创建默认模板文件
		if err := createDefaultTemplate(templatePath); err != nil {
			Logger.Errorf("创建默认模板文件失败: %v", err)
			fmt.Printf("❌ 创建默认模板文件失败: %v\n", err)
			return
		}
		fmt.Printf("⚠️  模板文件不存在，已创建默认文件: %s\n", templatePath)
	}

	// 3. 校验模板文件内容
	content, err := os.ReadFile(templatePath)
	if err != nil {
		Logger.Errorf("读取模板文件失败: %v", err)
		fmt.Printf("❌ 读取模板文件失败: %v\n", err)
		return
	}

	// 检查是否为空（全注释）
	if isTemplateEmpty(string(content)) {
		Logger.Warn("模板文件内容为空（全注释），请补充模板配置")
		fmt.Println("⚠️  模板文件内容为空（全注释），请补充模板配置")
		return
	}

	// 4. 解析模板文件（跳过 # 开头的注释行，避免注释内的 {{...}} 被当作模板语法解析报错）
	filteredLines := make([]string, 0)
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	filteredContent := strings.Join(filteredLines, "\n")

	tmpl, err := template.New("proxy").Parse(filteredContent)
	if err != nil {
		Logger.Errorf("解析模板文件失败: %v", err)
		fmt.Printf("❌ 解析模板文件失败: %v\n", err)
		return
	}

	TemplateMgr.tmpl = tmpl
	TemplateMgr.templateReady = true
	fmt.Printf("✅ 模板文件已加载: %s\n", templatePath)
}

// createDefaultTemplate 创建默认模板文件
func createDefaultTemplate(path string) error {
	defaultContent := `# nginx代理配置模板
# 此文件用于生成nginx反向代理配置
# 请不要删除此文件，并根据实际需求修改模板内容
#
# 可用变量:
#   {{.ServerName}} - 完整域名，如 testok.hzbxhd.com
#
# 示例:
# server {
#     listen 80;
#     server_name {{.ServerName}};
#     location / {
#         proxy_pass http://backend;
#     }
# }
`
	return os.WriteFile(path, []byte(defaultContent), 0644)
}

// isTemplateEmpty 检查模板是否为空（全注释）
func isTemplateEmpty(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return false
		}
	}
	return true
}

// IsReady 返回模板是否已就绪
func (tm *TemplateManager) IsReady() bool {
	return tm != nil && tm.templateReady
}

// Render 渲染模板，返回生成的配置内容
func (tm *TemplateManager) Render(data interface{}) (string, error) {
	if !tm.IsReady() {
		return "", fmt.Errorf("模板文件未加载，无法生成配置")
	}

	var buf bytes.Buffer
	if err := tm.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return buf.String(), nil
}

// WriteConfFile 将生成的配置覆盖写入 nginx 配置目录的指定文件中
// 如果文件不存在则创建，如果已存在则覆盖内容
func WriteConfFile(filename string, content string) (string, error) {
	nginxConf := config.GetNginxConfig()
	confDir := nginxConf.ConfDir

	// 确保目录存在
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	outputPath := filepath.Join(confDir, filename)

	// 以覆盖模式写入文件
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入配置文件失败: %w", err)
	}

	Logger.Infof("配置已写入: %s", outputPath)
	return outputPath, nil
}

// DeleteConfFile 删除 nginx 配置目录中的指定配置文件
func DeleteConfFile(filename string) error {
	nginxConf := config.GetNginxConfig()
	confDir := nginxConf.ConfDir

	filePath := filepath.Join(confDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", filename)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除配置文件失败: %w", err)
	}

	Logger.Infof("配置已删除: %s", filePath)
	return nil
}
