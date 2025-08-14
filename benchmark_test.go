package promise

import (
	"context"
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

// BenchmarkNormalExecution measures the performance of normal promise execution
func BenchmarkNormalExecution(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("test")
		})

		resultPromise := promise.Then(func(value string) any {
			return value + " processed"
		}, nil)

		result, err := resultPromise.Await()
		_ = result
		_ = err
	}
}

// BenchmarkPanicHandling measures the performance impact of panic handling
func BenchmarkPanicHandling(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("test")
		})

		resultPromise := promise.Then(func(value string) any {
			panic("intentional panic for testing")
		}, nil)

		_, err := resultPromise.Await()
		_ = err
	}
}

// BenchmarkCustomManagerCreation benchmarks custom manager creation
func BenchmarkCustomManagerCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mgr := NewPromiseMgrWithConfig(4, &MicrotaskConfig{
			BufferSize:  1000,
			WorkerCount: 2,
		})
		mgr.Close()
	}
}

// BenchmarkCustomManagerPromise benchmarks Promise creation with custom manager
func BenchmarkCustomManagerPromise(b *testing.B) {
	mgr := NewPromiseMgrWithConfig(4, &MicrotaskConfig{
		BufferSize:  1000,
		WorkerCount: 2,
	})
	defer mgr.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just create the Promise, don't wait for completion
		_ = NewWithMgr(mgr, func(resolve func(string), reject func(error)) {
			resolve("success")
		})
	}
}

// BenchmarkGlobalManagerPromise benchmarks Promise creation with global manager
func BenchmarkGlobalManagerPromise(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just create the Promise, don't wait for completion
		_ = New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})
	}
}

// BenchmarkManagerConfiguration benchmarks manager configuration changes
func BenchmarkManagerConfiguration(b *testing.B) {
	mgr := NewPromiseMgr(4)
	defer mgr.Close()

	config := &MicrotaskConfig{
		BufferSize:  1000,
		WorkerCount: 2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mgr.SetMicrotaskConfig(config)
		_ = mgr.SetExecutorWorker(4)
	}
}

// BenchmarkManagerIsolation benchmarks multiple managers working independently
func BenchmarkManagerIsolation(b *testing.B) {
	mgr1 := NewPromiseMgrWithConfig(2, &MicrotaskConfig{BufferSize: 500, WorkerCount: 1})
	mgr2 := NewPromiseMgrWithConfig(3, &MicrotaskConfig{BufferSize: 1000, WorkerCount: 2})
	defer mgr1.Close()
	defer mgr2.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create promises with different managers concurrently
		p1 := NewWithMgr(mgr1, func(resolve func(string), reject func(error)) {
			resolve("from mgr1")
		})

		p2 := NewWithMgr(mgr2, func(resolve func(string), reject func(error)) {
			resolve("from mgr2")
		})

		// Wait for both to complete with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		result1, err1 := p1.AwaitWithContext(ctx)
		result2, err2 := p2.AwaitWithContext(ctx)
		cancel()

		if err1 != nil {
			b.Errorf("Promise1 failed: %v", err1)
			continue
		}
		if err2 != nil {
			b.Errorf("Promise2 failed: %v", err2)
			continue
		}

		if result1 != "from mgr1" || result2 != "from mgr2" {
			b.Error("Unexpected result")
		}
	}
}

// BenchmarkDefaultManagerReset benchmarks resetting the default manager
func BenchmarkDefaultManagerReset(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResetDefaultMgr(4, &MicrotaskConfig{
			BufferSize:  1000,
			WorkerCount: 2,
		})
	}
}

// BenchmarkPromiseExecutionWithCustomManager benchmarks Promise execution with custom manager
func BenchmarkPromiseExecutionWithCustomManager(b *testing.B) {
	mgr := NewPromiseMgrWithConfig(4, &MicrotaskConfig{
		BufferSize:  1000,
		WorkerCount: 2,
	})
	defer mgr.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewWithMgr(mgr, func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Wait for completion with longer timeout
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := p.AwaitWithContext(ctx)
		cancel()

		if err != nil {
			b.Errorf("Promise failed: %v", err)
		}
	}
}

// BenchmarkPromiseExecutionWithGlobalManager benchmarks Promise execution with global manager
func BenchmarkPromiseExecutionWithGlobalManager(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Wait for completion with longer timeout
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := p.AwaitWithContext(ctx)
		cancel()

		if err != nil {
			b.Errorf("Promise failed: %v", err)
		}
	}
}

// BenchmarkSimplePromiseExecution benchmarks simple Promise execution without complex operations
func BenchmarkSimplePromiseExecution(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Simple wait without timeout
		result, err := p.Await()
		if err != nil {
			b.Errorf("Promise failed: %v", err)
		}
		if result != "success" {
			b.Errorf("Expected 'success', got %s", result)
		}
	}
}

// BenchmarkSimpleCustomManagerPromise benchmarks simple Promise execution with custom manager
func BenchmarkSimpleCustomManagerPromise(b *testing.B) {
	mgr := NewPromiseMgrWithConfig(4, &MicrotaskConfig{
		BufferSize:  1000,
		WorkerCount: 2,
	})
	defer mgr.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewWithMgr(mgr, func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Simple wait without timeout
		result, err := p.Await()
		if err != nil {
			b.Errorf("Promise failed: %v", err)
		}
		if result != "success" {
			b.Errorf("Expected 'success', got %s", result)
		}
	}
}

// BenchmarkPromiseCreationOnly benchmarks only Promise creation without execution
func BenchmarkPromiseCreationOnly(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})
	}
}

// BenchmarkCustomManagerPromiseCreationOnly benchmarks only Promise creation with custom manager
func BenchmarkCustomManagerPromiseCreationOnly(b *testing.B) {
	mgr := NewPromiseMgrWithConfig(4, &MicrotaskConfig{
		BufferSize:  1000,
		WorkerCount: 2,
	})
	defer mgr.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewWithMgr(mgr, func(resolve func(string), reject func(error)) {
			resolve("success")
		})
	}
}
