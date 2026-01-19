package main

import (
	"eip-plugs/common"
	"eip-plugs/router"
	"fmt"
	"net/http"
)

func main() {
	log := common.GetLogger()

	// 设置路由
	mux := router.SetupRoutes()

	// 启动服务器
	port := ":8070"
	log.Info("服务器启动在端口 %s", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Error("服务器启动失败: %v", err)
		fmt.Printf("服务器启动失败: %v\n", err)
	}
}
