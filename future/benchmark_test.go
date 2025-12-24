package future

import (
	"context"
	"sync"
	"testing"

	"github.com/xigexb/go-future/pool"
)

// ============ 模拟高性能池 ============

type BenchPool struct {
	work chan func()
	sem  chan struct{}
}

func NewBenchPool(size int) *BenchPool {
	p := &BenchPool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
	go p.dispatcher()
	return p
}

func (p *BenchPool) dispatcher() {
	for task := range p.work {
		p.sem <- struct{}{}
		go func(t func()) {
			defer func() { <-p.sem }()
			t()
		}(task)
	}
}

func (p *BenchPool) Submit(task pool.Runnable) {
	p.work <- task
}

var sharedBenchPool = NewBenchPool(1000)

// ============ Benchmark Case ============

func BenchmarkNative_Goroutine(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				_ = 1 + 1
				wg.Done()
			}()
			wg.Wait()
		}
	})
}

func BenchmarkFuture_DefaultPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 全自动 GC 回收，无 Free 调用
			f := SupplyAsync(func() int {
				return 1
			})
			_, _ = f.Join()
		}
	})
}

// 测试链式调用的内存开销 (无 Join)
// 预期：由于 doneChan 惰性加载，这里中间的 100 个 Future 不应该创建 channel
func BenchmarkChain_NoJoin_AutoFree(b *testing.B) {
	f := CompletedFuture(0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		curr := f
		for j := 0; j < 100; j++ {
			curr = ThenApply(curr, func(v int) int {
				return v + 1
			})
		}
		// 不调用 Join，让 curr 自然消亡，测试 GC 和内存分配情况
	}
}
func BenchmarkFuture_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 1. 创建 Future
			f := SupplyAsync(func() int {
				return 1 // 做一点点简单的计算
			})

			// 2. 必须等待结果，否则异步任务可能还没跑完 benchmark 就结束了
			// 同时，Get() 的返回值赋给一个局部变量，防止被优化
			_, _ = f.Get(context.TODO())
		}
	})
}
