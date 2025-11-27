package future

import (
	"fmt"

	"github.com/xigexb/go-future/pool"
)

// ============ 1. ThenApply (Map: T -> V) ============

// ThenApply 同步执行转换
func ThenApply[T any, V any](src *CompletableFuture[T], fn func(T) V) *CompletableFuture[V] {
	return uniApply(src, fn, false)
}

// ThenApplyAsync 异步执行转换
func ThenApplyAsync[T any, V any](src *CompletableFuture[T], fn func(T) V) *CompletableFuture[V] {
	return uniApply(src, fn, true)
}

func uniApply[T any, V any](src *CompletableFuture[T], fn func(T) V, async bool) *CompletableFuture[V] {
	dest := New[V]()
	src.whenCompleteInternal(func(val T, err error) {
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
			pool.GlobalExecutor.Submit(task)
		} else {
			task()
		}
	})
	return dest
}

// ============ 2. ThenAccept (Consumer: T -> Void) ============

func (f *CompletableFuture[T]) ThenAccept(fn func(T)) *CompletableFuture[struct{}] {
	return ThenApply(f, func(v T) struct{} { fn(v); return struct{}{} })
}

func (f *CompletableFuture[T]) ThenAcceptAsync(fn func(T)) *CompletableFuture[struct{}] {
	return ThenApplyAsync(f, func(v T) struct{} { fn(v); return struct{}{} })
}

// ============ 3. ThenRun (Runnable: Void -> Void) ============

func (f *CompletableFuture[T]) ThenRun(action func()) *CompletableFuture[struct{}] {
	return ThenApply(f, func(_ T) struct{} { action(); return struct{}{} })
}

func (f *CompletableFuture[T]) ThenRunAsync(action func()) *CompletableFuture[struct{}] {
	return ThenApplyAsync(f, func(_ T) struct{} { action(); return struct{}{} })
}

// ============ 4. ThenCompose (FlatMap: T -> Future[V]) ============

func ThenCompose[T any, V any](src *CompletableFuture[T], fn func(T) *CompletableFuture[V]) *CompletableFuture[V] {
	return uniCompose(src, fn, false)
}

func ThenComposeAsync[T any, V any](src *CompletableFuture[T], fn func(T) *CompletableFuture[V]) *CompletableFuture[V] {
	return uniCompose(src, fn, true)
}

func uniCompose[T any, V any](src *CompletableFuture[T], fn func(T) *CompletableFuture[V], async bool) *CompletableFuture[V] {
	dest := New[V]()
	src.whenCompleteInternal(func(val T, err error) {
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
			relay.whenCompleteInternal(func(v V, e error) {
				if e != nil {
					dest.CompleteExceptionally(e)
				} else {
					dest.Complete(v)
				}
			})
		}
		if async {
			pool.GlobalExecutor.Submit(task)
		} else {
			task()
		}
	})
	return dest
}

// ============ 5. WhenComplete (Peeking) ============

func (f *CompletableFuture[T]) WhenComplete(action func(T, error)) *CompletableFuture[T] {
	return uniWhenComplete(f, action, false)
}

func (f *CompletableFuture[T]) WhenCompleteAsync(action func(T, error)) *CompletableFuture[T] {
	return uniWhenComplete(f, action, true)
}

func uniWhenComplete[T any](src *CompletableFuture[T], action func(T, error), async bool) *CompletableFuture[T] {
	dest := New[T]()
	src.whenCompleteInternal(func(val T, err error) {
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
			pool.GlobalExecutor.Submit(task)
		} else {
			task()
		}
	})
	return dest
}
