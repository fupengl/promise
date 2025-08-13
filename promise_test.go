package promise

import (
	"errors"
	"fmt"
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
	if err.Error() != "promise timeout" {
		t.Errorf("Expected timeout error, but got: %v", err)
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

	p.Finally(func() {
		finallyCalled = true
	})

	result, err := p.Await()
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
