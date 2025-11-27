package future

import (
	"sync/atomic"
	"time"
)

// OrTimeout 对应 Java 9: orTimeout
func (f *CompletableFuture[T]) OrTimeout(d time.Duration) *CompletableFuture[T] {
	go func() {
		select {
		case <-time.After(d):
			// 1. 优化：先做一次无锁检查
			if f.IsDone() {
				return
			}

			f.mu.Lock()
			// 2. 锁内再次检查状态 (使用 state 代替 isDone)
			if f.state != stateDone {
				f.mu.Unlock()
				// 解锁后调用，CompleteExceptionally 内部有原子 CAS 检查，是安全的
				f.CompleteExceptionally(ErrTimeout)
			} else {
				f.mu.Unlock()
			}
		case <-f.doneChan:
		}
	}()
	return f
}

// CompleteOnTimeout 对应 Java 9: completeOnTimeout
func (f *CompletableFuture[T]) CompleteOnTimeout(value T, d time.Duration) *CompletableFuture[T] {
	go func() {
		select {
		case <-time.After(d):
			f.Complete(value)
		case <-f.doneChan:
		}
	}()
	return f
}

// Cancel 对应 Java: cancel(boolean)
func (f *CompletableFuture[T]) Cancel(mayInterruptIfRunning bool) bool {
	// 1. 快速检查
	if f.IsDone() {
		return false
	}

	f.mu.Lock()
	// 2. 锁内检查 (使用 state 代替 isDone)
	if f.state == stateDone {
		f.mu.Unlock()
		return false
	}

	// 3. 修改状态
	atomic.StoreInt32(&f.state, stateDone)
	f.err = ErrCanceled
	close(f.doneChan)
	cbs := f.callbacks
	f.callbacks = nil
	f.mu.Unlock()

	// 4. 触发 Context 取消 (级联取消)
	f.cancel()

	// 5. 触发回调
	var zero T
	for _, cb := range cbs {
		cb(zero, ErrCanceled)
	}
	return true
}
