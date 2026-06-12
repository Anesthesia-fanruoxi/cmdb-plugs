package common

import (
	"encoding/json"
	"fmt"
	"nginx-plugs/config"
	"os"
	"path/filepath"
	"sync"
)

// DNSRecordStore DNS记录持久化存储
// 文件名: conf_dir/.dns_records.json
type DNSRecordStore struct {
	Records map[string]DNSRecordEntry `json:"records"` // key: server_name
}

// DNSRecordEntry 单条DNS记录信息
type DNSRecordEntry struct {
	ServerName string `json:"server_name"` // 完整域名,如 ceshi.hzlsg.com
	SubDomain  string `json:"sub_domain"`  // 二级域名,如 ceshi
	Domain     string `json:"domain"`      // 主域名,如 hzlsg.com
	RecordId   string `json:"record_id"`   // 阿里云DNS RecordId
	RecordType string `json:"record_type"` // CNAME 或 A
}

var (
	storeMu    sync.Mutex
	storeCache *DNSRecordStore
)

// dnsStorePath 获取存储文件路径
func dnsStorePath() string {
	return filepath.Join(config.GetNginxConfig().ConfDir, ".dns_records.json")
}

// LoadDNSStore 从文件加载DNS记录映射
func LoadDNSStore() (*DNSRecordStore, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	if storeCache != nil {
		return storeCache, nil
	}

	store := &DNSRecordStore{Records: make(map[string]DNSRecordEntry)}

	data, err := os.ReadFile(dnsStorePath())
	if err != nil {
		if os.IsNotExist(err) {
			storeCache = store
			return store, nil
		}
		return nil, fmt.Errorf("读取DNS记录文件失败: %w", err)
	}

	if len(data) == 0 {
		storeCache = store
		return store, nil
	}

	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("解析DNS记录文件失败: %w", err)
	}

	storeCache = store
	return store, nil
}

// SaveDNSRecord 添加或更新一条DNS记录映射并持久化
func SaveDNSRecord(entry DNSRecordEntry) error {
	store, err := LoadDNSStore()
	if err != nil {
		return err
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	store.Records[entry.ServerName] = entry
	return flushStoreLocked(store)
}

// GetDNSRecord 根据 server_name 获取记录
func GetDNSRecord(serverName string) (DNSRecordEntry, bool) {
	store, err := LoadDNSStore()
	if err != nil {
		return DNSRecordEntry{}, false
	}
	entry, ok := store.Records[serverName]
	return entry, ok
}

// DeleteDNSRecord 删除一条DNS记录映射
func DeleteDNSRecord(serverName string) error {
	store, err := LoadDNSStore()
	if err != nil {
		return err
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	delete(store.Records, serverName)
	return flushStoreLocked(store)
}

// flushStoreLocked 写入存储文件(调用方需持有锁)
func flushStoreLocked(store *DNSRecordStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化DNS记录失败: %w", err)
	}

	path := dnsStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入DNS记录文件失败: %w", err)
	}

	storeCache = store
	return nil
}
