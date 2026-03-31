package main

import (
	"cron-plugs/router"
	"flag"
	"log"
	"net/http"
)

func main() {
	// 解析命令行参数
	port := flag.String("port", "8081", "监听端口号")
	flag.Parse()

	// 设置路由
	router.Setup()

	// 启动服务
	log.Printf("Agent服务启动在端口: %s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
