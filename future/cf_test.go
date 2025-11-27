package future

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// 辅助断言函数
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

// 1. 测试基本的异步执行和结果获取
func TestSupplyAsync_Get(t *testing.T) {
	f := SupplyAsync(func() int {
		return 100
	})

	val, err := f.Get(context.Background())
	assertNil(t, err)
	assertEqual(t, val, 100)
}

// 2. 测试链式调用 (Map / Apply)
func TestThenApply(t *testing.T) {
	f1 := SupplyAsync(func() int {
		return 10
	})

	// int -> string
	f2 := ThenApply(f1, func(v int) string {
		return fmt.Sprintf("val:%d", v)
	})

	// string -> string
	f3 := ThenApply(f2, func(s string) string {
		return s + "!"
	})

	res, err := f3.Join()
	assertNil(t, err)
	assertEqual(t, res, "val:10!")
}

// 3. 测试 FlatMap (ThenCompose)
func TestThenCompose(t *testing.T) {
	f := SupplyAsync(func() int { return 1 })

	f2 := ThenCompose(f, func(v int) *CompletableFuture[int] {
		// 返回一个新的 Future
		return SupplyAsync(func() int {
			return v + 10
		})
	})

	res, err := f2.Join()
	assertNil(t, err)
	assertEqual(t, res, 11)
}

// 4. 测试异常处理和 Panic 捕获
func TestPanicHandling(t *testing.T) {
	f := SupplyAsync(func() int {
		panic("boom")
		return 0
	})

	_, err := f.Join()
	if err == nil {
		t.Fatal("Expected error from panic, got nil")
	}
	// 验证错误信息包含 panic 内容
	if err.Error() != "panic: boom" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// 5. 测试 Exceptionally 恢复
func TestExceptionally(t *testing.T) {
	f := SupplyAsync(func() int {
		panic("fail")
		return 0
	})

	// 发生错误时返回 -1
	// 修改点：现在需要返回 (T, error)
	fRecovered := f.Exceptionally(func(err error) (int, error) {
		return -1, nil // nil 表示成功恢复
	})

	res, err := fRecovered.Join()
	assertNil(t, err)
	assertEqual(t, res, -1)
}

// 6. 测试 AllOf (所有成功)
func TestAllOf_Success(t *testing.T) {
	count := 10
	futures := make([]*CompletableFuture[int], count)
	for i := 0; i < count; i++ {
		i := i
		futures[i] = SupplyAsync(func() int {
			time.Sleep(10 * time.Millisecond)
			return i
		})
	}

	all := AllOf(futures...)
	_, err := all.Join()
	assertNil(t, err)

	for _, f := range futures {
		if !f.IsDone() {
			t.Error("Child future should be done")
		}
	}
}

// 7. 测试 AllOf (快速失败 Fail-Fast)
func TestAllOf_FailFast(t *testing.T) {
	// 任务1: 慢，成功
	f1 := SupplyAsync(func() int {
		time.Sleep(200 * time.Millisecond)
		return 1
	})

	// 任务2: 手动创建 Future
	f2 := New[int]()

	// 模拟 10ms 后发生错误
	go func() {
		time.Sleep(10 * time.Millisecond)
		f2.CompleteExceptionally(errors.New("fast fail"))
	}()

	start := time.Now()
	all := AllOf(f1, f2)
	_, err := all.Join()
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != "fast fail" {
		t.Errorf("Expected 'fast fail', got %v", err)
	}

	if duration > 100*time.Millisecond {
		t.Errorf("AllOf did not fail fast, took %v", duration)
	}
}

// 8. 测试超时控制 OrTimeout
func TestOrTimeout(t *testing.T) {
	f := SupplyAsync(func() int {
		time.Sleep(200 * time.Millisecond)
		return 1
	})

	f.OrTimeout(50 * time.Millisecond)

	_, err := f.Join()
	if err != ErrTimeout {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
}

// 9. 测试 AnyOf
func TestAnyOf(t *testing.T) {
	f1 := SupplyAsync(func() int {
		time.Sleep(200 * time.Millisecond)
		return 1
	})
	f2 := SupplyAsync(func() int {
		time.Sleep(10 * time.Millisecond)
		return 2
	})

	anyF := AnyOf(f1, f2)
	res, err := anyF.Join()
	assertNil(t, err)
	assertEqual(t, res, 2)
}

// 10. 测试取消 Cancel
func TestCancel(t *testing.T) {
	f := SupplyAsync(func() int {
		time.Sleep(1 * time.Second)
		return 1
	})

	go func() {
		time.Sleep(10 * time.Millisecond)
		f.Cancel(true)
	}()

	_, err := f.Join()
	if err != ErrCanceled {
		t.Errorf("Expected ErrCanceled, got %v", err)
	}
}

// 11. 测试 ThenCombine (合并 T 和 U)
func TestThenCombine(t *testing.T) {
	f1 := SupplyAsync(func() int { return 10 })
	f2 := SupplyAsync(func() string { return " apples" })

	// 合并 int 和 string -> string
	fCombined := ThenCombine(f1, f2, func(n int, s string) string {
		return fmt.Sprintf("%d%s", n, s)
	})

	res, err := fCombined.Join()
	assertNil(t, err)
	assertEqual(t, res, "10 apples")
}

// 12. 测试 ThenCombine 的异常处理
func TestThenCombine_Error(t *testing.T) {
	f1 := SupplyAsync(func() int { return 10 })
	f2 := SupplyAsync(func() string {
		panic("fail")
		return ""
	})

	fCombined := ThenCombine(f1, f2, func(n int, s string) string {
		return "should not happen"
	})

	_, err := fCombined.Join()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// 13. 测试 ApplyToEither (谁快用谁)
func TestApplyToEither(t *testing.T) {
	fSlow := SupplyAsync(func() int {
		time.Sleep(100 * time.Millisecond)
		return 1
	})
	fFast := SupplyAsync(func() int {
		time.Sleep(10 * time.Millisecond)
		return 2
	})

	fRes := ApplyToEither(fSlow, fFast, func(val int) string {
		return fmt.Sprintf("Winner is %d", val)
	})

	res, err := fRes.Join()
	assertNil(t, err)
	assertEqual(t, res, "Winner is 2")
}
