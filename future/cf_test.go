package future

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xigexb/go-future/pool"
)

// ============ 辅助工具 ============

// mockExecutor 用于验证任务是否提交到了指定的执行器
type mockExecutor struct {
	submitCount int32
}

func (m *mockExecutor) Submit(task pool.Runnable) {
	atomic.AddInt32(&m.submitCount, 1)
	go task()
}

func assertNil(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Fatalf("Expected nil error, got: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	if got != want {
		t.Helper()
		t.Errorf("Expected %v, got %v", want, got)
	}
}

// ============ 基础功能测试 ============

func TestSupplyAsync_Basic(t *testing.T) {
	f := SupplyAsync(func() int {
		return 100
	})
	val, err := f.Join()
	assertNil(t, err)
	assertEqual(t, val, 100)
}

func TestException_Handling(t *testing.T) {
	f := SupplyAsync(func() int {
		panic("boom")
	})

	// 验证 Exceptionally 恢复机制
	fRec := f.Exceptionally(func(err error) (int, error) {
		return -1, nil
	})

	val, err := fRec.Join()
	assertNil(t, err)
	assertEqual(t, val, -1)

	// 验证原始 Future 的错误状态
	if f.ExceptionNow() == nil {
		t.Error("Original future should have error")
	}
}

// ============ 新特性：自定义协程池测试 ============

func TestSupplyAsync_WithCustomExecutor(t *testing.T) {
	mock := &mockExecutor{}

	// 使用自定义池执行
	f := SupplyAsyncWithExecutor(mock, func() string {
		return "done"
	})

	res, err := f.Join()
	assertNil(t, err)
	assertEqual(t, res, "done")

	// 验证 mockExecutor 是否被调用
	if atomic.LoadInt32(&mock.submitCount) != 1 {
		t.Errorf("Expected custom executor to be called 1 time, got %d", mock.submitCount)
	}
}

func TestThenApplyAsync_WithCustomExecutor(t *testing.T) {
	mock := &mockExecutor{}

	f1 := SupplyAsync(func() int { return 1 })

	// 链式调用指定池
	f2 := ThenApplyAsyncWithExecutor(f1, mock, func(v int) int {
		return v + 1
	})

	val, _ := f2.Join()
	assertEqual(t, val, 2)

	// 验证 mockExecutor 是否被调用
	// 注意：SupplyAsync 用的是默认池，ThenApplyAsync 用的才是 mock
	if atomic.LoadInt32(&mock.submitCount) != 1 {
		t.Errorf("Expected custom executor to be called 1 time, got %d", mock.submitCount)
	}
}

func TestGlobalExecutor_Replacement(t *testing.T) {
	// 保存旧的，测试完恢复
	original := pool.GlobalExecutor
	defer pool.SetGlobalExecutor(original)

	mock := &mockExecutor{}
	pool.SetGlobalExecutor(mock)

	// 现在的普通 SupplyAsync 应该走 Mock 池
	SupplyAsync(func() int { return 1 }).Join()

	if atomic.LoadInt32(&mock.submitCount) != 1 {
		t.Error("Global executor replacement failed")
	}
}

// ============ 上下文与超时测试 ============

func TestContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建一个会阻塞的任务
	f := SupplyAsyncCtx(ctx, func() int {
		time.Sleep(1 * time.Second)
		return 1
	})

	// 立即取消
	cancel()

	_, err := f.Join()
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestOrTimeout_Legacy(t *testing.T) {
	f := SupplyAsync(func() int {
		time.Sleep(100 * time.Millisecond)
		return 1
	})

	// 50ms 超时
	f.OrTimeout(50 * time.Millisecond)

	_, err := f.Join()
	if err != ErrTimeout {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
}

// ============ 组合测试 (AllOf/AnyOf) ============

func TestAllOf_Concurrency(t *testing.T) {
	count := 50
	futures := make([]*CompletableFuture[int], count)

	for i := 0; i < count; i++ {
		i := i
		futures[i] = SupplyAsync(func() int {
			time.Sleep(time.Duration(i) * time.Millisecond)
			return i
		})
	}

	all := AllOf(futures...)
	_, err := all.Join()
	assertNil(t, err)

	for i, f := range futures {
		if val, _ := f.GetNow(-1); val != i {
			t.Errorf("Future %d result mismatch", i)
		}
	}
}

// ============ 竞态检测 (Run with -race) ============

// ============ 竞态检测 (Run with -race) ============

func TestRaceCondition(t *testing.T) {
	// 模拟高并发下的回调注册和完成
	f := New[int]()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			// 修正：Go 泛型不支持方法引入新类型参数，必须使用函数形式 ThenApply(f, ...)
			// 错误写法: f.ThenApply(...)
			// 正确写法:
			ThenApply(f, func(v int) int { return v })
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond)
		f.Complete(1)
	}()

	wg.Wait()
}
