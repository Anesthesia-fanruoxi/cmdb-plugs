package main

import (
	"fmt"
	"log"
	"net/http"

	"file-plugs/api"
	"file-plugs/config"
)

func main() {
	// 加载配置
	if err := config.Load("config/config.yaml"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 注册路由
	http.HandleFunc("/api/upload", api.UploadHandler)
	http.HandleFunc("/api/list", api.ListHandler)
	http.HandleFunc("/api/keys", api.ListKeysHandler)

	// 启动服务
	addr := fmt.Sprintf(":%d", config.Cfg.Server.Port)
	log.Printf("服务启动在 %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
