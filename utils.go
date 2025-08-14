package promise

import (
	"context"
	"errors"
	"time"
)

// Resolve creates a fulfilled Promise
func Resolve[T any](value T) *Promise[T] {
	p := &Promise[T]{
		state: Fulfilled,
		value: value,
		done:  make(chan struct{}),
	}
	close(p.done) // Signal completion immediately
	return p
}

// Reject creates a rejected Promise
func Reject[T any](err error) *Promise[T] {
	p := &Promise[T]{
		state: Rejected,
		err:   err,
		done:  make(chan struct{}),
	}
	close(p.done) // Signal completion immediately
	return p
}

// Delay creates a Promise that resolves after a delay
func Delay[T any](value T, delay time.Duration) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		time.Sleep(delay)
		resolve(value)
	})
}

// Timeout creates a Promise with timeout
func Timeout[T any](promise *Promise[T], timeout time.Duration) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Wait for Promise completion or timeout
		value, err := promise.AwaitWithContext(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				reject(errors.New("promise timeout"))
			} else {
				reject(err)
			}
		} else {
			resolve(value)
		}
	})
}

// Retry retries executing a function until success or max retries reached
func Retry[T any](fn func() (T, error), maxRetries int, delay time.Duration) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		var lastErr error

		for i := 0; i <= maxRetries; i++ {
			value, err := fn()
			if err == nil {
				resolve(value)
				return
			}

			lastErr = err
			if i < maxRetries {
				time.Sleep(delay)
			}
		}

		reject(lastErr)
	})
}
