package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nacos-plugs/common"
	"nacos-plugs/config"
)

var nacosClient *common.NacosClient

// Init 初始化 API
func Init(cfg *config.NacosConfig) {
	nacosClient = common.NewNacosClient(cfg)
}

// HandleGetConfig 获取配置
func HandleGetConfig(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dataId := req.URL.Query().Get("dataId")
	group := req.URL.Query().Get("group")
	if group == "" {
		group = "DEFAULT_GROUP"
	}

	content, err := nacosClient.GetConfig(dataId, group)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"content": content})
}

// HandleListConfigs 列出配置
func HandleListConfigs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pageNo, _ := strconv.Atoi(req.URL.Query().Get("pageNo"))
	pageSize, _ := strconv.Atoi(req.URL.Query().Get("pageSize"))
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	list, err := nacosClient.ListConfigs(pageNo, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// HandleSearchConfigs 搜索配置
func HandleSearchConfigs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dataId := req.URL.Query().Get("dataId")
	group := req.URL.Query().Get("group")
	pageNo, _ := strconv.Atoi(req.URL.Query().Get("pageNo"))
	pageSize, _ := strconv.Atoi(req.URL.Query().Get("pageSize"))
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	list, err := nacosClient.SearchConfigs(dataId, group, pageNo, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
