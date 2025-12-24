package future

import (
	"fmt"

	"github.com/xigexb/go-future/pool"
)

// ============ 1. ThenApply ============

func ThenApply[T any, V any](src *CompletableFuture[T], fn func(T) V) *CompletableFuture[V] {
	return uniApply(src, fn, false, nil)
}

func ThenApplyAsync[T any, V any](src *CompletableFuture[T], fn func(T) V) *CompletableFuture[V] {
	return uniApply(src, fn, true, nil)
}

func ThenApplyAsyncWithExecutor[T any, V any](src *CompletableFuture[T], executor pool.Executor, fn func(T) V) *CompletableFuture[V] {
	return uniApply(src, fn, true, executor)
}

func uniApply[T any, V any](src *CompletableFuture[T], fn func(T) V, async bool, executor pool.Executor) *CompletableFuture[V] {
	dest := New[V]()

	execTask := func(val T, err error) {
		if err != nil {
			dest.CompleteExceptionally(err)
			return
		}
		task := func() {
			res, panicErr := safecall(func() V { return fn(val) })
			if panicErr != nil {
				dest.CompleteExceptionally(panicErr)
			} else {
				dest.Complete(res)
			}
		}
		if async {
			exec := executor
			if exec == nil {
				exec = pool.GlobalExecutor
			}
			exec.Submit(task)
		} else {
			task()
		}
	}

	// 快速路径优化：如果上游已经完成，且不需要异步切换，直接执行
	if src.IsDone() {
		execTask(src.value, src.err)
	} else {
		src.whenCompleteInternal(execTask)
	}
	return dest
}

// ============ 2. ThenAccept ============

func (f *CompletableFuture[T]) ThenAccept(fn func(T)) *CompletableFuture[struct{}] {
	return ThenApply(f, func(v T) struct{} { fn(v); return struct{}{} })
}

func (f *CompletableFuture[T]) ThenAcceptAsync(fn func(T)) *CompletableFuture[struct{}] {
	return ThenApplyAsync(f, func(v T) struct{} { fn(v); return struct{}{} })
}

func (f *CompletableFuture[T]) ThenAcceptAsyncWithExecutor(executor pool.Executor, fn func(T)) *CompletableFuture[struct{}] {
	return ThenApplyAsyncWithExecutor(f, executor, func(v T) struct{} { fn(v); return struct{}{} })
}

// ============ 3. ThenRun ============

func (f *CompletableFuture[T]) ThenRun(action func()) *CompletableFuture[struct{}] {
	return ThenApply(f, func(_ T) struct{} { action(); return struct{}{} })
}

func (f *CompletableFuture[T]) ThenRunAsync(action func()) *CompletableFuture[struct{}] {
	return ThenApplyAsync(f, func(_ T) struct{} { action(); return struct{}{} })
}

func (f *CompletableFuture[T]) ThenRunAsyncWithExecutor(executor pool.Executor, action func()) *CompletableFuture[struct{}] {
	return ThenApplyAsyncWithExecutor(f, executor, func(_ T) struct{} { action(); return struct{}{} })
}

// ============ 4. ThenCompose ============

func ThenCompose[T any, V any](src *CompletableFuture[T], fn func(T) *CompletableFuture[V]) *CompletableFuture[V] {
	return uniCompose(src, fn, false, nil)
}

func ThenComposeAsync[T any, V any](src *CompletableFuture[T], fn func(T) *CompletableFuture[V]) *CompletableFuture[V] {
	return uniCompose(src, fn, true, nil)
}

func ThenComposeAsyncWithExecutor[T any, V any](src *CompletableFuture[T], executor pool.Executor, fn func(T) *CompletableFuture[V]) *CompletableFuture[V] {
	return uniCompose(src, fn, true, executor)
}

func uniCompose[T any, V any](src *CompletableFuture[T], fn func(T) *CompletableFuture[V], async bool, executor pool.Executor) *CompletableFuture[V] {
	dest := New[V]()

	execTask := func(val T, err error) {
		if err != nil {
			dest.CompleteExceptionally(err)
			return
		}
		task := func() {
			defer func() {
				if r := recover(); r != nil {
					dest.CompleteExceptionally(fmt.Errorf("panic in ThenCompose: %v", r))
				}
			}()

			relay := fn(val)
			if relay == nil {
				dest.CompleteExceptionally(ErrNilFunction)
				return
			}

			// Relay 也可以走快速路径
			if relay.IsDone() {
				v, e := relay.value, relay.err
				if e != nil {
					dest.CompleteExceptionally(e)
				} else {
					dest.Complete(v)
				}
			} else {
				relay.whenCompleteInternal(func(v V, e error) {
					if e != nil {
						dest.CompleteExceptionally(e)
					} else {
						dest.Complete(v)
					}
				})
			}
		}
		if async {
			exec := executor
			if exec == nil {
				exec = pool.GlobalExecutor
			}
			exec.Submit(task)
		} else {
			task()
		}
	}

	if src.IsDone() {
		execTask(src.value, src.err)
	} else {
		src.whenCompleteInternal(execTask)
	}
	return dest
}

// ============ 5. WhenComplete ============

func (f *CompletableFuture[T]) WhenComplete(action func(T, error)) *CompletableFuture[T] {
	return uniWhenComplete(f, action, false, nil)
}

func (f *CompletableFuture[T]) WhenCompleteAsync(action func(T, error)) *CompletableFuture[T] {
	return uniWhenComplete(f, action, true, nil)
}

func (f *CompletableFuture[T]) WhenCompleteAsyncWithExecutor(executor pool.Executor, action func(T, error)) *CompletableFuture[T] {
	return uniWhenComplete(f, action, true, executor)
}

func uniWhenComplete[T any](src *CompletableFuture[T], action func(T, error), async bool, executor pool.Executor) *CompletableFuture[T] {
	dest := New[T]()

	execTask := func(val T, err error) {
		task := func() {
			func() {
				defer func() { recover() }()
				action(val, err)
			}()
			if err != nil {
				dest.CompleteExceptionally(err)
			} else {
				dest.Complete(val)
			}
		}
		if async {
			exec := executor
			if exec == nil {
				exec = pool.GlobalExecutor
			}
			exec.Submit(task)
		} else {
			task()
		}
	}

	if src.IsDone() {
		execTask(src.value, src.err)
	} else {
		src.whenCompleteInternal(execTask)
	}
	return dest
}
