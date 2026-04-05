package common

import (
	"sync"
	"time"
)

// AlertManager 告警管理器
type AlertManager struct {
	mu               sync.RWMutex
	lastAlertTime    time.Time
	suppressDuration time.Duration
}

// NewAlertManager 创建告警管理器
func NewAlertManager(suppressHours int) *AlertManager {
	return &AlertManager{
		suppressDuration: time.Duration(suppressHours) * time.Hour,
	}
}

// ShouldAlert 判断是否应该发送告警
// 返回 true 表示应该发送告警，false 表示在抑制周期内
func (am *AlertManager) ShouldAlert() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// 如果从未告警过，应该发送
	if am.lastAlertTime.IsZero() {
		return true
	}

	// 判断是否超过抑制周期
	return time.Since(am.lastAlertTime) >= am.suppressDuration
}

// RecordAlert 记录告警时间
func (am *AlertManager) RecordAlert() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.lastAlertTime = time.Now()
}

// GetLastAlertTime 获取上次告警时间
func (am *AlertManager) GetLastAlertTime() time.Time {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.lastAlertTime
}
