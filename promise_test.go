package promise

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Simulate async operation for testing
func asyncTask(id int, delay time.Duration, shouldFail bool) *Promise[string] {
	return New(func(resolve func(string), reject func(error)) {
		time.Sleep(delay)
		if shouldFail {
			reject(errors.New(fmt.Sprintf("task %d failed", id)))
		} else {
			resolve(fmt.Sprintf("task %d completed", id))
		}
	})
}

func TestBasicPromise(t *testing.T) {
	// Create a successful Promise
	p1 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(100 * time.Millisecond)
		resolve("Hello, Promise!")
	})

	// Use Then to handle results
	p1.Then(func(value string) any {
		return value + " (processed)"
	}, func(err error) any {
		return nil
	})

	// Wait for completion
	result, err := p1.Await()
	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}
	if result != "Hello, Promise!" {
		t.Errorf("Expected result 'Hello, Promise!', but got: %s", result)
	}
}

func TestPromiseChain(t *testing.T) {
	p := New(func(resolve func(int), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		resolve(10)
	})

	// Test chain execution
	chainResult := p.Then(func(value int) any {
		return value * 2
	}, nil).Then(func(value any) any {
		if v, ok := value.(int); ok {
			return v + 5
		}
		return 0
	}, nil)

	result, err := chainResult.Await()
	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}
	expected := 25 // (10 * 2) + 5
	if result != expected {
		t.Errorf("Expected result %d, but got: %d", expected, result)
	}
}

func TestPromiseAll(t *testing.T) {
	promises := []*Promise[string]{
		asyncTask(1, 100*time.Millisecond, false),
		asyncTask(2, 200*time.Millisecond, false),
		asyncTask(3, 150*time.Millisecond, false),
	}

	allPromise := All(promises...)
	results, err := allPromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, but got: %d", len(results))
	}

	// Verify all tasks completed successfully
	for i, result := range results {
		expected := fmt.Sprintf("task %d completed", i+1)
		if result != expected {
			t.Errorf("Expected result '%s', but got: '%s'", expected, result)
		}
	}
}

func TestPromiseAllSettled(t *testing.T) {
	promises := []*Promise[string]{
		asyncTask(1, 100*time.Millisecond, false),
		asyncTask(2, 200*time.Millisecond, true), // This will fail
		asyncTask(3, 150*time.Millisecond, false),
	}

	settledPromise := AllSettled(promises...)
	results, err := settledPromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, but got: %d", len(results))
	}

	// Verify task 2 failed and others succeeded
	if results[1].Error == nil {
		t.Errorf("Expected task 2 to fail")
	}
	if results[0].Error != nil || results[2].Error != nil {
		t.Errorf("Expected tasks 1 and 3 to succeed")
	}
}

func TestPromiseRace(t *testing.T) {
	promises := []*Promise[string]{
		asyncTask(1, 300*time.Millisecond, false),
		asyncTask(2, 100*time.Millisecond, false), // This is the fastest
		asyncTask(3, 200*time.Millisecond, false),
	}

	racePromise := Race(promises...)
	result, err := racePromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	// The fastest task should win
	expected := "task 2 completed"
	if result != expected {
		t.Errorf("Expected result '%s', but got: '%s'", expected, result)
	}
}

func TestPromiseAny(t *testing.T) {
	promises := []*Promise[string]{
		asyncTask(1, 100*time.Millisecond, true),  // Will fail
		asyncTask(2, 200*time.Millisecond, false), // Will succeed
		asyncTask(3, 300*time.Millisecond, true),  // Will fail
	}

	anyPromise := Any(promises...)
	result, err := anyPromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	// Should get the first successful result
	expected := "task 2 completed"
	if result != expected {
		t.Errorf("Expected result '%s', but got: '%s'", expected, result)
	}
}

func TestPromiseTimeout(t *testing.T) {
	// Create a Promise that takes a long time
	slowPromise := New(func(resolve func(string), reject func(error)) {
		time.Sleep(2 * time.Second)
		resolve("Too slow")
	})

	// Set 1 second timeout
	timeoutPromise := Timeout(slowPromise, 1*time.Second)
	_, err := timeoutPromise.Await()

	if err == nil {
		t.Errorf("Expected timeout error, but didn't get one")
	}

	// Verify it's a timeout error
	if !strings.Contains(err.Error(), "Promise timeout") {
		t.Errorf("Expected timeout error message to contain 'Promise timeout', but got: %v", err)
	}

	// Verify it's a PromiseError with TimeoutError type
	if promiseErr, ok := err.(*PromiseError); ok {
		if promiseErr.Type != TimeoutError {
			t.Errorf("Expected TimeoutError type, but got: %v", promiseErr.Type)
		}
	} else {
		t.Errorf("Expected PromiseError type, but got: %T", err)
	}
}

