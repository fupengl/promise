# Go Promise Library

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 🌍 **Multi-language Support**: English | [中文](README.md)

A high-performance, type-safe Go Promise library inspired by JavaScript Promises.

## ✨ Features

- 🚀 **High Performance**: Based on microtask queue, avoiding goroutine leaks
- 🔒 **Type Safe**: Using Go generics with compile-time type checking
- 🛡️ **Safe & Reliable**: Built-in panic recovery, automatic error propagation
- 🔄 **Chainable**: Support for Promise chaining operations
- ⚡ **Concurrency Control**: Provides All, Race, Any and other concurrency methods
- 🎯 **Zero Dependencies**: Pure Go implementation, no external dependencies

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
BenchmarkPromiseCreation-12      3109549               350.9 ns/op           288 B/op          5 allocs/op
BenchmarkPromiseThen-12          1856625               646.2 ns/op           440 B/op          8 allocs/op
BenchmarkPromiseAwait-12        100000000               11.34 ns/op            0 B/op          0 allocs/op
BenchmarkMicrotaskQueue-12       3588987               346.4 ns/op           144 B/op          3 allocs/op
BenchmarkPromiseChain-12          250549              4303 ns/op            4687 B/op         74 allocs/op
BenchmarkNormalExecution-12      1325180               907.2 ns/op           759 B/op         13 allocs/op
BenchmarkPanicHandling-12        1000000              1025 ns/op             743 B/op         12 allocs/op
```

### Performance Analysis

| Operation | Performance | Memory Allocation | Description |
|-----------|-------------|-------------------|-------------|
| **Promise Creation** | 350.9 ns/op | 288 B/op | Basic Promise instance creation |
| **Microtask Scheduling** | 346.4 ns/op | 144 B/op | Microtask queue scheduling |
| **Promise Chain** | 4303 ns/op | 4687 B/op | 10-level Promise chaining |
| **Then Operation** | 646.2 ns/op | 440 B/op | Adding Then callback |
| **Await Wait** | 11.34 ns/op | 0 B/op | Waiting for completed Promise |
| **Normal Execution** | 907.2 ns/op | 759 B/op | Complete Promise execution flow |
| **Exception Handling** | 1025 ns/op | 743 B/op | Promise with panic recovery |

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

## 📖 Complete Documentation

- **API Reference**: [Go pkg.dev](https://pkg.go.dev/github.com/fupengl/promise)
- **中文文档**: [README.md](README.md)

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
