package future

import (
	"context"

	"github.com/xigexb/go-future/pool"
)

// SupplyAsync 对应 Java: supplyAsync
func SupplyAsync[T any](supplier func() T) *CompletableFuture[T] {
	return SupplyAsyncCtx(context.Background(), supplier)
}

// SupplyAsyncCtx 支持 Context 传递（Go 特有优化）
// 允许任务感知上游的取消信号 (TraceID, Timeout 等)
func SupplyAsyncCtx[T any](ctx context.Context, supplier func() T) *CompletableFuture[T] {
	f := NewWithContext[T](ctx)
	if supplier == nil {
		f.CompleteExceptionally(ErrNilFunction)
		return f
	}
	pool.GlobalExecutor.Submit(func() {
		// 任务开始前先检查上下文状态
		if ctx.Err() != nil {
			f.CompleteExceptionally(ctx.Err())
			return
		}
		val, err := safecall(func() T { return supplier() })
		if err != nil {
			f.CompleteExceptionally(err)
		} else {
			f.Complete(val)
		}
	})
	return f
}

// RunAsync 对应 Java: runAsync
func RunAsync(runnable func()) *CompletableFuture[struct{}] {
	return RunAsyncCtx(context.Background(), runnable)
}

// RunAsyncCtx 支持 Context 传递
func RunAsyncCtx(ctx context.Context, runnable func()) *CompletableFuture[struct{}] {
	f := NewWithContext[struct{}](ctx)
	if runnable == nil {
		f.CompleteExceptionally(ErrNilFunction)
		return f
	}
	pool.GlobalExecutor.Submit(func() {
		if ctx.Err() != nil {
			f.CompleteExceptionally(ctx.Err())
			return
		}
		_, err := safecall(func() int {
			runnable()
			return 0
		})
		if err != nil {
			f.CompleteExceptionally(err)
		} else {
			f.Complete(struct{}{})
		}
	})
	return f
}

// CompletedFuture 对应 Java: completedFuture
func CompletedFuture[T any](val T) *CompletableFuture[T] {
	f := New[T]()
	f.Complete(val)
	return f
}

// FailedFuture 对应 Java 9: failedFuture
func FailedFuture[T any](err error) *CompletableFuture[T] {
	f := New[T]()
	f.CompleteExceptionally(err)
	return f
}
