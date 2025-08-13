package promise

import (
	"context"
	"errors"
)

// New creates a new Promise
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	p := &Promise[T]{
		state:    Pending,
		handlers: make([]*handler[T], 0),
		done:     make(chan struct{}),
	}

	// Execute executor asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					p.reject(err)
				} else {
					p.reject(errors.New("panic occurred"))
				}
			}
		}()

		executor(p.resolve, p.reject)
	}()

	return p
}

// resolve fulfills the Promise
func (p *Promise[T]) resolve(value T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != Pending {
		return
	}

	p.state = Fulfilled
	p.value = value

	// Signal completion
	close(p.done)

	// Execute all fulfilled handlers using microtask queue
	for _, h := range p.handlers {
		if h.onFulfilled != nil {
			// Capture values to avoid race conditions
			handler := h.onFulfilled
			next := h.next
			val := value

			scheduleMicrotask(func() {
				safeCallback(handler, val, next)
			})
		}
	}
}

// reject rejects the Promise
func (p *Promise[T]) reject(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != Pending {
		return
	}

	p.state = Rejected
	p.err = err

	// Signal completion
	close(p.done)

	// Execute all rejected handlers using microtask queue
	for _, h := range p.handlers {
		if h.onRejected != nil {
			// Capture values to avoid race conditions
			handler := h.onRejected
			next := h.next
			errVal := err

			scheduleMicrotask(func() {
				safeErrorCallback(handler, errVal, next)
			})
		} else if h.next != nil {
			// If no rejected handler, pass error to next Promise
			// This follows JavaScript Promise behavior
			next := h.next
			errVal := err
			scheduleMicrotask(func() {
				next.reject(errVal)
			})
		}
	}
}

// Then adds fulfilled and rejected handlers
func (p *Promise[T]) Then(onFulfilled func(T) any, onRejected func(error) any) *Promise[any] {
	next := &Promise[any]{
		state:    Pending,
		handlers: make([]*handler[any], 0),
		done:     make(chan struct{}),
	}

	h := &handler[T]{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		next:        next,
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// If Promise is already completed, execute handlers using microtask queue
	if p.state == Fulfilled {
		if onFulfilled != nil {
			// Capture values to avoid race conditions
			handler := onFulfilled
			val := p.value

			scheduleMicrotask(func() {
				safeCallback(handler, val, next)
			})
		} else {
			next.resolve(p.value)
		}
	} else if p.state == Rejected {
		if onRejected != nil {
			// Capture values to avoid race conditions
			handler := onRejected
			errVal := p.err

			scheduleMicrotask(func() {
				safeErrorCallback(handler, errVal, next)
			})
		} else {
			next.reject(p.err)
		}
	} else {
		// Promise is still pending, add handler
		p.handlers = append(p.handlers, h)
	}

	return next
}

// Catch adds a rejected handler
func (p *Promise[T]) Catch(onRejected func(error) any) *Promise[any] {
	return p.Then(nil, onRejected)
}

// Finally adds a finally handler
func (p *Promise[T]) Finally(onFinally func()) *Promise[T] {
	next := &Promise[T]{
		state:    Pending,
		handlers: make([]*handler[T], 0),
		done:     make(chan struct{}),
	}

	// Create a handler that will execute the finally callback
	h := &handler[T]{
		onFulfilled: func(value T) any {
			safeFinallyCallback(onFinally, next, value, nil)
			return nil
		},
		onRejected: func(err error) any {
			safeFinallyCallback(onFinally, next, p.value, err)
			return nil
		},
		next: nil, // We don't need to chain to another promise here
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// If Promise is already completed, execute handlers using microtask queue
	if p.state == Fulfilled {
		// Capture values to avoid race conditions
		handler := h.onFulfilled
		val := p.value

		scheduleMicrotask(func() {
			handler(val)
		})
	} else if p.state == Rejected {
		// Capture values to avoid race conditions
		handler := h.onRejected
		errVal := p.err

		scheduleMicrotask(func() {
			handler(errVal)
		})
	} else {
		// Promise is still pending, add handler
		p.handlers = append(p.handlers, h)
	}

	return next
}

// Await waits for Promise completion and returns result
func (p *Promise[T]) Await() (T, error) {
	// Wait for completion signal instead of polling
	<-p.done

	if p.state == Fulfilled {
		return p.value, nil
	}
	var zero T
	return zero, p.err
}

// AwaitWithContext waits for Promise completion using context
func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-p.done:
		// Promise completed
	}

	if p.state == Fulfilled {
		return p.value, nil
	}
	var zero T
	return zero, p.err
}

// State gets the Promise state
func (p *Promise[T]) State() State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// IsPending checks if Promise is in pending state
func (p *Promise[T]) IsPending() bool {
	return p.State() == Pending
}

// IsFulfilled checks if Promise is fulfilled
func (p *Promise[T]) IsFulfilled() bool {
	return p.State() == Fulfilled
}

// IsRejected checks if Promise is rejected
func (p *Promise[T]) IsRejected() bool {
	return p.State() == Rejected
}

// safeCallback wraps a callback function with panic recovery
func safeCallback[T any](callback func(T) any, value T, next *Promise[any]) {
	if next == nil {
		// No next Promise, execute directly without panic handling
		callback(value)
		return
	}

	// Has next Promise, wrap with panic recovery
	defer func() {
		if r := recover(); r != nil {
			var err error
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = errors.New("panic occurred in callback")
			}
			next.reject(err)
		}
	}()

	result := callback(value)
	next.resolve(result)
}

// safeErrorCallback wraps an error callback function with panic recovery
func safeErrorCallback(callback func(error) any, err error, next *Promise[any]) {
	if next == nil {
		// No next Promise, execute directly without panic handling
		callback(err)
		return
	}

	// Has next Promise, wrap with panic recovery
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = errors.New("panic occurred in error callback")
			}
			next.reject(panicErr)
		}
	}()

	result := callback(err)
	next.resolve(result)
}

// safeFinallyCallback wraps a finally callback function with panic recovery
func safeFinallyCallback[T any](callback func(), next *Promise[T], value T, err error) {
	if next == nil {
		// No next Promise, execute directly without panic handling
		callback()
		return
	}

	// Has next Promise, wrap with panic recovery
	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = errors.New("panic occurred in finally callback")
			}
			next.reject(panicErr)
		}
	}()

	callback()

	// After finally callback, resolve/reject based on original state
	if err != nil {
		next.reject(err)
	} else {
		next.resolve(value)
	}
}