func TestPromiseRetry(t *testing.T) {
	attempts := 0
	fn := func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "retry success", nil
	}

	retryPromise := Retry(fn, 3, 50*time.Millisecond)
	result, err := retryPromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	if result != "retry success" {
		t.Errorf("Expected result 'retry success', but got: '%s'", result)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, but got: %d", attempts)
	}
}

func TestPromiseMap(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	mapPromise := Map(items, func(item int) *Promise[string] {
		return asyncTask(item, 50*time.Millisecond, false)
	})

	results, err := mapPromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, but got: %d", len(results))
	}

	// Verify all items were processed
	for i, result := range results {
		expected := fmt.Sprintf("task %d completed", i+1)
		if result != expected {
			t.Errorf("Expected result '%s', but got: '%s'", expected, result)
		}
	}
}

func TestPromiseReduce(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	reducePromise := Reduce(items, func(acc int, item int) *Promise[int] {
		return New(func(resolve func(int), reject func(error)) {
			time.Sleep(10 * time.Millisecond)
			resolve(acc + item)
		})
	}, 0)

	result, err := reducePromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	expected := 15 // 0 + 1 + 2 + 3 + 4 + 5
	if result != expected {
		t.Errorf("Expected result %d, but got: %d", expected, result)
	}
}

func TestPromiseCatch(t *testing.T) {
	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		reject(errors.New("intentional failure"))
	})

	// Test error handling
	caughtPromise := p.Catch(func(err error) any {
		return "error handled"
	})

	result, err := caughtPromise.Await()
	if err != nil {
		t.Errorf("Expected success after error handling, but got error: %v", err)
	}

	if result != "error handled" {
		t.Errorf("Expected result 'error handled', but got: '%v'", result)
	}
}

func TestPromiseFinally(t *testing.T) {
	finallyCalled := false

	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		resolve("successfully completed")
	})

	rep := p.Finally(func() {
		finallyCalled = true
	})

	result, err := rep.Await()
	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	if !finallyCalled {
		t.Errorf("Finally should have been called")
	}

	if result != "successfully completed" {
		t.Errorf("Expected result 'successfully completed', but got: '%s'", result)
	}
}

// TestPromiseFinallyRejected tests Finally with rejected Promise
func TestPromiseFinallyRejected(t *testing.T) {
	finallyCalled := false

	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		reject(errors.New("test error"))
	})

	rep := p.Finally(func() {
		finallyCalled = true
	})

	_, err := rep.Await()
	if err == nil {
		t.Error("Expected error, but got nil")
	}

	if !finallyCalled {
		t.Error("Finally should have been called even for rejected Promise")
	}
}

// TestPromiseThenRejected tests Then with rejected Promise
func TestPromiseThenRejected(t *testing.T) {
	onRejectedCalled := false
	onFulfilledCalled := false

	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		reject(errors.New("test error"))
	})

	resultPromise := p.Then(
		func(value string) any {
			onFulfilledCalled = true
			return value + " processed"
		},
		func(err error) any {
			onRejectedCalled = true
			return "error handled: " + err.Error()
		},
	)

	result, err := resultPromise.Await()
	if err != nil {
		t.Errorf("Expected success from error handler, but got error: %v", err)
	}

	if onFulfilledCalled {
		t.Error("onFulfilled should not be called for rejected Promise")
	}
	if !onRejectedCalled {
		t.Error("onRejected should be called for rejected Promise")
	}

	expected := "error handled: Promise rejected: test error"
	if result != expected {
		t.Errorf("Expected '%s', but got: '%v'", expected, result)
	}
}

