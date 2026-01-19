package config

// NacosConfig Nacos 连接配置
type NacosConfig struct {
	Host        string `json:"host" yaml:"host"`
	Port        int    `json:"port" yaml:"port"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Username    string `json:"username" yaml:"username"`
	Password    string `json:"password" yaml:"password"`
	ContextPath string `json:"contextPath" yaml:"contextPath"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *NacosConfig {
	return &NacosConfig{
		Host:        "127.0.0.1",
		Port:        8848,
		Namespace:   "public",
		ContextPath: "/nacos",
	}
}
