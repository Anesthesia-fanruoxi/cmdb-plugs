package main

import (
	"log"
	"net/http"

	"nacos-plugs/api"
	"nacos-plugs/config"
	"nacos-plugs/router"
)

func main() {
	cfg := config.LoadConfig("config/config.yml")

	password := "***"
	if cfg.Password != "" {
		password = "***"
	}
	log.Printf("Nacos配置: host=%s, port=%d, namespace=%s, username=%s, password=%s, contextPath=%s",
		cfg.Host, cfg.Port, cfg.Namespace, cfg.Username, password, cfg.ContextPath)

	api.Init(cfg)

	mux := router.Setup()

	addr := ":8080"
	log.Printf("服务启动在 %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
