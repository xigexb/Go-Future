package future

import (
	"errors"
	"github.com/xigexb/go-future/pool"
	"sync/atomic"
)

// ============ Multi-Future Aggregation ============

// AllOf (Fail-Fast)
func AllOf[T any](futures ...*CompletableFuture[T]) *CompletableFuture[struct{}] {
	dest := New[struct{}]()
	n := len(futures)
	if n == 0 {
		dest.Complete(struct{}{})
		return dest
	}
	var pending int32 = int32(n)
	var doneFlag int32 = 0
	for _, f := range futures {
		f.whenCompleteInternal(func(_ T, err error) {
			if atomic.LoadInt32(&doneFlag) == 1 {
				return
			}
			if err != nil {
				if atomic.CompareAndSwapInt32(&doneFlag, 0, 1) {
					dest.CompleteExceptionally(err)
				}
				return
			}
			if atomic.AddInt32(&pending, -1) == 0 {
				if atomic.CompareAndSwapInt32(&doneFlag, 0, 1) {
					dest.Complete(struct{}{})
				}
			}
		})
	}
	return dest
}

// AnyOf
func AnyOf[T any](futures ...*CompletableFuture[T]) *CompletableFuture[T] {
	dest := New[T]()
	if len(futures) == 0 {
		dest.CompleteExceptionally(errors.New("no futures"))
		return dest
	}
	var doneFlag int32 = 0
	for _, f := range futures {
		f.whenCompleteInternal(func(val T, err error) {
			if atomic.CompareAndSwapInt32(&doneFlag, 0, 1) {
				if err != nil {
					dest.CompleteExceptionally(err)
				} else {
					dest.Complete(val)
				}
			}
		})
	}
	return dest
}

// ============ Binary: AND (ThenCombine) ============

func ThenCombine[T any, U any, V any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], fn func(T, U) V) *CompletableFuture[V] {
	return biApply(f1, f2, fn, false)
}

func ThenCombineAsync[T any, U any, V any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], fn func(T, U) V) *CompletableFuture[V] {
	return biApply(f1, f2, fn, true)
}

// ThenAcceptBoth
func ThenAcceptBoth[T any, U any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], fn func(T, U)) *CompletableFuture[struct{}] {
	return ThenCombine(f1, f2, func(t T, u U) struct{} { fn(t, u); return struct{}{} })
}

func ThenAcceptBothAsync[T any, U any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], fn func(T, U)) *CompletableFuture[struct{}] {
	return ThenCombineAsync(f1, f2, func(t T, u U) struct{} { fn(t, u); return struct{}{} })
}

// RunAfterBoth
func RunAfterBoth[T any, U any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], action func()) *CompletableFuture[struct{}] {
	return ThenCombine(f1, f2, func(_ T, _ U) struct{} { action(); return struct{}{} })
}

func RunAfterBothAsync[T any, U any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], action func()) *CompletableFuture[struct{}] {
	return ThenCombineAsync(f1, f2, func(_ T, _ U) struct{} { action(); return struct{}{} })
}

func biApply[T any, U any, V any](f1 *CompletableFuture[T], f2 *CompletableFuture[U], fn func(T, U) V, async bool) *CompletableFuture[V] {
	dest := New[V]()
	// 简单的非阻塞实现：在一个新协程等待两者
	// 注：这里为了简化逻辑使用 Join，更底层的实现应该使用计数器回调
	pool.GlobalExecutor.Submit(func() {
		v1, err1 := f1.Join()
		if err1 != nil {
			dest.CompleteExceptionally(err1)
			return
		}
		v2, err2 := f2.Join()
		if err2 != nil {
			dest.CompleteExceptionally(err2)
			return
		}
		task := func() {
			res, panicErr := safecall(func() V { return fn(v1, v2) })
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

// ============ Binary: OR (ApplyToEither) ============

func ApplyToEither[T any, V any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], fn func(T) V) *CompletableFuture[V] {
	return orApply(f1, f2, fn, false)
}

func ApplyToEitherAsync[T any, V any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], fn func(T) V) *CompletableFuture[V] {
	return orApply(f1, f2, fn, true)
}

func AcceptEither[T any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], fn func(T)) *CompletableFuture[struct{}] {
	return ApplyToEither(f1, f2, func(t T) struct{} { fn(t); return struct{}{} })
}

func AcceptEitherAsync[T any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], fn func(T)) *CompletableFuture[struct{}] {
	return ApplyToEitherAsync(f1, f2, func(t T) struct{} { fn(t); return struct{}{} })
}

func RunAfterEither[T any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], action func()) *CompletableFuture[struct{}] {
	return ApplyToEither(f1, f2, func(_ T) struct{} { action(); return struct{}{} })
}

func RunAfterEitherAsync[T any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], action func()) *CompletableFuture[struct{}] {
	return ApplyToEitherAsync(f1, f2, func(_ T) struct{} { action(); return struct{}{} })
}

func orApply[T any, V any](f1 *CompletableFuture[T], f2 *CompletableFuture[T], fn func(T) V, async bool) *CompletableFuture[V] {
	dest := New[V]()
	var done int32 = 0
	cb := func(val T, err error) {
		if atomic.CompareAndSwapInt32(&done, 0, 1) {
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
		}
	}
	f1.whenCompleteInternal(cb)
	f2.whenCompleteInternal(cb)
	return dest
}
