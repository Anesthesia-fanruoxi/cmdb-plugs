package common

import (
	"bytes"
	"fmt"
	"nginx-plugs/config"
	"os"
	"path/filepath"
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
// 返回 error 不阻止服务启动，只标记模板不可用
func InitTemplate() {
	nginxConf := config.GetNginxConfig()
	templateFile := nginxConf.TemplateFile

	TemplateMgr = &TemplateManager{
		templateFile:  templateFile,
		templateReady: false,
	}

	// 检查模板文件是否存在
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		fmt.Printf("⚠️  未找到模板文件: %s\n", templateFile)
		fmt.Println("⚠️  服务将启动，但所有接口将返回「未找到模板文件」错误")
		fmt.Println("⚠️  请将 proxy.conf.template 放置到正确路径后重启服务")
		return
	}

	// 解析模板文件
	tmpl, err := template.New("proxy").ParseFiles(templateFile)
	if err != nil {
		fmt.Printf("❌ 解析模板文件失败: %s, 错误: %v\n", templateFile, err)
		return
	}

	TemplateMgr.tmpl = tmpl
	TemplateMgr.templateReady = true
	fmt.Printf("✅ 模板文件已加载: %s\n", templateFile)
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
	// 使用模板文件的基础名称作为执行模板的名称
	name := filepath.Base(tm.templateFile)
	if err := tm.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
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
