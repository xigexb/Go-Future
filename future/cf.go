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
)

const (
	statePending    int32 = 0
	stateCompleting int32 = 1
	stateDone       int32 = 2
)

// 全局复用的已关闭 Channel (Zero Alloc 的关键)
var closedChan = make(chan struct{})

func init() {
	close(closedChan)
}

type callback[T any] func(val T, err error)

type CompletableFuture[T any] struct {
	state int32

	value T
	err   error

	mu sync.Mutex

	cb0 callback[T]
	cbs []callback[T]

	doneChan chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	_ [8]uint64
}

func New[T any]() *CompletableFuture[T] {
	return NewWithContext[T](context.Background())
}

func NewWithContext[T any](parent context.Context) *CompletableFuture[T] {
	f := &CompletableFuture[T]{
		state: statePending,
	}
	if parent == nil {
		parent = context.Background()
	}
	if parent.Done() == nil {
		f.ctx = parent
	} else {
		f.ctx, f.cancel = context.WithCancel(parent)
	}
	return f
}

// ============ State Inspection ============

func (f *CompletableFuture[T]) IsDone() bool {
	return atomic.LoadInt32(&f.state) == stateDone
}

func (f *CompletableFuture[T]) IsCancelled() bool {
	if !f.IsDone() {
		return false
	}
	return f.err == ErrCanceled
}

func (f *CompletableFuture[T]) IsCompletedExceptionally() bool {
	if !f.IsDone() {
		return false
	}
	return f.err != nil && f.err != ErrCanceled
}

// ============ Result Retrieval ============

func (f *CompletableFuture[T]) ResultNow() T {
	if atomic.LoadInt32(&f.state) != stateDone {
		panic("Future not completed")
	}
	if f.err != nil {
		panic(fmt.Sprintf("Future completed exceptionally: %v", f.err))
	}
	return f.value
}

func (f *CompletableFuture[T]) ExceptionNow() error {
	if atomic.LoadInt32(&f.state) != stateDone {
		panic("Future not completed")
	}
	if f.err == nil {
		panic("Future completed normally")
	}
	return f.err
}

func (f *CompletableFuture[T]) GetNow(valueIfAbsent T) (T, error) {
	if atomic.LoadInt32(&f.state) == stateDone {
		return f.value, f.err
	}
	return valueIfAbsent, nil
}

func (f *CompletableFuture[T]) Join() (T, error) {
	if atomic.LoadInt32(&f.state) == stateDone {
		return f.value, f.err
	}
	<-f.getDoneChanLazy()
	return f.value, f.err
}

func (f *CompletableFuture[T]) Get(ctx context.Context) (T, error) {
	if atomic.LoadInt32(&f.state) == stateDone {
		return f.value, f.err
	}
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-f.getDoneChanLazy():
		return f.value, f.err
	}
}

// getDoneChanLazy 获取等待通道
// 【核心修复】：解决 close of closed channel Panic，并实现 Zero Alloc
func (f *CompletableFuture[T]) getDoneChanLazy() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 如果已经存在（无论是谁创建的），直接返回
	if f.doneChan != nil {
		return f.doneChan
	}

	// 如果当前已经是 Done 状态，且 doneChan 还是 nil
	// 说明 Complete 过程已经结束（或者正在结束且没创建 channel）
	// 此时直接返回全局 closedChan，千万不要赋值给 f.doneChan，避免 finishCompletion 误关
	if atomic.LoadInt32(&f.state) == stateDone {
		return closedChan
	}

	// 只有在 Pending 状态下，才真正创建 channel
	f.doneChan = make(chan struct{})
	return f.doneChan
}

// ============ Completion Actions ============

func (f *CompletableFuture[T]) Complete(val T) bool {
	if !atomic.CompareAndSwapInt32(&f.state, statePending, stateCompleting) {
		return false
	}
	f.value = val
	f.finishCompletion()
	return true
}

func (f *CompletableFuture[T]) CompleteExceptionally(err error) bool {
	if !atomic.CompareAndSwapInt32(&f.state, statePending, stateCompleting) {
		return false
	}
	f.err = err
	f.finishCompletion()
	return true
}

func (f *CompletableFuture[T]) finishCompletion() {
	atomic.StoreInt32(&f.state, stateDone)

	f.mu.Lock()
	// 只有当 channel 确实存在时才关闭
	// 由于 getDoneChanLazy 对 Done 状态不再创建 channel，这里是安全的
	if f.doneChan != nil {
		close(f.doneChan)
	}

	cb0 := f.cb0
	cbs := f.cbs
	f.cb0 = nil
	f.cbs = nil
	f.mu.Unlock()

	if cb0 != nil {
		cb0(f.value, f.err)
	}
	for _, cb := range cbs {
		cb(f.value, f.err)
	}
}

func (f *CompletableFuture[T]) CompleteAsync(supplier func() T) *CompletableFuture[T] {
	return f.CompleteAsyncWithExecutor(nil, supplier)
}

func (f *CompletableFuture[T]) CompleteAsyncWithExecutor(executor pool.Executor, supplier func() T) *CompletableFuture[T] {
	exec := executor
	if exec == nil {
		exec = pool.GlobalExecutor
	}
	exec.Submit(func() {
		res, err := safecall(func() T { return supplier() })
		if err != nil {
			f.CompleteExceptionally(err)
		} else {
			f.Complete(res)
		}
	})
	return f
}

func (f *CompletableFuture[T]) whenCompleteInternal(cb callback[T]) {
	if atomic.LoadInt32(&f.state) == stateDone {
		cb(f.value, f.err)
		return
	}

	f.mu.Lock()
	if atomic.LoadInt32(&f.state) == stateDone {
		f.mu.Unlock()
		cb(f.value, f.err)
		return
	}

	if f.cb0 == nil {
		f.cb0 = cb
	} else {
		if f.cbs == nil {
			f.cbs = make([]callback[T], 0, 2)
		}
		f.cbs = append(f.cbs, cb)
	}
	f.mu.Unlock()
}

func (f *CompletableFuture[T]) Cancel(mayInterruptIfRunning bool) bool {
	if !atomic.CompareAndSwapInt32(&f.state, statePending, stateCompleting) {
		return false
	}
	f.err = ErrCanceled
	if f.cancel != nil {
		f.cancel()
	}
	f.finishCompletion()
	return true
}

func (f *CompletableFuture[T]) ObtrudeValue(val T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.value = val
	f.err = nil
	atomic.StoreInt32(&f.state, stateDone)
}

func (f *CompletableFuture[T]) ObtrudeException(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var zero T
	f.value = zero
	f.err = err
	atomic.StoreInt32(&f.state, stateDone)
}

func safecall[R any](fn func() R) (result R, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn(), nil
}
