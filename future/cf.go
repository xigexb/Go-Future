package future

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/xigexb/go-future/pool"
)

var (
	ErrCanceled    = errors.New("completable future canceled")
	ErrTimeout     = errors.New("completable future timed out")
	ErrNilFunction = errors.New("function cannot be nil")
	ErrNoResult    = errors.New("future not completed")
)

// 状态常量 (用于原子操作)
const (
	statePending = 0
	stateDone    = 1
)

// Callback 内部回调链节点
type callback[T any] func(val T, err error)

// CompletableFuture 核心结构
type CompletableFuture[T any] struct {
	value     T
	err       error
	state     int32 // 原子状态标记 (0: Pending, 1: Done)
	mu        sync.Mutex
	doneChan  chan struct{}
	callbacks []callback[T]
	ctx       context.Context
	cancel    context.CancelFunc
}

// New 创建
func New[T any]() *CompletableFuture[T] {
	return NewWithContext[T](context.Background())
}

// NewWithContext 使用父 Context 创建 Future (支持链路追踪/级联取消)
func NewWithContext[T any](parent context.Context) *CompletableFuture[T] {
	ctx, cancel := context.WithCancel(parent)
	return &CompletableFuture[T]{
		state:     statePending,
		doneChan:  make(chan struct{}),
		callbacks: make([]callback[T], 0, 4),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// ============ State Inspection ============

// IsDone 原子检查，无锁，极大提高高并发下的轮询性能
func (f *CompletableFuture[T]) IsDone() bool {
	return atomic.LoadInt32(&f.state) == stateDone
}

func (f *CompletableFuture[T]) IsCancelled() bool {
	if !f.IsDone() {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err == ErrCanceled
}

func (f *CompletableFuture[T]) IsCompletedExceptionally() bool {
	if !f.IsDone() {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err != nil && f.err != ErrCanceled
}

// ResultNow 对应 Java 19: resultNow()
func (f *CompletableFuture[T]) ResultNow() T {
	if !f.IsDone() {
		panic("Future not completed")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		panic(fmt.Sprintf("Future completed exceptionally: %v", f.err))
	}
	return f.value
}

// ExceptionNow 对应 Java 19: exceptionNow()
func (f *CompletableFuture[T]) ExceptionNow() error {
	if !f.IsDone() {
		panic("Future not completed")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err == nil {
		panic("Future completed normally")
	}
	return f.err
}

// GetNow 对应 Java: getNow(T valueIfAbsent)
func (f *CompletableFuture[T]) GetNow(valueIfAbsent T) (T, error) {
	if !f.IsDone() {
		return valueIfAbsent, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.value, f.err
}

// ============ Completion Actions ============

func (f *CompletableFuture[T]) Complete(val T) bool {
	// 1. 快速原子检查
	if atomic.LoadInt32(&f.state) == stateDone {
		return false
	}

	f.mu.Lock()
	// 2. 双重检查
	if f.state == stateDone {
		f.mu.Unlock()
		return false
	}

	// 3. 修改状态
	atomic.StoreInt32(&f.state, stateDone)
	f.value = val
	close(f.doneChan)
	cbs := f.callbacks
	f.callbacks = nil
	f.mu.Unlock()

	// 4. 触发回调
	for _, cb := range cbs {
		cb(val, nil)
	}
	return true
}

func (f *CompletableFuture[T]) CompleteExceptionally(err error) bool {
	if atomic.LoadInt32(&f.state) == stateDone {
		return false
	}

	f.mu.Lock()
	if f.state == stateDone {
		f.mu.Unlock()
		return false
	}

	atomic.StoreInt32(&f.state, stateDone)
	f.err = err
	close(f.doneChan)
	cbs := f.callbacks
	f.callbacks = nil
	f.mu.Unlock()

	var zero T
	for _, cb := range cbs {
		cb(zero, err)
	}
	return true
}

// CompleteAsync 对应 Java 9: completeAsync(Supplier)
func (f *CompletableFuture[T]) CompleteAsync(supplier func() T) *CompletableFuture[T] {
	pool.GlobalExecutor.Submit(func() {
		res, err := safecall(func() T { return supplier() })
		if err != nil {
			f.CompleteExceptionally(err)
		} else {
			f.Complete(res)
		}
	})
	return f
}

// ============ Waiting ============

func (f *CompletableFuture[T]) Join() (T, error) {
	<-f.doneChan
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.value, f.err
}

func (f *CompletableFuture[T]) Get(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-f.doneChan:
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.value, f.err
	}
}

// ============ Obtrude (Forced completion) ============

func (f *CompletableFuture[T]) ObtrudeValue(val T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.value = val
	f.err = nil
	atomic.StoreInt32(&f.state, stateDone) // 强制标记完成
	// 注意：Obtrude 不重新触发回调
}

func (f *CompletableFuture[T]) ObtrudeException(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var zero T
	f.value = zero
	f.err = err
	atomic.StoreInt32(&f.state, stateDone)
}

// ============ Internal Logic ============

func (f *CompletableFuture[T]) whenCompleteInternal(cb callback[T]) {
	// 优化：如果已经完成，直接回调，不加锁
	if atomic.LoadInt32(&f.state) == stateDone {
		f.mu.Lock()
		v, e := f.value, f.err
		f.mu.Unlock()
		cb(v, e)
		return
	}

	f.mu.Lock()
	if f.state == stateDone {
		v, e := f.value, f.err
		f.mu.Unlock()
		cb(v, e)
		return
	}
	f.callbacks = append(f.callbacks, cb)
	f.mu.Unlock()
}

func safecall[R any](fn func() R) (result R, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn(), nil
}
