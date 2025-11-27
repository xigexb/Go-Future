package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 测试并发限制
// 我们创建一个容量为 2 的池子，提交 5 个长任务
// 理论上同一时间只有 2 个在运行
func TestBlockingExecutor_Limit(t *testing.T) {
	limit := 2
	executor := NewBlockingExecutor(limit)

	var runningCount int32
	var maxRunning int32
	var wg sync.WaitGroup

	taskCount := 10
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		executor.Submit(func() {
			defer wg.Done()

			// 记录当前运行数
			current := atomic.AddInt32(&runningCount, 1)

			// 更新历史最大并发数
			for {
				oldMax := atomic.LoadInt32(&maxRunning)
				if current <= oldMax {
					break
				}
				if atomic.CompareAndSwapInt32(&maxRunning, oldMax, current) {
					break
				}
			}

			// 模拟耗时，确保并发重叠
			time.Sleep(50 * time.Millisecond)

			atomic.AddInt32(&runningCount, -1)
		})
	}

	wg.Wait()

	if maxRunning > int32(limit) {
		t.Errorf("Pool exceeded concurrency limit. Limit: %d, Max Running: %d", limit, maxRunning)
	}

	t.Logf("Pool test passed. Limit: %d, Max Actual: %d", limit, maxRunning)
}

// 测试 Panic 恢复，防止池子里的 goroutine 挂掉导致池子不可用
func TestBlockingExecutor_PanicSafety(t *testing.T) {
	executor := NewBlockingExecutor(1)
	var wg sync.WaitGroup
	wg.Add(2)

	// 任务1：Panic
	executor.Submit(func() {
		defer wg.Done()
		panic("pool panic check")
	})

	// 任务2：正常任务
	// 如果任务1导致 worker 挂掉且没恢复，信号量可能泄露，或者 channel 堵塞，这个任务可能跑不了
	executor.Submit(func() {
		defer wg.Done()
		// 正常执行
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Pass
	case <-time.After(1 * time.Second):
		t.Fatal("Pool blocked or crashed after panic")
	}
}
