package models

// Connection Redis 连接配置
type Connection struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
	DB       int    `json:"db"`
}

// KeyNode Key 树节点
type KeyNode struct {
	Name     string     `json:"name"`
	FullKey  string     `json:"fullKey,omitempty"`
	IsLeaf   bool       `json:"isLeaf"`
	Children []*KeyNode `json:"children,omitempty"`
	Count    int        `json:"count"`
}

// KeyInfo Key 详细信息
type KeyInfo struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl"`
	Value interface{} `json:"value"`
	Size  int64       `json:"size"`
}

// APIResponse 统一 API 响应
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// KeyTreeRequest 获取 Key 树请求
type KeyTreeRequest struct {
	Pattern   string `json:"pattern"`
	Separator string `json:"separator"`
}

// KeyInfoRequest 获取 Key 详情请求
type KeyInfoRequest struct {
	Key string `json:"key" binding:"required"`
}

// DeleteKeyRequest 删除 Key 请求
type DeleteKeyRequest struct {
	Key string `json:"key" binding:"required"`
}
