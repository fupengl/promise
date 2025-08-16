# Go Promise Library

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 🌍 **Multi-language Support**: English | [中文](README_CN.md)

A high-performance, type-safe Go Promise library inspired by JavaScript Promises.

## ✨ Features

- 🚀 **High Performance**: Based on microtask queue, avoiding goroutine leaks
- 🔒 **Type Safe**: Using Go generics, compile-time type checking
- 🛡️ **Safe & Reliable**: Built-in panic recovery, automatic error propagation
- 🔄 **Chainable**: Support Promise chaining operations
- ⚡ **Concurrency Control**: Provide All, Race, Any and other concurrency methods
- 🎯 **Zero Dependencies**: Pure Go implementation, no external dependencies
- 🎛️ **Flexible Configuration**: Support global and custom Promise managers
- 🔧 **Resource Isolation**: Different managers don't affect each other, support independent configuration

## 📦 Installation

```bash
go get github.com/fupengl/promise
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/fupengl/promise"
)

func main() {
    // Create a Promise
    p := promise.New(func(resolve func(string), reject func(error)) {
        resolve("Hello, Promise!")
    })

    // Chain operations
    result := p.Then(func(value string) any {
        return value + " World!"
    }, nil)

    // Wait for result
    finalValue, _ := result.Await()
    fmt.Println(finalValue) // Output: Hello, Promise! World!
}
```

### Error Handling

```go
p := promise.New(func(resolve func(int), reject func(error)) {
    reject(errors.New("something went wrong"))
})

result, _ := p.Catch(func(err error) any {
    return 42 // Return default value
}).Await()

fmt.Printf("Result: %v\n", result) // Output: Result: 42
```

### Concurrency Control

```go
// Wait for all Promises to complete
promises := []*promise.Promise[string]{
    promise.Delay("First", 100*time.Millisecond),
    promise.Delay("Second", 200*time.Millisecond),
    promise.Delay("Third", 150*time.Millisecond),
}

results, _ := promise.All(promises...).Await()
fmt.Printf("All completed: %v\n", results)
```

## 📚 Core API

### Constructors

```go
// Create new Promise
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T]

// Create fulfilled Promise
func Resolve[T any](value T) *Promise[T]

// Create rejected Promise
func Reject[T any](err error) *Promise[T]
```

### Instance Methods

```go
// Add success/failure handlers
func (p *Promise[T]) Then(onFulfilled func(T) any, onRejected func(error) any) *Promise[any]

// Add error handler
func (p *Promise[T]) Catch(onRejected func(error) any) *Promise[any]

// Add finally handler
func (p *Promise[T]) Finally(onFinally func()) *Promise[T]

// Wait for Promise completion
func (p *Promise[T]) Await() (T, error)

// Wait with context
func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error)
```

### Static Methods

```go
// Wait for all Promises to complete
func All[T any](promises ...*Promise[T]) *Promise[[]T]

// Wait for all Promises to complete (regardless of success/failure)
func AllSettled[T any](promises ...*Promise[T]) *Promise[[]Result[T]]

// Return first completed Promise
func Race[T any](promises ...*Promise[T]) *Promise[T]

// Return first successful Promise
func Any[T any](promises ...*Promise[T]) *Promise[T]
```

### Utility Functions

```go
// Delayed Promise
func Delay[T any](value T, delay time.Duration) *Promise[T]

// Timeout control
func Timeout[T any](promise *Promise[T], timeout time.Duration) *Promise[T]

// Retry mechanism
func Retry[T any](fn func() (T, error), maxRetries int, delay time.Duration) *Promise[T]

// Array mapping
func Map[T any, R any](items []T, fn func(T) *Promise[R]) *Promise[[]R]

// Array reduction
func Reduce[T any, R any](items []T, fn func(R, T) *Promise[R], initial R) *Promise[R]
```

## 📊 Performance Test Results

### Test Environment
- **CPU**: Apple M2 Max
- **Go Version**: 1.21.4
- **Test Command**: `go test -bench=. -benchmem`

### Benchmark Results

