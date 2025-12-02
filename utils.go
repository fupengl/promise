package promise

import (
	"context"
	"time"
)

func Resolve[T any](value T) *Promise[T] {
	p := &Promise[T]{
		done:    make(chan struct{}),
		manager: GetDefaultMgr(),
	}

	// Use helper functions to set state and value atomically
	p.setState(Fulfilled)
	p.setValue(value)
	close(p.done)
	return p
}

func Reject[T any](err error) *Promise[T] {
	p := &Promise[T]{
		done:    make(chan struct{}),
		manager: GetDefaultMgr(),
	}

	// Use helper functions to set state and error atomically
	p.setState(Rejected)
	p.setError(err)
	close(p.done)
	return p
}

func Delay[T any](value T, delay time.Duration) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		time.Sleep(delay)
		resolve(value)
	})
}

// Timeout creates a Promise with timeout
func Timeout[T any](promise *Promise[T], timeout time.Duration) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		value, err := promise.AwaitWithContext(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				reject(&PromiseError{
					Message: "Promise timeout",
					Cause:   err,
					Type:    TimeoutError,
				})
			} else {
				reject(wrapErrorIfNeeded(err, "Promise operation failed", RejectionError))
			}
			return
		}
		resolve(value)
	})
}

// Retry retries executing a function until success or max retries reached
func Retry[T any](fn func() (T, error), maxRetries int, delay time.Duration) *Promise[T] {
	return RetryWithContext(context.Background(), fn, maxRetries, delay)
}

// RetryWithContext retries executing a function with context support for cancellation
func RetryWithContext[T any](ctx context.Context, fn func() (T, error), maxRetries int, delay time.Duration) *Promise[T] {
	return New(func(resolve func(T), reject func(error)) {
		var lastErr error

		for i := 0; i <= maxRetries; i++ {
			// Check context cancellation before each attempt
			select {
			case <-ctx.Done():
				reject(&PromiseError{
					Message: "Retry cancelled",
					Cause:   ctx.Err(),
					Type:    RejectionError,
				})
				return
			default:
			}

			value, err := fn()
			if err == nil {
				resolve(value)
				return
			}

			lastErr = err
			if i < maxRetries {
				// Use context-aware sleep for delay
				select {
				case <-ctx.Done():
					reject(&PromiseError{
						Message: "Retry cancelled during delay",
						Cause:   ctx.Err(),
						Type:    RejectionError,
					})
					return
				case <-time.After(delay):
				}
			}
		}

		reject(lastErr)
	})
}

// Promisify converts a function that returns (T, error) to a function that returns *Promise[T]
// This is the most common pattern in Go where functions return (value, error)
func Promisify[T any](fn func() (T, error)) func() *Promise[T] {
	return PromisifyWithMgr(GetDefaultMgr(), fn)
}

// PromisifyWithMgr converts a function that returns (T, error) to a function that returns *Promise[T]
// This is the most common pattern in Go where functions return (value, error)
func PromisifyWithMgr[T any](mgr *PromiseMgr, fn func() (T, error)) func() *Promise[T] {
	return func() *Promise[T] {
		return NewWithMgr(mgr, func(resolve func(T), reject func(error)) {
			value, err := fn()
			if err != nil {
				reject(err)
			} else {
				resolve(value)
			}
		})
	}
}

// wrapErrorIfNeeded wraps an error as PromiseError only if it's not already one
// This prevents double wrapping and improves error handling efficiency
func wrapErrorIfNeeded(err error, message string, errorType ErrorType) error {
	if err == nil {
		return &PromiseError{
			Message: message,
			Cause:   nil,
			Type:    errorType,
		}
	}

	// Check if already a PromiseError to avoid double wrapping
	if promiseErr, ok := err.(*PromiseError); ok {
		return promiseErr
	}

	// Wrap as PromiseError
	return &PromiseError{
		Message: message,
		Cause:   err,
		Type:    errorType,
	}
}