// TestPromiseStateChecks tests the state checking methods
func TestPromiseStateChecks(t *testing.T) {
	// Test pending state
	pendingPromise := New(func(resolve func(string), reject func(error)) {
		// Do nothing, keep it pending
	})

	if !pendingPromise.IsPending() {
		t.Error("New Promise should be pending")
	}
	if pendingPromise.IsFulfilled() {
		t.Error("New Promise should not be fulfilled")
	}
	if pendingPromise.IsRejected() {
		t.Error("New Promise should not be rejected")
	}

	// Test fulfilled state
	fulfilledPromise := New(func(resolve func(string), reject func(error)) {
		resolve("success")
	})

	// Wait for completion
	_, _ = fulfilledPromise.Await()

	if fulfilledPromise.IsPending() {
		t.Error("Resolved Promise should not be pending")
	}
	if !fulfilledPromise.IsFulfilled() {
		t.Error("Resolved Promise should be fulfilled")
	}
	if fulfilledPromise.IsRejected() {
		t.Error("Resolved Promise should not be rejected")
	}

	// Test rejected state
	rejectedPromise := New(func(resolve func(string), reject func(error)) {
		reject(errors.New("error"))
	})

	// Wait for completion
	_, _ = rejectedPromise.Await()

	if rejectedPromise.IsPending() {
		t.Error("Rejected Promise should not be pending")
	}
	if rejectedPromise.IsFulfilled() {
		t.Error("Rejected Promise should not be fulfilled")
	}
	if !rejectedPromise.IsRejected() {
		t.Error("Rejected Promise should be rejected")
	}
}

// TestRejectFunction tests the Reject utility function
func TestRejectFunction(t *testing.T) {
	err := errors.New("test error")
	rejectedPromise := Reject[string](err)

	if rejectedPromise.IsPending() {
		t.Error("Reject should create a non-pending Promise")
	}
	if !rejectedPromise.IsRejected() {
		t.Error("Reject should create a rejected Promise")
	}

	// Test Await returns the error
	_, resultErr := rejectedPromise.Await()
	if resultErr == nil {
		t.Error("Rejected Promise should return error")
	}
	if resultErr.Error() != err.Error() {
		t.Errorf("Expected error %v, got %v", err, resultErr)
	}
}

// TestMicrotaskConfig tests the microtask configuration
func TestMicrotaskConfig(t *testing.T) {
	// Test default config
	defaultConfig := DefaultMicrotaskConfig()
	if defaultConfig.BufferSize != 10000 {
		t.Errorf("Expected default BufferSize 10000, got %d", defaultConfig.BufferSize)
	}
	if defaultConfig.WorkerCount != runtime.NumCPU()*2 {
		t.Errorf("Expected default WorkerCount %d, got %d", runtime.NumCPU()*2, defaultConfig.WorkerCount)
	}

	// Test that we can get the default manager
	defaultMgr := GetDefaultMgr()
	if defaultMgr == nil {
		t.Error("Default manager should not be nil")
	}

	// Test that we can get the current config
	currentConfig := defaultMgr.GetMicrotaskConfig()
	if currentConfig.BufferSize != 10000 {
		t.Errorf("Expected default BufferSize 10000, got %d", currentConfig.BufferSize)
	}
}

// TestPromiseAwaitWithContextTimeout tests AwaitWithContext with timeout
func TestPromiseAwaitWithContextTimeout(t *testing.T) {
	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(200 * time.Millisecond) // Longer than context timeout
		resolve("success")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.AwaitWithContext(ctx)
	if err == nil {
		t.Error("Expected timeout error, but got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, but got: %v", err)
	}
}

// TestGlobalManagerConfiguration tests global manager configuration
func TestGlobalManagerConfiguration(t *testing.T) {
	// Test default manager
	defaultMgr := GetDefaultMgr()
	if defaultMgr == nil {
		t.Error("Default manager should not be nil")
	}

	// Test setting microtask config
	customConfig := &MicrotaskConfig{
		BufferSize:  2000,
		WorkerCount: 4,
	}

	err := defaultMgr.SetMicrotaskConfig(customConfig)
	if err != nil {
		t.Errorf("Failed to set microtask config: %v", err)
	}

	// Verify config was set
	currentConfig := defaultMgr.GetMicrotaskConfig()
	if currentConfig.BufferSize != 2000 {
		t.Errorf("Expected BufferSize 2000, got %d", currentConfig.BufferSize)
	}
	if currentConfig.WorkerCount != 4 {
		t.Errorf("Expected WorkerCount 4, got %d", currentConfig.WorkerCount)
	}

	// Test setting executor workers
	err = defaultMgr.SetExecutorWorker(8)
	if err != nil {
		t.Errorf("Failed to set executor workers: %v", err)
	}

	if defaultMgr.Workers() != 8 {
		t.Errorf("Expected 8 workers, got %d", defaultMgr.Workers())
	}
}

