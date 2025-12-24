package future

import (
	"context"

	"github.com/xigexb/go-future/pool"
)

// ============ SupplyAsync (有返回值) ============

func SupplyAsync[T any](supplier func() T) *CompletableFuture[T] {
	return SupplyAsyncCtxWithExecutor(context.Background(), nil, supplier)
}

func SupplyAsyncCtx[T any](ctx context.Context, supplier func() T) *CompletableFuture[T] {
	return SupplyAsyncCtxWithExecutor(ctx, nil, supplier)
}

func SupplyAsyncWithExecutor[T any](executor pool.Executor, supplier func() T) *CompletableFuture[T] {
	return SupplyAsyncCtxWithExecutor(context.Background(), executor, supplier)
}

func SupplyAsyncCtxWithExecutor[T any](ctx context.Context, executor pool.Executor, supplier func() T) *CompletableFuture[T] {
	// 自动创建，无需 Pool 复用逻辑
	f := NewWithContext[T](ctx)
	if supplier == nil {
		f.CompleteExceptionally(ErrNilFunction)
		return f
	}

	exec := executor
	if exec == nil {
		exec = pool.GlobalExecutor
	}

	exec.Submit(func() {
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

// ============ RunAsync (无返回值) ============

func RunAsync(runnable func()) *CompletableFuture[struct{}] {
	return RunAsyncCtxWithExecutor(context.Background(), nil, runnable)
}

func RunAsyncCtx(ctx context.Context, runnable func()) *CompletableFuture[struct{}] {
	return RunAsyncCtxWithExecutor(ctx, nil, runnable)
}

func RunAsyncWithExecutor(executor pool.Executor, runnable func()) *CompletableFuture[struct{}] {
	return RunAsyncCtxWithExecutor(context.Background(), executor, runnable)
}

func RunAsyncCtxWithExecutor(ctx context.Context, executor pool.Executor, runnable func()) *CompletableFuture[struct{}] {
	f := NewWithContext[struct{}](ctx)
	if runnable == nil {
		f.CompleteExceptionally(ErrNilFunction)
		return f
	}

	exec := executor
	if exec == nil {
		exec = pool.GlobalExecutor
	}

	exec.Submit(func() {
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

func CompletedFuture[T any](val T) *CompletableFuture[T] {
	f := New[T]()
	f.Complete(val)
	return f
}

func FailedFuture[T any](err error) *CompletableFuture[T] {
	f := New[T]()
	f.CompleteExceptionally(err)
	return f
}
