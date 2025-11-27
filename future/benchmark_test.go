package future

import (
	"sync"
	"testing"
)

// 基准测试：原生 Goroutine + WaitGroup (基准线)
func BenchmarkNativeGoroutine(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				// 模拟微小任务
				_ = 1 + 1
				wg.Done()
			}()
			wg.Wait()
		}
	})
}

// 基准测试：Go-Future 的 SupplyAsync + Join
func BenchmarkFutureSupplyAsync(b *testing.B) {
	// 防止优化
	var res int

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f := SupplyAsync(func() int {
				return 1 + 1
			})
			val, _ := f.Join()
			res = val
		}
	})
	_ = res
}

// 基准测试：链式调用开销
func BenchmarkFutureChaining(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f := SupplyAsync(func() int {
				return 1
			})
			f2 := ThenApply(f, func(v int) int {
				return v + 1
			})
			_, _ = f2.Join()
		}
	})
}
