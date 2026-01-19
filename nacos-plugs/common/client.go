package common

import (
	"fmt"
	"net/http"
	"time"

	"nacos-plugs/config"
)

// NacosClient Nacos HTTP 客户端
type NacosClient struct {
	Config     *config.NacosConfig
	HttpClient *http.Client
	BaseURL    string
}

// NewNacosClient 创建 Nacos 客户端
func NewNacosClient(cfg *config.NacosConfig) *NacosClient {
	baseURL := fmt.Sprintf("http://%s:%d%s", cfg.Host, cfg.Port, cfg.ContextPath)
	return &NacosClient{
		Config: cfg,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		BaseURL: baseURL,
	}
}

// SetAuth 设置认证信息
func (c *NacosClient) SetAuth(req *http.Request) {
	if c.Config.Username != "" && c.Config.Password != "" {
		req.SetBasicAuth(c.Config.Username, c.Config.Password)
	}
}
