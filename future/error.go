package future

import (
	"fmt"
	"github.com/xigexb/go-future/pool"
)

// Exceptionally
func (f *CompletableFuture[T]) Exceptionally(fn func(error) (T, error)) *CompletableFuture[T] {
	return uniExceptionally(f, fn, false)
}

func (f *CompletableFuture[T]) ExceptionallyAsync(fn func(error) (T, error)) *CompletableFuture[T] {
	return uniExceptionally(f, fn, true)
}

func uniExceptionally[T any](f *CompletableFuture[T], fn func(error) (T, error), async bool) *CompletableFuture[T] {
	dest := New[T]()
	f.whenCompleteInternal(func(val T, err error) {
		if err == nil {
			dest.Complete(val)
			return
		}
		task := func() {
			defer func() {
				if r := recover(); r != nil {
					dest.CompleteExceptionally(fmt.Errorf("panic in Exceptionally: %v", r))
				}
			}()
			v, e := fn(err)
			if e != nil {
				dest.CompleteExceptionally(e)
			} else {
				dest.Complete(v)
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

// ExceptionallyCompose 对应 Java 12: exceptionallyCompose
func (f *CompletableFuture[T]) ExceptionallyCompose(fn func(error) *CompletableFuture[T]) *CompletableFuture[T] {
	return uniExceptionallyCompose(f, fn, false)
}

func (f *CompletableFuture[T]) ExceptionallyComposeAsync(fn func(error) *CompletableFuture[T]) *CompletableFuture[T] {
	return uniExceptionallyCompose(f, fn, true)
}

func uniExceptionallyCompose[T any](f *CompletableFuture[T], fn func(error) *CompletableFuture[T], async bool) *CompletableFuture[T] {
	dest := New[T]()
	f.whenCompleteInternal(func(val T, err error) {
		if err == nil {
			dest.Complete(val)
			return
		}
		task := func() {
			defer func() {
				if r := recover(); r != nil {
					dest.CompleteExceptionally(fmt.Errorf("panic in ExceptionallyCompose: %v", r))
				}
			}()
			relay := fn(err)
			if relay == nil {
				dest.CompleteExceptionally(ErrNilFunction)
				return
			}
			relay.whenCompleteInternal(func(v T, e error) {
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

// Handle
func (f *CompletableFuture[T]) Handle(fn func(T, error) T) *CompletableFuture[T] {
	return uniHandle(f, fn, false)
}

func (f *CompletableFuture[T]) HandleAsync(fn func(T, error) T) *CompletableFuture[T] {
	return uniHandle(f, fn, true)
}

func uniHandle[T any](f *CompletableFuture[T], fn func(T, error) T, async bool) *CompletableFuture[T] {
	dest := New[T]()
	f.whenCompleteInternal(func(val T, err error) {
		task := func() {
			defer func() {
				if r := recover(); r != nil {
					dest.CompleteExceptionally(fmt.Errorf("panic in Handle: %v", r))
				}
			}()
			res := fn(val, err)
			dest.Complete(res)
		}
		if async {
			pool.GlobalExecutor.Submit(task)
		} else {
			task()
		}
	})
	return dest
}
