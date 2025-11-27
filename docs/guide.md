# Go-Future 深度使用指南 🚀

欢迎使用 **Go-Future**！这是一个完全对齐 **JDK 21/25** 标准的 `CompletableFuture` 实现。

它利用 Go 1.18+ 泛型，提供了**类型安全**、**高性能**（亚微秒级开销）的异步编程体验。

---

## 目录

1. [快速开始](#1-快速开始)
2. [同步 vs 异步 (Sync vs Async)](#2-同步-vs-异步-sync-vs-async)
3. [链式转换](#3-链式转换-mapflatmap)
4. [组合任务 (And/Or/All)](#4-组合任务-andorall)
5. [异常处理](#5-异常处理)
6. [高级控制 (GetNow/Obtrude)](#6-高级控制)

---

## 1. 快速开始

```go
package main

import (
    "fmt"
    "time"
    "github.com/xigexb/go-future/future"
)

func main() {
    // 1. 开启异步任务
    f := future.SupplyAsync(func() int {
        // 模拟耗时任务
        return 100
    })

    // 2. 链式处理
    f.ThenApply(func(v int) string {
        return fmt.Sprintf("Result: %d", v)
    }).ThenAccept(func(s string) {
        fmt.Println(s)
    })

    // 3. 阻塞等待结果
    f.Join()
}
```

---

## 2. 同步 vs 异步 (Sync vs Async)

这是本库与 Java 标准对齐的核心特性。大多数方法都有两个版本：

* **默认版本 (如 `ThenApply`)**:
    * 在**上一个任务完成的线程（Goroutine）**中立即执行。
    * **优点**: 性能极高（无调度开销），适合轻量级转换（如数据计算、字段提取）。
    * **注意**: 避免在里面做阻塞操作，否则会卡住回调链。

* **Async 版本 (如 `ThenApplyAsync`)**:
    * 将任务提交到**全局协程池**中执行。
    * **优点**: 适合耗时操作（I/O、复杂计算），防止阻塞主链路。

```go
// 极快，在回调中直接执行
f.ThenApply(func(v int) int { return v + 1 })

// 提交到池中执行，适合重活
f.ThenApplyAsync(func(v int) int {
    time.Sleep(100 * time.Millisecond)
    return v + 1
})
```

---

## 3. 链式转换 (Map/FlatMap)

### 3.1 ThenApply (一对一转换)
*对应 Java `thenApply`*。

```go
f1 := future.SupplyAsync(func() int { return 10 })

// int -> string
// 注意：Go 泛型限制，类型转换需使用顶层函数 future.ThenApply
f2 := future.ThenApply(f1, func(v int) string {
    return fmt.Sprintf("ID: %d", v)
})
```

### 3.2 ThenCompose (扁平化转换)
*对应 Java `thenCompose`*。当你的回调函数也返回一个 Future 时使用。

```go
future.ThenCompose(f1, func(id int) *future.CompletableFuture[string] {
    // 返回一个新的异步任务
    return future.SupplyAsync(func() string {
        return getUserById(id)
    })
})
```

---

## 4. 组合任务 (And/Or/All)

### 4.1 AllOf (等待所有)
等待所有任务完成。**Fail-Fast 机制**：只要有一个失败，整体立即失败。

```go
f1 := future.SupplyAsync(task1)
f2 := future.SupplyAsync(task2)

future.AllOf(f1, f2).Join()
```

### 4.2 AnyOf (任意一个)
谁先结束（无论成败），就返回谁的结果。

```go
future.AnyOf(f1, f2).ThenAccept(func(v any) {
    fmt.Println("First result:", v)
})
```

### 4.3 ThenCombine (两个都完成)
*对应 Java `thenCombine`*。等待 A 和 B 都完成，合并计算结果。

```go
future.ThenCombine(f1, f2, func(a int, b int) int {
    return a + b
})
```

### 4.4 ApplyToEither (两个竞速)
*对应 Java `applyToEither`*。A 或 B 谁先成功，就用谁的结果进行转换。

```go
future.ApplyToEither(f1, f2, func(v int) string {
    return "Winner: " + strconv.Itoa(v)
})
```

---

## 5. 异常处理

本库支持更符合 Go 习惯的错误处理（支持返回 error）。

### 5.1 Exceptionally (捕获并恢复)

```go
f.Exceptionally(func(err error) (int, error) {
    if isRecoverable(err) {
        return 0, nil // 吞掉错误，返回默认值 0
    }
    return 0, err // 无法恢复，继续抛出错误
})
```

### 5.2 Handle (无论成败)
类似 `finally`，同时获取结果和错误。

```go
f.Handle(func(val int, err error) int {
    if err != nil {
        return -1
    }
    return val
})
```

---

## 6. 高级控制

### 6.1 ResultNow / ExceptionNow (Java 19+)
如果你确定任务已完成，可以非阻塞地直接拿结果。如果没完成会 Panic。

```go
if f.IsDone() {
    val := f.ResultNow() // 直接取值
    fmt.Println(val)
}
```

### 6.2 GetNow
尝试立即获取，没完成则返回默认值。

```go
val, _ := f.GetNow(999) // 如果没做完，返回 999
```

### 6.3 Obtrude (强制写入)
强制修改 Future 的结果（即使它已经完成了）。常用于测试或故障恢复。

```go
f.ObtrudeValue(100) // 强行把结果改为 100
```
