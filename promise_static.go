package promise

import (
	"errors"
	"sync/atomic"
)

// All waits for all Promises to complete, rejects if any Promise is rejected
func All[T any](promises ...*Promise[T]) *Promise[[]T] {
	return New[[]T](func(resolve func([]T), reject func(error)) {
		if len(promises) == 0 {
			resolve([]T{})
			return
		}

		results := make([]T, len(promises))
		completed := 0
		hasError := false

		for i, p := range promises {
			go func(index int, promise *Promise[T]) {
				value, err := promise.Await()
				if err != nil {
					if !hasError {
						hasError = true
						reject(err)
					}
					return
				}

				results[index] = value
				completed++

				if completed == len(promises) && !hasError {
					resolve(results)
				}
			}(i, p)
		}
	})
}

// AllSettled waits for all Promises to complete, regardless of success or failure
func AllSettled[T any](promises ...*Promise[T]) *Promise[[]Result[T]] {
	return New(func(resolve func([]Result[T]), reject func(error)) {
		if len(promises) == 0 {
			resolve([]Result[T]{})
			return
		}

		results := make([]Result[T], len(promises))
		completed := 0

		for i, p := range promises {
			go func(index int, promise *Promise[T]) {
				value, err := promise.Await()

				results[index] = Result[T]{
					Value: value,
					Error: err,
					Index: index,
				}

				completed++
				if completed == len(promises) {
					resolve(results)
				}
			}(i, p)
		}
	})
}

// Race returns the result of the first completed Promise
func Race[T any](promises ...*Promise[T]) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		if len(promises) == 0 {
			reject(errors.New("no promises provided"))
			return
		}

		// Create a channel to receive the first completed result
		resultChan := make(chan Result[T], 1)
		var completed int32

		// Start all Promises
		for i, p := range promises {
			go func(index int, promise *Promise[T]) {
				value, err := promise.Await()

				// Use atomic operation to ensure only one goroutine can send
				if atomic.CompareAndSwapInt32(&completed, 0, 1) {
					select {
					case resultChan <- Result[T]{Value: value, Error: err, Index: index}:
						// Successfully sent result
					default:
						// This should not happen with atomic check
					}
				}
			}(i, p)
		}

		// Wait for the first result
		result := <-resultChan

		if result.Error != nil {
			reject(result.Error)
		} else {
			resolve(result.Value)
		}
	})
}

// Any returns the result of the first successful Promise, rejects if all fail
func Any[T any](promises ...*Promise[T]) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		if len(promises) == 0 {
			reject(errors.New("no promises provided"))
			return
		}

		errs := make([]error, len(promises))
		completed := 0

		for i, p := range promises {
			go func(index int, promise *Promise[T]) {
				value, err := promise.Await()

				if err == nil {
					// Success, resolve immediately
					resolve(value)
					return
				}

				errs[index] = err
				completed++

				// If all Promises failed
				if completed == len(promises) {
					reject(errors.New("all promises rejected"))
				}
			}(i, p)
		}
	})
}

// Map executes an async function on each element of an array
func Map[T any, R any](items []T, fn func(T) *Promise[R]) *Promise[[]R] {
	if len(items) == 0 {
		return Resolve[[]R]([]R{})
	}

	promises := make([]*Promise[R], len(items))
	for i, item := range items {
		promises[i] = fn(item)
	}

	return All(promises...)
}

// Reduce performs async reduction operation on array elements
func Reduce[T any, R any](items []T, fn func(R, T) *Promise[R], initial R) *Promise[R] {
	if len(items) == 0 {
		return Resolve(initial)
	}

	result := initial
	for _, item := range items {
		promise := fn(result, item)
		value, err := promise.Await()
		if err != nil {
			return Reject[R](err)
		}
		result = value
	}

	return Resolve(result)
}
