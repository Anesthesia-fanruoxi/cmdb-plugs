package model

// GenerateRequest 生成nginx配置的请求参数
type GenerateRequest struct {
	// 基础配置
	ServerName string `json:"server_name"` // 服务器名称（域名），如 example.com
	Listen     int    `json:"listen"`      // 监听端口，默认80

	// 上游服务配置
	UpstreamName   string `json:"upstream_name"`   // 上游名称，如 backend
	UpstreamServer string `json:"upstream_server"` // 上游服务器地址，如 127.0.0.1:8080

	// 代理配置
	Location     string `json:"location"`       // 代理路径，默认 /
	ProxyPass    string `json:"proxy_pass"`     // 代理目标，如 http://backend（为空则自动用 http://upstream_name）
	HostHeader   string `json:"host_header"`    // Host 头，默认 $host
	RealIPHeader string `json:"real_ip_header"` // X-Real-IP 头，默认 $remote_addr

	// 输出配置
	OutputFile string `json:"output_file"` // 输出文件名，如 example.conf（为空则自动用 server_name.conf）

	// 额外选项
	SSL           bool   `json:"ssl"`             // 是否启用SSL
	SSLCert       string `json:"ssl_cert"`        // SSL证书路径
	SSLKey        string `json:"ssl_key"`         // SSL私钥路径
	ExtraConfig   string `json:"extra_config"`    // 额外的nginx配置片段（直接追加到server块内）
	ClientMaxBody string `json:"client_max_body"` // 客户端最大请求体，如 "50m"
}

// PreviewRequest 预览配置的请求参数（不写文件，直接返回内容）
type PreviewRequest struct {
	GenerateRequest
}
