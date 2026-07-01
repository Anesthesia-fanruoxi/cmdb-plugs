package common

import (
	"fmt"
	"sync"
)

// TaskManager 异步任务管理器，通过信号量控制并发数
type TaskManager struct {
	semaphore chan struct{}
	mu        sync.Mutex
	active    int
}

var taskManager *TaskManager

// InitTaskManager 初始化全局任务管理器
func InitTaskManager(maxConcurrent int) {
	taskManager = &TaskManager{
		semaphore: make(chan struct{}, maxConcurrent),
	}
	Logger.Infof("异步导出任务管理器已初始化，最大并发: %d", maxConcurrent)
}

// GetTaskManager 获取全局任务管理器
func GetTaskManager() *TaskManager {
	return taskManager
}

// Submit 非阻塞提交任务，并发已满时返回错误
func (tm *TaskManager) Submit(fn func()) error {
	select {
	case tm.semaphore <- struct{}{}:
		tm.mu.Lock()
		tm.active++
		tm.mu.Unlock()

		go func() {
			defer func() {
				<-tm.semaphore
				tm.mu.Lock()
				tm.active--
				tm.mu.Unlock()
			}()
			fn()
		}()
		return nil
	default:
		return fmt.Errorf("当前导出任务已满（%d/%d），请稍后重试", len(tm.semaphore), cap(tm.semaphore))
	}
}

// ActiveCount 返回当前活跃任务数
func (tm *TaskManager) ActiveCount() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.active
}
