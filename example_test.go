package promise

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// Simulate async operation
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
	fmt.Println("=== Testing Basic Promise Functionality ===")

	// Create a successful Promise
	p1 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(100 * time.Millisecond)
		resolve("Hello, Promise!")
	})

	// Use Then to handle results
	p1.Then(func(value string) any {
		fmt.Printf("Success: %s\n", value)
		return value + " (processed)"
	}, func(err error) any {
		fmt.Printf("Failure: %v\n", err)
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
	fmt.Println("\n=== Testing Promise Chain Calls ===")

	p := New(func(resolve func(int), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		resolve(10)
	})

	p.Then(func(value int) any {
		fmt.Printf("Step 1: %d\n", value)
		return value * 2
	}, nil).Then(func(value any) any {
		fmt.Printf("Step 2: %v\n", value)
		return value.(int) + 5
	}, nil).Then(func(value any) any {
		fmt.Printf("Step 3: %v\n", value)
		return value
	}, nil)

	result, err := p.Await()
	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}
	if result != 10 {
		t.Errorf("Expected result 10, but got: %d", result)
	}
}

func TestPromiseAll(t *testing.T) {
	fmt.Println("\n=== Testing Promise.All ===")

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

	fmt.Printf("All results: %v\n", results)
}

func TestPromiseAllSettled(t *testing.T) {
	fmt.Println("\n=== Testing Promise.AllSettled ===")

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

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Task %d failed: %v\n", result.Index, result.Error)
		} else {
			fmt.Printf("Task %d succeeded: %s\n", result.Index, result.Value)
		}
	}
}

func TestPromiseRace(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Race ===")

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

	fmt.Printf("Race winner: %s\n", result)
}

func TestPromiseAny(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Any ===")

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

	fmt.Printf("Any success result: %s\n", result)
}

func TestPromiseTimeout(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Timeout ===")

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

	fmt.Printf("Timeout error: %v\n", err)
}

func TestPromiseRetry(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Retry ===")

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

	fmt.Printf("Retry result: %s (attempts: %d)\n", result, attempts)
}

func TestPromiseMap(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Map ===")

	items := []int{1, 2, 3, 4, 5}

	mapPromise := Map(items, func(item int) *Promise[string] {
		return asyncTask(item, 50*time.Millisecond, false)
	})

	results, err := mapPromise.Await()

	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	fmt.Printf("Map results: %v\n", results)
}

func TestPromiseReduce(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Reduce ===")

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

	fmt.Printf("Reduce result: %d\n", result)
}

func TestPromiseCatch(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Catch ===")

	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		reject(errors.New("intentional failure"))
	})

	p.Catch(func(err error) any {
		fmt.Printf("Caught error: %v\n", err)
		return "error handled"
	})

	_, err := p.Await()
	if err == nil {
		t.Errorf("Expected error, but didn't get one")
	}
}

func TestPromiseFinally(t *testing.T) {
	fmt.Println("\n=== Testing Promise.Finally ===")

	finallyCalled := false

	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		resolve("successfully completed")
	})

	p.Finally(func() {
		finallyCalled = true
		fmt.Println("Finally was called")
	})

	result, err := p.Await()
	if err != nil {
		t.Errorf("Expected success, but got error: %v", err)
	}

	if !finallyCalled {
		t.Errorf("Finally should have been called")
	}

	fmt.Printf("Result: %s\n", result)
}
