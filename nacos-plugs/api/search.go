package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nacos-plugs/common"
	"nacos-plugs/config"
)

var nacosClient *common.NacosClient

func Init(cfg *config.NacosConfig) {
	nacosClient = common.NewNacosClient(cfg)
}

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func HandleGetConfig(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Code: 405, Msg: "Method not allowed"})
		return
	}

	dataId := req.URL.Query().Get("dataId")
	group := req.URL.Query().Get("group")
	if group == "" {
		group = "DEFAULT_GROUP"
	}

	content, err := nacosClient.GetConfig(dataId, group)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Code: 0, Msg: "success", Data: map[string]string{"content": content}})
}

func HandleListConfigs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Code: 405, Msg: "Method not allowed"})
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
		writeJSON(w, http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Code: 0, Msg: "success", Data: list})
}

func HandleSearchConfigs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Code: 405, Msg: "Method not allowed"})
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
		writeJSON(w, http.StatusInternalServerError, Response{Code: 500, Msg: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Code: 0, Msg: "success", Data: list})
}