```
BenchmarkPromiseCreation-12              2100846               559.3 ns/op           448 B/op          8 allocs/op
BenchmarkPromiseThen-12                  3609886               342.6 ns/op           336 B/op          7 allocs/op
BenchmarkPromiseAwait-12                90184309                14.07 ns/op            0 B/op          0 allocs/op
BenchmarkMicrotaskQueue-12               9050398               130.2 ns/op            24 B/op          2 allocs/op
BenchmarkPromiseChain-12                  152283             14239 ns/op            4227 B/op         72 allocs/op
BenchmarkSimplePromiseChain-12            208448              6225 ns/op            2551 B/op         42 allocs/op
```

### Performance Analysis

| Operation | Performance | Memory Allocation | Description |
|-----------|-------------|-------------------|-------------|
| **Promise Creation** | 559.3 ns/op | 448 B/op | Basic Promise instance creation |
| **Then Operation** | 342.6 ns/op | 336 B/op | Adding Then callback |
| **Promise Await** | 14.07 ns/op | 0 B/op | Promise await completion |
| **Microtask Scheduling** | 130.2 ns/op | 24 B/op | Microtask queue scheduling |
| **Long Promise Chain (10)** | 14,239 ns/op | 4,227 B/op | 10-level Promise chaining |
| **Simple Promise Chain (5)** | 6,225 ns/op | 2,551 B/op | 5-level Promise chaining |

### Performance Highlights

- ⭐ **Excellent Promise Await Performance**: Only 14.07 nanoseconds, can handle 90 million operations per second
- ⭐ **Efficient Microtask Scheduling**: 130.2 nanoseconds scheduling time, suitable for high-frequency async operations
- ⭐ **Reasonable Memory Allocation**: Each Promise is about 448 bytes, controllable memory overhead
- ⭐ **Smooth Chaining Operations**: Each Then operation only takes 342.6 nanoseconds

## 🧪 Testing

### Functional Testing

```bash
go test -v
```

### Example Code

```bash
go test -v -run Example
```

### Performance Testing

```bash
go test -bench=. -benchmem
```

## 🔧 Configuration

### Microtask Queue Configuration

```go
import "github.com/fupengl/promise"

// Configure microtask queue
promise.SetMicrotaskConfig(&promise.MicrotaskConfig{
    BufferSize:  2000,        // Task buffer size
    WorkerCount: 8,           // Worker goroutine count
})
```

### Promise Manager Configuration

```go
import "github.com/fupengl/promise"

// Method 1: Configure through global manager
promise.GetDefaultMgr().SetMicrotaskConfig(&promise.MicrotaskConfig{
    BufferSize:  3000,
    WorkerCount: 6,
})
promise.GetDefaultMgr().SetExecutorWorker(8)

// Method 2: Create custom manager
customMgr := promise.NewPromiseMgrWithConfig(4, &promise.MicrotaskConfig{
    BufferSize:  1000,
    WorkerCount: 2,
})

// Create Promise using custom manager
p := promise.NewWithMgr(customMgr, func(resolve func(string), reject func(error)) {
    resolve("Hello from custom manager!")
})

// Cleanup resources
defer customMgr.Close()
```

### Manager API

```go
// Get global default manager
defaultMgr := promise.GetDefaultMgr()

// Configure microtask
defaultMgr.SetMicrotaskConfig(config)
defaultMgr.GetMicrotaskConfig()

// Configure executor worker count
defaultMgr.SetExecutorWorker(workers)

// Reset default manager
promise.ResetDefaultMgr(workers, microtaskConfig)
```

## 📖 Complete Documentation

- **API Reference**: [Go pkg.dev](https://pkg.go.dev/github.com/fupengl/promise)
- **中文文档**: [README_CN.md](README_CN.md)

## 🤝 Contributing

Issues and Pull Requests are welcome!

### Development Requirements

- Go 1.21+
- Go modules support

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Contact

- GitHub: [@fupengl](https://github.com/fupengl)
- Issues: [GitHub Issues](https://github.com/fupengl/promise/issues)

---

⭐ If this project helps you, please give us a Star!