// TestCustomManager tests custom manager functionality
func TestCustomManager(t *testing.T) {
	// Create custom manager
	customConfig := &MicrotaskConfig{
		BufferSize:  1000,
		WorkerCount: 2,
	}

	customMgr := NewPromiseMgrWithConfig(4, customConfig)
	if customMgr == nil {
		t.Error("Custom manager should not be nil")
	}

	// Verify custom config
	config := customMgr.GetMicrotaskConfig()
	if config.BufferSize != 1000 {
		t.Errorf("Expected BufferSize 1000, got %d", config.BufferSize)
	}
	if config.WorkerCount != 2 {
		t.Errorf("Expected WorkerCount 2, got %d", config.WorkerCount)
	}

	// Verify worker count
	if customMgr.Workers() != 4 {
		t.Errorf("Expected 4 workers, got %d", customMgr.Workers())
	}

	// Test Promise creation with custom manager
	p := NewWithMgr(customMgr, func(resolve func(string), reject func(error)) {
		resolve("success")
	})

	if p == nil {
		t.Error("Promise should not be nil")
	}

	// Wait for completion
	result, err := p.Await()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %s", result)
	}

	// Cleanup
	customMgr.Close()
}

// TestManagerIsolation tests that different managers are isolated
func TestManagerIsolation(t *testing.T) {
	// Create two different managers
	mgr1 := NewPromiseMgrWithConfig(2, &MicrotaskConfig{BufferSize: 500, WorkerCount: 1})
	mgr2 := NewPromiseMgrWithConfig(3, &MicrotaskConfig{BufferSize: 1000, WorkerCount: 2})

	// Verify they have different configurations
	if mgr1.Workers() == mgr2.Workers() {
		t.Error("Managers should have different worker counts")
	}

	config1 := mgr1.GetMicrotaskConfig()
	config2 := mgr2.GetMicrotaskConfig()

	if config1.BufferSize == config2.BufferSize {
		t.Error("Managers should have different buffer sizes")
	}

	// Create promises with different managers
	p1 := NewWithMgr(mgr1, func(resolve func(string), reject func(error)) {
		resolve("from mgr1")
	})

	p2 := NewWithMgr(mgr2, func(resolve func(string), reject func(error)) {
		resolve("from mgr2")
	})

	// Both should work independently
	result1, _ := p1.Await()
	result2, _ := p2.Await()

	if result1 != "from mgr1" {
		t.Errorf("Expected 'from mgr1', got %s", result1)
	}
	if result2 != "from mgr2" {
		t.Errorf("Expected 'from mgr2', got %s", result2)
	}

	// Cleanup
	mgr1.Close()
	mgr2.Close()
}

// TestResetDefaultManager tests resetting the default manager
func TestResetDefaultManager(t *testing.T) {
	// Get initial manager
	initialMgr := GetDefaultMgr()

	// Reset with new configuration
	ResetDefaultMgr(6, &MicrotaskConfig{BufferSize: 1500, WorkerCount: 3})

	// Get new manager
	newMgr := GetDefaultMgr()

	// Should be different instance
	if initialMgr == newMgr {
		t.Error("Manager should be reset to new instance")
	}

	// Verify new configuration
	if newMgr.Workers() != 6 {
		t.Errorf("Expected 6 workers, got %d", newMgr.Workers())
	}

	config := newMgr.GetMicrotaskConfig()
	if config.BufferSize != 1500 {
		t.Errorf("Expected BufferSize 1500, got %d", config.BufferSize)
	}
	if config.WorkerCount != 3 {
		t.Errorf("Expected WorkerCount 3, got %d", config.WorkerCount)
	}
}

