package model

type SQLSearchRequest struct {
	QueryID string `json:"query_id"`
	DB      string `json:"dbName"`
	Query   string `json:"query"`
}

type CancelRequest struct {
	QueryID string `json:"query_id"`
	DB      string `json:"dbName"`
}

type StructureRequest struct {
	Type string                 `json:"type"`
	Op   map[string]interface{} `json:"op"`
}
