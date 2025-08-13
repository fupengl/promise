package promise

import (
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
		scheduleMicrotask(func() {
			// Simple operation
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

		// Create a chain of 10 promises
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

		_, _ = currentPromise.Await()
	}
}

// BenchmarkConcurrentPromises measures the performance of concurrent promise operations
func BenchmarkConcurrentPromises(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create 100 concurrent promises
		promises := make([]*Promise[string], 100)
		for j := 0; j < 100; j++ {
			promises[j] = New(func(resolve func(string), reject func(error)) {
				time.Sleep(1 * time.Millisecond)
				resolve("concurrent")
			})
		}

		// Wait for all to complete
		_, _ = All(promises...).Await()
	}
}

// BenchmarkPanicHandling measures the performance impact of panic handling
func BenchmarkPanicHandling(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("test")
		})

		// Add a callback that might panic (but doesn't in this case)
		promise.Then(func(value string) any {
			return value + " safe"
		}, nil)

		_, _ = promise.Await()
	}
}