// TestMicrotaskPanicHandling tests panic handling in microtask callbacks
func TestMicrotaskPanicHandling(t *testing.T) {
	t.Run("Panic in Then callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Add a Then callback that will panic
		nextPromise := promise.Then(func(value string) any {
			panic("intentional panic in Then callback")
		}, nil)

		// The next promise should be rejected due to panic
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if !strings.Contains(err.Error(), "panic in fulfilled callback") {
			t.Errorf("Expected panic error message to contain 'panic in fulfilled callback', but got: %v", err)
		}
	})

	t.Run("Panic in Catch callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("original error"))
		})

		// Add a Catch callback that will panic
		nextPromise := promise.Catch(func(err error) any {
			panic("intentional panic in Catch callback")
		})

		// The next promise should be rejected due to panic
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if !strings.Contains(err.Error(), "panic in error callback") {
			t.Errorf("Expected panic error message to contain 'panic in error callback', but got: %v", err)
		}
	})

	t.Run("Panic in Finally callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Add a Finally callback that will panic
		nextPromise := promise.Finally(func() {
			panic("intentional panic in Finally callback")
		})

		// The next promise should be rejected due to panic
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if !strings.Contains(err.Error(), "panic in finally callback") {
			t.Errorf("Expected panic error message to contain 'panic in finally callback', but got: %v", err)
		}
	})

	t.Run("Error panic in callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Add a Then callback that will panic with an error
		nextPromise := promise.Then(func(value string) any {
			panic(errors.New("custom error panic"))
		}, nil)

		// The next promise should be rejected with the custom error
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if !strings.Contains(err.Error(), "custom error panic") {
			t.Errorf("Expected panic error message to contain 'custom error panic', but got: %v", err)
		}
	})
}

// TestRetryWithContextCancellation tests the cancellation functionality of RetryWithContext
func TestRetryWithContextCancellation(t *testing.T) {
	tests := []struct {
		name           string
		cancelDelay    time.Duration
		expectedError  string
		expectedType   ErrorType
		shouldComplete bool
	}{
		{
			name:           "Cancel before first attempt",
			cancelDelay:    0,
			expectedError:  "Retry cancelled",
			expectedType:   RejectionError,
			shouldComplete: false,
		},
		{
			name:           "Cancel during delay",
			cancelDelay:    50 * time.Millisecond,
			expectedError:  "Retry cancelled during delay",
			expectedType:   RejectionError,
			shouldComplete: false,
		},
		{
			name:           "Complete without cancellation",
			cancelDelay:    0, // Don't cancel, let it complete
			expectedError:  "",
			expectedType:   0,
			shouldComplete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			fn := func() (string, error) {
				attempts++
				if attempts < 3 {
					return "", errors.New("temporary failure")
				}
				return "success", nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start retry operation
			retryPromise := RetryWithContext(ctx, fn, 3, 100*time.Millisecond)

			// Cancel after specified delay
			if tt.cancelDelay > 0 {
				time.AfterFunc(tt.cancelDelay, cancel)
			} else if tt.cancelDelay == 0 && !tt.shouldComplete {
				cancel() // Cancel immediately for cancellation tests
			}
			// For shouldComplete=true, don't cancel

			// Wait for result
			result, err := retryPromise.Await()

			if tt.shouldComplete {
				// Should complete successfully
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if result != "success" {
					t.Errorf("Expected result 'success', but got: %s", result)
				}
			} else {
				// Should be cancelled
				if err == nil {
					t.Error("Expected cancellation error, but got none")
					return
				}

				// Check error type and message
				if promiseErr, ok := err.(*PromiseError); ok {
					if promiseErr.Type != tt.expectedType {
						t.Errorf("Expected error type: %v, but got: %v", tt.expectedType, promiseErr.Type)
					}

					if !strings.Contains(promiseErr.Message, tt.expectedError) {
						t.Errorf("Expected error message to contain: %s, but got: %s", tt.expectedError, promiseErr.Message)
					}

					if promiseErr.Cause == nil {
						t.Error("Expected cause error, but got nil")
					} else if !strings.Contains(promiseErr.Cause.Error(), "context canceled") {
						t.Errorf("Expected cause to be context canceled, but got: %v", promiseErr.Cause)
					}
				} else {
					t.Errorf("Expected PromiseError type, but got: %T", err)
				}
			}
		})
	}
}

