# Go Promise Library

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 🌍 **多语言支持**: [English](README_EN.md) | 中文

一个高性能、类型安全的Go语言Promise库，参考JavaScript Promise设计。

## ✨ 特性

- 🚀 **高性能**: 基于微任务队列，避免goroutine泄漏
- 🔒 **类型安全**: 使用Go泛型，编译时类型检查
- 🛡️ **安全可靠**: 内置panic恢复，错误自动传播
- 🔄 **链式调用**: 支持Promise链式操作
- ⚡ **并发控制**: 提供All、Race、Any等并发方法
- 🎯 **零依赖**: 纯Go实现，无外部依赖

## 📦 安装

```bash
go get github.com/fupengl/promise
```

## 🚀 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/fupengl/promise"
)

func main() {
    // 创建Promise
    p := promise.New(func(resolve func(string), reject func(error)) {
        resolve("Hello, Promise!")
    })

    // 链式调用
    result := p.Then(func(value string) any {
        return value + " World!"
    }, nil)

    // 等待结果
    finalValue, _ := result.Await()
    fmt.Println(finalValue) // 输出: Hello, Promise! World!
}
```

### 错误处理

```go
p := promise.New(func(resolve func(int), reject func(error)) {
    reject(errors.New("something went wrong"))
})

result, _ := p.Catch(func(err error) any {
    return 42 // 返回默认值
}).Await()

fmt.Printf("Result: %v\n", result) // 输出: Result: 42
```

### 并发控制

```go
// 等待所有Promise完成
promises := []*promise.Promise[string]{
    promise.Delay("First", 100*time.Millisecond),
    promise.Delay("Second", 200*time.Millisecond),
    promise.Delay("Third", 150*time.Millisecond),
}

results, _ := promise.All(promises...).Await()
fmt.Printf("All completed: %v\n", results)
```

## 📚 核心API

### 构造函数

```go
// 创建新Promise
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T]

// 创建已完成的Promise
func Resolve[T any](value T) *Promise[T]

// 创建已拒绝的Promise
func Reject[T any](err error) *Promise[T]
```

### 实例方法

```go
// 添加成功/失败处理函数
func (p *Promise[T]) Then(onFulfilled func(T) any, onRejected func(error) any) *Promise[any]

// 添加错误处理函数
func (p *Promise[T]) Catch(onRejected func(error) any) *Promise[any]

// 添加最终处理函数
func (p *Promise[T]) Finally(onFinally func()) *Promise[T]

// 等待Promise完成
func (p *Promise[T]) Await() (T, error)

// 带上下文的等待
func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error)
```

### 静态方法

```go
// 等待所有Promise完成
func All[T any](promises ...*Promise[T]) *Promise[[]T]

// 等待所有Promise完成（无论成功失败）
func AllSettled[T any](promises ...*Promise[T]) *Promise[[]Result[T]]

// 返回第一个完成的Promise
func Race[T any](promises ...*Promise[T]) *Promise[T]

// 返回第一个成功的Promise
func Any[T any](promises ...*Promise[T]) *Promise[T]
```

### 工具函数

```go
// 延迟Promise
func Delay[T any](value T, delay time.Duration) *Promise[T]

// 超时控制
func Timeout[T any](promise *Promise[T], timeout time.Duration) *Promise[T]

// 重试机制
func Retry[T any](fn func() (T, error), maxRetries int, delay time.Duration) *Promise[T]

// 数组映射
func Map[T any, R any](items []T, fn func(T) *Promise[R]) *Promise[[]R]

// 数组归约
func Reduce[T any, R any](items []T, fn func(R, T) *Promise[R], initial R) *Promise[R]
```

## 📊 性能测试结果

### 测试环境
- **CPU**: Apple M2 Max
- **Go版本**: 1.21.4
- **测试命令**: `go test -bench=. -benchmem`

### 基准测试结果

```
BenchmarkPromiseCreation-12      3109549               350.9 ns/op           288 B/op          5 allocs/op
BenchmarkPromiseThen-12          1856625               646.2 ns/op           440 B/op          8 allocs/op
BenchmarkPromiseAwait-12        100000000               11.34 ns/op            0 B/op          0 allocs/op
BenchmarkMicrotaskQueue-12       3588987               346.4 ns/op           144 B/op          3 allocs/op
BenchmarkPromiseChain-12          250549              4303 ns/op            4687 B/op         74 allocs/op
BenchmarkNormalExecution-12      1325180               907.2 ns/op           759 B/op         13 allocs/op
BenchmarkPanicHandling-12        1000000              1025 ns/op             743 B/op         12 allocs/op
```

### 性能分析

| 操作 | 性能 | 内存分配 | 说明 |
|------|------|----------|------|
| **Promise创建** | 350.9 ns/op | 288 B/op | 基础Promise实例创建 |
| **微任务调度** | 346.4 ns/op | 144 B/op | 微任务队列调度 |
| **Promise链** | 4303 ns/op | 4687 B/op | 10级Promise链式调用 |
| **Then操作** | 646.2 ns/op | 440 B/op | 添加Then回调 |
| **Await等待** | 11.34 ns/op | 0 B/op | 等待已完成的Promise |
| **正常执行** | 907.2 ns/op | 759 B/op | 完整Promise执行流程 |
| **异常处理** | 1025 ns/op | 743 B/op | 包含panic恢复的Promise |



## 🧪 测试

### 功能测试

```bash
go test -v
```

### 示例代码

```bash
go test -v -run Example
```

### 性能测试

```bash
go test -bench=. -benchmem
```

## 🔧 配置

### 微任务队列配置

```go
import "github.com/fupengl/promise"

// 配置微任务队列
promise.SetMicrotaskConfig(&promise.MicrotaskConfig{
    BufferSize:  2000,        // 任务缓冲区大小
    WorkerCount: 8,           // 工作协程数量
})
```

## 📖 完整文档

- **API参考**: [Go pkg.dev](https://pkg.go.dev/github.com/fupengl/promise)

## 🤝 贡献

欢迎提交Issue和Pull Request！

### 开发环境要求

- Go 1.21+
- 支持Go modules

## 📄 许可证

本项目采用MIT许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系方式

- GitHub: [@fupengl](https://github.com/fupengl)
- Issues: [GitHub Issues](https://github.com/fupengl/promise/issues)

---

⭐ 如果这个项目对您有帮助，请给我们一个Star！
