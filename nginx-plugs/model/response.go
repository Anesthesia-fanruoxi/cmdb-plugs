package model

// GenerateResponse 生成配置响应
type GenerateResponse struct {
	ServerName string `json:"server_name"` // 服务器名称
	OutputFile string `json:"output_file"` // 生成的配置文件路径
	Content    string `json:"content"`     // 生成的配置内容（预览时返回）
}

// DeleteResponse 删除配置响应
type DeleteResponse struct {
	ServerName string `json:"server_name"` // 服务器名称
	Filename   string `json:"filename"`    // 删除的文件名
}

// ListResponse 列出已生成配置文件的响应
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
