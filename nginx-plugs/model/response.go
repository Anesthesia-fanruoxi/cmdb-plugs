package model

// GenerateResponse 生成配置响应
type GenerateResponse struct {
	ServerName string `json:"server_name"` // 服务器名称
	OutputFile string `json:"output_file"` // 生成的配置文件路径
	Content    string `json:"content"`     // 生成的配置内容（预览时返回）
	ConfDir    string `json:"conf_dir"`    // nginx配置目录
}

// ListResponse 列出已生成配置文件的响应
type ListResponse struct {
	ConfDir string     `json:"conf_dir"` // nginx配置目录
	Files   []FileInfo `json:"files"`    // 配置文件列表
	Total   int        `json:"total"`    // 文件总数
}

// FileInfo 文件信息
type FileInfo struct {
	Name    string `json:"name"`     // 文件名
	Path    string `json:"path"`     // 完整路径
	Size    int64  `json:"size"`     // 文件大小（字节）
	ModTime string `json:"mod_time"` // 最后修改时间
}
