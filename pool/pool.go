package pool

import (
	"log"
	"runtime"
	"sync"
)

// Runnable 任务函数定义
type Runnable func()

// Executor 执行器接口
type Executor interface {
	Submit(task Runnable)
}

// blockingExecutor 限制并发数的执行器
type blockingExecutor struct {
	sem  chan struct{} // 信号量
	wait sync.WaitGroup
}

// GlobalExecutor 全局默认执行器
var GlobalExecutor Executor

func init() {
	// 默认并发数为 CPU 核心数 * 2，模拟 Java ForkJoinPool.commonPool
	cpus := runtime.NumCPU() * 2
	if cpus < 4 {
		cpus = 4
	}
	GlobalExecutor = NewBlockingExecutor(cpus)
}

// NewBlockingExecutor 创建一个带并发限制的执行器
func NewBlockingExecutor(limit int) Executor {
	return &blockingExecutor{
		sem: make(chan struct{}, limit),
	}
}

func (e *blockingExecutor) Submit(task Runnable) {
	// 获取信号量，如果满了会阻塞，起到背压作用
	// 在生产环境中，也可以改为非阻塞丢弃或入队，这里为了可靠性选择阻塞等待资源
	e.sem <- struct{}{}
	go func() {
		defer func() {
			<-e.sem // 释放信号量
			if r := recover(); r != nil {
				log.Printf("[Pool] Panic recovered: %v", r)
			}
		}()
		task()
	}()
}

// DirectExecutor 直接在当前 goroutine 或新 goroutine 执行，不限制（用于测试或极轻量任务）
type DirectExecutor struct{}

func (d *DirectExecutor) Submit(task Runnable) {
	go task()
}
