package model

// ConfigItem 配置项
type ConfigItem struct {
	DataId  string `json:"dataId"`
	Group   string `json:"group"`
	Content string `json:"content"`
	Tenant  string `json:"tenant"`
}

// ConfigListResponse 配置列表响应
type ConfigListResponse struct {
	TotalCount     int          `json:"totalCount"`
	PageNumber     int          `json:"pageNumber"`
	PagesAvailable int          `json:"pagesAvailable"`
	PageItems      []ConfigItem `json:"pageItems"`
}
