package api

import (
	"eip-plugs/common"
	"eip-plugs/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var log = common.GetLogger()

// GetPublicIP 获取公网IP地址
func GetPublicIP() string {
	urls := []string{
		"https://ident.me",
		"https://ipv4.icanhazip.com",
		"http://myip.ipip.net/ip",
	}

	var wg sync.WaitGroup
	results := make(chan string, len(urls))

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			client := &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					DisableKeepAlives: true,
				},
			}

			req, err := http.NewRequest("GET", u, nil)
			if err != nil {
				return
			}

			req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			req.Header.Set("Pragma", "no-cache")
			req.Header.Set("Expires", "0")
			req.Header.Set("User-Agent", fmt.Sprintf("IP-Reporter-%d", time.Now().Unix()))

			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			var ip string

			if strings.Contains(u, "ipip.net/ip") {
				var ipResp model.IPResponse
				if err := json.Unmarshal(body, &ipResp); err == nil {
					ip = ipResp.IP
				}
			} else {
				ip = strings.TrimSpace(string(body))
			}

			if ip != "" {
				results <- ip
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	ipSet := make(map[string]bool)
	for ip := range results {
		ipSet[ip] = true
	}

	for ip := range ipSet {
		return ip
	}

	return ""
}

// GetIPHandler 处理IP查询请求
func GetIPHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("收到IP查询请求")

	ip := GetPublicIP()
	if ip == "" {
		log.Error("获取公网IP失败")
		common.Error(w, 500, "获取公网IP失败")
		return
	}

	log.Info("成功获取公网IP: %s", ip)
	common.Success(w, map[string]string{
		"ip": ip,
	})
}
