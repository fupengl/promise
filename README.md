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
- 🚦 **Concurrency Safe**: Fixed all data race issues, thread-safe operations

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

### External Control with WithResolvers

```go
// Create Promise with external control (fastest method)
promise, resolve, reject := promise.WithResolvers[string]()

// Control Promise state from external code
go func() {
    time.Sleep(100 * time.Millisecond)
    resolve("Hello from external control!")
}()

// Wait for result
result, _ := promise.Await()
fmt.Println(result) // Output: Hello from external control!
```

### Converting Go Functions with Promisify

```go
// Convert a function that returns (T, error) to a Promise function
fetchData := func() (string, error) {
    return "data from API", nil
}

// Convert to Promise function
promiseFn := promise.Promisify(fetchData)

// Execute and get result
result, _ := promiseFn().Await()
fmt.Println(result) // Output: data from API
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

// Create Promise with external control (fastest method)
func WithResolvers[T any]() (*Promise[T], func(T), func(error))

// Create Promise with custom manager and external control
func WithResolversWithMgr[T any](manager *PromiseMgr) (*Promise[T], func(T), func(error))
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

// Convert (T, error) function to Promise function
func Promisify[T any](fn func() (T, error)) func() *Promise[T]

// Array mapping
func Map[T any, R any](items []T, fn func(T) *Promise[R]) *Promise[[]R]

// Array reduction
func Reduce[T any, R any](items []T, fn func(R, T) *Promise[R], initial R) *Promise[R]
```

## 📊 Performance Test Results

### Test Environment
- **CPU**: Apple M2 Max
- **Go Version**: 1.24.2
- **Test Command**: `go test -bench=. -benchmem`

### Benchmark Results

```
BenchmarkPromiseCreation-12              1,278,765 ops    1,013 ns/op    437 B/op    7 allocs/op
BenchmarkPromiseThen-12                  1,918,182 ops      633.1 ns/op    448 B/op    7 allocs/op
BenchmarkPromiseAwait-12                39,064,831 ops      32.87 ns/op      0 B/op    0 allocs/op
BenchmarkMicrotaskQueue-12               3,176,100 ops      370.3 ns/op    134 B/op    2 allocs/op
BenchmarkPromiseChain-12                   171,643 ops   13,399 ns/op    5,096 B/op   72 allocs/op
BenchmarkSimplePromiseChain-12             279,733 ops    6,951 ns/op    2,975 B/op   42 allocs/op
BenchmarkWithResolvers-12                5,637,927 ops      213.1 ns/op    288 B/op    5 allocs/op
BenchmarkWithResolversWithMgr-12         5,736,646 ops      209.4 ns/op    288 B/op    5 allocs/op
BenchmarkResolveMultipleTimes-12         4,016,504 ops      298.4 ns/op    320 B/op    7 allocs/op
BenchmarkRejectMultipleTimes-12          2,891,620 ops      410.3 ns/op    560 B/op   10 allocs/op
BenchmarkMemoryAllocation-12               517,053 ops    2,842 ns/op    1,417 B/op  21 allocs/op
BenchmarkConcurrentPromiseCreation-12     1,000,000 ops    1,201 ns/op      416 B/op    6 allocs/op
BenchmarkTaskPoolReuse-12                1,609,479 ops      758.4 ns/op    440 B/op    8 allocs/op
```

### Performance Analysis

| Operation | Performance | Memory Allocation | Description |
|-----------|-------------|-------------------|-------------|
| **Promise Creation** | 1,013 ns/op | 437 B/op, 7 allocs/op | Basic Promise instance creation |
| **Then Operation** | 633.1 ns/op | 448 B/op, 7 allocs/op | Adding Then callback |
| **Promise Await** | 32.87 ns/op | 0 B/op, 0 allocs/op | Promise await completion |
| **Microtask Scheduling** | 370.3 ns/op | 134 B/op, 2 allocs/op | Microtask queue scheduling |
| **Long Promise Chain (10)** | 13,399 ns/op | 5,096 B/op, 72 allocs/op | 10-level Promise chaining |
| **Simple Promise Chain (5)** | 6,951 ns/op | 2,975 B/op, 42 allocs/op | 5-level Promise chaining |
| **WithResolvers** | 213.1 ns/op | 288 B/op, 5 allocs/op | **Fastest Promise creation method** |
| **WithResolversWithMgr** | 209.4 ns/op | 288 B/op, 5 allocs/op | **Fastest with custom manager** |
| **Memory Allocation Test** | 2,842 ns/op | 1,417 B/op, 21 allocs/op | Complex chaining memory usage |
| **Concurrent Promise Creation** | 1,201 ns/op | 416 B/op, 6 allocs/op | Concurrent creation performance |
| **Task Pool Reuse** | 758.4 ns/op | 440 B/op, 8 allocs/op | Task object pool reuse effect |

### Performance Highlights

- ⭐ **Excellent Promise Await Performance**: Only 32.87 nanoseconds, can handle 30 million operations per second
- ⭐ **Fastest Promise Creation**: WithResolversWithMgr achieves 209.4 ns/op, **4.8x faster** than traditional creation
- ⭐ **Efficient Microtask Scheduling**: 370.3 nanoseconds scheduling time, suitable for high-frequency async operations
- ⭐ **Optimized Memory Usage**: WithResolvers uses only 288 B/op, **34.1% less memory** than traditional creation
- ⭐ **Smooth Chaining Operations**: Each Then operation only takes 633.1 nanoseconds
- ⭐ **Significant Task Pool Reuse Effect**: Performance improvement of about 1.3x through object pool reuse
- ⭐ **Excellent Concurrent Performance**: Stable concurrent creation performance, suitable for high-concurrency scenarios

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

### Race Detection Testing

```bash
go test -race -v
```

## 🔧 Configuration

### Promise Manager Configuration

```go
import "github.com/fupengl/promise"

// Method 1: Configure through global manager
defaultMgr := promise.GetDefaultMgr()
defaultMgr.SetMicrotaskConfig(6, 3000)  // workers, queueSize
defaultMgr.SetExecutorConfig(8, 32)     // workers, queueSize

// Method 2: Create custom manager
customMgr := promise.NewPromiseMgrWithConfig(&promise.PromiseMgrConfig{
    ExecutorWorkers:    4,
    ExecutorQueueSize:  16,
    MicrotaskWorkers:   2,
    MicrotaskQueueSize: 1000,
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

// Configure microtask (workers, queueSize)
defaultMgr.SetMicrotaskConfig(6, 3000)

// Configure executor (workers, queueSize)
defaultMgr.SetExecutorConfig(8, 32)

// Get current configuration
config := defaultMgr.GetConfig()

// Reset default manager configurations
promise.ResetDefaultMgrExecutor(6, 24)      // Reset executor only
promise.ResetDefaultMgrMicrotask(3, 1500)  // Reset microtask only
```

### Configuration Structure

```go
type PromiseMgrConfig struct {
    ExecutorWorkers    int // Number of executor worker goroutines
    ExecutorQueueSize  int // Size of executor task queue
    MicrotaskWorkers   int // Number of microtask worker goroutines
    MicrotaskQueueSize int // Size of microtask queue
}

// Default configuration (automatically calculated based on CPU cores)
func DefaultPromiseMgrConfig() *PromiseMgrConfig
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