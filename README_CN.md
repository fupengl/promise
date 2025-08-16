# Go Promise Library

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 🌍 **多语言支持**: [English](README.md) | 中文

一个高性能、类型安全的Go语言Promise库，参考JavaScript Promise设计。

## ✨ 特性

- 🚀 **高性能**: 基于微任务队列，避免goroutine泄漏
- 🔒 **类型安全**: 使用Go泛型，编译时类型检查
- 🛡️ **安全可靠**: 内置panic恢复，错误自动传播
- 🔄 **链式调用**: 支持Promise链式操作
- ⚡ **并发控制**: 提供All、Race、Any等并发方法
- 🎯 **零依赖**: 纯Go实现，无外部依赖
- 🎛️ **灵活配置**: 支持全局和自定义Promise管理器
- 🔧 **资源隔离**: 不同管理器之间互不影响，支持独立配置

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

### 使用自定义管理器

```go
package main

import (
    "fmt"
    "github.com/fupengl/promise"
)

func main() {
    // 创建自定义管理器
    customMgr := promise.NewPromiseMgrWithConfig(4, &promise.MicrotaskConfig{
        BufferSize:  1000,
        WorkerCount: 2,
    })
    defer customMgr.Close()

    // 使用自定义管理器创建Promise
    p := promise.NewWithMgr(customMgr, func(resolve func(string), reject func(error)) {
        resolve("Hello from custom manager!")
    })

    // 等待结果
    result, _ := p.Await()
    fmt.Println(result) // 输出: Hello from custom manager!
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

// 使用指定管理器创建Promise
func NewWithMgr[T any](manager *PromiseMgr, executor func(resolve func(T), reject func(error))) *Promise[T]

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

### 管理器函数

```go
// 获取全局默认管理器
func GetDefaultMgr() *PromiseMgr

// 重置默认管理器配置
func ResetDefaultMgr(workers int, microtaskConfig *MicrotaskConfig)

// 创建Promise管理器
func NewPromiseMgr(workers int) *PromiseMgr

// 创建带配置的Promise管理器
func NewPromiseMgrWithConfig(workers int, microtaskConfig *MicrotaskConfig) *PromiseMgr
```

## 📊 性能测试结果

### 测试环境
- **CPU**: Apple M2 Max
- **Go版本**: 1.21.4
- **测试命令**: `go test -bench=. -benchmem`

### 基准测试结果

```
BenchmarkPromiseCreation-12              2100846               559.3 ns/op           448 B/op          8 allocs/op
BenchmarkPromiseThen-12                  3609886               342.6 ns/op           336 B/op          7 allocs/op
BenchmarkPromiseAwait-12                90184309                14.07 ns/op            0 B/op          0 allocs/op
BenchmarkMicrotaskQueue-12               9050398               130.2 ns/op            24 B/op          2 allocs/op
BenchmarkPromiseChain-12                  152283             14239 ns/op            4227 B/op         72 allocs/op
BenchmarkSimplePromiseChain-12            208448              6225 ns/op            2551 B/op         42 allocs/op
```

### 性能分析

| 操作 | 性能 | 内存分配 | 说明 |
|------|------|----------|------|
| **Promise创建** | 559.3 ns/op | 448 B/op | 基础Promise实例创建 |
| **Then操作** | 342.6 ns/op | 336 B/op | 添加Then回调 |
| **Promise等待** | 14.07 ns/op | 0 B/op | Promise等待完成 |
| **微任务调度** | 130.2 ns/op | 24 B/op | 微任务队列调度 |
| **长Promise链(10个)** | 14,239 ns/op | 4,227 B/op | 10级Promise链式调用 |
| **简单Promise链(5个)** | 6,225 ns/op | 2,551 B/op | 5级Promise链式调用 |

### 性能亮点

- ⭐ **Promise等待性能极佳**: 仅需14.07纳秒，每秒可处理9000万次
- ⭐ **微任务调度高效**: 130.2纳秒的调度时间，适合高频异步操作
- ⭐ **内存分配合理**: 每个Promise约448字节，内存开销可控
- ⭐ **链式操作流畅**: 每个Then操作仅需342.6纳秒



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

### Promise管理器配置

```go
import "github.com/fupengl/promise"

// 方式1：通过全局管理器配置
promise.GetDefaultMgr().SetMicrotaskConfig(&promise.MicrotaskConfig{
    BufferSize:  3000,
    WorkerCount: 6,
})
promise.GetDefaultMgr().SetExecutorWorker(8)

// 方式2：创建自定义管理器
customMgr := promise.NewPromiseMgrWithConfig(4, &promise.MicrotaskConfig{
    BufferSize:  1000,
    WorkerCount: 2,
})

// 使用自定义管理器创建Promise
p := promise.NewWithMgr(customMgr, func(resolve func(string), reject func(error)) {
    resolve("Hello from custom manager!")
})

// 清理资源
defer customMgr.Close()
```

### 管理器API

```go
// 获取全局默认管理器
defaultMgr := promise.GetDefaultMgr()

// 配置微任务
defaultMgr.SetMicrotaskConfig(config)
defaultMgr.GetMicrotaskConfig()

// 配置executor worker数量
defaultMgr.SetExecutorWorker(workers)

// 重置默认管理器
promise.ResetDefaultMgr(workers, microtaskConfig)
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
