package promise

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Example functions for Go documentation

func ExampleNew() {
	// Create a new Promise
	p := New(func(resolve func(string), reject func(error)) {
		// Simulate async operation
		resolve("Hello, Promise!")
	})

	// Wait for result
	result, _ := p.Await()
	fmt.Println(result)
	// Output: Hello, Promise!
}

func ExamplePromise_Then() {
	p := New(func(resolve func(int), reject func(error)) {
		resolve(10)
	})

	// Chain promises
	result := p.Then(func(value int) any {
		return value * 2
	}, nil).Then(func(value any) any {
		if v, ok := value.(int); ok {
			return v + 5
		}
		return 0
	}, nil)

	finalResult, _ := result.Await()
	fmt.Println(finalResult)
	// Output: 25
}

func ExamplePromise_Catch() {
	p := New(func(resolve func(string), reject func(error)) {
		reject(errors.New("something went wrong"))
	})

	// Handle error
	result, _ := p.Catch(func(err error) any {
		return "error handled: " + err.Error()
	}).Await()

	fmt.Println(result)
	// Output: error handled: something went wrong
}

func ExamplePromise_Finally() {
	p := New(func(resolve func(string), reject func(error)) {
		resolve("success")
	})

	// Always execute cleanup
	p.Finally(func() {
		fmt.Println("cleanup completed")
	})

	result, _ := p.Await()
	fmt.Println(result)
	// Output: cleanup completed
	// success
}

func ExampleAll() {
	promises := []*Promise[string]{
		Resolve("first"),
		Resolve("second"),
		Resolve("third"),
	}

	results, _ := All(promises...).Await()
	fmt.Println(results)
	// Output: [first second third]
}

func ExampleRace() {
	fast := Delay("fast", 100*time.Millisecond)
	slow := Delay("slow", 500*time.Millisecond)

	result, _ := Race(fast, slow).Await()
	fmt.Println(result)
	// Output: fast
}

func ExampleTimeout() {
	slowPromise := Delay("too slow", 2*time.Second)
	timeoutPromise := Timeout(slowPromise, 1*time.Second)

	_, err := timeoutPromise.Await()
	if err != nil {
		fmt.Println("Promise timeout")
	}
	// Output: Promise timeout
}

func ExampleRetry() {
	attempts := 0
	fn := func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}

	result, _ := Retry(fn, 3, 10*time.Millisecond).Await()
	fmt.Printf("Result: %s (attempts: %d)\n", result, attempts)
	// Output: Result: success (attempts: 3)
}

func ExamplePromise_State() {
	p := New(func(resolve func(string), reject func(error)) {
		resolve("completed")
	})

	// Check initial state
	fmt.Printf("Initial state: %v\n", p.State())

	// Wait for completion
	_, _ = p.Await()

	// Check final state
	fmt.Printf("Final state: %v\n", p.State())
	// Output: Initial state: 0
	// Final state: 1
}

func ExamplePromise_AwaitWithContext() {
	p := New(func(resolve func(string), reject func(error)) {
		time.Sleep(2 * time.Second)
		resolve("completed")
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Wait with context
	_, err := p.AwaitWithContext(ctx)
	if err != nil {
		fmt.Println("Context deadline exceeded")
	}
	// Output: Context deadline exceeded
}

func ExampleMap() {
	items := []int{1, 2, 3, 4, 5}

	// Map each item to a Promise
	mapPromise := Map(items, func(item int) *Promise[string] {
		return Delay(fmt.Sprintf("item_%d", item), 10*time.Millisecond)
	})

	results, _ := mapPromise.Await()
	fmt.Println(results)
	// Output: [item_1 item_2 item_3 item_4 item_5]
}

func ExampleReduce() {
	items := []int{1, 2, 3, 4, 5}

	// Reduce array with async operations
	reducePromise := Reduce(items, func(acc int, item int) *Promise[int] {
		return New(func(resolve func(int), reject func(error)) {
			time.Sleep(5 * time.Millisecond)
			resolve(acc + item)
		})
	}, 0)

	result, _ := reducePromise.Await()
	fmt.Printf("Sum: %d\n", result)
	// Output: Sum: 15
}
