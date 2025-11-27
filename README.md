# Go-Future ⚡

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-1.18+-blue.svg?style=flat-square" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green.svg?style=flat-square" alt="License"></a>
  <a href="https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/concurrent/CompletableFuture.html"><img src="https://img.shields.io/badge/API-Java%2021%2F25-orange.svg?style=flat-square" alt="Java Parity"></a>
  <a href="#"><img src="https://img.shields.io/badge/coverage-95%25-brightgreen.svg?style=flat-square" alt="Coverage"></a>
</p>

<p align="center">
  <strong>A production-ready, high-performance, zero-dependency `CompletableFuture` implementation for Go.</strong>
  <br>
  一个生产就绪、高性能、零依赖的 Go 语言 `CompletableFuture` 实现。
</p>

---

## 📖 Introduction (简介)

**Go-Future** brings the powerful, fluent asynchronous programming model of Java's `CompletableFuture` to Go.

While Go's `channel` and `goroutine` are powerful primitives, orchestrating complex asynchronous workflows (DAGs) can
still be verbose and error-prone. Go-Future bridges this gap by providing a rich, type-safe, and composable API aligned
with **JDK 21/25** standards.

**Go-Future** 将 Java `CompletableFuture` 强大且流畅的异步编程模型带入了 Go 语言。

虽然 Go 的 `channel` 和 `goroutine` 是强大的原语，但在编排复杂的异步工作流（DAG）时，代码往往会变得冗长且容易出错。Go-Future
通过提供一套与 **JDK 21/25** 标准对齐的、类型安全且可组合的 API，填补了这一空白。

## ✨ Features (特性)

* 🚀 **Full API Parity**: Supports 50+ methods including `SupplyAsync`, `ThenCompose`, `ThenCombine`, `AllOf`, `AnyOf`,
  `Exceptionally`, `ObtrudeValue`, etc.
    * *完全对齐 Java API，支持 50+ 种方法。*
* ⚡ **High Performance**: Built on `sync/atomic` for lock-free state checks. The overhead is sub-microsecond (~400ns).
    * *高性能：基于原子操作的状态管理，额外开销仅为亚微秒级。*
* 🛡️ **Production Ready**: Built-in **Goroutine Pool** (Backpressure protection) and **Panic Recovery**.
    * *生产就绪：内置协程池防止资源耗尽，自动捕获 Panic。*
* 🌐 **Go-Native**: Optimized for Go ecosystem with `Context` propagation (Cancellation & Tracing).
    * *Go 原生优化：支持 Context 传递，完美支持超时控制与链路追踪。*
* 🧩 **Type Safe**: Fully generic code (Go 1.18+).
    * *类型安全：纯泛型实现。*

## 🛠️ Installation (安装)

```bash
go get github.com/xigexb/go-future
```

## 🚀 Quick Start (快速开始)

### Basic Usage (基础用法)

```go
package main

import (
    "fmt"
    "github.com/xigexb/go-future/future"
)

func main() {
    // 1. Async execution
    // 1. 异步执行
    f := future.SupplyAsync(func() int {
        return 10
    })

    // 2. Chaining transformations
    // 2. 链式转换
    f.ThenApply(func(v int) string {
        return fmt.Sprintf("Result: %d", v*2)
    }).ThenAccept(func(s string) {
        fmt.Println(s) // Output: Result: 20
    })

    // 3. Block and wait
    // 3. 阻塞等待
    f.Join()
}
```

### Context & Timeout (上下文与超时)

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

// Support Context for tracing/cancellation
// 支持 Context 用于链路追踪或取消
future.SupplyAsyncCtx(ctx, func () string {
// do something heavy
return "ok"
}).ThenAccept(func (s string) {
fmt.Println(s)
}).Join()
```

## 📚 Documentation (文档)

For detailed usage, patterns, and best practices, please refer to the Guide:
<br>
👉 **[Go-Future Deep Dive / 深度使用指南](docs/guide.md)**

## 📊 Benchmarks (基准测试)

Environment: Intel i9-11900KF @ 3.50GHz, Go 1.20, Windows.

| Benchmark Case         | Time/Op     | Alloc/Op | Description                             |
|:-----------------------|:------------|:---------|:----------------------------------------|
| **Native Goroutine**   | ~69 ns      | 32 B     | Baseline (Physical limit of Go)         |
| **Future SupplyAsync** | **~399 ns** | 408 B    | Includes pool scheduling & context init |
| **Future Chaining**    | **~506 ns** | 840 B    | Full async callback execution           |

> **Conclusion**: The overhead introduced by Go-Future is negligible (**< 0.4µs**) compared to typical I/O operations (
> ms level).
>
> **结论**: 相比原生协程，本库带来的额外开销极低（小于 0.4 微秒），在实际业务中可忽略不计。

## 🤝 Contributing (贡献)

Contributions are welcome! Please feel free to submit a Pull Request.

欢迎提交 Issue 和 PR 参与共建！

## 📄 License

MIT © [xigexb](https://github.com/xigexb) [website](https://www.xigexb.com)
