# Go-Future Guide

> A complete, accurate usage guide for `github.com/xigexb/go-future`.

This document covers **all public APIs**, their semantics, and real-world usage
patterns. The behavior is intentionally aligned with **Java CompletableFuture (JDK 21/25)**, adapted to Go idioms.

**Important Note on Syntax**:

- **Functions (`future.Func(f, ...)`)**: Used when the operation changes the generic type (e.g., `T` -> `V`). Go methods
  do not support new type parameters.
- **Methods (`f.Method(...)`)**: Used when the operation maintains the same type `T` or returns a fixed type (like
  `struct{}`).

---

## 1. Creating Futures

### 1.1 SupplyAsync (with result)

```go
f := future.SupplyAsync(func () int {
return 42
})
```

With `context.Context`:

```go
ctx := context.Background()
f := future.SupplyAsyncCtx(ctx, func () string {
return "ok"
})
```

With custom executor:

```go
f := future.SupplyAsyncWithExecutor(exec, func () int {
return 100
})
```

---

### 1.2 RunAsync (no result)

Returns `*CompletableFuture[struct{}]`.

```go
future.RunAsync(func () {
fmt.Println("side effect")
})
```

---

### 1.3 Pre-completed futures

```go
f1 := future.CompletedFuture(1)
f2 := future.FailedFuture[int](errors.New("failed"))
```

---

## 2. Waiting for Results

### 2.1 Join (blocking)

Blocks until done. Returns value and error.

```go
value, err := f.Join()
```

### 2.2 Get (context-aware)

Waits with a context (timeout/cancel).

```go
value, err := f.Get(ctx)
```

### 2.3 Non-blocking access

```go
value, err := f.GetNow(-1) // Returns default if not done
```

```go
value := f.ResultNow() // Panics if not completed or exceptional
err := f.ExceptionNow() // Panics if completed normally
```

---

## 3. Transformations (Chaining)

### 3.1 ThenApply (Map)

**Type:** Function
**Signature:** `T -> V`

```go
f1 := future.SupplyAsync(func () int {
return 10
})

// Correct: Use function call for type change
f2 := future.ThenApply(f1, func (v int) string {
return fmt.Sprintf("Value: %d", v * 2)
})
```

Async variants:

```go
future.ThenApplyAsync(f, fn)
future.ThenApplyAsyncWithExecutor(f, exec, fn)
```

---

### 3.2 ThenAccept (Consumer)

**Type:** Method
**Signature:** `T -> void`

```go
f.ThenAccept(func (v int) {
fmt.Println(v)
})
```

---

### 3.3 ThenRun (Runnable)

**Type:** Method
**Signature:** `void -> void`

```go
f.ThenRun(func () {
fmt.Println("done")
})
```

---

## 4. ThenCompose (FlatMap)

Used when the mapping function itself returns a Future.

**Type:** Function

```go
f := future.SupplyAsync(func () int {
return 1
})

// f is *CompletableFuture[int]
// Result is *CompletableFuture[string]
f2 := future.ThenCompose(f, func (v int) *future.CompletableFuture[string] {
// Return a new Future
return future.SupplyAsync(func () string {
return fmt.Sprintf("order-%d", v)
})
})
```

---

## 5. WhenComplete

Observe completion without changing the result.

**Type:** Method

```go
f.WhenComplete(func (v int, err error) {
if err != nil {
log.Println("failed:", err)
}
})
```

---

## 6. Exception Handling

### 6.1 Exceptionally (Recover)

Recovers from an error by returning a fallback value.

**Type:** Method

```go
future.SupplyAsync(func () int {
panic("db down")
}).Exceptionally(func (err error) (int, error) {
return -1, nil // Fallback value
})
```

---

### 6.2 ExceptionallyCompose

Recovers by returning a new Future (fallback via async operation).

**Type:** Method

```go
f.ExceptionallyCompose(func (err error) *future.CompletableFuture[int] {
return future.SupplyAsync(func () int {
return 0 // Async fallback
})
})
```

---

### 6.3 Handle

Always executes regardless of success or failure.

**Type:** Method (Note: In this Go implementation, `Handle` does not change the type `T`)

```go
f.Handle(func (v int, err error) int {
if err != nil {
return -1 // Recover
}
return v // Pass through
})
```

---

## 7. Multiple Futures

### 7.1 AllOf (Void)

Waits for all futures to complete. Fail-fast if any fails.

```go
future.AllOf(f1, f2, f3).Join()
```

---

### 7.2 AnyOf (Race)

Returns the result of the first future to complete.

```go
vFuture := future.AnyOf(f1, f2) // Returns *CompletableFuture[T]
v, err := vFuture.Join()
```

---

## 8. Binary Combinators

### 8.1 ThenCombine (AND)

Combines results of two independent futures.

**Type:** Function

```go
f3 := future.ThenCombine(f1, f2, func (a int, b int) int {
return a + b
})
```

---

### 8.2 ApplyToEither (OR)

Takes the result of whichever future finishes first.

**Type:** Function

```go
f3 := future.ApplyToEither(f1, f2, func (v int) string {
return fmt.Sprintf("winner: %d", v)
})
```

---

## 9. Timeout Control

### 9.1 OrTimeout

**Type:** Method

```go
f.OrTimeout(50 * time.Millisecond)
```

Returns `ErrTimeout` if triggered.

---

### 9.2 CompleteOnTimeout

**Type:** Method

```go
f.CompleteOnTimeout(-1, 50*time.Millisecond)
```

---

## 10. Executors & Thread Pools

### 10.1 Global Executor

By default, async methods use a global `BlockingExecutor` (Limit = CPU * 2).

```go
future.SupplyAsync(fn)
```

### 10.2 Custom Executor

Pass a custom executor (implementing `Submit(func())`) to control concurrency.

```go
exec := pool.NewBlockingExecutor(8)
future.SupplyAsyncWithExecutor(exec, fn)
```

### 10.3 Replace Global Executor

You can replace the default pool at application startup:

```go
pool.SetGlobalExecutor(myCustomPool)
```

---

## 11. Full Example

```go
package main

import (
    "fmt"
    "github.com/xigexb/go-future/future"
)

func main() {
    // 1. Start Async Task
    f1 := future.SupplyAsync(func() int {
        return 10
    })

    // 2. Transform (int -> int) using ThenApply (Function)
    f2 := future.ThenApply(f1, func(v int) int {
        return v * 10
    })

    // 3. Compose (int -> string) using ThenCompose (Function)
    f3 := future.ThenCompose(f2, func(v int) *future.CompletableFuture[string] {
        return future.SupplyAsync(func() string {
            return fmt.Sprintf("Order: %d Shipped", v)
        })
    })

    // 4. Handle Errors & Consume (Methods)
    f3.Exceptionally(func(err error) (string, error) {
        return "Fallback Order", nil
    }).ThenAccept(func(msg string) {
        fmt.Println(msg)
    }).Join()
}
```

---

End of guide.