// TestRetryWithContextTimeout tests timeout-based cancellation
func TestRetryWithContextTimeout(t *testing.T) {
	attempts := 0
	fn := func() (string, error) {
		attempts++
		if attempts < 5 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// Start retry operation with long delay
	retryPromise := RetryWithContext(ctx, fn, 5, 100*time.Millisecond)

	// Wait for result
	_, err := retryPromise.Await()

	// Should timeout before completion
	if err == nil {
		t.Error("Expected timeout error, but got none")
		return
	}

	// Check error details
	if promiseErr, ok := err.(*PromiseError); ok {
		if promiseErr.Type != RejectionError {
			t.Errorf("Expected error type: %v, but got: %v", RejectionError, promiseErr.Type)
		}

		if !strings.Contains(promiseErr.Message, "Retry cancelled") {
			t.Errorf("Expected error message to contain 'Retry cancelled', but got: %s", promiseErr.Message)
		}

		if promiseErr.Cause == nil {
			t.Error("Expected cause error, but got nil")
		} else if !strings.Contains(promiseErr.Cause.Error(), "deadline exceeded") {
			t.Errorf("Expected cause to be deadline exceeded, but got: %v", promiseErr.Cause)
		}
	} else {
		t.Errorf("Expected PromiseError type, but got: %T", err)
	}

	// Should not complete all attempts due to timeout
	if attempts >= 5 {
		t.Errorf("Expected less than 5 attempts due to timeout, but got: %d", attempts)
	}
}

// TestRetryWithContextImmediateSuccess tests successful completion without cancellation
func TestRetryWithContextImmediateSuccess(t *testing.T) {
	attempts := 0
	fn := func() (string, error) {
		attempts++
		if attempts == 1 {
			return "success on first try", nil
		}
		return "", errors.New("should not reach here")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start retry operation
	retryPromise := RetryWithContext(ctx, fn, 3, 100*time.Millisecond)

	// Wait for result
	result, err := retryPromise.Await()

	// Should complete successfully
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	if result != "success on first try" {
		t.Errorf("Expected result 'success on first try', but got: %s", result)
	}

	// Should only attempt once
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, but got: %d", attempts)
	}
}

// TestNewWithMgrPanicHandling tests panic handling in NewWithMgr executor
func TestNewWithMgrPanicHandling(t *testing.T) {
	t.Run("Error panic in executor", func(t *testing.T) {
		manager := NewPromiseMgr(1)
		defer manager.Close()

		promise := NewWithMgr(manager, func(resolve func(string), reject func(error)) {
			panic(errors.New("custom error panic in executor"))
		})

		_, err := promise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}

		// Verify it's a PromiseError with PanicError type
		if promiseErr, ok := err.(*PromiseError); ok {
			if promiseErr.Type != PanicError {
				t.Errorf("Expected PanicError type, but got: %v", promiseErr.Type)
			}
			if !strings.Contains(promiseErr.Message, "panic in executor") {
				t.Errorf("Expected error message to contain 'panic in executor', but got: %v", promiseErr.Message)
			}
		} else {
			t.Errorf("Expected PromiseError type, but got: %T", err)
		}
	})

	t.Run("Non-error panic in executor", func(t *testing.T) {
		manager := NewPromiseMgr(1)
		defer manager.Close()

		promise := NewWithMgr(manager, func(resolve func(string), reject func(error)) {
			panic("string panic in executor")
		})

		_, err := promise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}

		// Verify it's a PromiseError with PanicError type
		if promiseErr, ok := err.(*PromiseError); ok {
			if promiseErr.Type != PanicError {
				t.Errorf("Expected PanicError type, but got: %v", promiseErr.Type)
			}
			if !strings.Contains(promiseErr.Message, "panic in executor") {
				t.Errorf("Expected error message to contain 'panic in executor', but got: %v", promiseErr.Message)
			}
			if promiseErr.Value != "string panic in executor" {
				t.Errorf("Expected panic value to be 'string panic in executor', but got: %v", promiseErr.Value)
			}
		} else {
			t.Errorf("Expected PromiseError type, but got: %T", err)
		}
	})
}

// TestPromiseCreationAfterManagerShutdown tests that Promise creation fails gracefully when manager is shutdown
func TestPromiseCreationAfterManagerShutdown(t *testing.T) {
	manager := NewPromiseMgr(1)

	// Close the manager first
	manager.Close()

	// Verify manager is shutdown
	if !manager.IsShutdown() {
		t.Error("Expected manager to be shutdown")
	}

	// Try to create a Promise after manager shutdown
	promise := NewWithMgr(manager, func(resolve func(string), reject func(error)) {
		resolve("success")
	})

	// The promise should be rejected immediately
	_, err := promise.Await()
	if err == nil {
		t.Error("Expected error due to manager shutdown, but got none")
	}

	// Verify it's a PromiseError with correct message
	if promiseErr, ok := err.(*PromiseError); ok {
		if promiseErr.Type != RejectionError {
			t.Errorf("Expected RejectionError type, but got: %v", promiseErr.Type)
		}
		if !strings.Contains(promiseErr.Message, "manager is shutdown") {
			t.Errorf("Expected error message to contain 'manager is shutdown', but got: %v", promiseErr.Message)
		}
	} else {
		t.Errorf("Expected PromiseError type, but got: %T", err)
	}
}

