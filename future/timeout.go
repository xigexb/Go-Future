package future

import (
	"time"
)

// OrTimeout 如果在指定时间内未完成，则抛出 ErrTimeout 异常
func (f *CompletableFuture[T]) OrTimeout(d time.Duration) *CompletableFuture[T] {
	if f.IsDone() {
		return f
	}

	go func() {
		select {
		case <-time.After(d):
			// 尝试以超时失败结束
			// 利用 CAS 机制保证线程安全，无需手动加锁
			f.CompleteExceptionally(ErrTimeout)
		case <-f.getDoneChanLazy():
			// 任务在超时前已完成，无需操作
		}
	}()
	return f
}

// CompleteOnTimeout 如果在指定时间内未完成，则使用给定的默认值完成
func (f *CompletableFuture[T]) CompleteOnTimeout(value T, d time.Duration) *CompletableFuture[T] {
	if f.IsDone() {
		return f
	}

	go func() {
		select {
		case <-time.After(d):
			// 尝试写入默认值
			f.Complete(value)
		case <-f.getDoneChanLazy():
			// 任务在超时前已完成
		}
	}()
	return f
}
