package main

import (
	"log"
	"net/http"

	"nacos-plugs/api"
	"nacos-plugs/config"
	"nacos-plugs/router"
)

func main() {
	// 加载配置
	cfg := config.DefaultConfig()

	// 初始化 API
	api.Init(cfg)

	// 设置路由
	mux := router.Setup()

	// 启动服务
	addr := ":8080"
	log.Printf("服务启动在 %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