// TestWithResolvers tests the WithResolvers function
func TestWithResolvers(t *testing.T) {
	t.Run("Resolve from external", func(t *testing.T) {
		promise, resolve, _ := WithResolvers[string]()

		// Resolve from external code
		go func() {
			time.Sleep(10 * time.Millisecond)
			resolve("external resolve")
		}()

		result, err := promise.Await()
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if result != "external resolve" {
			t.Errorf("Expected 'external resolve', but got: %v", result)
		}
	})

	t.Run("Reject from external", func(t *testing.T) {
		promise, _, reject := WithResolvers[string]()

		// Reject from external code
		go func() {
			time.Sleep(10 * time.Millisecond)
			reject(errors.New("external reject"))
		}()

		_, err := promise.Await()
		if err == nil {
			t.Error("Expected error, but got none")
		}
		if !strings.Contains(err.Error(), "external reject") {
			t.Errorf("Expected error to contain 'external reject', but got: %v", err)
		}
	})

	t.Run("Multiple resolves should only take first", func(t *testing.T) {
		promise, resolve, _ := WithResolvers[string]()

		// Multiple resolves
		go func() {
			time.Sleep(5 * time.Millisecond)
			resolve("first")
		}()
		go func() {
			time.Sleep(10 * time.Millisecond)
			resolve("second")
		}()

		result, err := promise.Await()
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if result != "first" {
			t.Errorf("Expected 'first', but got: %v", result)
		}
	})

	t.Run("Multiple rejects should only take first", func(t *testing.T) {
		promise, _, reject := WithResolvers[string]()

		// Multiple rejects
		go func() {
			time.Sleep(5 * time.Millisecond)
			reject(errors.New("first reject"))
		}()
		go func() {
			time.Sleep(10 * time.Millisecond)
			reject(errors.New("second reject"))
		}()

		_, err := promise.Await()
		if err == nil {
			t.Error("Expected error, but got none")
		}
		if !strings.Contains(err.Error(), "first reject") {
			t.Errorf("Expected error to contain 'first reject', but got: %v", err)
		}
	})
}

// TestWithResolversWithMgr tests the WithResolversWithMgr function
func TestWithResolversWithMgr(t *testing.T) {
	t.Run("With custom manager", func(t *testing.T) {
		manager := NewPromiseMgr(1)
		defer manager.Close()

		promise, resolve, _ := WithResolversWithMgr[string](manager)

		// Resolve from external code
		go func() {
			time.Sleep(10 * time.Millisecond)
			resolve("custom manager resolve")
		}()

		result, err := promise.Await()
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if result != "custom manager resolve" {
			t.Errorf("Expected 'custom manager resolve', but got: %v", result)
		}
	})

	t.Run("With shutdown manager", func(t *testing.T) {
		manager := NewPromiseMgr(1)
		manager.Close()

		promise, resolve, _ := WithResolversWithMgr[string](manager)

		// Try to resolve after manager shutdown
		resolve("should not work")

		_, err := promise.Await()
		if err == nil {
			t.Error("Expected error due to manager shutdown, but got none")
		}
		if !strings.Contains(err.Error(), "manager is shutdown") {
			t.Errorf("Expected error to contain 'manager is shutdown', but got: %v", err)
		}
	})
}

// TestPromisify tests the Promisify function
func TestPromisify(t *testing.T) {
	// Test successful function conversion
	fn := func() (string, error) {
		return "success result", nil
	}

	promiseFn := Promisify(fn)
	promise := promiseFn()
	result, err := promise.Await()

	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if result != "success result" {
		t.Errorf("Expected 'success result', but got: %v", result)
	}

	// Test error function conversion
	errFn := func() (string, error) {
		return "", errors.New("test error")
	}

	errPromiseFn := Promisify(errFn)
	errPromise := errPromiseFn()
	_, err = errPromise.Await()

	if err == nil {
		t.Error("Expected error, but got none")
	}
	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("Expected error to contain 'test error', but got: %v", err)
	}
}
