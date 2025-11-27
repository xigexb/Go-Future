# Go-Future ⚡

[![Go Version](https://img.shields.io/badge/go-1.20+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)
[![API](https://img.shields.io/badge/API-Java%2021%2F25-orange.svg)](https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/concurrent/CompletableFuture.html)

一个**生产就绪**、**高性能**、**零依赖**的 Go 语言 `CompletableFuture` 实现。

完全对齐 **JDK 21/25** API 标准，包含 `Async` 变体、`Obtrude` 控制、`ResultNow` 状态检查等 50+ 个方法。

## 📚 文档

👉 **[点击查看详细使用教程 (Tutorial)](docs/guide.md)**

## ✨ 特性

- **完整 API**: 1:1 复刻 Java API，包括 `SupplyAsync`, `ThenCompose`, `ThenCombine`, `ApplyToEither`, `ExceptionallyCompose` 等。
- **Sync & Async**: 每个操作都支持同步（高性能）和异步（`*Async` 提交到池）两种模式。
- **类型安全**: 基于 Go 1.18+ 泛型。
- **高性能**: 链式调用开销仅 **~400ns** (比原生 Goroutine 仅多 0.3µs)。
- **生产级防护**: 内置基于信号量的协程池，支持 `Panic` 自动捕获。

## 🚀 快速开始[
]()
```bash
go get [github.com/xigexb/go-future](https://github.com/xigexb/go-future)
```

```go
package main

import (
    "fmt"
    "[github.com/xigexb/go-future/future](https://github.com/xigexb/go-future/future)"
)

func main() {
    // 1. 异步任务
    f1 := future.SupplyAsync(func() int { return 10 })
    
    // 2. 异步链式 (Async 提交到协程池)
    f2 := future.ThenApplyAsync(f1, func(v int) int {
        return v * 2
    })
    
    // 3. 阻塞获取
    val, err := f2.Join()
    if err != nil {
        panic(err)
    }
    fmt.Println(val) // 20
}
```

## 📊 性能基准 (Benchmark)

基于 Intel i9-11900KF @ 3.50GHz 测试：

| 测试场景 | 平均耗时 (ns/op) | 说明 |
| :--- | :--- | :--- |
| **Native Goroutine** | ~69 ns | Go 语言物理极限 |
| **Future SupplyAsync** | **~387 ns** | 包含协程池调度开销 |
| **Future Chaining** | **~498 ns** | 极速回调执行 |

## 🤝 贡献

欢迎提交 Issue 和 PR！