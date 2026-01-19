package main

import (
	"log"

	"redis-plugs/config"
	"redis-plugs/router"
)

func main() {
	r := router.Setup()

	log.Printf("Redis Viewer 启动在 %s", config.DefaultConfig.Port)
	if err := r.Run(config.DefaultConfig.Port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
