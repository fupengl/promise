package promise

import (
	"context"
	"errors"
	"testing"
	"time"
)

// BenchmarkPromiseCreation measures the performance of creating promises
func BenchmarkPromiseCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(func(resolve func(string), reject func(error)) {
			resolve("test")
		})
	}
}

// BenchmarkPromiseThen measures the performance of chaining promises
func BenchmarkPromiseThen(b *testing.B) {
	promise := New(func(resolve func(string), reject func(error)) {
		resolve("test")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise.Then(func(value string) any {
			return value + " processed"
		}, nil)
	}
}

// BenchmarkPromiseAwait measures the performance of awaiting promises
func BenchmarkPromiseAwait(b *testing.B) {
	promise := New(func(resolve func(string), reject func(error)) {
		resolve("test")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = promise.Await()
	}
}

// BenchmarkMicrotaskQueue measures the performance of microtask scheduling
func BenchmarkMicrotaskQueue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetDefaultMgr().scheduleMicrotask(func() {
			_ = i * 2
		})
	}
}

// BenchmarkPromiseChain measures the performance of long promise chains
func BenchmarkPromiseChain(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise := New(func(resolve func(int), reject func(error)) {
			resolve(0)
		})

		var currentPromise *Promise[any] = promise.Then(func(value int) any {
			return value + 1
		}, nil)

		for j := 1; j < 10; j++ {
			currentPromise = currentPromise.Then(func(value any) any {
				if v, ok := value.(int); ok {
					return v + 1
				}
				return 0
			}, nil)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, _ = currentPromise.AwaitWithContext(ctx)
	}
}

// BenchmarkSimplePromiseChain measures the performance of simple promise chains
func BenchmarkSimplePromiseChain(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise := New(func(resolve func(int), reject func(error)) {
			resolve(0)
		})

		var currentPromise *Promise[any] = promise.Then(func(value int) any {
			return value + 1
		}, nil).Then(func(value any) any {
			if v, ok := value.(int); ok {
				return v + 1
			}
			return 0
		}, nil).Then(func(value any) any {
			if v, ok := value.(int); ok {
				return v + 1
			}
			return 0
		}, nil).Then(func(value any) any {
			if v, ok := value.(int); ok {
				return v + 1
			}
			return 0
		}, nil).Then(func(value any) any {
			if v, ok := value.(int); ok {
				return v + 1
			}
			return 0
		}, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, _ = currentPromise.AwaitWithContext(ctx)
	}
}

// BenchmarkWithResolvers measures the performance of WithResolvers function
func BenchmarkWithResolvers(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise, resolve, _ := WithResolvers[string]()
		resolve("test")
		_, _ = promise.Await()
	}
}

// BenchmarkWithResolversWithMgr measures the performance of WithResolversWithMgr function
func BenchmarkWithResolversWithMgr(b *testing.B) {
	manager := NewPromiseMgr(4)
	defer manager.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise, resolve, _ := WithResolversWithMgr[string](manager)
		resolve("test")
		_, _ = promise.Await()
	}
}

// BenchmarkResolveMultipleTimes measures the performance impact of multiple resolve calls
func BenchmarkResolveMultipleTimes(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise, resolve, _ := WithResolvers[string]()

		// Call resolve multiple times to test the channel check overhead
		resolve("first")
		resolve("second") // This should be ignored
		resolve("third")  // This should be ignored

		_, _ = promise.Await()
	}
}

// BenchmarkRejectMultipleTimes measures the performance impact of multiple reject calls
func BenchmarkRejectMultipleTimes(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise, _, reject := WithResolvers[string]()

		// Call reject multiple times to test the channel check overhead
		reject(errors.New("first"))
		reject(errors.New("second")) // This should be ignored
		reject(errors.New("third"))  // This should be ignored

		_, _ = promise.Await()
	}
}
