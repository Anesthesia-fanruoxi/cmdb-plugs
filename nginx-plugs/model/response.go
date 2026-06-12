package model

// AddResponse 添加域名解析响应（全流程结果）
type AddResponse struct {
	ServerName    string            `json:"server_name"`    // 完整域名，如 api.hzlsg.com
	SubDomain     string            `json:"sub_domain"`     // 二级域名
	Domain        string            `json:"domain"`         // 主域名
	DNSRecordId   string            `json:"dns_record_id"`  // DNS记录ID
	OutputFile    string            `json:"output_file"`    // nginx配置文件路径
	ReloadResults []SSHReloadResult `json:"reload_results"` // 每台服务器reload结果
}

// GenerateResponse 生成配置响应（保留兼容）
type GenerateResponse struct {
	ServerName string `json:"server_name"` // 服务器名称
	OutputFile string `json:"output_file"` // 生成的配置文件路径
	Content    string `json:"content"`     // 生成的配置内容（预览时返回）
}

// DeleteResponse 删除配置响应（全流程结果）
type DeleteResponse struct {
	ServerName    string            `json:"server_name"`    // 完整域名
	Filename      string            `json:"filename"`       // 删除的nginx配置文件名
	DNSDeleted    bool              `json:"dns_deleted"`    // DNS记录是否已删除
	ReloadResults []SSHReloadResult `json:"reload_results"` // 每台服务器reload结果
}

// SSHReloadResult SSH远程重载结果
type SSHReloadResult struct {
	Name   string `json:"name"`            // 服务器名称
	Host   string `json:"host"`            // 服务器地址
	Status string `json:"status"`          // success 或 failed
	Error  string `json:"error,omitempty"` // 错误信息（如有）
}

// ListResponse 列出已有解析的响应
type ListResponse struct {
	Files []FileInfo `json:"files"` // 配置文件列表
	Total int        `json:"total"` // 文件总数
}

// FileInfo 文件信息
type FileInfo struct {
	Domain    string `json:"domain"`     // 域名
	Path      string `json:"path"`       // 完整路径
	CreatedAt string `json:"created_at"` // 创建时间
}

// OptionsResponse 下拉选项响应
type OptionsResponse struct {
	Domains []DomainItem `json:"domains"` // 可选主域名列表
}

// DomainItem 域名选项
type DomainItem struct {
	Name string `json:"name"` // 域名，如 hzlsg.com
}
