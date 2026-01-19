package common

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"redis-plugs/config"
)

var (
	client   *redis.Client
	clientMu sync.RWMutex
)

func init() {
	InitRedis()
}

// InitRedis 初始化 Redis 连接
func InitRedis() {
	cfg := config.DefaultRedis

	clientMu.Lock()
	defer clientMu.Unlock()

	client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis 连接失败: %v", err)
		return
	}

	log.Printf("Redis 连接成功: %s:%d", cfg.Host, cfg.Port)
}

// GetClient 获取客户端
func GetClient() (*redis.Client, error) {
	clientMu.RLock()
	defer clientMu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("Redis 未连接")
	}
	return client, nil
}

// CloseRedis 关闭连接
func CloseRedis() {
	clientMu.Lock()
	defer clientMu.Unlock()

	if client != nil {
		client.Close()
		client = nil
	}
}

// GetRedisInfo 获取 Redis 状态信息
func GetRedisInfo(ctx context.Context) (map[string]interface{}, error) {
	clientMu.RLock()
	defer clientMu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("Redis 未连接")
	}

	info, err := client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	dbSize, _ := client.DBSize(ctx).Result()

	// 解析 info 字符串
	parsed := parseRedisInfo(info)

	result := map[string]interface{}{
		"connected": true,
		"dbSize":    dbSize,
		"server": map[string]interface{}{
			"version":    parsed["redis_version"],
			"mode":       parsed["redis_mode"],
			"os":         parsed["os"],
			"uptimeDays": parsed["uptime_in_days"],
			"port":       parsed["tcp_port"],
		},
		"memory": map[string]interface{}{
			"used":      parsed["used_memory_human"],
			"peak":      parsed["used_memory_peak_human"],
			"total":     parsed["total_system_memory_human"],
			"fragRatio": parsed["mem_fragmentation_ratio"],
		},
		"clients": map[string]interface{}{
			"connected": parsed["connected_clients"],
			"blocked":   parsed["blocked_clients"],
			"maxClient": parsed["maxclients"],
		},
		"stats": map[string]interface{}{
			"totalConnections": parsed["total_connections_received"],
			"totalCommands":    parsed["total_commands_processed"],
			"opsPerSec":        parsed["instantaneous_ops_per_sec"],
			"keyspaceHits":     parsed["keyspace_hits"],
			"keyspaceMisses":   parsed["keyspace_misses"],
		},
		"replication": map[string]interface{}{
			"role":            parsed["role"],
			"connectedSlaves": parsed["connected_slaves"],
		},
		"keyspace": parseKeyspace(parsed),
	}

	return result, nil
}

// parseRedisInfo 解析 Redis INFO 字符串
func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\r\n")

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// parseKeyspace 解析 keyspace 信息
func parseKeyspace(info map[string]string) map[string]interface{} {
	keyspace := make(map[string]interface{})
	for k, v := range info {
		if strings.HasPrefix(k, "db") {
			keyspace[k] = v
		}
	}
	return keyspace
}
