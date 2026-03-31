package common

import (
	"net/http"
	"time"

	"nacos-plugs/config"
)

type NacosClient struct {
	Config     *config.NacosConfig
	HttpClient *http.Client
	BaseURL    string
}

func NewNacosClient(cfg *config.NacosConfig) *NacosClient {
	return &NacosClient{
		Config: cfg,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		BaseURL: cfg.GetServerAddress(),
	}
}

func (c *NacosClient) SetAuth(req *http.Request) {
	if c.Config.Username != "" && c.Config.Password != "" {
		req.SetBasicAuth(c.Config.Username, c.Config.Password)
	}
}
